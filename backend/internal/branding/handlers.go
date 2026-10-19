package branding

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"backend/internal/auth"
	"backend/internal/httpx"
)

type Handler struct {
	service     *Service
	authService *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings/branding", h.get)
	mux.HandleFunc("PUT /api/settings/branding", h.update)
}

// get is public, no auth - the system's accent color and logo aren't
// sensitive, and the voter-facing pages (which have no dashboard session)
// need to read this as the fallback for an election that hasn't set its
// own branding.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.Get(r.Context())
	if err != nil {
		log.Printf("getting branding settings failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to get branding settings")
		return
	}
	httpx.JSON(w, http.StatusOK, settings)
}

type updateRequest struct {
	AccentColor          string `json:"accentColor"`
	AccentColorSecondary string `json:"accentColorSecondary"`
	AccentColorTertiary  string `json:"accentColorTertiary"`
	LogoURL              string `json:"logoUrl"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	_, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can change branding")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	settings, err := h.service.Update(r.Context(), req.AccentColor, req.AccentColorSecondary, req.AccentColorTertiary, req.LogoURL)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, settings)

	case errors.Is(err, ErrInvalidAccentColor), errors.Is(err, ErrInvalidLogoURL):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("updating branding settings failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to update branding settings")
	}
}
