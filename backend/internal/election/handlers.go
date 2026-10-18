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
	mux.HandleFunc("PUT /api/elections/{id}", h.updateElection)
	mux.HandleFunc("GET /api/elections/{id}/electorate", h.getElectorateSummary)
	mux.HandleFunc("GET /api/elections/{electionId}/portfolios", h.listPortfolios)
	mux.HandleFunc("POST /api/elections/{electionId}/portfolios", h.createPortfolio)
	mux.HandleFunc("PUT /api/elections/{electionId}/portfolios/order", h.reorderPortfolios)
	mux.HandleFunc("DELETE /api/elections/{electionId}/portfolios/{portfolioId}", h.deletePortfolio)
	mux.HandleFunc("POST /api/elections/{electionId}/portfolios/{portfolioId}/candidates", h.createCandidate)
	mux.HandleFunc("PUT /api/elections/{electionId}/portfolios/{portfolioId}/candidates/order", h.reorderCandidates)
	mux.HandleFunc("PUT /api/elections/{electionId}/portfolios/{portfolioId}/candidates/{candidateId}", h.updateCandidate)
	mux.HandleFunc("DELETE /api/elections/{electionId}/portfolios/{portfolioId}/candidates/{candidateId}", h.deleteCandidate)
	mux.HandleFunc("GET /api/elections/{electionId}/admins", h.listAssignedAdmins)
	mux.HandleFunc("POST /api/elections/{electionId}/admins", h.assignAdmin)
	mux.HandleFunc("DELETE /api/elections/{electionId}/admins/{userId}", h.unassignAdmin)
	mux.HandleFunc("POST /api/elections/{id}/submit", h.submitForApproval)
	mux.HandleFunc("POST /api/elections/{id}/approve", h.approveElection)
	mux.HandleFunc("POST /api/elections/{id}/reject", h.rejectElection)
	mux.HandleFunc("DELETE /api/elections/{id}", h.deleteElection)
	mux.HandleFunc("POST /api/elections/{id}/archive", h.archiveElection)
	mux.HandleFunc("PUT /api/elections/{id}/schedule", h.updateSchedule)
	mux.HandleFunc("GET /api/elections/{id}/schedule-changes", h.listScheduleChanges)
}

// authorizeBallotEditor allows a systemadmin unconditionally, or an admin
// who is specifically assigned to this election. An approver never edits a
// ballot - reviewing one is a different role entirely.
func (h *Handler) authorizeBallotEditor(w http.ResponseWriter, r *http.Request, electionID string) (*auth.User, error) {
	return AuthorizeBallotEditor(h.authService, h.service, w, r, electionID)
}

// AuthorizeBallotEditor is the package-level form of the same check, for
// other handler packages (voter credential issuance, specifically) that
// need the identical "systemadmin or assigned admin" rule without taking
// on a dependency on election.Handler itself.
func AuthorizeBallotEditor(authService *auth.Service, service *Service, w http.ResponseWriter, r *http.Request, electionID string) (*auth.User, error) {
	actor, err := authService.Authorize(w, r, dashboardRoles...)
	if err != nil {
		return nil, err
	}
	if actor.Role == "SYSTEMADMIN" {
		return actor, nil
	}
	if actor.Role == "ADMIN" {
		assigned, err := service.IsAssigned(r.Context(), electionID, actor.ID)
		if err != nil {
			return nil, err
		}
		if assigned {
			return actor, nil
		}
	}
	return nil, auth.ErrForbidden
}

func writeBallotAuthError(w http.ResponseWriter, err error) {
	WriteBallotAuthError(w, err)
}

// WriteBallotAuthError is the package-level form of the same error mapping,
// paired with AuthorizeBallotEditor for reuse outside this package.
func WriteBallotAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "only a systemadmin or an admin assigned to this election can edit the ballot")
	case errors.Is(err, auth.ErrUnauthorized):
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
	default:
		log.Printf("ballot authorization check failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to authorize")
	}
}

type createRequest struct {
	Title                    string            `json:"title"`
	Description              string            `json:"description"`
	StartTime                string            `json:"startTime"`
	EndTime                  string            `json:"endTime"`
	ElectorateDatabaseID     string            `json:"electorateDatabaseId"`
	ElectorateFilter         map[string]string `json:"electorateFilter"`
	VoterVerificationColumns []string          `json:"voterVerificationColumns"`
	AccentColor              string            `json:"accentColor"`
	LogoURL                  string            `json:"logoUrl"`
	BallotStyle              string            `json:"ballotStyle"`
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
		Title:                    req.Title,
		Description:              req.Description,
		StartTime:                startTime,
		EndTime:                  endTime,
		ElectorateDatabaseID:     req.ElectorateDatabaseID,
		ElectorateFilter:         req.ElectorateFilter,
		VoterVerificationColumns: req.VoterVerificationColumns,
		AccentColor:              req.AccentColor,
		LogoURL:                  req.LogoURL,
		BallotStyle:              req.BallotStyle,
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

	case errors.Is(err, ErrElectorateNotFound):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("election creation failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to create election")
	}
}

func (h *Handler) updateElection(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("id")
	actor, err := h.authorizeBallotEditor(w, r, electionID)
	if err != nil {
		writeBallotAuthError(w, err)
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

	result, err := h.service.Update(r.Context(), electionID, CreateInput{
		Title:                    req.Title,
		Description:              req.Description,
		StartTime:                startTime,
		EndTime:                  endTime,
		ElectorateDatabaseID:     req.ElectorateDatabaseID,
		ElectorateFilter:         req.ElectorateFilter,
		VoterVerificationColumns: req.VoterVerificationColumns,
		AccentColor:              req.AccentColor,
		LogoURL:                  req.LogoURL,
		BallotStyle:              req.BallotStyle,
	})

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_UPDATED",
			Metadata: map[string]any{
				"election_id": result.ID,
				"title":       result.Title,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrElectorateNotFound):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("election update failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to update election")
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

func (h *Handler) getElectorateSummary(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, dashboardRoles...); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	summary, err := h.service.GetElectorateSummary(r.Context(), r.PathValue("id"))
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, summary)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")
	default:
		log.Printf("getting electorate summary failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to get electorate summary")
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
	actor, err := h.authorizeBallotEditor(w, r, r.PathValue("electionId"))
	if err != nil {
		writeBallotAuthError(w, err)
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
	electionID := r.PathValue("electionId")
	actor, err := h.authorizeBallotEditor(w, r, electionID)
	if err != nil {
		writeBallotAuthError(w, err)
		return
	}

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

func (h *Handler) reorderPortfolios(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("electionId")
	actor, err := h.authorizeBallotEditor(w, r, electionID)
	if err != nil {
		writeBallotAuthError(w, err)
		return
	}

	var order []OrderEntry
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.ReorderPortfolios(r.Context(), electionID, order)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "PORTFOLIOS_REORDERED",
			Metadata:    map[string]any{"election_id": electionID},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "reordered"})

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrUnknownOrderItem):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("reordering portfolios failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to reorder portfolios")
	}
}

type candidateRequest struct {
	FullName     string   `json:"fullName"`
	PhotoURL     string   `json:"photoUrl"`
	RunningMates []string `json:"runningMates"`
}

func (h *Handler) createCandidate(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("electionId")
	actor, err := h.authorizeBallotEditor(w, r, electionID)
	if err != nil {
		writeBallotAuthError(w, err)
		return
	}

	var req candidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	portfolioID := r.PathValue("portfolioId")
	result, err := h.service.CreateCandidate(r.Context(), electionID, portfolioID, req.FullName, req.PhotoURL, req.RunningMates)

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

func (h *Handler) updateCandidate(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("electionId")
	actor, err := h.authorizeBallotEditor(w, r, electionID)
	if err != nil {
		writeBallotAuthError(w, err)
		return
	}

	var req candidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	portfolioID := r.PathValue("portfolioId")
	candidateID := r.PathValue("candidateId")
	result, err := h.service.UpdateCandidate(r.Context(), electionID, portfolioID, candidateID, req.FullName, req.PhotoURL, req.RunningMates)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "CANDIDATE_UPDATED",
			Metadata: map[string]any{
				"election_id":  electionID,
				"portfolio_id": portfolioID,
				"candidate_id": result.ID,
				"full_name":    result.FullName,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such candidate")

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("updating candidate failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to update candidate")
	}
}

func (h *Handler) deleteCandidate(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("electionId")
	actor, err := h.authorizeBallotEditor(w, r, electionID)
	if err != nil {
		writeBallotAuthError(w, err)
		return
	}

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

func (h *Handler) reorderCandidates(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("electionId")
	actor, err := h.authorizeBallotEditor(w, r, electionID)
	if err != nil {
		writeBallotAuthError(w, err)
		return
	}

	portfolioID := r.PathValue("portfolioId")
	var order []OrderEntry
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.ReorderCandidates(r.Context(), electionID, portfolioID, order)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "CANDIDATES_REORDERED",
			Metadata:    map[string]any{"election_id": electionID, "portfolio_id": portfolioID},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "reordered"})

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrUnknownOrderItem):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("reordering candidates failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to reorder candidates")
	}
}

func (h *Handler) listAssignedAdmins(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, dashboardRoles...); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	assignments, err := h.service.ListAssignedAdmins(r.Context(), r.PathValue("electionId"))
	if err != nil {
		log.Printf("listing assigned admins failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to list assigned admins")
		return
	}

	httpx.JSON(w, http.StatusOK, assignments)
}

type assignAdminRequest struct {
	UserID string `json:"userId"`
}

func (h *Handler) assignAdmin(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can assign admins")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	var req assignAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	electionID := r.PathValue("electionId")
	result, err := h.service.AssignAdmin(r.Context(), electionID, req.UserID, actor.ID)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_ADMIN_ASSIGNED",
			Metadata: map[string]any{
				"election_id":    electionID,
				"assigned_user":  result.UserID,
				"assigned_email": result.Email,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusCreated, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrUserNotFound):
		httpx.Error(w, http.StatusNotFound, "no such user")

	case errors.Is(err, ErrNotAnAdmin):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrAlreadyAssigned):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("assigning admin failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to assign admin")
	}
}

func (h *Handler) unassignAdmin(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can unassign admins")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	electionID := r.PathValue("electionId")
	userID := r.PathValue("userId")
	err = h.service.UnassignAdmin(r.Context(), electionID, userID)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_ADMIN_UNASSIGNED",
			Metadata: map[string]any{
				"election_id":     electionID,
				"unassigned_user": userID,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "unassigned"})

	case errors.Is(err, ErrNotAssigned):
		httpx.Error(w, http.StatusNotFound, err.Error())

	default:
		log.Printf("unassigning admin failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to unassign admin")
	}
}

func (h *Handler) submitForApproval(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("id")
	actor, err := h.authorizeBallotEditor(w, r, electionID)
	if err != nil {
		writeBallotAuthError(w, err)
		return
	}

	result, err := h.service.SubmitForApproval(r.Context(), electionID)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_SUBMITTED",
			Metadata:    map[string]any{"election_id": electionID, "title": result.Title},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrEmptyBallot), errors.Is(err, ErrPortfolioEmpty), errors.Is(err, ErrNoElectorateDatabase),
		errors.Is(err, ErrNoEligibleVoters), errors.Is(err, ErrElectionAlreadyEnded):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("submitting election failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to submit election")
	}
}

func (h *Handler) approveElection(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "APPROVER")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only an approver can approve an election")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	electionID := r.PathValue("id")
	result, err := h.service.Approve(r.Context(), electionID, actor.ID)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_APPROVED",
			Metadata:    map[string]any{"election_id": electionID, "title": result.Title},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrNotPendingApproval):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrElectionAlreadyEnded):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("approving election failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to approve election")
	}
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) rejectElection(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "APPROVER")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only an approver can reject an election")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	var req rejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	electionID := r.PathValue("id")
	result, err := h.service.Reject(r.Context(), electionID, req.Reason)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_REJECTED",
			Metadata: map[string]any{
				"election_id": electionID,
				"title":       result.Title,
				"reason":      req.Reason,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrNotPendingApproval):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("rejecting election failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to reject election")
	}
}

func (h *Handler) deleteElection(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can delete an election")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	electionID := r.PathValue("id")
	err = h.service.Delete(r.Context(), electionID)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_DELETED",
			Metadata:    map[string]any{"election_id": electionID},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrElectionNotDraft):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("deleting election failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to delete election")
	}
}

func (h *Handler) archiveElection(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can archive an election")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return
	}

	electionID := r.PathValue("id")
	result, err := h.service.Archive(r.Context(), electionID)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_ARCHIVED",
			Metadata:    map[string]any{"election_id": electionID, "title": result.Title},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrElectionNotClosed):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("archiving election failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to archive election")
	}
}

type updateScheduleRequest struct {
	EndTime string `json:"endTime"`
	Reason  string `json:"reason"`
}

func (h *Handler) updateSchedule(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("id")
	actor, err := h.authorizeBallotEditor(w, r, electionID)
	if err != nil {
		writeBallotAuthError(w, err)
		return
	}

	var req updateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	newEndTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "end time is invalid")
		return
	}

	result, err := h.service.UpdateSchedule(r.Context(), electionID, actor.ID, newEndTime, req.Reason)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTION_SCHEDULE_CHANGED",
			Metadata: map[string]any{
				"election_id":  electionID,
				"title":        result.Title,
				"new_end_time": result.EndTime,
				"reason":       req.Reason,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrElectionNotLive):
		httpx.Error(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrScheduleUnchanged), errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("updating election schedule failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to update schedule")
	}
}

func (h *Handler) listScheduleChanges(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, dashboardRoles...); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	changes, err := h.service.ListScheduleChanges(r.Context(), r.PathValue("id"))
	if err != nil {
		log.Printf("listing schedule changes failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to list schedule changes")
		return
	}

	httpx.JSON(w, http.StatusOK, changes)
}

func (h *Handler) logAudit(ctx context.Context, entry audit.Entry) {
	if err := audit.Log(ctx, h.pool, entry); err != nil {
		log.Printf("failed to write audit log for action %q: %v", entry.Action, err)
	}
}
