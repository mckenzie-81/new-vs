package electorate

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxNameLength = 200

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrNotFound       = errors.New("electorate database not found")
	ErrInUse          = errors.New("this electorate database is linked to at least one election and can't be deleted")
	ErrColumnMismatch = errors.New("this file's columns don't match the database's existing columns")
	ErrMissingColumns = errors.New("the file must include a name column and an email column")
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type Database struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Columns     []string  `json:"columns"`
	VoterCount  int       `json:"voterCount"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (s *Service) CreateDatabase(ctx context.Context, actorID, name, description string) (*Database, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if len(name) > maxNameLength {
		return nil, fmt.Errorf("%w: name must be %d characters or fewer", ErrInvalidInput, maxNameLength)
	}

	var desc *string
	if trimmed := strings.TrimSpace(description); trimmed != "" {
		desc = &trimmed
	}

	d := Database{Name: name, Description: desc, Columns: []string{}, CreatedBy: actorID}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO electorate_databases (name, description, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, name, desc, actorID).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating electorate database: %w", err)
	}

	return &d, nil
}

func (s *Service) ListDatabases(ctx context.Context) ([]Database, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.id, d.name, d.description, d.columns, d.created_by, d.created_at, d.updated_at,
			coalesce((SELECT count(*) FROM electorate_voters v WHERE v.electorate_database_id = d.id), 0)
		FROM electorate_databases d
		ORDER BY d.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing electorate databases: %w", err)
	}
	defer rows.Close()

	databases := []Database{}
	for rows.Next() {
		var d Database
		if err := rows.Scan(
			&d.ID, &d.Name, &d.Description, &d.Columns, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.VoterCount,
		); err != nil {
			return nil, fmt.Errorf("scanning electorate database: %w", err)
		}
		databases = append(databases, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating electorate databases: %w", err)
	}

	return databases, nil
}

func (s *Service) GetDatabase(ctx context.Context, id string) (*Database, error) {
	var d Database
	err := s.pool.QueryRow(ctx, `
		SELECT d.id, d.name, d.description, d.columns, d.created_by, d.created_at, d.updated_at,
			coalesce((SELECT count(*) FROM electorate_voters v WHERE v.electorate_database_id = d.id), 0)
		FROM electorate_databases d
		WHERE d.id = $1
	`, id).Scan(&d.ID, &d.Name, &d.Description, &d.Columns, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.VoterCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("looking up electorate database: %w", err)
	}

	return &d, nil
}

// DatabaseExists is used by the election package to validate a linked
// database id at election-creation time without taking on a full
// dependency on Database's shape.
func (s *Service) DatabaseExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM electorate_databases WHERE id = $1)
	`, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("checking electorate database: %w", err)
	}
	return exists, nil
}

func (s *Service) DeleteDatabase(ctx context.Context, id string) error {
	var inUse bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM elections WHERE electorate_database_id = $1)
	`, id).Scan(&inUse); err != nil {
		return fmt.Errorf("checking electorate database usage: %w", err)
	}
	if inUse {
		return ErrInUse
	}

	tag, err := s.pool.Exec(ctx, `DELETE FROM electorate_databases WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting electorate database: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

type UploadResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

func normalizeHeader(h string) string {
	return strings.ToLower(strings.TrimSpace(h))
}

func equalColumns(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if normalizeHeader(a[i]) != normalizeHeader(b[i]) {
			return false
		}
	}
	return true
}

// UploadVoters parses a CSV and adds its rows to a database. The first
// upload sets the database's column set from the file's header row; every
// later upload must match those columns exactly, since voter records in
// one database are expected to share a consistent shape for filtering to
// make sense. The file must have a name and an email column (matched
// case-insensitively, kept in their original casing); everything else is
// carried through as-is, whatever an institution's own export calls it.
// Rows with a missing name/email, a malformed email, or an email already
// present in this database are skipped rather than failing the whole
// upload.
func (s *Service) UploadVoters(ctx context.Context, databaseID string, r io.Reader) (*UploadResult, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("%w: couldn't read the file's header row", ErrInvalidInput)
	}
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
	}
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], string([]byte{0xEF, 0xBB, 0xBF}))
	}

	nameIdx, emailIdx := -1, -1
	for i, h := range header {
		switch normalizeHeader(h) {
		case "name":
			nameIdx = i
		case "email":
			emailIdx = i
		}
	}
	if nameIdx == -1 || emailIdx == -1 {
		return nil, ErrMissingColumns
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var existingColumns []string
	err = tx.QueryRow(ctx, `SELECT columns FROM electorate_databases WHERE id = $1 FOR UPDATE`, databaseID).Scan(&existingColumns)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("looking up electorate database: %w", err)
	}

	if len(existingColumns) > 0 {
		if !equalColumns(existingColumns, header) {
			return nil, ErrColumnMismatch
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE electorate_databases SET columns = $1, updated_at = now() WHERE id = $2
		`, header, databaseID); err != nil {
			return nil, fmt.Errorf("setting electorate database columns: %w", err)
		}
	}

	result := &UploadResult{}
	seen := map[string]bool{}
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: malformed row in file", ErrInvalidInput)
		}
		if len(record) != len(header) {
			result.Skipped++
			continue
		}

		name := strings.TrimSpace(record[nameIdx])
		email := strings.TrimSpace(record[emailIdx])
		if name == "" || email == "" || !strings.Contains(email, "@") {
			result.Skipped++
			continue
		}
		emailKey := strings.ToLower(email)
		if seen[emailKey] {
			result.Skipped++
			continue
		}
		seen[emailKey] = true

		data := make(map[string]string, len(header))
		for i, col := range header {
			data[col] = strings.TrimSpace(record[i])
		}
		dataJSON, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("encoding voter row: %w", err)
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO electorate_voters (electorate_database_id, data)
			VALUES ($1, $2)
			ON CONFLICT (electorate_database_id, (lower(data ->> 'email'))) DO NOTHING
		`, databaseID, dataJSON)
		if err != nil {
			return nil, fmt.Errorf("inserting voter: %w", err)
		}
		if tag.RowsAffected() > 0 {
			result.Imported++
		} else {
			result.Skipped++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing upload: %w", err)
	}

	return result, nil
}

type Voter struct {
	ID   string            `json:"id"`
	Data map[string]string `json:"data"`
}

func buildFilterClause(filter map[string]string, args []any) (string, []any) {
	clause := ""
	for field, value := range filter {
		args = append(args, field, value)
		clause += fmt.Sprintf(" AND data ->> $%d = $%d", len(args)-1, len(args))
	}
	return clause, args
}

// CountEligible returns how many voters in a database match a filter - an
// empty/nil filter means everyone in the database.
func (s *Service) CountEligible(ctx context.Context, databaseID string, filter map[string]string) (int, error) {
	args := []any{databaseID}
	clause, args := buildFilterClause(filter, args)
	query := `SELECT count(*) FROM electorate_voters WHERE electorate_database_id = $1` + clause

	var count int
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting eligible voters: %w", err)
	}
	return count, nil
}

// ListVoters returns up to limit voters matching a filter, for display -
// this is a sample/browse view, not meant for exporting the full roll.
func (s *Service) ListVoters(ctx context.Context, databaseID string, filter map[string]string, limit int) ([]Voter, error) {
	args := []any{databaseID}
	clause, args := buildFilterClause(filter, args)
	query := fmt.Sprintf(`
		SELECT id, data FROM electorate_voters
		WHERE electorate_database_id = $1%s
		ORDER BY created_at ASC
		LIMIT %d
	`, clause, limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing voters: %w", err)
	}
	defer rows.Close()

	voters := []Voter{}
	for rows.Next() {
		var v Voter
		var raw []byte
		if err := rows.Scan(&v.ID, &raw); err != nil {
			return nil, fmt.Errorf("scanning voter: %w", err)
		}
		if err := json.Unmarshal(raw, &v.Data); err != nil {
			return nil, fmt.Errorf("decoding voter data: %w", err)
		}
		voters = append(voters, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating voters: %w", err)
	}

	return voters, nil
}

// DistinctValues lists the values that actually occur for one column in a
// database, for building a filter picker from real data rather than free
// text that might not match anything.
func (s *Service) DistinctValues(ctx context.Context, databaseID, column string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT data ->> $2
		FROM electorate_voters
		WHERE electorate_database_id = $1 AND data ->> $2 IS NOT NULL AND data ->> $2 <> ''
		ORDER BY 1
	`, databaseID, column)
	if err != nil {
		return nil, fmt.Errorf("listing distinct values: %w", err)
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scanning distinct value: %w", err)
		}
		values = append(values, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating distinct values: %w", err)
	}

	return values, nil
}
