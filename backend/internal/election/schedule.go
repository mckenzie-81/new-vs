package election

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrElectionNotLive   = errors.New("only a live election's schedule can be changed this way")
	ErrScheduleUnchanged = errors.New("the new end time must be different from the current one")
)

type ScheduleChange struct {
	ID              string    `json:"id"`
	ElectionID      string    `json:"electionId"`
	ActorUserID     string    `json:"actorUserId"`
	ActorName       string    `json:"actorName"`
	PreviousEndTime time.Time `json:"previousEndTime"`
	NewEndTime      time.Time `json:"newEndTime"`
	Reason          string    `json:"reason"`
	CreatedAt       time.Time `json:"createdAt"`
}

// UpdateSchedule ends a live election early or extends it - a new end
// time earlier than the current one closes it (the very next read applies
// fixupLifecycle and flips status to CLOSED, same as reaching the
// original end time naturally would), a later one extends voting. Only
// end time can change this way, and only while LIVE: changing when
// something already LIVE "started" is meaningless, and a schedule not yet
// in effect belongs to the ordinary draft-edit flow instead. Every change
// is recorded with who, when, and why - this alters the terms voters were
// told applied, so it isn't optional bookkeeping.
func (s *Service) UpdateSchedule(ctx context.Context, electionID, actorID string, newEndTime time.Time, reason string) (*Election, error) {
	e, err := s.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}
	if e.Status != "LIVE" {
		return nil, ErrElectionNotLive
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("%w: a reason is required", ErrInvalidInput)
	}
	if len(reason) > maxRejectionReasonLength {
		return nil, fmt.Errorf("%w: reason must be %d characters or fewer", ErrInvalidInput, maxRejectionReasonLength)
	}
	if newEndTime.Equal(e.EndTime) {
		return nil, ErrScheduleUnchanged
	}
	if !newEndTime.After(e.StartTime) {
		return nil, fmt.Errorf("%w: end time must be after start time", ErrInvalidInput)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE elections SET end_time = $1, updated_at = now() WHERE id = $2
	`, newEndTime, electionID); err != nil {
		return nil, fmt.Errorf("updating schedule: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO election_schedule_changes (election_id, actor_user_id, previous_end_time, new_end_time, reason)
		VALUES ($1, $2, $3, $4, $5)
	`, electionID, actorID, e.EndTime, newEndTime, reason); err != nil {
		return nil, fmt.Errorf("recording schedule change: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing schedule change: %w", err)
	}

	return s.Get(ctx, electionID)
}

// ListScheduleChanges returns an election's schedule-change history,
// newest first, with the acting admin's name for the dashboard's benefit -
// the public pulse page shows the same history through its own, actor-free
// projection instead of calling this directly.
func (s *Service) ListScheduleChanges(ctx context.Context, electionID string) ([]ScheduleChange, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.election_id, c.actor_user_id, u.name, c.previous_end_time, c.new_end_time, c.reason, c.created_at
		FROM election_schedule_changes c
		JOIN users u ON u.id = c.actor_user_id
		WHERE c.election_id = $1
		ORDER BY c.created_at DESC
	`, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing schedule changes: %w", err)
	}
	defer rows.Close()

	changes := []ScheduleChange{}
	for rows.Next() {
		var c ScheduleChange
		if err := rows.Scan(
			&c.ID, &c.ElectionID, &c.ActorUserID, &c.ActorName, &c.PreviousEndTime, &c.NewEndTime, &c.Reason, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning schedule change: %w", err)
		}
		changes = append(changes, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating schedule changes: %w", err)
	}
	return changes, nil
}
