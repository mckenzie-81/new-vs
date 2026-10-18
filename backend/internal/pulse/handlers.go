package pulse

import (
	"errors"
	"log"
	"net/http"

	"backend/internal/election"
	"backend/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/elections/{id}/pulse", h.getPulse)
}

// getPulse is deliberately the one handler in this codebase with no
// h.authService.Authorize call anywhere in it - that absence is the
// entire point of this route, not an oversight. See the package doc for
// why that's safe: it never touches the real voting tables, only the
// pre-computed snapshot.
func (h *Handler) getPulse(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.GetSnapshot(r.Context(), r.PathValue("id"))
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, snapshot)

	case errors.Is(err, election.ErrNotFound), errors.Is(err, ErrNotPublic):
		// Identical status and body for both cases, on purpose - an
		// anonymous caller shouldn't be able to tell "never existed"
		// apart from "exists, just not public yet".
		httpx.Error(w, http.StatusNotFound, "no such election")

	default:
		log.Printf("getting pulse snapshot failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to get pulse data")
	}
}
