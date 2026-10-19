// Package auditlog exposes the hash-chained audit log (internal/audit) to
// the dashboard - a separate package from audit itself so this can depend
// on internal/auth for authorization without audit depending back on
// auth (which already depends on audit to write its own entries).
package auditlog

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"backend/internal/audit"
	"backend/internal/auth"
	"backend/internal/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool           *pgxpool.Pool
	authService    *auth.Service
	checkpointPath string
}

func NewHandler(pool *pgxpool.Pool, authService *auth.Service, checkpointPath string) *Handler {
	return &Handler{pool: pool, authService: authService, checkpointPath: checkpointPath}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/audit-log", h.list)
	mux.HandleFunc("GET /api/audit-log/verify", h.verify)
	mux.HandleFunc("GET /api/audit-log/checkpoints", h.checkpoints)
	mux.HandleFunc("GET /api/audit-log/alert", h.getAlert)
	mux.HandleFunc("POST /api/audit-log/alerts/{id}/acknowledge", h.acknowledgeAlert)
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
	Ok      bool `json:"ok"`
	Entries int  `json:"entries"`
	// BrokenAtSeq is set when the chain itself is internally
	// inconsistent - the strongest possible finding, since even a
	// careful rewrite can't produce this.
	BrokenAtSeq *int64 `json:"brokenAtSeq,omitempty"`
	// CheckpointsOk and CheckpointMismatchAtSeq come from cross-checking
	// the external anchor file: a rewrite thorough enough to keep the
	// chain internally consistent (so BrokenAtSeq stays nil) still can't
	// also go back and edit a checkpoint already written outside
	// Postgres, so it shows up here instead.
	CheckpointsOk           bool   `json:"checkpointsOk"`
	CheckpointMismatchAtSeq *int64 `json:"checkpointMismatchAtSeq,omitempty"`
	CheckpointsChecked      int    `json:"checkpointsChecked"`
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

	checkpointsOk, mismatchAtSeq, checked, err := audit.VerifyAgainstCheckpoints(r.Context(), h.pool, h.checkpointPath)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to verify audit log checkpoints")
		return
	}

	httpx.JSON(w, http.StatusOK, verifyResponse{
		Ok:                      ok,
		Entries:                 entries,
		BrokenAtSeq:             brokenAtSeq,
		CheckpointsOk:           checkpointsOk,
		CheckpointMismatchAtSeq: mismatchAtSeq,
		CheckpointsChecked:      checked,
	})
}

// checkpoints lists every checkpoint ever taken, so an admin can see the
// anchor history directly - and, if they want real protection beyond
// this server's own disk, copy it somewhere further out of reach
// themselves (their own notes, a separate repo, email to themselves).
func (h *Handler) checkpoints(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, "SYSTEMADMIN"); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	checkpoints, err := audit.ReadCheckpoints(h.checkpointPath)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to read checkpoints")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"checkpoints": checkpoints})
}

// getAlert is the cheap read a dashboard-wide banner polls on every page
// load - a single indexed lookup, not a re-verify. It's how "automatic"
// verification actually surfaces to a human instead of only ever living
// in server logs nobody's watching.
func (h *Handler) getAlert(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, "SYSTEMADMIN"); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	alert, err := audit.LatestUnacknowledgedAlert(r.Context(), h.pool)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to check for integrity alerts")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"alert": alert})
}

func (h *Handler) acknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	actor, err := h.authService.Authorize(w, r, "SYSTEMADMIN")
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	err = audit.AcknowledgeAlert(r.Context(), h.pool, r.PathValue("id"), actor.ID)
	switch {
	case err == nil:
		if err := audit.LogStandalone(r.Context(), h.pool, audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "AUDIT_INTEGRITY_ALERT_ACKNOWLEDGED",
			Metadata:    map[string]any{"alert_id": r.PathValue("id")},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		}); err != nil {
			log.Printf("failed to write audit log for alert acknowledgement: %v", err)
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})

	case errors.Is(err, audit.ErrAlertNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())

	default:
		httpx.Error(w, http.StatusInternalServerError, "failed to acknowledge alert")
	}
}
