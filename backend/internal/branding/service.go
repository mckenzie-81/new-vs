// Package branding holds the system-wide default look (accent color,
// logo) that any election not supplying its own falls back to. It's
// deliberately narrow: only an accent color and a logo URL, nothing that
// touches the neutral text/background palette or anything carrying
// semantic meaning (success/warning/danger) - those stay fixed regardless
// of brand, since they're relied on to mean the same thing everywhere.
package branding

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidAccentColor = errors.New("accent color must be a hex color like #2e4e8f with enough contrast for white button text")
	ErrInvalidLogoURL     = errors.New("logo URL is too long")
)

const maxLogoURLLength = 2048

// minContrastRatio is the WCAG AA threshold for normal text - the accent
// color is always used as a background with white text on top (see
// Button's accent variant), so this is the one real accessibility risk an
// accent-only override doesn't already eliminate on its own.
const minContrastRatio = 4.5

type Settings struct {
	AccentColor *string   `json:"accentColor"`
	LogoURL     *string   `json:"logoUrl"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Get(ctx context.Context) (*Settings, error) {
	var settings Settings
	if err := s.pool.QueryRow(ctx, `
		SELECT accent_color, logo_url, updated_at FROM system_settings WHERE id = true
	`).Scan(&settings.AccentColor, &settings.LogoURL, &settings.UpdatedAt); err != nil {
		return nil, fmt.Errorf("getting system settings: %w", err)
	}
	return &settings, nil
}

// Update takes raw form input - an empty accentColor or logoUrl clears
// that field back to unset (no override), matching the same
// empty-string-means-unset convention already used for an election's
// electorate database link.
func (s *Service) Update(ctx context.Context, accentColor, logoURL string) (*Settings, error) {
	var accent *string
	if trimmed := strings.TrimSpace(accentColor); trimmed != "" {
		normalized, err := ValidateAccentColor(trimmed)
		if err != nil {
			return nil, err
		}
		accent = &normalized
	}

	var logo *string
	if trimmed := strings.TrimSpace(logoURL); trimmed != "" {
		if len(trimmed) > maxLogoURLLength {
			return nil, ErrInvalidLogoURL
		}
		logo = &trimmed
	}

	var settings Settings
	if err := s.pool.QueryRow(ctx, `
		UPDATE system_settings SET accent_color = $1, logo_url = $2, updated_at = now() WHERE id = true
		RETURNING accent_color, logo_url, updated_at
	`, accent, logo).Scan(&settings.AccentColor, &settings.LogoURL, &settings.UpdatedAt); err != nil {
		return nil, fmt.Errorf("updating system settings: %w", err)
	}
	return &settings, nil
}

// ValidateAccentColor normalizes a hex color (lowercase, #-prefixed) and
// rejects one that wouldn't leave white button text readable on top of
// it. Exported so internal/election applies the exact same rule to a
// per-election override instead of a second, possibly-drifting copy of it.
func ValidateAccentColor(hex string) (string, error) {
	hex = strings.ToLower(strings.TrimSpace(hex))
	if !strings.HasPrefix(hex, "#") {
		hex = "#" + hex
	}
	if len(hex) != 7 {
		return "", ErrInvalidAccentColor
	}

	r, err1 := strconv.ParseUint(hex[1:3], 16, 8)
	g, err2 := strconv.ParseUint(hex[3:5], 16, 8)
	b, err3 := strconv.ParseUint(hex[5:7], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return "", ErrInvalidAccentColor
	}

	if contrastAgainstWhite(float64(r), float64(g), float64(b)) < minContrastRatio {
		return "", ErrInvalidAccentColor
	}

	return hex, nil
}

func contrastAgainstWhite(r, g, b float64) float64 {
	luminance := relativeLuminance(r, g, b)
	// White's own relative luminance is exactly 1.0.
	return (1.0 + 0.05) / (luminance + 0.05)
}

func relativeLuminance(r, g, b float64) float64 {
	adjust := func(c float64) float64 {
		c = c / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*adjust(r) + 0.7152*adjust(g) + 0.0722*adjust(b)
}
