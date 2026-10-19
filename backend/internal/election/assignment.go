package election

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrUserNotFound    = errors.New("no such user")
	ErrNotAnAdmin      = errors.New("only an active admin can be assigned to an election")
	ErrAlreadyAssigned = errors.New("this admin is already assigned to this election")
	ErrNotAssigned     = errors.New("this admin is not assigned to this election")
)

type AdminAssignment struct {
	UserID     string    `json:"userId"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	AssignedAt time.Time `json:"assignedAt"`
}

// AssignAdmin gives an admin day-to-day access to one election's ballot -
// it does not touch who can create or approve elections, only who can edit
// this one's portfolios and candidates alongside a systemadmin. Only an
// ACTIVE user with the ADMIN role can be assigned; a systemadmin already
// has access everywhere, and an approver's job is reviewing, not editing.
func (s *Service) AssignAdmin(ctx context.Context, electionID, userID, assignedBy string) (*AdminAssignment, error) {
	if _, err := s.Get(ctx, electionID); err != nil {
		return nil, err
	}

	var role, status, name, email string
	err := s.pool.QueryRow(ctx, `
		SELECT role, status, name, email FROM users WHERE id = $1
	`, userID).Scan(&role, &status, &name, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}
	if role != "ADMIN" || status != "ACTIVE" {
		return nil, ErrNotAnAdmin
	}

	var alreadyAssigned bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM election_admins WHERE election_id = $1 AND user_id = $2)
	`, electionID, userID).Scan(&alreadyAssigned); err != nil {
		return nil, fmt.Errorf("checking existing assignment: %w", err)
	}
	if alreadyAssigned {
		return nil, ErrAlreadyAssigned
	}

	var assignedAt time.Time
	err = s.pool.QueryRow(ctx, `
		INSERT INTO election_admins (election_id, user_id, assigned_by)
		VALUES ($1, $2, $3)
		RETURNING assigned_at
	`, electionID, userID, assignedBy).Scan(&assignedAt)
	if err != nil {
		return nil, fmt.Errorf("assigning admin: %w", err)
	}

	return &AdminAssignment{UserID: userID, Name: name, Email: email, AssignedAt: assignedAt}, nil
}

// UnassignAdmin returns the unassigned admin's name/email on success -
// not used by the removal itself, but it's the one place that still
// knows who this was, for a caller that wants to notify them afterward
// without a second round-trip to look it up again.
func (s *Service) UnassignAdmin(ctx context.Context, electionID, userID string) (*AdminAssignment, error) {
	var name, email string
	if err := s.pool.QueryRow(ctx, `
		SELECT name, email FROM users WHERE id = $1
	`, userID).Scan(&name, &email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	tag, err := s.pool.Exec(ctx, `
		DELETE FROM election_admins WHERE election_id = $1 AND user_id = $2
	`, electionID, userID)
	if err != nil {
		return nil, fmt.Errorf("unassigning admin: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotAssigned
	}
	return &AdminAssignment{UserID: userID, Name: name, Email: email}, nil
}

func (s *Service) ListAssignedAdmins(ctx context.Context, electionID string) ([]AdminAssignment, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.name, u.email, ea.assigned_at
		FROM election_admins ea
		JOIN users u ON u.id = ea.user_id
		WHERE ea.election_id = $1
		ORDER BY ea.assigned_at ASC
	`, electionID)
	if err != nil {
		return nil, fmt.Errorf("listing assigned admins: %w", err)
	}
	defer rows.Close()

	assignments := []AdminAssignment{}
	for rows.Next() {
		var a AdminAssignment
		if err := rows.Scan(&a.UserID, &a.Name, &a.Email, &a.AssignedAt); err != nil {
			return nil, fmt.Errorf("scanning assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating assignments: %w", err)
	}

	return assignments, nil
}

// IsAssigned is the check that lets an assigned admin past the
// systemadmin-only gate on ballot edits.
func (s *Service) IsAssigned(ctx context.Context, electionID, userID string) (bool, error) {
	var assigned bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM election_admins WHERE election_id = $1 AND user_id = $2)
	`, electionID, userID).Scan(&assigned)
	if err != nil {
		return false, fmt.Errorf("checking assignment: %w", err)
	}
	return assigned, nil
}
