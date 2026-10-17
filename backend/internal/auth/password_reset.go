package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	passwordResetTTL  = 1 * time.Hour
	minPasswordLength = 12
)

var (
	ErrInvalidResetToken = errors.New("invalid password reset token")
	ErrResetTokenExpired = errors.New("password reset token expired")
	ErrPasswordTooShort  = fmt.Errorf("password must be at least %d characters", minPasswordLength)
)

type PasswordResetRequestResult struct {
	UserID   string
	Name     string
	Email    string
	RawToken string
}

// RequestPasswordReset issues a reset token for an active account with this
// email. A nil result with a nil error means no matching active account
// exists - callers must give an identical response either way, so that
// whether an email is registered is never observable from the outside.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (*PasswordResetRequestResult, error) {
	rawToken, tokenHash, err := generateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("generating reset token: %w", err)
	}
	expiresAt := time.Now().Add(passwordResetTTL)

	var result PasswordResetRequestResult
	err = s.pool.QueryRow(ctx, `
		UPDATE users
		SET password_reset_token_hash = $2, password_reset_expires_at = $3
		WHERE lower(email) = lower($1) AND status = 'ACTIVE'
		RETURNING id, name, email
	`, email, tokenHash, expiresAt).Scan(&result.UserID, &result.Name, &result.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("requesting password reset: %w", err)
	}
	result.RawToken = rawToken

	return &result, nil
}

type PasswordResetConfirmResult struct {
	UserID string
	Name   string
	Email  string
}

// ConfirmPasswordReset sets a new password and, since the account may have
// been compromised, also signs out every existing session for it - a
// bootstrap or invite confirmation never needs to do this, because no
// sessions exist yet for an account that isn't active.
func (s *Service) ConfirmPasswordReset(ctx context.Context, rawToken, newPassword string) (*PasswordResetConfirmResult, error) {
	if len(newPassword) < minPasswordLength {
		return nil, ErrPasswordTooShort
	}

	tokenHash := hashSessionToken(rawToken)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var userID, name, email string
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, name, email, password_reset_expires_at
		FROM users
		WHERE password_reset_token_hash = $1
		FOR UPDATE
	`, tokenHash).Scan(&userID, &name, &email, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidResetToken
	}
	if err != nil {
		return nil, fmt.Errorf("looking up reset token: %w", err)
	}

	if time.Now().After(expiresAt) {
		if _, err := tx.Exec(ctx, `
			UPDATE users SET password_reset_token_hash = NULL, password_reset_expires_at = NULL WHERE id = $1
		`, userID); err != nil {
			return nil, fmt.Errorf("clearing expired reset token: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing expired reset cleanup: %w", err)
		}
		return nil, ErrResetTokenExpired
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing new password: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, password_reset_token_hash = NULL, password_reset_expires_at = NULL,
			failed_login_attempts = 0, locked_until = NULL
		WHERE id = $1
	`, userID, newHash); err != nil {
		return nil, fmt.Errorf("updating password: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
		return nil, fmt.Errorf("clearing sessions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing password reset: %w", err)
	}

	return &PasswordResetConfirmResult{UserID: userID, Name: name, Email: email}, nil
}
