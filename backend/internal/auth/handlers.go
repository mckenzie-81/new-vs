package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"backend/internal/audit"
	"backend/internal/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionCookieName = "session"

type Handler struct {
	service      *Service
	pool         *pgxpool.Pool
	cookieSecure bool
	appBaseURL   string
}

func NewHandler(service *Service, pool *pgxpool.Pool, cookieSecure bool, appBaseURL string) *Handler {
	return &Handler{service: service, pool: pool, cookieSecure: cookieSecure, appBaseURL: appBaseURL}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/login", h.login)
	mux.HandleFunc("POST /api/login/totp", h.loginTOTP)
	mux.HandleFunc("POST /api/logout", h.logout)
	mux.HandleFunc("GET /api/session", h.session)
	mux.HandleFunc("POST /api/password-reset/request", h.requestPasswordReset)
	mux.HandleFunc("POST /api/password-reset/confirm", h.confirmPasswordReset)
	mux.HandleFunc("GET /api/auth/totp/status", h.totpStatus)
	mux.HandleFunc("POST /api/auth/totp/setup", h.totpSetup)
	mux.HandleFunc("POST /api/auth/totp/confirm", h.totpConfirm)
	mux.HandleFunc("POST /api/auth/totp/disable", h.totpDisable)
}

// dashboardRoles mirrors the identical private list in internal/election
// and internal/voter - small enough to duplicate rather than export just
// for this one call site. Self-service 2FA is opt-in for every
// dashboard role, not just SYSTEMADMIN.
var dashboardRoles = []string{"SYSTEMADMIN", "ADMIN", "APPROVER"}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	result, err := h.service.Login(r.Context(), req.Email, req.Password, ip, userAgent)

	switch {
	case err == nil && result.TOTPRequired:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &result.User.ID,
			Action:      "LOGIN_AWAITING_TOTP",
			Metadata:    map[string]any{"email": result.User.Email},
			IPAddress:   ip,
			UserAgent:   userAgent,
		})
		httpx.JSON(w, http.StatusOK, map[string]any{
			"status":     "totp_required",
			"loginToken": result.PendingLoginToken,
		})

	case err == nil:
		setSessionCookie(w, result.RawToken, result.ExpiresAt, h.cookieSecure)
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &result.User.ID,
			Action:      "LOGIN_SUCCEEDED",
			Metadata:    map[string]any{"email": result.User.Email},
			IPAddress:   ip,
			UserAgent:   userAgent,
		})
		httpx.JSON(w, http.StatusOK, map[string]any{
			"id":    result.User.ID,
			"name":  result.User.Name,
			"email": result.User.Email,
			"role":  result.User.Role,
		})

	case errors.Is(err, ErrAccountLocked):
		h.logAudit(r.Context(), audit.Entry{
			Action:    "LOGIN_BLOCKED_LOCKED",
			Metadata:  map[string]any{"email_attempted": req.Email},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.Error(w, http.StatusLocked, "Too many failed attempts. Try again in a few minutes.")

	case errors.Is(err, ErrAccountNotVerified):
		httpx.Error(w, http.StatusForbidden, "Please verify your email before signing in.")

	case errors.Is(err, ErrAccountDisabled):
		h.logAudit(r.Context(), audit.Entry{
			Action:    "LOGIN_BLOCKED_DISABLED",
			Metadata:  map[string]any{"email_attempted": req.Email},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.Error(w, http.StatusForbidden, "This account has been disabled.")

	case errors.Is(err, ErrInvalidCredentials):
		h.logAudit(r.Context(), audit.Entry{
			Action:    "LOGIN_FAILED",
			Metadata:  map[string]any{"email_attempted": req.Email},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.Error(w, http.StatusUnauthorized, "Invalid email or password.")

	default:
		log.Printf("login failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to log in")
	}
}

type loginTOTPRequest struct {
	LoginToken string `json:"loginToken"`
	Code       string `json:"code"`
}

func (h *Handler) loginTOTP(w http.ResponseWriter, r *http.Request) {
	var req loginTOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	result, err := h.service.CompleteTOTPLogin(r.Context(), req.LoginToken, req.Code, ip, userAgent)

	switch {
	case err == nil:
		setSessionCookie(w, result.RawToken, result.ExpiresAt, h.cookieSecure)
		action := "LOGIN_SUCCEEDED"
		if result.UsedRecoveryCode {
			action = "LOGIN_SUCCEEDED_VIA_RECOVERY_CODE"
		}
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &result.User.ID,
			Action:      action,
			Metadata:    map[string]any{"email": result.User.Email},
			IPAddress:   ip,
			UserAgent:   userAgent,
		})
		httpx.JSON(w, http.StatusOK, map[string]any{
			"id":               result.User.ID,
			"name":             result.User.Name,
			"email":            result.User.Email,
			"role":             result.User.Role,
			"usedRecoveryCode": result.UsedRecoveryCode,
		})

	case errors.Is(err, ErrTOTPLoginExpired):
		httpx.Error(w, http.StatusUnauthorized, err.Error())

	case errors.Is(err, ErrTOTPLoginLocked):
		h.logAudit(r.Context(), audit.Entry{
			Action:    "LOGIN_TOTP_LOCKED",
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.Error(w, http.StatusLocked, err.Error())

	case errors.Is(err, ErrTOTPInvalidCode):
		h.logAudit(r.Context(), audit.Entry{
			Action:    "LOGIN_TOTP_FAILED",
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.Error(w, http.StatusUnauthorized, err.Error())

	case errors.Is(err, ErrAccountDisabled):
		httpx.Error(w, http.StatusForbidden, "This account has been disabled.")

	case errors.Is(err, ErrAccountNotVerified):
		httpx.Error(w, http.StatusForbidden, "Please verify your email before signing in.")

	default:
		log.Printf("totp login failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to sign in")
	}
}

func (h *Handler) totpStatus(w http.ResponseWriter, r *http.Request) {
	actor, err := h.service.Authorize(w, r, dashboardRoles...)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	status, err := h.service.GetTOTPStatus(r.Context(), actor.ID)
	if err != nil {
		log.Printf("getting totp status failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to check 2FA status")
		return
	}
	httpx.JSON(w, http.StatusOK, status)
}

func (h *Handler) totpSetup(w http.ResponseWriter, r *http.Request) {
	actor, err := h.service.Authorize(w, r, dashboardRoles...)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	setup, err := h.service.BeginTOTPSetup(r.Context(), actor.ID, actor.Email)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, map[string]any{
			"secret":    setup.Secret,
			"uri":       setup.URI,
			"qrCodePng": setup.QRCodePNG,
		})
	case errors.Is(err, ErrTOTPAlreadyEnabled):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		log.Printf("beginning totp setup failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to start 2FA setup")
	}
}

type totpConfirmRequest struct {
	Code string `json:"code"`
}

func (h *Handler) totpConfirm(w http.ResponseWriter, r *http.Request) {
	actor, err := h.service.Authorize(w, r, dashboardRoles...)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req totpConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	recoveryCodes, err := h.service.ConfirmTOTPSetup(r.Context(), actor.ID, req.Code)
	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "TOTP_ENABLED",
			Metadata:    map[string]any{"email": actor.Email},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]any{"recoveryCodes": recoveryCodes})

	case errors.Is(err, ErrTOTPSetupNotFound):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrTOTPAlreadyEnabled):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrTOTPInvalidCode):
		httpx.Error(w, http.StatusUnauthorized, err.Error())

	default:
		log.Printf("confirming totp setup failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to enable 2FA")
	}
}

type totpDisableRequest struct {
	Password string `json:"password"`
}

func (h *Handler) totpDisable(w http.ResponseWriter, r *http.Request) {
	actor, err := h.service.Authorize(w, r, dashboardRoles...)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req totpDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.DisableTOTP(r.Context(), actor.ID, req.Password)
	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "TOTP_DISABLED",
			Metadata:    map[string]any{"email": actor.Email},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "disabled"})

	case errors.Is(err, ErrTOTPNotEnabled):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, "incorrect password")

	default:
		log.Printf("disabling totp failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to disable 2FA")
	}
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	rawToken := readSessionCookie(r)
	if err := h.service.Logout(r.Context(), rawToken); err != nil {
		log.Printf("logout failed: %v", err)
	}
	clearSessionCookie(w, h.cookieSecure)
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	rawToken := readSessionCookie(r)
	user, err := h.service.ValidateSession(r.Context(), rawToken, httpx.ClientIP(r), r.Header.Get("User-Agent"))
	if err != nil {
		clearSessionCookie(w, h.cookieSecure)
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	})
}

func setSessionCookie(w http.ResponseWriter, rawToken string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func readSessionCookie(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

type requestPasswordResetRequest struct {
	Email string `json:"email"`
}

func (h *Handler) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req requestPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	result, err := h.service.RequestPasswordReset(r.Context(), req.Email)
	if err != nil {
		log.Printf("password reset request failed: %v", err)
		// Deliberately still returns the generic response below - a server
		// error here must not behave observably differently from "no such
		// account", or the response code itself becomes a way to probe
		// which emails are registered.
	}

	if result != nil {
		sendPasswordResetEmailStub(h.appBaseURL, result.Email, result.RawToken)
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &result.UserID,
			Action:      "PASSWORD_RESET_REQUESTED",
			Metadata:    map[string]any{"email": result.Email},
			IPAddress:   ip,
			UserAgent:   userAgent,
		})
	}

	// Same response whether or not the email matched an account - the
	// whole point of this endpoint is that it can't be used to tell.
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type confirmPasswordResetRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

func (h *Handler) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req confirmPasswordResetRequest
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

	result, err := h.service.ConfirmPasswordReset(r.Context(), req.Token, req.NewPassword)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &result.UserID,
			Action:      "PASSWORD_RESET_COMPLETED",
			Metadata:    map[string]any{"email": result.Email},
			IPAddress:   ip,
			UserAgent:   userAgent,
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "reset"})

	case errors.Is(err, ErrInvalidResetToken):
		httpx.Error(w, http.StatusBadRequest, "invalid or expired reset link")

	case errors.Is(err, ErrResetTokenExpired):
		httpx.Error(w, http.StatusGone, "this reset link has expired")

	case errors.Is(err, ErrPasswordTooShort):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("password reset confirm failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to reset password")
	}
}

// sendPasswordResetEmailStub stands in for real transactional email, same
// as the bootstrap and invite stubs - it just surfaces the link a real
// email would contain.
func sendPasswordResetEmailStub(appBaseURL, email, rawToken string) {
	link := appBaseURL + "/reset-password?token=" + rawToken
	log.Printf("[EMAIL STUB] password reset link for %s: %s", email, link)
}

func (h *Handler) logAudit(ctx context.Context, entry audit.Entry) {
	if err := audit.LogStandalone(ctx, h.pool, entry); err != nil {
		log.Printf("failed to write audit log for action %q: %v", entry.Action, err)
	}
}
