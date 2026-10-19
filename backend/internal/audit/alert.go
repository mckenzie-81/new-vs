package audit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAlertNotFound = errors.New("no such alert, or it's already been acknowledged")

type Alert struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"createdAt"`
}

// RunIntegrityCheck is what the background ticker in cmd/server calls
// every few minutes - it's the whole reason "automatic verification"
// means something instead of only ever running when an admin remembers
// to click the button. It checks both the chain's internal consistency
// and, more importantly, every past checkpoint against today's
// database, recording an alert the moment either disagrees. The new
// checkpoint is taken last, so it never papers over a problem this same
// run just found.
func RunIntegrityCheck(ctx context.Context, pool *pgxpool.Pool, checkpointPath string) error {
	ok, brokenAtSeq, _, err := VerifyChain(ctx, pool)
	if err != nil {
		return fmt.Errorf("verifying chain: %w", err)
	}
	if !ok {
		if err := recordAlert(ctx, pool, "CHAIN_BROKEN",
			fmt.Sprintf("the chain is internally inconsistent starting at entry #%d", *brokenAtSeq)); err != nil {
			return err
		}
	}

	checkpointsOk, mismatchAtSeq, _, err := VerifyAgainstCheckpoints(ctx, pool, checkpointPath)
	if err != nil {
		return fmt.Errorf("verifying checkpoints: %w", err)
	}
	if !checkpointsOk {
		if err := recordAlert(ctx, pool, "CHECKPOINT_MISMATCH",
			fmt.Sprintf("entry #%d no longer matches what was checkpointed outside the database earlier", *mismatchAtSeq)); err != nil {
			return err
		}
	}

	if _, err := WriteCheckpoint(ctx, pool, checkpointPath); err != nil {
		return fmt.Errorf("writing checkpoint: %w", err)
	}
	return nil
}

// recordAlert only inserts when there's no unacknowledged alert of the
// same kind already open - otherwise an unresolved problem would add a
// new row every single tick forever, burying the one that matters.
func recordAlert(ctx context.Context, pool *pgxpool.Pool, kind, detail string) error {
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM audit_integrity_alerts WHERE kind = $1 AND acknowledged_at IS NULL)
	`, kind).Scan(&exists); err != nil {
		return fmt.Errorf("checking for an existing alert: %w", err)
	}
	if exists {
		return nil
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_integrity_alerts (kind, detail) VALUES ($1, $2)
	`, kind, detail); err != nil {
		return fmt.Errorf("recording alert: %w", err)
	}
	return nil
}

// LatestUnacknowledgedAlert is the cheap, indexed read the dashboard
// polls on every page load - nothing like the full-table-scan a real
// verify does, which is exactly why it's safe to call that often.
func LatestUnacknowledgedAlert(ctx context.Context, pool *pgxpool.Pool) (*Alert, error) {
	var a Alert
	err := pool.QueryRow(ctx, `
		SELECT id, kind, detail, created_at FROM audit_integrity_alerts
		WHERE acknowledged_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&a.ID, &a.Kind, &a.Detail, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading latest alert: %w", err)
	}
	return &a, nil
}

// AcknowledgeAlert never deletes the row - the point of this whole
// feature is a permanent record, so dismissing an alert just marks it
// seen, by whom and when, instead of making it disappear.
func AcknowledgeAlert(ctx context.Context, pool *pgxpool.Pool, alertID, userID string) error {
	tag, err := pool.Exec(ctx, `
		UPDATE audit_integrity_alerts SET acknowledged_at = now(), acknowledged_by = $1
		WHERE id = $2 AND acknowledged_at IS NULL
	`, userID, alertID)
	if err != nil {
		return fmt.Errorf("acknowledging alert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlertNotFound
	}
	return nil
}
