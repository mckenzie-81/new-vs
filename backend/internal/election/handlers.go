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

func (h *Handler) logAudit(ctx context.Context, entry audit.Entry) {
	if err := audit.Log(ctx, h.pool, entry); err != nil {
		log.Printf("failed to write audit log for action %q: %v", entry.Action, err)
	}
}
