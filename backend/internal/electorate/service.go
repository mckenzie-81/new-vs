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
	"golang.org/x/crypto/bcrypt"
)

const maxNameLength = 200

var (
	ErrInvalidInput            = errors.New("invalid input")
	ErrNotFound                = errors.New("electorate database not found")
	ErrInUse                   = errors.New("this electorate database is linked to at least one election and can't be deleted")
	ErrColumnMismatch          = errors.New("this file's columns don't match the database's existing columns")
	ErrMissingColumns          = errors.New("the file must include a name column and an email column")
	ErrSensitiveColumnMismatch = errors.New("this file's sensitive columns don't match the database's existing sensitive columns")
	ErrSensitiveColumn         = errors.New("this column holds hashed values and can't be read back")
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type Database struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      *string   `json:"description"`
	Columns          []string  `json:"columns"`
	SensitiveColumns []string  `json:"sensitiveColumns"`
	VoterCount       int       `json:"voterCount"`
	CreatedBy        string    `json:"createdBy"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
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

	d := Database{Name: name, Description: desc, Columns: []string{}, SensitiveColumns: []string{}, CreatedBy: actorID}
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
		SELECT d.id, d.name, d.description, d.columns, d.sensitive_columns, d.created_by, d.created_at, d.updated_at,
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
			&d.ID, &d.Name, &d.Description, &d.Columns, &d.SensitiveColumns, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.VoterCount,
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
		SELECT d.id, d.name, d.description, d.columns, d.sensitive_columns, d.created_by, d.created_at, d.updated_at,
			coalesce((SELECT count(*) FROM electorate_voters v WHERE v.electorate_database_id = d.id), 0)
		FROM electorate_databases d
		WHERE d.id = $1
	`, id).Scan(&d.ID, &d.Name, &d.Description, &d.Columns, &d.SensitiveColumns, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.VoterCount)
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

// equalColumnSets compares two column lists as sets rather than sequences -
// used for sensitive columns, which are a subset marker rather than a
// positional header.
func equalColumnSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	normalized := make(map[string]bool, len(a))
	for _, c := range a {
		normalized[normalizeHeader(c)] = true
	}
	for _, c := range b {
		if !normalized[normalizeHeader(c)] {
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
//
// sensitiveColumns names columns (matched case-insensitively against the
// header, excluding name/email) whose values are secrets - a PIN or a
// second student ID, say - rather than plain metadata. Those values are
// bcrypt-hashed before they're ever written to the database and can't be
// read back in plaintext afterward; see ListVoters and DistinctValues. Like
// the column set itself, this is fixed by the first upload and every later
// upload must match it exactly.
func (s *Service) UploadVoters(ctx context.Context, databaseID string, r io.Reader, sensitiveColumns []string) (*UploadResult, error) {
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

	headerSet := make(map[string]bool, len(header))
	for _, h := range header {
		headerSet[normalizeHeader(h)] = true
	}
	sensitiveSet := make(map[string]bool, len(sensitiveColumns))
	for _, c := range sensitiveColumns {
		normalized := normalizeHeader(c)
		if normalized == "name" || normalized == "email" {
			return nil, fmt.Errorf("%w: name and email can't be marked sensitive", ErrInvalidInput)
		}
		if !headerSet[normalized] {
			return nil, fmt.Errorf("%w: sensitive column %q isn't in the file's header", ErrInvalidInput, c)
		}
		sensitiveSet[normalized] = true
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var existingColumns, existingSensitive []string
	err = tx.QueryRow(ctx, `
		SELECT columns, sensitive_columns FROM electorate_databases WHERE id = $1 FOR UPDATE
	`, databaseID).Scan(&existingColumns, &existingSensitive)
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
		if !equalColumnSets(existingSensitive, sensitiveColumns) {
			return nil, ErrSensitiveColumnMismatch
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE electorate_databases SET columns = $1, sensitive_columns = $2, updated_at = now() WHERE id = $3
		`, header, sensitiveColumns, databaseID); err != nil {
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

		data := make(map[string]string, len(header)+1)
		for i, col := range header {
			value := strings.TrimSpace(record[i])
			if sensitiveSet[normalizeHeader(col)] {
				hash, err := bcrypt.GenerateFromPassword([]byte(value), bcrypt.DefaultCost)
				if err != nil {
					return nil, fmt.Errorf("hashing sensitive value: %w", err)
				}
				value = string(hash)
			}
			data[col] = value
		}
		// The unique index and ON CONFLICT clause below key on the exact
		// JSON key "email", but the loop above preserves the header's
		// original casing (e.g. "Email") as the map key - without this,
		// any database whose header isn't literally lowercase "email"
		// silently loses deduplication on re-upload.
		data["email"] = email
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

const sensitiveValuePlaceholder = "••••••••"

// maskSensitive overwrites sensitive column values in place with a fixed
// placeholder, never the bcrypt hash itself - the hash material is never
// sent to a browser, not even an admin's.
func maskSensitive(data map[string]string, sensitiveColumns []string) {
	for _, col := range sensitiveColumns {
		if _, ok := data[col]; ok {
			data[col] = sensitiveValuePlaceholder
		}
	}
}

// ListVoters returns up to limit voters matching a filter, for display -
// this is a sample/browse view, not meant for exporting the full roll.
func (s *Service) ListVoters(ctx context.Context, databaseID string, filter map[string]string, limit int) ([]Voter, error) {
	var sensitiveColumns []string
	if err := s.pool.QueryRow(ctx, `
		SELECT sensitive_columns FROM electorate_databases WHERE id = $1
	`, databaseID).Scan(&sensitiveColumns); err != nil {
		return nil, fmt.Errorf("looking up sensitive columns: %w", err)
	}

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
		maskSensitive(v.Data, sensitiveColumns)
		voters = append(voters, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating voters: %w", err)
	}

	return voters, nil
}

// EligibleVoter is the minimal projection needed to issue voter
// credentials - just enough to send an invite, never the full data blob
// (which may hold bcrypt hashes that have no business loading into memory
// for a bulk pass over possibly thousands of rows).
type EligibleVoter struct {
	ID    string
	Email string
}

// ListEligibleVoters returns every voter matching a filter, uncapped -
// unlike ListVoters (a browse view capped for the admin UI), this backs
// voter-credential issuance, where silently truncating the roll at some
// limit would be a correctness bug, not a display nicety.
func (s *Service) ListEligibleVoters(ctx context.Context, databaseID string, filter map[string]string) ([]EligibleVoter, error) {
	args := []any{databaseID}
	clause, args := buildFilterClause(filter, args)
	query := `SELECT id, data ->> 'email' FROM electorate_voters WHERE electorate_database_id = $1` + clause

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing eligible voters: %w", err)
	}
	defer rows.Close()

	voters := []EligibleVoter{}
	for rows.Next() {
		var v EligibleVoter
		if err := rows.Scan(&v.ID, &v.Email); err != nil {
			return nil, fmt.Errorf("scanning eligible voter: %w", err)
		}
		voters = append(voters, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating eligible voters: %w", err)
	}

	return voters, nil
}

// DistinctValues lists the values that actually occur for one column in a
// database, for building a filter picker from real data rather than free
// text that might not match anything. Refuses a sensitive column outright -
// every value in one would be a unique bcrypt hash, meaningless as a filter
// choice and not something to hand back to a browser regardless.
func (s *Service) DistinctValues(ctx context.Context, databaseID, column string) ([]string, error) {
	var sensitiveColumns []string
	if err := s.pool.QueryRow(ctx, `
		SELECT sensitive_columns FROM electorate_databases WHERE id = $1
	`, databaseID).Scan(&sensitiveColumns); err != nil {
		return nil, fmt.Errorf("looking up sensitive columns: %w", err)
	}
	for _, c := range sensitiveColumns {
		if normalizeHeader(c) == normalizeHeader(column) {
			return nil, ErrSensitiveColumn
		}
	}

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
