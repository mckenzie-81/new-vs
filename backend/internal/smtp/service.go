// Package smtp stores an institution's own SMTP credentials (their
// existing Google Workspace/Office 365/etc. account) so mail goes out
// genuinely as them - passes their own SPF/DKIM, lands in their own sent
// mail, costs this platform nothing. Nothing in this package wires that
// sending into the platform's actual notification flows yet (election
// submitted/approved/rejected, admin assigned, voter credentials, etc.
// are all still log-only stubs elsewhere) - this is just the settings
// storage, the test-mode guard, and a way for an admin to confirm their
// own credentials actually work before anything real depends on them.
//
// TestMode exists specifically to decouple "SMTP is configured" from
// "we're now mailing real people": while it's on (the default), nothing
// outside this package's own SendTestEmail ever sends real mail,
// regardless of what's configured. Only SendTestEmail bypasses it, and
// only ever to the admin's own address, confirming their own session, by
// its own design - never an arbitrary recipient.
package smtp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wneessen/go-mail"
)

var (
	ErrInvalidHost      = errors.New("host is required")
	ErrInvalidPort      = errors.New("port must be between 1 and 65535")
	ErrInvalidFromEmail = errors.New("from email must look like a real email address")
	ErrNotConfigured    = errors.New("SMTP isn't configured yet")
)

type Settings struct {
	Host        *string   `json:"host"`
	Port        *int      `json:"port"`
	Username    *string   `json:"username"`
	HasPassword bool      `json:"hasPassword"`
	FromName    *string   `json:"fromName"`
	FromEmail   *string   `json:"fromEmail"`
	TestMode    bool      `json:"testMode"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Service struct {
	pool          *pgxpool.Pool
	credentialKey []byte
}

// NewService never fails outright - an absent or malformed
// SMTP_CREDENTIAL_KEY just means credentialKey is nil, which Update
// turns into ErrCredentialKeyNotConfigured only if an admin actually
// tries to save a password. Everything else (reading settings, toggling
// test mode) works regardless.
func NewService(pool *pgxpool.Pool, credentialKeyBase64 string) *Service {
	return &Service{pool: pool, credentialKey: decodeKey(credentialKeyBase64)}
}

func (s *Service) Get(ctx context.Context) (*Settings, error) {
	var settings Settings
	var passwordEncrypted *string
	if err := s.pool.QueryRow(ctx, `
		SELECT smtp_host, smtp_port, smtp_username, smtp_password_encrypted,
		       smtp_from_name, smtp_from_email, smtp_test_mode, updated_at
		FROM system_settings WHERE id = true
	`).Scan(
		&settings.Host, &settings.Port, &settings.Username, &passwordEncrypted,
		&settings.FromName, &settings.FromEmail, &settings.TestMode, &settings.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("getting smtp settings: %w", err)
	}
	settings.HasPassword = passwordEncrypted != nil && *passwordEncrypted != ""
	return &settings, nil
}

// Update takes raw form input. An empty host clears SMTP configuration
// entirely, same empty-means-unset convention the branding settings
// use. An empty password leaves whatever's already stored untouched -
// there's no reason to force re-entering a working credential just to
// change the from name, and the password is never sent back to the
// browser to be re-submitted unchanged in the first place.
func (s *Service) Update(ctx context.Context, host string, port int, username, password, fromName, fromEmail string, testMode bool) (*Settings, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		if _, err := s.pool.Exec(ctx, `
			UPDATE system_settings SET
				smtp_host = NULL, smtp_port = NULL, smtp_username = NULL,
				smtp_password_encrypted = NULL, smtp_from_name = NULL, smtp_from_email = NULL,
				smtp_test_mode = $1, updated_at = now()
			WHERE id = true
		`, testMode); err != nil {
			return nil, fmt.Errorf("clearing smtp settings: %w", err)
		}
		return s.Get(ctx)
	}

	if port < 1 || port > 65535 {
		return nil, ErrInvalidPort
	}
	fromEmail = strings.TrimSpace(fromEmail)
	if fromEmail == "" || !strings.Contains(fromEmail, "@") {
		return nil, ErrInvalidFromEmail
	}

	var passwordEncrypted *string
	if password != "" {
		if s.credentialKey == nil {
			return nil, ErrCredentialKeyNotConfigured
		}
		encrypted, err := encrypt(s.credentialKey, password)
		if err != nil {
			return nil, fmt.Errorf("encrypting password: %w", err)
		}
		passwordEncrypted = &encrypted
	}

	username = strings.TrimSpace(username)
	fromName = strings.TrimSpace(fromName)

	var query string
	var args []any
	if passwordEncrypted != nil {
		query = `
			UPDATE system_settings SET
				smtp_host = $1, smtp_port = $2, smtp_username = $3, smtp_password_encrypted = $4,
				smtp_from_name = $5, smtp_from_email = $6, smtp_test_mode = $7, updated_at = now()
			WHERE id = true
		`
		args = []any{host, port, username, *passwordEncrypted, fromName, fromEmail, testMode}
	} else {
		// Deliberately doesn't touch smtp_password_encrypted - see the
		// doc comment above.
		query = `
			UPDATE system_settings SET
				smtp_host = $1, smtp_port = $2, smtp_username = $3,
				smtp_from_name = $4, smtp_from_email = $5, smtp_test_mode = $6, updated_at = now()
			WHERE id = true
		`
		args = []any{host, port, username, fromName, fromEmail, testMode}
	}
	if _, err := s.pool.Exec(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("updating smtp settings: %w", err)
	}

	return s.Get(ctx)
}

// SendTestEmail is the one path in this entire package that sends real
// mail, deliberately independent of TestMode - verifying your own
// SMTP credentials actually work is not the same thing as mailing real
// voters, and an admin confirming their own session's address is not an
// open relay: toEmail is never taken from request input.
func (s *Service) SendTestEmail(ctx context.Context, toEmail string) error {
	var host, username, passwordEncrypted, fromName, fromEmail string
	var port int
	err := s.pool.QueryRow(ctx, `
		SELECT smtp_host, smtp_port, smtp_username, smtp_password_encrypted, smtp_from_name, smtp_from_email
		FROM system_settings WHERE id = true
	`).Scan(&host, &port, &username, &passwordEncrypted, &fromName, &fromEmail)
	if err != nil {
		return fmt.Errorf("reading smtp settings: %w", err)
	}
	if host == "" {
		return ErrNotConfigured
	}
	if s.credentialKey == nil {
		return ErrCredentialKeyNotConfigured
	}
	password, err := decrypt(s.credentialKey, passwordEncrypted)
	if err != nil {
		return fmt.Errorf("decrypting password: %w", err)
	}

	opts := []mail.Option{
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTimeout(15 * time.Second),
	}
	if port == 465 {
		opts = append(opts, mail.WithSSL())
	}
	client, err := mail.NewClient(host, opts...)
	if err != nil {
		return fmt.Errorf("creating smtp client: %w", err)
	}

	msg := mail.NewMsg()
	display := fromEmail
	if fromName != "" {
		display = fromName
	}
	if err := msg.FromFormat(display, fromEmail); err != nil {
		return fmt.Errorf("setting from address: %w", err)
	}
	if err := msg.To(toEmail); err != nil {
		return fmt.Errorf("setting recipient: %w", err)
	}
	msg.Subject("Test email from your voting system")
	msg.SetBodyString(mail.TypeTextPlain,
		"This confirms your SMTP settings are working. Nothing else uses this connection yet - "+
			"real voter and admin notifications are still being tested before they're switched on.")

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("sending test email: %w", err)
	}
	return nil
}
