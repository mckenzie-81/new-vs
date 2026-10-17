package election

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

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

var dashboardRoles = []string{"SYSTEMADMIN", "ADMIN", "APPROVER"}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/elections", h.create)
	mux.HandleFunc("GET /api/elections", h.list)
	mux.HandleFunc("GET /api/elections/{id}", h.get)
	mux.HandleFunc("GET /api/elections/{electionId}/portfolios", h.listPortfolios)
	mux.HandleFunc("POST /api/elections/{electionId}/portfolios", h.createPortfolio)
	mux.HandleFunc("DELETE /api/elections/{electionId}/portfolios/{portfolioId}", h.deletePortfolio)
	mux.HandleFunc("POST /api/elections/{electionId}/portfolios/{portfolioId}/candidates", h.createCandidate)
	mux.HandleFunc("DELETE /api/elections/{electionId}/portfolios/{portfolioId}/candidates/{candidateId}", h.deleteCandidate)
}

type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can create elections")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "start time is invalid")
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "end time is invalid")
		return
	}

	ip := httpx.ClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	result, err := h.service.Create(r.Context(), actor.ID, CreateInput{
		Title:       req.Title,
		Description: req.Description,
		StartTime:   startTime,
		EndTime:     endTime,
	})

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_CREATED",
			Metadata: map[string]any{
				"election_id": result.ID,
				"title":       result.Title,
			},
			IPAddress: ip,
			UserAgent: userAgent,
		})
		httpx.JSON(w, http.StatusCreated, result)

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("election creation failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to create election")
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, dashboardRoles...); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	elections, err := h.service.List(r.Context())
	if err != nil {
		log.Printf("listing elections failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to list elections")
		return
	}

	httpx.JSON(w, http.StatusOK, elections)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, dashboardRoles...); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	election, err := h.service.Get(r.Context(), r.PathValue("id"))
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, election)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")
	default:
		log.Printf("getting election failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to get election")
	}
}

func (h *Handler) listPortfolios(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, dashboardRoles...); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	portfolios, err := h.service.ListPortfolios(r.Context(), r.PathValue("electionId"))
	if err != nil {
		log.Printf("listing portfolios failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to list portfolios")
		return
	}

	httpx.JSON(w, http.StatusOK, portfolios)
}

type createPortfolioRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h *Handler) createPortfolio(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can edit the ballot")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	var req createPortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	electionID := r.PathValue("electionId")
	result, err := h.service.CreatePortfolio(r.Context(), electionID, req.Title, req.Description)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "PORTFOLIO_CREATED",
			Metadata: map[string]any{
				"election_id":  electionID,
				"portfolio_id": result.ID,
				"title":        result.Title,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusCreated, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("creating portfolio failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to create portfolio")
	}
}

func (h *Handler) deletePortfolio(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can edit the ballot")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	electionID := r.PathValue("electionId")
	portfolioID := r.PathValue("portfolioId")
	err = h.service.DeletePortfolio(r.Context(), electionID, portfolioID)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "PORTFOLIO_DELETED",
			Metadata: map[string]any{
				"election_id":  electionID,
				"portfolio_id": portfolioID,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such portfolio")

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("deleting portfolio failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to delete portfolio")
	}
}

type createCandidateRequest struct {
	FullName  string `json:"fullName"`
	Manifesto string `json:"manifesto"`
}

func (h *Handler) createCandidate(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can edit the ballot")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	var req createCandidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	electionID := r.PathValue("electionId")
	portfolioID := r.PathValue("portfolioId")
	result, err := h.service.CreateCandidate(r.Context(), electionID, portfolioID, req.FullName, req.Manifesto)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "CANDIDATE_CREATED",
			Metadata: map[string]any{
				"election_id":  electionID,
				"portfolio_id": portfolioID,
				"candidate_id": result.ID,
				"full_name":    result.FullName,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusCreated, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such portfolio")

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("creating candidate failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to create candidate")
	}
}

func (h *Handler) deleteCandidate(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can edit the ballot")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	electionID := r.PathValue("electionId")
	portfolioID := r.PathValue("portfolioId")
	candidateID := r.PathValue("candidateId")
	err = h.service.DeleteCandidate(r.Context(), electionID, portfolioID, candidateID)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "CANDIDATE_DELETED",
			Metadata: map[string]any{
				"election_id":  electionID,
				"portfolio_id": portfolioID,
				"candidate_id": candidateID,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such candidate")

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("deleting candidate failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to delete candidate")
	}
}

func (h *Handler) logAudit(ctx context.Context, entry audit.Entry) {
	if err := audit.Log(ctx, h.pool, entry); err != nil {
		log.Printf("failed to write audit log for action %q: %v", entry.Action, err)
	}
}
