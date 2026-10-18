package storage

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

var dashboardRoles = []string{"SYSTEMADMIN", "ADMIN", "APPROVER"}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/uploads/presign", h.presign)
}

type presignRequest struct {
	Kind        string `json:"kind"`
	ContentType string `json:"contentType"`
}

// presign requires an authenticated dashboard user, same as the rest of
// the admin-facing API - an upload URL is only ever requested from a
// candidate/branding editing form, never anonymously.
func (h *Handler) presign(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, dashboardRoles...); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req presignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	upload, err := h.service.PresignUpload(r.Context(), Kind(req.Kind), req.ContentType)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, upload)
	case errors.Is(err, ErrNotConfigured):
		httpx.Error(w, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, ErrUnsupportedType):
		httpx.Error(w, http.StatusBadRequest, "unsupported file type - use PNG, JPEG, WEBP, GIF, or SVG")
	default:
		log.Printf("presigning upload failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to prepare upload")
	}
}
