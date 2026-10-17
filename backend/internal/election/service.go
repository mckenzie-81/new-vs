package election

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxTitleLength = 200

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("election not found")
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type Election struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
}

func (s *Service) Create(ctx context.Context, actorID string, in CreateInput) (*Election, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if len(title) > maxTitleLength {
		return nil, fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalidInput, maxTitleLength)
	}
	if !in.EndTime.After(in.StartTime) {
		return nil, fmt.Errorf("%w: end time must be after start time", ErrInvalidInput)
	}

	var description *string
	if trimmed := strings.TrimSpace(in.Description); trimmed != "" {
		description = &trimmed
	}

	var e Election
	err := s.pool.QueryRow(ctx, `
		INSERT INTO elections (title, description, start_time, end_time, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, status, start_time, end_time, created_by, created_at, updated_at
	`, title, description, in.StartTime, in.EndTime, actorID).Scan(
		&e.ID, &e.Title, &e.Description, &e.Status, &e.StartTime, &e.EndTime,
		&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating election: %w", err)
	}

	return &e, nil
}

// List returns every election, newest first. No filtering by who's
// assigned to what yet - that's a separate piece once admin assignment
// exists. For now, anyone with a dashboard account can see the full list.
func (s *Service) List(ctx context.Context) ([]Election, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, description, status, start_time, end_time, created_by, created_at, updated_at
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
			&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt,
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
		SELECT id, title, description, status, start_time, end_time, created_by, created_at, updated_at
		FROM elections
		WHERE id = $1
	`, id).Scan(
		&e.ID, &e.Title, &e.Description, &e.Status, &e.StartTime, &e.EndTime,
		&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("looking up election: %w", err)
	}

	return &e, nil
}
