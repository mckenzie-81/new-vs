package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL     string
	BootstrapSecret string
	Port            string
	AppBaseURL      string
	CookieSecure    bool

	// R2* configure uploads (candidate photos, logos) to Cloudflare R2.
	// Deliberately optional - an unconfigured deployment still starts and
	// serves everything else; only the upload endpoint itself refuses
	// requests until these are set, via storage.ErrNotConfigured.
	R2AccountID     string
	R2AccessKeyID   string
	R2SecretKey     string
	R2Bucket        string
	R2PublicBaseURL string

	// AuditCheckpointPath is a local file the audit chain's head hash gets
	// appended to on a timer, outside Postgres entirely - see
	// internal/audit's package doc for why that matters.
	AuditCheckpointPath string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		BootstrapSecret: os.Getenv("BOOTSTRAP_SECRET"),
		Port:            os.Getenv("PORT"),
		AppBaseURL:      os.Getenv("APP_BASE_URL"),
		R2AccountID:     os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:   os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretKey:     os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:        os.Getenv("R2_BUCKET"),
		R2PublicBaseURL: strings.TrimRight(os.Getenv("R2_PUBLIC_BASE_URL"), "/"),

		AuditCheckpointPath: os.Getenv("AUDIT_CHECKPOINT_PATH"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.BootstrapSecret == "" {
		return nil, fmt.Errorf("BOOTSTRAP_SECRET is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.AppBaseURL == "" {
		cfg.AppBaseURL = "http://localhost:3000"
	}
	cfg.CookieSecure = strings.HasPrefix(cfg.AppBaseURL, "https://")
	if cfg.AuditCheckpointPath == "" {
		cfg.AuditCheckpointPath = "./audit-checkpoints.log"
	}

	return cfg, nil
}
