// Package auditlog exposes the hash-chained audit log (internal/audit) to
// the dashboard - a separate package from audit itself so this can depend
// on internal/auth for authorization without audit depending back on
// auth (which already depends on audit to write its own entries).
package auditlog

import (
	"net/http"
	"strconv"
	"time"

	"backend/internal/audit"
	"backend/internal/auth"
	"backend/internal/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool        *pgxpool.Pool
	authService *auth.Service
}

func NewHandler(pool *pgxpool.Pool, authService *auth.Service) *Handler {
	return &Handler{pool: pool, authService: authService}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/audit-log", h.list)
	mux.HandleFunc("GET /api/audit-log/verify", h.verify)
}

const defaultLimit = 50
const maxLimit = 200

type entryResponse struct {
	Seq         int64          `json:"seq"`
	ActorUserID *string        `json:"actorUserId"`
	ActorName   *string        `json:"actorName"`
	ActorEmail  *string        `json:"actorEmail"`
	Action      string         `json:"action"`
	Metadata    map[string]any `json:"metadata"`
	IPAddress   *string        `json:"ipAddress"`
	UserAgent   *string        `json:"userAgent"`
	CreatedAt   time.Time      `json:"createdAt"`
}

// list is the only audit-log read this handler exposes - deliberately
// restricted to SYSTEMADMIN. Even an election's own assigned admins don't
// get this: it spans every election and every actor on the platform, not
// just theirs.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, "SYSTEMADMIN"); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	limit := defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= maxLimit {
			limit = n
		}
	}

	var before *int64
	if raw := r.URL.Query().Get("before"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
			before = &n
		}
	}

	var electionID *string
	if raw := r.URL.Query().Get("electionId"); raw != "" {
		electionID = &raw
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT a.seq, a.actor_user_id, u.name, u.email, a.action, a.metadata, a.ip_address, a.user_agent, a.created_at
		FROM audit_log a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE ($1::bigint IS NULL OR a.seq < $1)
		AND ($2::text IS NULL OR a.metadata->>'election_id' = $2)
		ORDER BY a.seq DESC
		LIMIT $3
	`, before, electionID, limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list audit log")
		return
	}
	defer rows.Close()

	entries := []entryResponse{}
	for rows.Next() {
		var e entryResponse
		if err := rows.Scan(&e.Seq, &e.ActorUserID, &e.ActorName, &e.ActorEmail, &e.Action, &e.Metadata, &e.IPAddress, &e.UserAgent, &e.CreatedAt); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "failed to read audit log")
			return
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to read audit log")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"entries": entries})
}

type verifyResponse struct {
	Ok          bool   `json:"ok"`
	Entries     int    `json:"entries"`
	BrokenAtSeq *int64 `json:"brokenAtSeq,omitempty"`
}

// verify re-walks and recomputes the entire chain on demand - an admin
// action, not something run on every page load, since it's a full table
// scan.
func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, "SYSTEMADMIN"); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	ok, brokenAtSeq, entries, err := audit.VerifyChain(r.Context(), h.pool)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to verify audit log")
		return
	}

	httpx.JSON(w, http.StatusOK, verifyResponse{Ok: ok, Entries: entries, BrokenAtSeq: brokenAtSeq})
}
