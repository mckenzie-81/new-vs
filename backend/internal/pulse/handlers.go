package pulse

import (
	"errors"
	"log"
	"net/http"

	"backend/internal/auth"
	"backend/internal/election"
	"backend/internal/httpx"
	"backend/internal/voter"
)

type Handler struct {
	service     *Service
	authService *auth.Service
	election    *election.Service
}

func NewHandler(service *Service, authService *auth.Service, electionService *election.Service) *Handler {
	return &Handler{service: service, authService: authService, election: electionService}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/elections/{id}/pulse", h.getPulse)
	mux.HandleFunc("POST /api/elections/{id}/publish-results", h.publishResults)
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

// publishResults uses the same "systemadmin or assigned admin" rule as
// editing the ballot - certifying final standings for public release is at
// least as consequential as building the ballot was.
func (h *Handler) publishResults(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("id")
	if _, err := election.AuthorizeBallotEditor(h.authService, h.election, w, r, electionID); err != nil {
		election.WriteBallotAuthError(w, err)
		return
	}

	results, err := h.service.PublishResults(r.Context(), electionID)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, results)
	case errors.Is(err, election.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")
	case errors.Is(err, voter.ErrResultsNotReady):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		log.Printf("publishing results failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to publish results")
	}
}
