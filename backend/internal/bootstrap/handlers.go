package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"backend/internal/audit"
	"backend/internal/auth"
	"backend/internal/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	service     *Service
	authService *auth.Service
	pool        *pgxpool.Pool
	appBaseURL  string
}

func NewHandler(service *Service, authService *auth.Service, pool *pgxpool.Pool, appBaseURL string) *Handler {
	return &Handler{service: service, authService: authService, pool: pool, appBaseURL: appBaseURL}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/bootstrap/status", h.status)
	mux.HandleFunc("POST /api/bootstrap/initiate", h.initiate)
	mux.HandleFunc("GET /api/bootstrap/confirm", h.confirm)
	mux.HandleFunc("POST /api/invite", h.createInvite)
	mux.HandleFunc("POST /api/invite/accept", h.acceptInvite)
	mux.HandleFunc("GET /api/people", h.listPeople)
	mux.HandleFunc("DELETE /api/people/{id}", h.revokeInvite)
	mux.HandleFunc("POST /api/people/{id}/disable", h.disableAccount)
	mux.HandleFunc("POST /api/people/{id}/reactivate", h.reactivateAccount)
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	available, err := h.service.Available(r.Context())
	if err != nil {
		log.Printf("bootstrap status check failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to check bootstrap status")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"available": available})
}

type initiateRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Secret   string `json:"secret"`
}

func (h *Handler) initiate(w http.ResponseWriter, r *http.Request) {
	var req initiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	result, err := h.service.Initiate(r.Context(), InitiateInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Secret:   req.Secret,
	})

	switch {
	case err == nil:
		// outside the DB transaction: stand-in for real email delivery,
		// which is its own piece we haven't built yet.
		sendVerificationEmailStub(h.appBaseURL, result.Email, result.RawToken)

		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &result.UserID,
			Action:      "BOOTSTRAP_INITIATED",
			Metadata: map[string]any{
				"name":  result.Name,
				"email": result.Email,
			},
			IPAddress: ip,
			UserAgent: userAgent,
		})

		httpx.JSON(w, http.StatusCreated, map[string]string{"status": "pending_verification"})

	case errors.Is(err, ErrAlreadyBootstrapped):
		httpx.Error(w, http.StatusConflict, "bootstrap is no longer available")

	case errors.Is(err, ErrInvalidSecret):
		h.logAudit(r.Context(), audit.Entry{
			Action: "BOOTSTRAP_INITIATE_REJECTED",
			Metadata: map[string]any{
				"reason":          "invalid_secret",
				"email_attempted": req.Email,
			},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.Error(w, http.StatusForbidden, "invalid bootstrap secret")

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("bootstrap initiate failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to initiate bootstrap")
	}
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		httpx.Error(w, http.StatusBadRequest, "token is required")
		return
	}

	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	result, err := h.service.Confirm(r.Context(), token, "")

	switch {
	case err == nil:
		action := "SYSTEMADMIN_BOOTSTRAPPED"
		if result.AlreadyActive {
			// Same token confirmed again (email link scanner, double-click,
			// a retried request) - worth a trace in the audit log, but it
			// must never look like the account was activated twice.
			action = "BOOTSTRAP_CONFIRM_REPEATED"
		}
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &result.UserID,
			Action:      action,
			Metadata: map[string]any{
				"name":  result.Name,
				"email": result.Email,
			},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "active"})

	case errors.Is(err, ErrInvalidToken):
		httpx.Error(w, http.StatusBadRequest, "invalid verification token")

	case errors.Is(err, ErrTokenExpired):
		httpx.Error(w, http.StatusGone, "verification token has expired, please restart setup")

	default:
		log.Printf("bootstrap confirm failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to confirm bootstrap")
	}
}

type createInviteRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *Handler) createInvite(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can send invites")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	var req createInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	result, err := h.service.InviteUser(r.Context(), InviteInput{
		Name:  req.Name,
		Email: req.Email,
		Role:  req.Role,
	})

	switch {
	case err == nil:
		sendInviteEmailStub(h.appBaseURL, result.Email, result.Role, result.RawToken)

		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "INVITE_CREATED",
			Metadata: map[string]any{
				"invited_name":  result.Name,
				"invited_email": result.Email,
				"invited_role":  result.Role,
			},
			IPAddress: ip,
			UserAgent: userAgent,
		})

		httpx.JSON(w, http.StatusCreated, map[string]string{"status": "invited"})

	case errors.Is(err, ErrInvalidRole):
		httpx.Error(w, http.StatusBadRequest, "invalid role")

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrSystemadminCapReached):
		httpx.Error(w, http.StatusConflict, "maximum number of systemadmins already exist")

	case errors.Is(err, ErrEmailInUse):
		httpx.Error(w, http.StatusConflict, "an account with this email already exists")

	default:
		log.Printf("invite creation failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to create invite")
	}
}

type acceptInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	var req acceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" {
		httpx.Error(w, http.StatusBadRequest, "token is required")
		return
	}

	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	result, err := h.service.Confirm(r.Context(), req.Token, req.Password)

	switch {
	case err == nil:
		action := "INVITE_ACCEPTED"
		if result.AlreadyActive {
			action = "INVITE_ACCEPT_REPEATED"
		}
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &result.UserID,
			Action:      action,
			Metadata: map[string]any{
				"name":  result.Name,
				"email": result.Email,
				"role":  result.Role,
			},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "active"})

	case errors.Is(err, ErrInvalidToken):
		httpx.Error(w, http.StatusBadRequest, "invalid invite token")

	case errors.Is(err, ErrTokenExpired):
		httpx.Error(w, http.StatusGone, "this invite has expired")

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("invite accept failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to accept invite")
	}
}

func (h *Handler) listPeople(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, "SYSTEMADMIN"); err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "not authorized")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	people, err := h.service.ListPeople(r.Context())
	if err != nil {
		log.Printf("listing people failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to list people")
		return
	}

	httpx.JSON(w, http.StatusOK, people)
}

func (h *Handler) revokeInvite(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "not authorized")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	id := r.PathValue("id")
	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	revoked, err := h.service.RevokeInvite(r.Context(), id)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "INVITE_REVOKED",
			Metadata: map[string]any{
				"revoked_name":  revoked.Name,
				"revoked_email": revoked.Email,
				"revoked_role":  revoked.Role,
			},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})

	case errors.Is(err, ErrPersonNotFound):
		httpx.Error(w, http.StatusNotFound, "no such person")

	case errors.Is(err, ErrNotPending):
		httpx.Error(w, http.StatusConflict, "only a pending invite can be revoked")

	default:
		log.Printf("invite revoke failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to revoke invite")
	}
}

func (h *Handler) disableAccount(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "not authorized")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	id := r.PathValue("id")
	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	disabled, err := h.service.DisableAccount(r.Context(), actor.ID, id)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ACCOUNT_DISABLED",
			Metadata: map[string]any{
				"disabled_name":  disabled.Name,
				"disabled_email": disabled.Email,
				"disabled_role":  disabled.Role,
			},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "disabled"})

	case errors.Is(err, ErrCannotDisableSelf):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrPersonNotFound):
		httpx.Error(w, http.StatusNotFound, "no such person")

	case errors.Is(err, ErrAccountNotActive):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrLastActiveSystemadmin):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("account disable failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to disable account")
	}
}

func (h *Handler) reactivateAccount(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "not authorized")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	id := r.PathValue("id")
	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	reactivated, err := h.service.ReactivateAccount(r.Context(), id)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ACCOUNT_REACTIVATED",
			Metadata: map[string]any{
				"reactivated_name":  reactivated.Name,
				"reactivated_email": reactivated.Email,
				"reactivated_role":  reactivated.Role,
			},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "active"})

	case errors.Is(err, ErrPersonNotFound):
		httpx.Error(w, http.StatusNotFound, "no such person")

	case errors.Is(err, ErrAccountNotDisabled):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrSystemadminCapReached):
		httpx.Error(w, http.StatusConflict, "maximum number of systemadmins already exist")

	default:
		log.Printf("account reactivate failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to reactivate account")
	}
}

func (h *Handler) logAudit(ctx context.Context, entry audit.Entry) {
	if err := audit.Log(ctx, h.pool, entry); err != nil {
		log.Printf("failed to write audit log for action %q: %v", entry.Action, err)
	}
}

// sendVerificationEmailStub stands in for real transactional email, which is
// a separate piece of the system not yet designed. For now it just surfaces
// the link a real email would contain.
func sendVerificationEmailStub(appBaseURL, email, rawToken string) {
	link := appBaseURL + "/bootstrap/confirm?token=" + rawToken
	log.Printf("[EMAIL STUB] verification link for %s: %s", email, link)
}

func sendInviteEmailStub(appBaseURL, email, role, rawToken string) {
	link := appBaseURL + "/invite/accept?token=" + rawToken
	log.Printf("[EMAIL STUB] invite link for %s (%s): %s", email, role, link)
}
