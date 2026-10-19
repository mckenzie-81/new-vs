package electorate

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/audit"
	"backend/internal/auth"
	"backend/internal/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxUploadSize = 10 << 20 // 10MB - generous for a CSV of plausible voter-list size

type Handler struct {
	service     *Service
	authService *auth.Service
	pool        *pgxpool.Pool
}

func NewHandler(service *Service, authService *auth.Service, pool *pgxpool.Pool) *Handler {
	return &Handler{service: service, authService: authService, pool: pool}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/electorate-databases", h.create)
	mux.HandleFunc("GET /api/electorate-databases", h.list)
	mux.HandleFunc("GET /api/electorate-databases/{id}", h.get)
	mux.HandleFunc("DELETE /api/electorate-databases/{id}", h.deleteDatabase)
	mux.HandleFunc("POST /api/electorate-databases/{id}/upload", h.upload)
	mux.HandleFunc("GET /api/electorate-databases/{id}/voters", h.listVoters)
	mux.HandleFunc("GET /api/electorate-databases/{id}/distinct", h.distinctValues)
	mux.HandleFunc("GET /api/electorate-databases/{id}/duplicates", h.duplicates)
	mux.HandleFunc("POST /api/electorate-databases/{id}/voters/remove", h.removeVoters)
}

// Electorate data is PII for everyone eligible to vote, not just the
// handful of accounts in this system - every endpoint here is
// systemadmin-only, more conservative than ballot access.
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) (*auth.User, error) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "only a systemadmin can manage electorate databases")
		} else {
			httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		}
		return nil, err
	}
	return actor, nil
}

type createRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authorize(w, r)
	if err != nil {
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.CreateDatabase(r.Context(), actor.ID, req.Name, req.Description)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTORATE_DATABASE_CREATED",
			Metadata:    map[string]any{"database_id": result.ID, "name": result.Name},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusCreated, result)

	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("creating electorate database failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to create electorate database")
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authorize(w, r); err != nil {
		return
	}

	databases, err := h.service.ListDatabases(r.Context())
	if err != nil {
		log.Printf("listing electorate databases failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to list electorate databases")
		return
	}

	httpx.JSON(w, http.StatusOK, databases)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authorize(w, r); err != nil {
		return
	}

	database, err := h.service.GetDatabase(r.Context(), r.PathValue("id"))
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, database)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such electorate database")
	default:
		log.Printf("getting electorate database failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to get electorate database")
	}
}

func (h *Handler) deleteDatabase(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authorize(w, r)
	if err != nil {
		return
	}

	id := r.PathValue("id")
	err = h.service.DeleteDatabase(r.Context(), id)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTORATE_DATABASE_DELETED",
			Metadata:    map[string]any{"database_id": id},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such electorate database")

	case errors.Is(err, ErrInUse):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("deleting electorate database failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to delete electorate database")
	}
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authorize(w, r)
	if err != nil {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		httpx.Error(w, http.StatusBadRequest, "file is too large or the upload is malformed")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "no file was uploaded")
		return
	}
	defer file.Close()

	var sensitiveColumns []string
	if raw := r.FormValue("sensitiveColumns"); raw != "" {
		for _, c := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(c); trimmed != "" {
				sensitiveColumns = append(sensitiveColumns, trimmed)
			}
		}
	}

	id := r.PathValue("id")
	result, err := h.service.UploadVoters(r.Context(), id, file, sensitiveColumns)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTORATE_VOTERS_UPLOADED",
			Metadata: map[string]any{
				"database_id": id,
				"imported":    result.Imported,
				"skipped":     result.Skipped,
			},
			IPAddress: httpx.ClientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, result)

	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such electorate database")

	case errors.Is(err, ErrMissingColumns), errors.Is(err, ErrColumnMismatch),
		errors.Is(err, ErrSensitiveColumnMismatch), errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	default:
		log.Printf("uploading voters failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to upload voters")
	}
}

func (h *Handler) listVoters(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authorize(w, r); err != nil {
		return
	}

	filter := map[string]string{}
	for key, values := range r.URL.Query() {
		if key == "limit" || len(values) == 0 {
			continue
		}
		filter[key] = values[0]
	}

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	voters, err := h.service.ListVoters(r.Context(), r.PathValue("id"), filter, limit)
	if err != nil {
		log.Printf("listing voters failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to list voters")
		return
	}

	httpx.JSON(w, http.StatusOK, voters)
}

func (h *Handler) duplicates(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authorize(w, r); err != nil {
		return
	}

	pairs, err := h.service.ExactDuplicates(r.Context(), r.PathValue("id"))
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, map[string]any{"pairs": pairs})
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such electorate database")
	default:
		log.Printf("checking for duplicate voters failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to check for duplicates")
	}
}

type removeVotersRequest struct {
	VoterIDs []string `json:"voterIds"`
}

func (h *Handler) removeVoters(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authorize(w, r)
	if err != nil {
		return
	}

	var req removeVotersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id := r.PathValue("id")
	removed, err := h.service.RemoveVoters(r.Context(), id, req.VoterIDs)

	switch {
	case err == nil:
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "ELECTORATE_VOTERS_REMOVED",
			Metadata:    map[string]any{"database_id": id, "removed": removed},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, map[string]any{"removed": removed})

	case errors.Is(err, ErrVotersLocked):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("removing voters failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to remove voters")
	}
}

func (h *Handler) distinctValues(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authorize(w, r); err != nil {
		return
	}

	column := r.URL.Query().Get("column")
	if column == "" {
		httpx.Error(w, http.StatusBadRequest, "column is required")
		return
	}

	values, err := h.service.DistinctValues(r.Context(), r.PathValue("id"), column)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, values)
	case errors.Is(err, ErrSensitiveColumn):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("listing distinct values failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to list distinct values")
	}
}

func (h *Handler) logAudit(ctx context.Context, entry audit.Entry) {
	if err := audit.LogStandalone(ctx, h.pool, entry); err != nil {
		log.Printf("failed to write audit log for action %q: %v", entry.Action, err)
	}
}
