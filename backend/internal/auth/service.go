package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionIdleDuration = 12 * time.Hour
	sessionAbsoluteMax  = 7 * 24 * time.Hour
	lockoutThreshold    = 5
	lockoutDuration     = 15 * time.Minute
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountLocked      = errors.New("account temporarily locked")
	ErrAccountNotVerified = errors.New("account not verified")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrSessionInvalid     = errors.New("invalid or expired session")
)

type Service struct {
	pool         *pgxpool.Pool
	cookieSecure bool
}

func NewService(pool *pgxpool.Pool, cookieSecure bool) *Service {
	return &Service{pool: pool, cookieSecure: cookieSecure}
}

type User struct {
	ID    string
	Name  string
	Email string
	Role  string
}

type LoginResult struct {
	User      User
	RawToken  string
	ExpiresAt time.Time

	// TOTPRequired means the password checked out but this account has
	// 2FA enabled - RawToken is empty and PendingLoginToken instead
	// identifies the in-progress login for CompleteTOTPLogin. No session
	// exists yet.
	TOTPRequired      bool
	PendingLoginToken string

	// UsedRecoveryCode is set when CompleteTOTPLogin accepted a backup
	// code instead of a live 6-digit code - worth surfacing to the
	// handler so it can nudge the user to regenerate their codes.
	UsedRecoveryCode bool
}

func (s *Service) Login(ctx context.Context, email, password, ip, userAgent string) (*LoginResult, error) {
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		id, name, dbEmail, role, status, passwordHash string
		failedAttempts                                int
		lockedUntil                                   *time.Time
		totpEnabledAt                                 *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, name, email, role, status, password_hash, failed_login_attempts, locked_until, totp_enabled_at
		FROM users
		WHERE lower(email) = lower($1)
		FOR UPDATE
	`, email).Scan(&id, &name, &dbEmail, &role, &status, &passwordHash, &failedAttempts, &lockedUntil, &totpEnabledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	if lockedUntil != nil && time.Now().Before(*lockedUntil) {
		return nil, ErrAccountLocked
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		newAttempts := failedAttempts + 1
		if newAttempts >= lockoutThreshold {
			lockUntil := time.Now().Add(lockoutDuration)
			if _, err := tx.Exec(ctx, `
				UPDATE users SET failed_login_attempts = 0, locked_until = $2 WHERE id = $1
			`, id, lockUntil); err != nil {
				return nil, fmt.Errorf("locking account: %w", err)
			}
		} else {
			if _, err := tx.Exec(ctx, `
				UPDATE users SET failed_login_attempts = $2 WHERE id = $1
			`, id, newAttempts); err != nil {
				return nil, fmt.Errorf("recording failed attempt: %w", err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing failed login: %w", err)
		}
		return nil, ErrInvalidCredentials
	}

	if status != "ACTIVE" {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing: %w", err)
		}
		if status == "DISABLED" {
			return nil, ErrAccountDisabled
		}
		return nil, ErrAccountNotVerified
	}

	// The password is proven correct at this point regardless of what
	// happens next, so the lockout counter resets here either way -
	// whether 2FA sends this account on to a pending-code step or
	// straight to a session.
	if _, err := tx.Exec(ctx, `
		UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE id = $1
	`, id); err != nil {
		return nil, fmt.Errorf("clearing lockout state: %w", err)
	}

	if totpEnabledAt != nil {
		pendingToken, err := s.beginPendingTOTPLogin(ctx, tx, id, ip, userAgent)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing pending login: %w", err)
		}
		return &LoginResult{
			User:              User{ID: id, Name: name, Email: dbEmail, Role: role},
			TOTPRequired:      true,
			PendingLoginToken: pendingToken,
		}, nil
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id); err != nil {
		return nil, fmt.Errorf("recording successful login: %w", err)
	}

	rawToken, expiresAt, err := s.createSession(ctx, tx, id, ip, userAgent)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing login: %w", err)
	}

	return &LoginResult{
		User:      User{ID: id, Name: name, Email: dbEmail, Role: role},
		RawToken:  rawToken,
		ExpiresAt: expiresAt,
	}, nil
}

// createSession is shared by Login (no 2FA) and CompleteTOTPLogin (2FA) -
// the one place a session row actually gets inserted, so there's only
// one definition of what "logged in" means to check for race conditions
// or expiry-math bugs in.
func (s *Service) createSession(ctx context.Context, tx pgx.Tx, userID, ip, userAgent string) (rawToken string, expiresAt time.Time, err error) {
	rawToken, tokenHash, err := generateSessionToken()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generating session token: %w", err)
	}

	now := time.Now()
	idleExpiry := now.Add(sessionIdleDuration)

	if _, err := tx.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, created_at, last_seen_at, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $3, $4, $5, $6)
	`, userID, tokenHash, now, idleExpiry, ip, userAgent); err != nil {
		return "", time.Time{}, fmt.Errorf("creating session: %w", err)
	}

	return rawToken, now.Add(sessionAbsoluteMax), nil
}

// ValidateSession checks a session token and, if it's still valid, extends
// its idle deadline. A session is rejected once it passes either its
// rolling idle deadline or the fixed absolute ceiling from when it was
// created, whichever comes first, or if the account it belongs to is no
// longer ACTIVE.
func (s *Service) ValidateSession(ctx context.Context, rawToken, ip, userAgent string) (*User, error) {
	if rawToken == "" {
		return nil, ErrSessionInvalid
	}
	tokenHash := hashSessionToken(rawToken)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		sessionID                         string
		userID, name, email, role, status string
		createdAt, expiresAt              time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT s.id, u.id, u.name, u.email, u.role, u.status, s.created_at, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		FOR UPDATE OF s
	`, tokenHash).Scan(&sessionID, &userID, &name, &email, &role, &status, &createdAt, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("looking up session: %w", err)
	}

	now := time.Now()
	if now.After(expiresAt) || now.After(createdAt.Add(sessionAbsoluteMax)) || status != "ACTIVE" {
		if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID); err != nil {
			return nil, fmt.Errorf("clearing invalid session: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing session cleanup: %w", err)
		}
		return nil, ErrSessionInvalid
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET expires_at = $2, last_seen_at = $3, ip_address = $4, user_agent = $5
		WHERE id = $1
	`, sessionID, now.Add(sessionIdleDuration), now, ip, userAgent); err != nil {
		return nil, fmt.Errorf("extending session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing session extension: %w", err)
	}

	return &User{ID: userID, Name: name, Email: email, Role: role}, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	tokenHash := hashSessionToken(rawToken)
	if _, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

func generateSessionToken() (rawToken string, tokenHash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	rawToken = hex.EncodeToString(buf)
	return rawToken, hashSessionToken(rawToken), nil
}

func hashSessionToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
