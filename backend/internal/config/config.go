package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL     string
	BootstrapSecret string
	Port            string
	AppBaseURL      string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		BootstrapSecret: os.Getenv("BOOTSTRAP_SECRET"),
		Port:            os.Getenv("PORT"),
		AppBaseURL:      os.Getenv("APP_BASE_URL"),
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

	return cfg, nil
}
