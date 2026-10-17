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
	ID                       string            `json:"id"`
	Title                    string            `json:"title"`
	Description              *string           `json:"description"`
	Status                   string            `json:"status"`
	StartTime                time.Time         `json:"startTime"`
	EndTime                  time.Time         `json:"endTime"`
	CreatedBy                string            `json:"createdBy"`
	ApprovedBy               *string           `json:"approvedBy"`
	RejectionReason          *string           `json:"rejectionReason"`
	ElectorateDatabaseID     *string           `json:"electorateDatabaseId"`
	ElectorateFilter         map[string]string `json:"electorateFilter"`
	VoterVerificationColumns []string          `json:"voterVerificationColumns"`
	CreatedAt                time.Time         `json:"createdAt"`
	UpdatedAt                time.Time         `json:"updatedAt"`
}

type CreateInput struct {
	Title                    string
	Description              string
	StartTime                time.Time
	EndTime                  time.Time
	ElectorateDatabaseID     string
	ElectorateFilter         map[string]string
	VoterVerificationColumns []string
}

const maxVerificationColumns = 2

// validateElectionInput checks the fields every election needs regardless
// of lifecycle stage. The electorate link is deliberately not one of them -
// a draft can exist without a database attached yet, since that's meant to
// be addable later while managing the draft. It's only required before
// submitting for approval (see SubmitForApproval).
func (s *Service) validateElectionInput(ctx context.Context, in CreateInput) (string, *string, *string, map[string]string, []string, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return "", nil, nil, nil, nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if len(title) > maxTitleLength {
		return "", nil, nil, nil, nil, fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalidInput, maxTitleLength)
	}
	if !in.EndTime.After(in.StartTime) {
		return "", nil, nil, nil, nil, fmt.Errorf("%w: end time must be after start time", ErrInvalidInput)
	}

	var electorateDatabaseID *string
	var allColumns, sensitiveColumns []string
	if trimmed := strings.TrimSpace(in.ElectorateDatabaseID); trimmed != "" {
		db, err := s.electorate.GetDatabase(ctx, trimmed)
		if errors.Is(err, electorate.ErrNotFound) {
			return "", nil, nil, nil, nil, ErrElectorateNotFound
		}
		if err != nil {
			return "", nil, nil, nil, nil, fmt.Errorf("checking electorate database: %w", err)
		}
		electorateDatabaseID = &trimmed
		allColumns = db.Columns
		sensitiveColumns = db.SensitiveColumns
	}

	// name/email aren't eligible as a verification factor: email is already
	// the magic-link channel itself, and name is routinely displayed and
	// far too guessable to prove anything.
	verifiableColumns := make([]string, 0, len(allColumns))
	for _, c := range allColumns {
		normalized := strings.ToLower(c)
		if normalized != "name" && normalized != "email" {
			verifiableColumns = append(verifiableColumns, c)
		}
	}

	// Filtering eligibility by a sensitive column can never match anything
	// (its stored values are per-row bcrypt hashes), so this would silently
	// produce "0 eligible" rather than fail loudly - catch it here instead.
	for field := range in.ElectorateFilter {
		if containsFold(sensitiveColumns, field) {
			return "", nil, nil, nil, nil, fmt.Errorf("%w: %q holds hashed values and can't be used as an eligibility filter", ErrInvalidInput, field)
		}
	}

	if len(in.VoterVerificationColumns) > maxVerificationColumns {
		return "", nil, nil, nil, nil, fmt.Errorf("%w: at most %d verification columns are allowed", ErrInvalidInput, maxVerificationColumns)
	}
	verificationColumns := make([]string, 0, len(in.VoterVerificationColumns))
	seen := map[string]bool{}
	for _, col := range in.VoterVerificationColumns {
		trimmed := strings.TrimSpace(col)
		if trimmed == "" {
			continue
		}
		if !containsFold(verifiableColumns, trimmed) {
			return "", nil, nil, nil, nil, fmt.Errorf("%w: %q isn't a column on the linked electorate database", ErrInvalidInput, trimmed)
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		verificationColumns = append(verificationColumns, trimmed)
	}

	var description *string
	if trimmed := strings.TrimSpace(in.Description); trimmed != "" {
		description = &trimmed
	}

	filter := in.ElectorateFilter
	if filter == nil {
		filter = map[string]string{}
	}

	return title, description, electorateDatabaseID, filter, verificationColumns, nil
}

func containsFold(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func (s *Service) Create(ctx context.Context, actorID string, in CreateInput) (*Election, error) {
	title, description, electorateDatabaseID, filter, verificationColumns, err := s.validateElectionInput(ctx, in)
	if err != nil {
		return nil, err
	}

	var e Election
	err = s.pool.QueryRow(ctx, `
		INSERT INTO elections (title, description, start_time, end_time, created_by, electorate_database_id, electorate_filter, voter_verification_columns)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, title, description, status, start_time, end_time, created_by, approved_by, rejection_reason,
			electorate_database_id, COALESCE(electorate_filter, '{}'::jsonb), voter_verification_columns, created_at, updated_at
	`, title, description, in.StartTime, in.EndTime, actorID, electorateDatabaseID, filter, verificationColumns).Scan(
		&e.ID, &e.Title, &e.Description, &e.Status, &e.StartTime, &e.EndTime,
		&e.CreatedBy, &e.ApprovedBy, &e.RejectionReason,
		&e.ElectorateDatabaseID, &e.ElectorateFilter, &e.VoterVerificationColumns, &e.CreatedAt, &e.UpdatedAt,
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

	title, description, electorateDatabaseID, filter, verificationColumns, err := s.validateElectionInput(ctx, in)
	if err != nil {
		return nil, err
	}

	var updated Election
	err = s.pool.QueryRow(ctx, `
		UPDATE elections
		SET title = $1, description = $2, start_time = $3, end_time = $4,
			electorate_database_id = $5, electorate_filter = $6, voter_verification_columns = $7, updated_at = now()
		WHERE id = $8
		RETURNING id, title, description, status, start_time, end_time, created_by, approved_by, rejection_reason,
			electorate_database_id, COALESCE(electorate_filter, '{}'::jsonb), voter_verification_columns, created_at, updated_at
	`, title, description, in.StartTime, in.EndTime, electorateDatabaseID, filter, verificationColumns, electionID).Scan(
		&updated.ID, &updated.Title, &updated.Description, &updated.Status, &updated.StartTime, &updated.EndTime,
		&updated.CreatedBy, &updated.ApprovedBy, &updated.RejectionReason,
		&updated.ElectorateDatabaseID, &updated.ElectorateFilter, &updated.VoterVerificationColumns, &updated.CreatedAt, &updated.UpdatedAt,
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
	if err := s.fixupLifecycle(ctx); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, title, description, status, start_time, end_time, created_by, approved_by, rejection_reason,
			electorate_database_id, COALESCE(electorate_filter, '{}'::jsonb), voter_verification_columns, created_at, updated_at
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
			&e.ElectorateDatabaseID, &e.ElectorateFilter, &e.VoterVerificationColumns, &e.CreatedAt, &e.UpdatedAt,
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
	if err := s.fixupLifecycle(ctx); err != nil {
		return nil, err
	}

	var e Election
	err := s.pool.QueryRow(ctx, `
		SELECT id, title, description, status, start_time, end_time, created_by, approved_by, rejection_reason,
			electorate_database_id, COALESCE(electorate_filter, '{}'::jsonb), voter_verification_columns, created_at, updated_at
		FROM elections
		WHERE id = $1
	`, id).Scan(
		&e.ID, &e.Title, &e.Description, &e.Status, &e.StartTime, &e.EndTime,
		&e.CreatedBy, &e.ApprovedBy, &e.RejectionReason,
		&e.ElectorateDatabaseID, &e.ElectorateFilter, &e.VoterVerificationColumns, &e.CreatedAt, &e.UpdatedAt,
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
	Linked        bool              `json:"linked"`
	DatabaseID    *string           `json:"databaseId"`
	DatabaseName  *string           `json:"databaseName"`
	Filter        map[string]string `json:"filter"`
	EligibleCount int               `json:"eligibleCount"`
}

func (s *Service) GetElectorateSummary(ctx context.Context, electionID string) (*ElectorateSummary, error) {
	e, err := s.Get(ctx, electionID)
	if err != nil {
		return nil, err
	}

	if e.ElectorateDatabaseID == nil {
		return &ElectorateSummary{Linked: false, Filter: map[string]string{}}, nil
	}

	db, err := s.electorate.GetDatabase(ctx, *e.ElectorateDatabaseID)
	if err != nil {
		return nil, fmt.Errorf("looking up electorate database: %w", err)
	}

	count, err := s.electorate.CountEligible(ctx, *e.ElectorateDatabaseID, e.ElectorateFilter)
	if err != nil {
		return nil, fmt.Errorf("counting eligible voters: %w", err)
	}

	return &ElectorateSummary{
		Linked:        true,
		DatabaseID:    &db.ID,
		DatabaseName:  &db.Name,
		Filter:        e.ElectorateFilter,
		EligibleCount: count,
	}, nil
}
