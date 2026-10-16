package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const verificationTTL = 12 * time.Hour

const advisoryLockKey = "bootstrap:systemadmin"

var (
	ErrAlreadyBootstrapped = errors.New("an active systemadmin already exists")
	ErrInvalidSecret       = errors.New("invalid bootstrap secret")
	ErrInvalidInput        = errors.New("invalid input")
	ErrInvalidToken        = errors.New("invalid verification token")
	ErrTokenExpired        = errors.New("verification token expired")
)

type Service struct {
	pool            *pgxpool.Pool
	bootstrapSecret string
}

func NewService(pool *pgxpool.Pool, bootstrapSecret string) *Service {
	return &Service{pool: pool, bootstrapSecret: bootstrapSecret}
}

// Available reports whether bootstrap is currently open, i.e. no active
// systemadmin exists yet. It is safe to expose publicly.
func (s *Service) Available(ctx context.Context) (bool, error) {
	var hasActive bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users WHERE role = 'SYSTEMADMIN' AND status = 'ACTIVE'
		)
	`).Scan(&hasActive)
	if err != nil {
		return false, fmt.Errorf("checking systemadmin existence: %w", err)
	}
	return !hasActive, nil
}

type InitiateInput struct {
	Name     string
	Email    string
	Password string
	Secret   string
}

type InitiateResult struct {
	UserID    string
	Name      string
	Email     string
	RawToken  string
	ExpiresAt time.Time
}

func (s *Service) Initiate(ctx context.Context, in InitiateInput) (*InitiateResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Serialize every bootstrap attempt against every other one. This is an
	// exceedingly rare operation, so a global lock has no real contention
	// cost, and it closes the race that a plain read-then-write would leave
	// open between two concurrent initiate calls.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, advisoryLockKey); err != nil {
		return nil, fmt.Errorf("acquiring bootstrap lock: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT id, status FROM users WHERE role = 'SYSTEMADMIN' FOR UPDATE
	`)
	if err != nil {
		return nil, fmt.Errorf("checking existing systemadmins: %w", err)
	}

	var stalePendingIDs []string
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning systemadmin row: %w", err)
		}
		if status == "ACTIVE" {
			rows.Close()
			return nil, ErrAlreadyBootstrapped
		}
		// PENDING_VERIFICATION: a new initiate attempt supersedes it
		// (latest attempt wins; the slot isn't permanently stuck behind an
		// unconfirmed email).
		stalePendingIDs = append(stalePendingIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating systemadmin rows: %w", err)
	}

	if len(stalePendingIDs) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = ANY($1)`, stalePendingIDs); err != nil {
			return nil, fmt.Errorf("clearing stale pending bootstrap: %w", err)
		}
	}

	if subtle.ConstantTimeCompare([]byte(s.bootstrapSecret), []byte(in.Secret)) != 1 {
		return nil, ErrInvalidSecret
	}

	if err := validateInitiateInput(in); err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	rawToken, tokenHash, err := generateVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("generating verification token: %w", err)
	}

	expiresAt := time.Now().Add(verificationTTL)

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, status, verification_token_hash, verification_expires_at)
		VALUES ($1, $2, $3, 'SYSTEMADMIN', 'PENDING_VERIFICATION', $4, $5)
		RETURNING id
	`, in.Name, in.Email, string(passwordHash), tokenHash, expiresAt).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("creating pending systemadmin: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing bootstrap initiation: %w", err)
	}

	return &InitiateResult{
		UserID:    userID,
		Name:      in.Name,
		Email:     in.Email,
		RawToken:  rawToken,
		ExpiresAt: expiresAt,
	}, nil
}

type ConfirmResult struct {
	UserID string
	Name   string
	Email  string
	// AlreadyActive is true when this call found the account already
	// activated by an earlier confirm of the same token (a duplicate
	// delivery - email link scanners, a double-click, a dev-mode double
	// effect - rather than this call being the one that activated it).
	AlreadyActive bool
}

// Confirm is idempotent with respect to a single token: confirming the same
// token twice succeeds both times rather than succeeding once and erroring
// the second time. A one-time link can genuinely be fetched more than once
// (corporate email security scanners pre-fetch links before a human clicks
// them), so "already used" cannot mean "reject it" - only "no such token at
// all" does.
func (s *Service) Confirm(ctx context.Context, rawToken string) (*ConfirmResult, error) {
	tokenHash := hashToken(rawToken)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		userID    string
		name      string
		email     string
		status    string
		expiresAt *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, name, email, status, verification_expires_at
		FROM users
		WHERE verification_token_hash = $1
		FOR UPDATE
	`, tokenHash).Scan(&userID, &name, &email, &status, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("looking up verification token: %w", err)
	}

	switch status {
	case "ACTIVE":
		// A repeat confirm of a token that already did its job. Nothing to
		// change; report success without touching the row again.
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing repeat confirmation: %w", err)
		}
		return &ConfirmResult{UserID: userID, Name: name, Email: email, AlreadyActive: true}, nil

	case "PENDING_VERIFICATION":
		if expiresAt != nil && time.Now().After(*expiresAt) {
			// Self-heal: drop the stale pending row now rather than waiting
			// for the next initiate attempt to clean it up, so the slot
			// reopens immediately.
			if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
				return nil, fmt.Errorf("clearing expired bootstrap attempt: %w", err)
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("committing expiry cleanup: %w", err)
			}
			return nil, ErrTokenExpired
		}

		// The token hash is deliberately kept (not nulled) after
		// activation, so a later repeat confirm of the same token can be
		// recognized above instead of looking indistinguishable from a
		// bogus one.
		if _, err := tx.Exec(ctx, `
			UPDATE users
			SET status = 'ACTIVE', updated_at = now()
			WHERE id = $1
		`, userID); err != nil {
			return nil, fmt.Errorf("activating systemadmin: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing confirmation: %w", err)
		}

		return &ConfirmResult{UserID: userID, Name: name, Email: email}, nil

	default:
		return nil, ErrInvalidToken
	}
}

func validateInitiateInput(in InitiateInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if !strings.Contains(in.Email, "@") || len(in.Email) < 5 {
		return fmt.Errorf("%w: email is invalid", ErrInvalidInput)
	}
	if len(in.Password) < 12 {
		return fmt.Errorf("%w: password must be at least 12 characters", ErrInvalidInput)
	}
	return nil
}

func generateVerificationToken() (rawToken string, tokenHash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	rawToken = hex.EncodeToString(buf)
	return rawToken, hashToken(rawToken), nil
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
