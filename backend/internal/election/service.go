package election

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/electorate"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxTitleLength = 200

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrNotFound           = errors.New("election not found")
	ErrElectorateNotFound = errors.New("no such electorate database")
)

type Service struct {
	pool       *pgxpool.Pool
	electorate *electorate.Service
}

func NewService(pool *pgxpool.Pool, electorateService *electorate.Service) *Service {
	return &Service{pool: pool, electorate: electorateService}
}

type Election struct {
	ID                   string            `json:"id"`
	Title                string            `json:"title"`
	Description          *string           `json:"description"`
	Status               string            `json:"status"`
	StartTime            time.Time         `json:"startTime"`
	EndTime              time.Time         `json:"endTime"`
	CreatedBy            string            `json:"createdBy"`
	ApprovedBy           *string           `json:"approvedBy"`
	RejectionReason      *string           `json:"rejectionReason"`
	ElectorateDatabaseID string            `json:"electorateDatabaseId"`
	ElectorateFilter     map[string]string `json:"electorateFilter"`
	CreatedAt            time.Time         `json:"createdAt"`
	UpdatedAt            time.Time         `json:"updatedAt"`
}

type CreateInput struct {
	Title                string
	Description          string
	StartTime            time.Time
	EndTime              time.Time
	ElectorateDatabaseID string
	ElectorateFilter     map[string]string
}

func (s *Service) validateElectionInput(ctx context.Context, in CreateInput) (string, *string, map[string]string, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return "", nil, nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if len(title) > maxTitleLength {
		return "", nil, nil, fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalidInput, maxTitleLength)
	}
	if !in.EndTime.After(in.StartTime) {
		return "", nil, nil, fmt.Errorf("%w: end time must be after start time", ErrInvalidInput)
	}
	if strings.TrimSpace(in.ElectorateDatabaseID) == "" {
		return "", nil, nil, fmt.Errorf("%w: an electorate database is required", ErrInvalidInput)
	}
	exists, err := s.electorate.DatabaseExists(ctx, in.ElectorateDatabaseID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("checking electorate database: %w", err)
	}
	if !exists {
		return "", nil, nil, ErrElectorateNotFound
	}

	var description *string
	if trimmed := strings.TrimSpace(in.Description); trimmed != "" {
		description = &trimmed
	}

	filter := in.ElectorateFilter
	if filter == nil {
		filter = map[string]string{}
	}

	return title, description, filter, nil
}

func (s *Service) Create(ctx context.Context, actorID string, in CreateInput) (*Election, error) {
	title, description, filter, err := s.validateElectionInput(ctx, in)
	if err != nil {
		return nil, err
	}

	var e Election
	err = s.pool.QueryRow(ctx, `
		INSERT INTO elections (title, description, start_time, end_time, created_by, electorate_database_id, electorate_filter)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, description, status, start_time, end_time, created_by, approved_by, rejection_reason,
			electorate_database_id, electorate_filter, created_at, updated_at
	`, title, description, in.StartTime, in.EndTime, actorID, in.ElectorateDatabaseID, filter).Scan(
		&e.ID, &e.Title, &e.Description, &e.Status, &e.StartTime, &e.EndTime,
		&e.CreatedBy, &e.ApprovedBy, &e.RejectionReason,
		&e.ElectorateDatabaseID, &e.ElectorateFilter, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating election: %w", err)
	}

	return &e, nil
}

// Update changes an election's own details - title, description, schedule,
// and electorate link. Like every other ballot-adjacent edit, this only
// works while the election is still a draft.
func (s *Service) Update(ctx context.Context, electionID string, in CreateInput) (*Election, error) {
	e, err := s.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}
	if e.Status != "DRAFT" {
		return nil, ErrElectionNotDraft
	}

	title, description, filter, err := s.validateElectionInput(ctx, in)
	if err != nil {
		return nil, err
	}

	var updated Election
	err = s.pool.QueryRow(ctx, `
		UPDATE elections
		SET title = $1, description = $2, start_time = $3, end_time = $4,
			electorate_database_id = $5, electorate_filter = $6, updated_at = now()
		WHERE id = $7
		RETURNING id, title, description, status, start_time, end_time, created_by, approved_by, rejection_reason,
			electorate_database_id, electorate_filter, created_at, updated_at
	`, title, description, in.StartTime, in.EndTime, in.ElectorateDatabaseID, filter, electionID).Scan(
		&updated.ID, &updated.Title, &updated.Description, &updated.Status, &updated.StartTime, &updated.EndTime,
		&updated.CreatedBy, &updated.ApprovedBy, &updated.RejectionReason,
		&updated.ElectorateDatabaseID, &updated.ElectorateFilter, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("updating election: %w", err)
	}

	return &updated, nil
}

// List returns every election, newest first. No filtering by who's
// assigned to what yet - that's a separate piece once admin assignment
// exists. For now, anyone with a dashboard account can see the full list.
func (s *Service) List(ctx context.Context) ([]Election, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, description, status, start_time, end_time, created_by, approved_by, rejection_reason,
			electorate_database_id, electorate_filter, created_at, updated_at
		FROM elections
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing elections: %w", err)
	}
	defer rows.Close()

	elections := []Election{}
	for rows.Next() {
		var e Election
		if err := rows.Scan(
			&e.ID, &e.Title, &e.Description, &e.Status, &e.StartTime, &e.EndTime,
			&e.CreatedBy, &e.ApprovedBy, &e.RejectionReason,
			&e.ElectorateDatabaseID, &e.ElectorateFilter, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning election: %w", err)
		}
		elections = append(elections, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating elections: %w", err)
	}

	return elections, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Election, error) {
	var e Election
	err := s.pool.QueryRow(ctx, `
		SELECT id, title, description, status, start_time, end_time, created_by, approved_by, rejection_reason,
			electorate_database_id, electorate_filter, created_at, updated_at
		FROM elections
		WHERE id = $1
	`, id).Scan(
		&e.ID, &e.Title, &e.Description, &e.Status, &e.StartTime, &e.EndTime,
		&e.CreatedBy, &e.ApprovedBy, &e.RejectionReason,
		&e.ElectorateDatabaseID, &e.ElectorateFilter, &e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("looking up election: %w", err)
	}

	return &e, nil
}

type ElectorateSummary struct {
	DatabaseID    string            `json:"databaseId"`
	DatabaseName  string            `json:"databaseName"`
	Filter        map[string]string `json:"filter"`
	EligibleCount int               `json:"eligibleCount"`
}

func (s *Service) GetElectorateSummary(ctx context.Context, electionID string) (*ElectorateSummary, error) {
	e, err := s.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}

	db, err := s.electorate.GetDatabase(ctx, e.ElectorateDatabaseID)
	if err != nil {
		return nil, fmt.Errorf("looking up electorate database: %w", err)
	}

	count, err := s.electorate.CountEligible(ctx, e.ElectorateDatabaseID, e.ElectorateFilter)
	if err != nil {
		return nil, fmt.Errorf("counting eligible voters: %w", err)
	}

	return &ElectorateSummary{
		DatabaseID:    db.ID,
		DatabaseName:  db.Name,
		Filter:        e.ElectorateFilter,
		EligibleCount: count,
	}, nil
}
