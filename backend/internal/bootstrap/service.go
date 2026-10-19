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

const (
	verificationTTL   = 12 * time.Hour
	inviteTokenTTL    = 3 * 24 * time.Hour
	minPasswordLength = 12
	maxSystemadmins   = 2
)

const (
	bootstrapAdvisoryLockKey = "bootstrap:systemadmin"
	inviteAdvisoryLockKey    = "invite:create"
)

var (
	ErrAlreadyBootstrapped   = errors.New("an active systemadmin already exists")
	ErrInvalidSecret         = errors.New("invalid bootstrap secret")
	ErrInvalidInput          = errors.New("invalid input")
	ErrInvalidToken          = errors.New("invalid verification token")
	ErrTokenExpired          = errors.New("verification token expired")
	ErrInvalidRole           = errors.New("invalid role")
	ErrSystemadminCapReached = errors.New("maximum number of systemadmins already exist")
	ErrEmailInUse            = errors.New("an active account with this email already exists")
	ErrPersonNotFound        = errors.New("no such person")
	ErrNotPending            = errors.New("only a pending invite can be revoked")
	ErrCannotDisableSelf     = errors.New("you cannot disable your own account")
	ErrAccountNotActive      = errors.New("only an active account can be disabled")
	ErrAccountNotDisabled    = errors.New("only a disabled account can be reactivated")
	ErrLastActiveSystemadmin = errors.New("cannot disable the last active systemadmin")
)

var invitableRoles = map[string]bool{
	"SYSTEMADMIN": true,
	"ADMIN":       true,
	"APPROVER":    true,
}

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
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, bootstrapAdvisoryLockKey); err != nil {
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

type InviteInput struct {
	Name  string
	Email string
	Role  string
}

type InviteResult struct {
	UserID    string
	Name      string
	Email     string
	Role      string
	RawToken  string
	ExpiresAt time.Time
}

// InviteUser creates a pending account for a role an authenticated
// Systemadmin is granting - the same pending-account machinery Initiate
// uses, just gated by "who's asking" instead of a shared secret, and with
// no password collected yet: the invitee sets their own password when they
// accept, via Confirm.
func (s *Service) InviteUser(ctx context.Context, in InviteInput) (*InviteResult, error) {
	if !invitableRoles[in.Role] {
		return nil, ErrInvalidRole
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if !strings.Contains(in.Email, "@") || len(in.Email) < 5 {
		return nil, fmt.Errorf("%w: email is invalid", ErrInvalidInput)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Same reasoning as bootstrap's lock: invites are rare enough that a
	// global lock costs nothing, and it closes the same class of race
	// between two simultaneous invites (e.g. two systemadmins inviting the
	// same email, or both granting the second systemadmin slot at once).
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, inviteAdvisoryLockKey); err != nil {
		return nil, fmt.Errorf("acquiring invite lock: %w", err)
	}

	if in.Role == "SYSTEMADMIN" {
		var count int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM users WHERE role = 'SYSTEMADMIN' AND status IN ('ACTIVE', 'PENDING_VERIFICATION')
		`).Scan(&count); err != nil {
			return nil, fmt.Errorf("counting systemadmins: %w", err)
		}
		if count >= maxSystemadmins {
			return nil, ErrSystemadminCapReached
		}
	}

	var alreadyActive bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE lower(email) = lower($1) AND status = 'ACTIVE')
	`, in.Email).Scan(&alreadyActive); err != nil {
		return nil, fmt.Errorf("checking existing account: %w", err)
	}
	if alreadyActive {
		return nil, ErrEmailInUse
	}

	// Latest invite wins per email: a new invite to someone who hasn't
	// accepted their last one supersedes it, rather than stacking.
	if _, err := tx.Exec(ctx, `
		DELETE FROM users WHERE lower(email) = lower($1) AND status = 'PENDING_VERIFICATION'
	`, in.Email); err != nil {
		return nil, fmt.Errorf("clearing prior pending invite: %w", err)
	}

	rawToken, tokenHash, err := generateVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("generating invite token: %w", err)
	}

	expiresAt := time.Now().Add(inviteTokenTTL)

	// password_hash starts empty - there is no password yet, the invitee
	// sets one when they accept. Confirm treats an empty hash as "this
	// account still needs a password" and requires one at that point.
	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, status, verification_token_hash, verification_expires_at)
		VALUES ($1, $2, '', $3, 'PENDING_VERIFICATION', $4, $5)
		RETURNING id
	`, in.Name, in.Email, in.Role, tokenHash, expiresAt).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("creating invited account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing invite: %w", err)
	}

	return &InviteResult{
		UserID:    userID,
		Name:      in.Name,
		Email:     in.Email,
		Role:      in.Role,
		RawToken:  rawToken,
		ExpiresAt: expiresAt,
	}, nil
}

type ConfirmResult struct {
	UserID string
	Name   string
	Email  string
	Role   string
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
//
// It serves two different pending-account origins through one path:
// self-service bootstrap accounts already have a password set at this
// point, so password is ignored; invited accounts don't, so one is
// required here - accepting an invite is what sets it.
func (s *Service) Confirm(ctx context.Context, rawToken string, password string) (*ConfirmResult, error) {
	tokenHash := hashToken(rawToken)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		userID       string
		name         string
		email        string
		role         string
		status       string
		passwordHash string
		expiresAt    *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, name, email, role, status, password_hash, verification_expires_at
		FROM users
		WHERE verification_token_hash = $1
		FOR UPDATE
	`, tokenHash).Scan(&userID, &name, &email, &role, &status, &passwordHash, &expiresAt)
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
		return &ConfirmResult{UserID: userID, Name: name, Email: email, Role: role, AlreadyActive: true}, nil

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

		if passwordHash == "" {
			// Invited account: no password yet, this is what sets it.
			if len(password) < minPasswordLength {
				return nil, fmt.Errorf("%w: password must be at least %d characters", ErrInvalidInput, minPasswordLength)
			}
			newHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return nil, fmt.Errorf("hashing password: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				UPDATE users
				SET status = 'ACTIVE', password_hash = $2, updated_at = now()
				WHERE id = $1
			`, userID, string(newHash)); err != nil {
				return nil, fmt.Errorf("activating invited account: %w", err)
			}
		} else {
			// Self-service bootstrap account: password was already set at
			// Initiate time. The token hash is deliberately kept (not
			// nulled) after activation, so a later repeat confirm of the
			// same token can be recognized above instead of looking
			// indistinguishable from a bogus one.
			if _, err := tx.Exec(ctx, `
				UPDATE users
				SET status = 'ACTIVE', updated_at = now()
				WHERE id = $1
			`, userID); err != nil {
				return nil, fmt.Errorf("activating systemadmin: %w", err)
			}
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing confirmation: %w", err)
		}

		return &ConfirmResult{UserID: userID, Name: name, Email: email, Role: role}, nil

	default:
		return nil, ErrInvalidToken
	}
}

type Person struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	TOTPEnabled bool      `json:"totpEnabled"`
}

// ListPeople returns everyone who has ever been invited or bootstrapped,
// newest first. There's no pagination yet - fine at the scale this system
// has accounts for (tens, not thousands).
func (s *Service) ListPeople(ctx context.Context) ([]Person, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, email, role, status, created_at, totp_enabled_at IS NOT NULL
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing people: %w", err)
	}
	defer rows.Close()

	people := []Person{}
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.Name, &p.Email, &p.Role, &p.Status, &p.CreatedAt, &p.TOTPEnabled); err != nil {
			return nil, fmt.Errorf("scanning person: %w", err)
		}
		people = append(people, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating people: %w", err)
	}

	return people, nil
}

type RevokedPerson struct {
	Name  string
	Email string
	Role  string
}

// RevokeInvite cancels a pending invite before it's accepted. It only ever
// touches PENDING_VERIFICATION rows - revoking access from someone already
// active is a materially different, more consequential action and isn't
// what this does.
func (s *Service) RevokeInvite(ctx context.Context, id string) (*RevokedPerson, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var p RevokedPerson
	var status string
	err = tx.QueryRow(ctx, `
		SELECT name, email, role, status FROM users WHERE id = $1 FOR UPDATE
	`, id).Scan(&p.Name, &p.Email, &p.Role, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPersonNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("looking up person: %w", err)
	}

	if status != "PENDING_VERIFICATION" {
		return nil, ErrNotPending
	}

	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, id); err != nil {
		return nil, fmt.Errorf("revoking invite: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing revoke: %w", err)
	}

	return &p, nil
}

// DisableAccount removes access from an already-active administrator -
// unlike RevokeInvite, this is about someone who's actually using the
// system, not an unsent invite. A systemadmin can never target their own
// account (it would kill their own session mid-action, and is simply not
// what this is for), and the active-systemadmin count is protected from
// ever dropping to zero even under two concurrent disable requests.
func (s *Service) DisableAccount(ctx context.Context, actorID, targetID string) (*RevokedPerson, error) {
	if actorID == targetID {
		return nil, ErrCannotDisableSelf
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Same lock as invite creation: both touch the active-systemadmin
	// population, and sharing it closes the race where two systemadmins
	// simultaneously disable each other, leaving zero behind.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, inviteAdvisoryLockKey); err != nil {
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}

	var p RevokedPerson
	var status string
	err = tx.QueryRow(ctx, `
		SELECT name, email, role, status FROM users WHERE id = $1 FOR UPDATE
	`, targetID).Scan(&p.Name, &p.Email, &p.Role, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPersonNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("looking up person: %w", err)
	}

	if status != "ACTIVE" {
		return nil, ErrAccountNotActive
	}

	if p.Role == "SYSTEMADMIN" {
		var count int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM users WHERE role = 'SYSTEMADMIN' AND status = 'ACTIVE'
		`).Scan(&count); err != nil {
			return nil, fmt.Errorf("counting active systemadmins: %w", err)
		}
		if count <= 1 {
			return nil, ErrLastActiveSystemadmin
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET status = 'DISABLED' WHERE id = $1`, targetID); err != nil {
		return nil, fmt.Errorf("disabling account: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, targetID); err != nil {
		return nil, fmt.Errorf("clearing sessions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing disable: %w", err)
	}

	return &p, nil
}

// ReactivateAccount restores a disabled account to ACTIVE. For a
// systemadmin target it's still subject to the same population cap invite
// creation enforces - reactivating shouldn't be a backdoor past it.
func (s *Service) ReactivateAccount(ctx context.Context, targetID string) (*RevokedPerson, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, inviteAdvisoryLockKey); err != nil {
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}

	var p RevokedPerson
	var status string
	err = tx.QueryRow(ctx, `
		SELECT name, email, role, status FROM users WHERE id = $1 FOR UPDATE
	`, targetID).Scan(&p.Name, &p.Email, &p.Role, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPersonNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("looking up person: %w", err)
	}

	if status != "DISABLED" {
		return nil, ErrAccountNotDisabled
	}

	if p.Role == "SYSTEMADMIN" {
		var count int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM users
			WHERE role = 'SYSTEMADMIN' AND status IN ('ACTIVE', 'PENDING_VERIFICATION') AND id <> $1
		`, targetID).Scan(&count); err != nil {
			return nil, fmt.Errorf("counting systemadmins: %w", err)
		}
		if count >= maxSystemadmins {
			return nil, ErrSystemadminCapReached
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE users SET status = 'ACTIVE', failed_login_attempts = 0, locked_until = NULL WHERE id = $1
	`, targetID); err != nil {
		return nil, fmt.Errorf("reactivating account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing reactivate: %w", err)
	}

	return &p, nil
}

func validateInitiateInput(in InitiateInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if !strings.Contains(in.Email, "@") || len(in.Email) < 5 {
		return fmt.Errorf("%w: email is invalid", ErrInvalidInput)
	}
	if len(in.Password) < minPasswordLength {
		return fmt.Errorf("%w: password must be at least %d characters", ErrInvalidInput, minPasswordLength)
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
