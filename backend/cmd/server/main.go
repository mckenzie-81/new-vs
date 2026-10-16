package main

import (
	"context"
	"log"
	"net/http"

	"backend/internal/auth"
	"backend/internal/bootstrap"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/httpx"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection error: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	bootstrapService := bootstrap.NewService(pool, cfg.BootstrapSecret)
	bootstrapHandler := bootstrap.NewHandler(bootstrapService, pool, cfg.AppBaseURL)

	authService := auth.NewService(pool)
	authHandler := auth.NewHandler(authService, pool, cfg.CookieSecure)

	mux := http.NewServeMux()
	bootstrapHandler.Register(mux)
	authHandler.Register(mux)

	handler := httpx.CORS(cfg.AppBaseURL, mux)

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
