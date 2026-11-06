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

// ValidateAccentColor normalizes a hex color (lowercase, #-prefixed).
// Exported so internal/election applies the exact same rule to a
// per-election override instead of a second, possibly-drifting copy of
// it. An admin can pick any color here, including one that leaves white
// button text hard to read - that's a visual choice for them to notice
// and change, not something the backend blocks.
func ValidateAccentColor(hex string) (string, error) {
	return ValidateAccentColorFormat(hex)
}

// ValidateAccentColorFormat is the hex-format check shared by all three
// accent shades.
func ValidateAccentColorFormat(hex string) (string, error) {
	hex = strings.ToLower(strings.TrimSpace(hex))
	if !strings.HasPrefix(hex, "#") {
		hex = "#" + hex
	}
	if len(hex) != 7 {
		return "", ErrInvalidAccentColor
	}

	if _, err := strconv.ParseUint(hex[1:3], 16, 8); err != nil {
		return "", ErrInvalidAccentColor
	}
	if _, err := strconv.ParseUint(hex[3:5], 16, 8); err != nil {
		return "", ErrInvalidAccentColor
	}
	if _, err := strconv.ParseUint(hex[5:7], 16, 8); err != nil {
		return "", ErrInvalidAccentColor
	}

	return hex, nil
}
