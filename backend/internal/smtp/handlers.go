package smtp

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
}

func NewHandler(service *Service, authService *auth.Service, pool *pgxpool.Pool) *Handler {
	return &Handler{service: service, authService: authService, pool: pool}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings/smtp", h.get)
	mux.HandleFunc("PUT /api/settings/smtp", h.update)
	mux.HandleFunc("POST /api/settings/smtp/test", h.sendTest)
}

// Unlike branding, SMTP settings are SYSTEMADMIN-only to even read - a
// host and username are infra detail with no reason to be public, and
// the password never comes back in the response regardless of who asks.
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) (*auth.User, error) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can manage email settings")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return nil, err
	}
	return actor, nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authorize(w, r); err != nil {
		return
	}

	settings, err := h.service.Get(r.Context())
	if err != nil {
		log.Printf("getting smtp settings failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to get email settings")
		return
	}
	httpx.JSON(w, http.StatusOK, settings)
}

type updateRequest struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromName  string `json:"fromName"`
	FromEmail string `json:"fromEmail"`
	TestMode  bool   `json:"testMode"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authorize(w, r)
	if err != nil {
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	settings, err := h.service.Update(r.Context(), req.Host, req.Port, req.Username, req.Password, req.FromName, req.FromEmail, req.TestMode)
	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "SMTP_SETTINGS_UPDATED",
			// Deliberately no host/username here either - this is
			// visible to every SYSTEMADMIN via the audit log, and
			// infra detail about a mail server is exactly the kind of
			// thing not to spread into a second place unnecessarily.
			// configured/testMode is the useful, low-sensitivity signal.
			Metadata:  map[string]any{"configured": settings.Host != nil, "test_mode": settings.TestMode},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, settings)

	case errors.Is(err, ErrInvalidPort), errors.Is(err, ErrInvalidFromEmail):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrCredentialKeyNotConfigured):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("updating smtp settings failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to update email settings")
	}
}

func (h *Handler) sendTest(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authorize(w, r)
	if err != nil {
		return
	}

	err = h.service.SendTestEmail(r.Context(), actor.Email)
	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "SMTP_TEST_EMAIL_SENT",
			Metadata:    map[string]any{"to": actor.Email},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent", "to": actor.Email})

	case errors.Is(err, ErrNotConfigured), errors.Is(err, ErrCredentialKeyNotConfigured):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("sending test email failed: %v", err)
		httpx.Error(w, http.StatusBadGateway, "failed to send test email - check your host, port, and credentials")
	}
}

func (h *Handler) logAudit(ctx context.Context, entry audit.Entry) {
	if err := audit.LogStandalone(ctx, h.pool, entry); err != nil {
		log.Printf("failed to write audit log for action %q: %v", entry.Action, err)
	}
}
