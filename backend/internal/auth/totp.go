package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	pendingTOTPLoginDuration  = 5 * time.Minute
	totpLoginLockoutThreshold = 5
	recoveryCodeCount         = 8
)

var (
	ErrTOTPAlreadySetUp   = errors.New("a 2FA secret is already pending confirmation or already enabled")
	ErrTOTPSetupNotFound  = errors.New("start 2FA setup again before confirming")
	ErrTOTPAlreadyEnabled = errors.New("two-factor authentication is already enabled")
	ErrTOTPNotEnabled     = errors.New("two-factor authentication is not enabled on this account")
	ErrTOTPInvalidCode    = errors.New("that code didn't work - check your authenticator app and try again")
	ErrTOTPLoginExpired   = errors.New("this sign-in attempt has expired - please sign in again")
	ErrTOTPLoginLocked    = errors.New("too many failed attempts - please sign in again")
)

type TOTPSetup struct {
	Secret    string // base32, for manual entry if the camera won't scan
	URI       string // otpauth:// URI the QR code encodes
	QRCodePNG string // base64-encoded PNG, "" if image generation failed - manual entry still works either way
}

// BeginTOTPSetup generates a brand new secret and stores it unconfirmed.
// It deliberately doesn't turn 2FA on by itself - see ConfirmTOTPSetup -
// so starting setup can never be the thing that locks someone out; only
// proving the secret actually works can enable it. Calling this again
// before confirming just replaces the pending secret with a fresh one.
func (s *Service) BeginTOTPSetup(ctx context.Context, userID, accountEmail string) (*TOTPSetup, error) {
	var enabledAt *time.Time
	if err := s.pool.QueryRow(ctx, `SELECT totp_enabled_at FROM users WHERE id = $1`, userID).Scan(&enabledAt); err != nil {
		return nil, fmt.Errorf("checking existing totp status: %w", err)
	}
	if enabledAt != nil {
		return nil, ErrTOTPAlreadyEnabled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Voting System",
		AccountName: accountEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("generating totp secret: %w", err)
	}

	if _, err := s.pool.Exec(ctx, `
		UPDATE users SET totp_secret = $1, totp_enabled_at = NULL WHERE id = $2
	`, key.Secret(), userID); err != nil {
		return nil, fmt.Errorf("storing totp secret: %w", err)
	}

	setup := &TOTPSetup{Secret: key.Secret(), URI: key.URL()}
	if img, err := key.Image(240, 240); err == nil {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err == nil {
			setup.QRCodePNG = base64.StdEncoding.EncodeToString(buf.Bytes())
		}
	}
	return setup, nil
}

// ConfirmTOTPSetup is the only thing that actually enables 2FA, and only
// after validating a real code from the secret BeginTOTPSetup handed
// out - proof it made it into the user's authenticator app correctly.
// Mints recovery codes at the same time, returned raw exactly once;
// only their bcrypt hashes are ever stored.
func (s *Service) ConfirmTOTPSetup(ctx context.Context, userID, code string) ([]string, error) {
	var secret string
	var enabledAt *time.Time
	if err := s.pool.QueryRow(ctx, `
		SELECT totp_secret, totp_enabled_at FROM users WHERE id = $1
	`, userID).Scan(&secret, &enabledAt); err != nil {
		return nil, fmt.Errorf("looking up totp secret: %w", err)
	}
	if enabledAt != nil {
		return nil, ErrTOTPAlreadyEnabled
	}
	if secret == "" {
		return nil, ErrTOTPSetupNotFound
	}
	if !totp.Validate(code, secret) {
		return nil, ErrTOTPInvalidCode
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rawCodes := make([]string, recoveryCodeCount)
	for i := range rawCodes {
		raw, err := generateRecoveryCode()
		if err != nil {
			return nil, fmt.Errorf("generating recovery code: %w", err)
		}
		// Hash the normalized form (no hyphens) since that's what
		// checkTOTPOrRecoveryCode compares against later - the hyphens
		// in raw are purely cosmetic for display, never part of the
		// value that actually gets checked.
		hash, err := bcrypt.GenerateFromPassword([]byte(normalizeRecoveryCode(raw)), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hashing recovery code: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO totp_recovery_codes (user_id, code_hash) VALUES ($1, $2)
		`, userID, string(hash)); err != nil {
			return nil, fmt.Errorf("storing recovery code: %w", err)
		}
		rawCodes[i] = raw
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET totp_enabled_at = now() WHERE id = $1`, userID); err != nil {
		return nil, fmt.Errorf("enabling totp: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing totp confirmation: %w", err)
	}
	return rawCodes, nil
}

// DisableTOTP requires the current password - re-proving who you are,
// not just holding an open session - so a hijacked browser tab can't
// silently turn off someone's second factor on its own.
func (s *Service) DisableTOTP(ctx context.Context, userID, password string) error {
	var passwordHash string
	var enabledAt *time.Time
	if err := s.pool.QueryRow(ctx, `
		SELECT password_hash, totp_enabled_at FROM users WHERE id = $1
	`, userID).Scan(&passwordHash, &enabledAt); err != nil {
		return fmt.Errorf("looking up account: %w", err)
	}
	if enabledAt == nil {
		return ErrTOTPNotEnabled
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	return s.clearTOTP(ctx, userID)
}

// AdminResetTOTP is the escape hatch for someone who's lost both their
// authenticator device and their recovery codes, with nowhere else to
// turn - a SYSTEMADMIN clears it on their behalf so they can set it up
// fresh on their next login. Same shape as the voter-credential unlock:
// a security feature is only as good as the recovery path it has, or
// doesn't.
func (s *Service) AdminResetTOTP(ctx context.Context, userID string) (name, email string, err error) {
	var enabledAt *time.Time
	if err := s.pool.QueryRow(ctx, `
		SELECT name, email, totp_enabled_at FROM users WHERE id = $1
	`, userID).Scan(&name, &email, &enabledAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrTOTPNotEnabled
		}
		return "", "", fmt.Errorf("looking up account: %w", err)
	}
	if enabledAt == nil {
		return "", "", ErrTOTPNotEnabled
	}
	if err := s.clearTOTP(ctx, userID); err != nil {
		return "", "", err
	}
	return name, email, nil
}

func (s *Service) clearTOTP(ctx context.Context, userID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE users SET totp_secret = NULL, totp_enabled_at = NULL WHERE id = $1
	`, userID); err != nil {
		return fmt.Errorf("clearing totp secret: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM totp_recovery_codes WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clearing recovery codes: %w", err)
	}
	return tx.Commit(ctx)
}

// TOTPStatus is what the Security settings page renders - just enough
// to know whether to show "set up 2FA" or "2FA is on, here's how to turn
// it off", never the secret itself again once setup is confirmed.
type TOTPStatus struct {
	Enabled bool `json:"enabled"`
}

func (s *Service) GetTOTPStatus(ctx context.Context, userID string) (*TOTPStatus, error) {
	var enabledAt *time.Time
	if err := s.pool.QueryRow(ctx, `SELECT totp_enabled_at FROM users WHERE id = $1`, userID).Scan(&enabledAt); err != nil {
		return nil, fmt.Errorf("looking up totp status: %w", err)
	}
	return &TOTPStatus{Enabled: enabledAt != nil}, nil
}

// beginPendingTOTPLogin is called from Login once a password has
// checked out for an account with 2FA enabled. It deliberately creates
// no session - only CompleteTOTPLogin does that - so there is no path
// where a password alone, without a valid code, ever produces a usable
// session cookie for a 2FA-enabled account.
func (s *Service) beginPendingTOTPLogin(ctx context.Context, tx pgx.Tx, userID, ip, userAgent string) (string, error) {
	rawToken, tokenHash, err := generateSessionToken()
	if err != nil {
		return "", fmt.Errorf("generating pending login token: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO pending_totp_logins (user_id, token_hash, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, tokenHash, time.Now().Add(pendingTOTPLoginDuration), ip, userAgent); err != nil {
		return "", fmt.Errorf("creating pending login: %w", err)
	}
	return rawToken, nil
}

// CompleteTOTPLogin is the second step of login for a 2FA-enabled
// account: a valid 6-digit code or an unused recovery code, checked
// against the pending login beginPendingTOTPLogin created. Wrong codes
// count toward a lockout exactly like a wrong password does - a 6-digit
// code is only ~1,000,000 possibilities, so without this a code could be
// brute-forced inside its own 30-second validity window.
func (s *Service) CompleteTOTPLogin(ctx context.Context, pendingRawToken, code, ip, userAgent string) (*LoginResult, error) {
	if pendingRawToken == "" || code == "" {
		return nil, ErrTOTPLoginExpired
	}
	tokenHash := hashSessionToken(pendingRawToken)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		pendingID, userID string
		failedAttempts    int
		expiresAt         time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, user_id, failed_attempts, expires_at FROM pending_totp_logins
		WHERE token_hash = $1 FOR UPDATE
	`, tokenHash).Scan(&pendingID, &userID, &failedAttempts, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTOTPLoginExpired
	}
	if err != nil {
		return nil, fmt.Errorf("looking up pending login: %w", err)
	}
	if time.Now().After(expiresAt) {
		return nil, ErrTOTPLoginExpired
	}

	var (
		name, email, role, status, secret string
	)
	if err := tx.QueryRow(ctx, `
		SELECT name, email, role, status, totp_secret FROM users WHERE id = $1
	`, userID).Scan(&name, &email, &role, &status, &secret); err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}
	if status != "ACTIVE" {
		if _, err := tx.Exec(ctx, `DELETE FROM pending_totp_logins WHERE id = $1`, pendingID); err != nil {
			return nil, fmt.Errorf("clearing pending login: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing: %w", err)
		}
		if status == "DISABLED" {
			return nil, ErrAccountDisabled
		}
		return nil, ErrAccountNotVerified
	}

	ok, recoveryCodeID, err := s.checkTOTPOrRecoveryCode(ctx, tx, userID, secret, code)
	if err != nil {
		return nil, err
	}
	if !ok {
		newAttempts := failedAttempts + 1
		if newAttempts >= totpLoginLockoutThreshold {
			if _, err := tx.Exec(ctx, `DELETE FROM pending_totp_logins WHERE id = $1`, pendingID); err != nil {
				return nil, fmt.Errorf("clearing locked pending login: %w", err)
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("committing: %w", err)
			}
			return nil, ErrTOTPLoginLocked
		}
		if _, err := tx.Exec(ctx, `
			UPDATE pending_totp_logins SET failed_attempts = $1 WHERE id = $2
		`, newAttempts, pendingID); err != nil {
			return nil, fmt.Errorf("recording failed totp attempt: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing: %w", err)
		}
		return nil, ErrTOTPInvalidCode
	}

	if recoveryCodeID != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE totp_recovery_codes SET used_at = now() WHERE id = $1
		`, recoveryCodeID); err != nil {
			return nil, fmt.Errorf("marking recovery code used: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM pending_totp_logins WHERE id = $1`, pendingID); err != nil {
		return nil, fmt.Errorf("clearing completed pending login: %w", err)
	}

	rawToken, sessionExpiresAt, err := s.createSession(ctx, tx, userID, ip, userAgent)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing totp login: %w", err)
	}

	result := &LoginResult{
		User:             User{ID: userID, Name: name, Email: email, Role: role},
		RawToken:         rawToken,
		ExpiresAt:        sessionExpiresAt,
		UsedRecoveryCode: recoveryCodeID != "",
	}
	return result, nil
}

// checkTOTPOrRecoveryCode tries the code as a live TOTP code first, then
// falls back to matching it against any unused recovery code - the two
// never collide in practice (6 digits vs. a longer alphanumeric string),
// so trying both costs nothing and keeps the login form a single field.
func (s *Service) checkTOTPOrRecoveryCode(ctx context.Context, tx pgx.Tx, userID, secret, code string) (ok bool, usedRecoveryCodeID string, err error) {
	if totp.Validate(code, secret) {
		return true, "", nil
	}

	rows, err := tx.Query(ctx, `
		SELECT id, code_hash FROM totp_recovery_codes WHERE user_id = $1 AND used_at IS NULL
	`, userID)
	if err != nil {
		return false, "", fmt.Errorf("reading recovery codes: %w", err)
	}
	defer rows.Close()

	normalized := normalizeRecoveryCode(code)
	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return false, "", fmt.Errorf("scanning recovery code: %w", err)
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(normalized)) == nil {
			rows.Close()
			return true, id, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, "", fmt.Errorf("iterating recovery codes: %w", err)
	}
	return false, "", nil
}

// normalizeRecoveryCode strips the cosmetic hyphens and case-folds a
// recovery code, so "lmh5-vs3i-2zrh-a6rx" and "LMH5VS3I2ZRHA6RX" hash and
// compare identically - the grouping in generateRecoveryCode's output is
// purely for the user to read and type, never part of the stored value.
func normalizeRecoveryCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
}

func generateRecoveryCode() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	raw := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)[:16]
	// Grouped for readability, same convention as how these are usually
	// shown (e.g. backup codes in most authenticator setups) - purely
	// cosmetic, normalized away again before comparison.
	return raw[:4] + "-" + raw[4:8] + "-" + raw[8:12] + "-" + raw[12:16], nil
}
