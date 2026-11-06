// Package branding holds the system-wide default look (a three-shade
// accent palette, logo) that any election not supplying its own falls
// back to. It's deliberately narrow: only accent shades and a logo URL,
// nothing that touches the neutral text/background palette or anything
// carrying semantic meaning (success/warning/danger) - those stay fixed
// regardless of brand, since they're relied on to mean the same thing
// everywhere. Three shades, not one, so a brand reads as more than a
// single flat color (primary action, secondary action, subtle accents),
// while staying inside that same accent-only guarantee.
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
	ErrInvalidAccentColor  = errors.New("accent color must be a hex color like #2e4e8f")
	ErrInvalidLogoURL      = errors.New("logo URL is too long")
	ErrInvalidPlatformName = errors.New("platform name is too long")
)

const maxLogoURLLength = 2048
const maxPlatformNameLength = 60

type Settings struct {
	AccentColor          *string   `json:"accentColor"`
	AccentColorSecondary *string   `json:"accentColorSecondary"`
	AccentColorTertiary  *string   `json:"accentColorTertiary"`
	LogoURL              *string   `json:"logoUrl"`
	PlatformName         *string   `json:"platformName"`
	UpdatedAt            time.Time `json:"updatedAt"`
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
		SELECT accent_color, accent_color_secondary, accent_color_tertiary, logo_url, platform_name, updated_at
		FROM system_settings WHERE id = true
	`).Scan(
		&settings.AccentColor, &settings.AccentColorSecondary, &settings.AccentColorTertiary,
		&settings.LogoURL, &settings.PlatformName, &settings.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("getting system settings: %w", err)
	}
	return &settings, nil
}

// Update takes raw form input - an empty color or logoUrl clears that
// field back to unset (no override), matching the same
// empty-string-means-unset convention already used for an election's
// electorate database link. The secondary and tertiary shades are each
// independently optional - a brand might only have one color decided and
// that's fine, it just means those slots inherit nothing special.
func (s *Service) Update(ctx context.Context, accentColor, accentColorSecondary, accentColorTertiary, logoURL, platformName string) (*Settings, error) {
	accent, err := normalizeOptionalColor(accentColor, ValidateAccentColor)
	if err != nil {
		return nil, err
	}
	secondary, err := normalizeOptionalColor(accentColorSecondary, ValidateAccentColorFormat)
	if err != nil {
		return nil, err
	}
	tertiary, err := normalizeOptionalColor(accentColorTertiary, ValidateAccentColorFormat)
	if err != nil {
		return nil, err
	}

	var logo *string
	if trimmed := strings.TrimSpace(logoURL); trimmed != "" {
		if len(trimmed) > maxLogoURLLength {
			return nil, ErrInvalidLogoURL
		}
		logo = &trimmed
	}

	var name *string
	if trimmed := strings.TrimSpace(platformName); trimmed != "" {
		if len(trimmed) > maxPlatformNameLength {
			return nil, ErrInvalidPlatformName
		}
		name = &trimmed
	}

	var settings Settings
	if err := s.pool.QueryRow(ctx, `
		UPDATE system_settings SET
			accent_color = $1, accent_color_secondary = $2, accent_color_tertiary = $3,
			logo_url = $4, platform_name = $5, updated_at = now()
		WHERE id = true
		RETURNING accent_color, accent_color_secondary, accent_color_tertiary, logo_url, platform_name, updated_at
	`, accent, secondary, tertiary, logo, name).Scan(
		&settings.AccentColor, &settings.AccentColorSecondary, &settings.AccentColorTertiary,
		&settings.LogoURL, &settings.PlatformName, &settings.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("updating system settings: %w", err)
	}
	return &settings, nil
}

func normalizeOptionalColor(hex string, validate func(string) (string, error)) (*string, error) {
	trimmed := strings.TrimSpace(hex)
	if trimmed == "" {
		return nil, nil
	}
	normalized, err := validate(trimmed)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

// ValidateAccentColor normalizes a hex color (lowercase, #-prefixed) and
// rejects one that wouldn't leave white button text readable on top of
// it. Exported so internal/election applies the exact same rule to a
// per-election override instead of a second, possibly-drifting copy of
// it. Reserved for the one primary accent shade, which is the only one
// actually used as a white-text button background (see Button's accent
// variant) - the secondary/tertiary shades use ValidateAccentColorFormat
// instead, since neither ever carries white text today.
func ValidateAccentColor(hex string) (string, error) {
	normalized, r, g, b, err := normalizeHexColor(hex)
	if err != nil {
		return "", err
	}
	if contrastAgainstWhite(r, g, b) < minContrastRatio {
		return "", ErrInvalidAccentColor
	}
	return normalized, nil
}

// ValidateAccentColorFormat is the same hex normalization as
// ValidateAccentColor, without the white-text contrast requirement - for
// a secondary/tertiary brand shade that's only ever used as text, a
// border, or a background with no text over it.
func ValidateAccentColorFormat(hex string) (string, error) {
	normalized, _, _, _, err := normalizeHexColor(hex)
	return normalized, err
}

func normalizeHexColor(hex string) (normalized string, r, g, b float64, err error) {
	hex = strings.ToLower(strings.TrimSpace(hex))
	if !strings.HasPrefix(hex, "#") {
		hex = "#" + hex
	}
	if len(hex) != 7 {
		return "", 0, 0, 0, ErrInvalidAccentColor
	}

	ri, err1 := strconv.ParseUint(hex[1:3], 16, 8)
	gi, err2 := strconv.ParseUint(hex[3:5], 16, 8)
	bi, err3 := strconv.ParseUint(hex[5:7], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return "", 0, 0, 0, ErrInvalidAccentColor
	}

	return hex, float64(ri), float64(gi), float64(bi), nil
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
