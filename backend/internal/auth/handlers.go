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
}

func NewHandler(service *Service, pool *pgxpool.Pool, cookieSecure bool) *Handler {
	return &Handler{service: service, pool: pool, cookieSecure: cookieSecure}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/login", h.login)
	mux.HandleFunc("POST /api/logout", h.logout)
	mux.HandleFunc("GET /api/session", h.session)
}

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
	case err == nil:
		h.setSessionCookie(w, result.RawToken, result.ExpiresAt)
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

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	rawToken := readSessionCookie(r)
	if err := h.service.Logout(r.Context(), rawToken); err != nil {
		log.Printf("logout failed: %v", err)
	}
	h.clearSessionCookie(w)
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	rawToken := readSessionCookie(r)
	user, err := h.service.ValidateSession(r.Context(), rawToken, httpx.ClientIP(r), r.Header.Get("User-Agent"))
	if err != nil {
		h.clearSessionCookie(w)
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

func (h *Handler) setSessionCookie(w http.ResponseWriter, rawToken string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
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

func (h *Handler) logAudit(ctx context.Context, entry audit.Entry) {
	if err := audit.Log(ctx, h.pool, entry); err != nil {
		log.Printf("failed to write audit log for action %q: %v", entry.Action, err)
	}
}
