package main

import (
	"context"
	"log"
	"net/http"

	"backend/internal/auth"
	"backend/internal/bootstrap"
	"backend/internal/branding"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/election"
	"backend/internal/electorate"
	"backend/internal/httpx"
	"backend/internal/pulse"
	"backend/internal/voter"
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

	authService := auth.NewService(pool, cfg.CookieSecure)
	authHandler := auth.NewHandler(authService, pool, cfg.CookieSecure, cfg.AppBaseURL)

	bootstrapService := bootstrap.NewService(pool, cfg.BootstrapSecret)
	bootstrapHandler := bootstrap.NewHandler(bootstrapService, authService, pool, cfg.AppBaseURL)

	electorateService := electorate.NewService(pool)
	electorateHandler := electorate.NewHandler(electorateService, authService, pool)

	electionService := election.NewService(pool, electorateService)
	electionHandler := election.NewHandler(electionService, authService, pool)

	voterService := voter.NewService(pool, electionService, electorateService)
	voterHandler := voter.NewHandler(voterService, electionService, authService, pool, cfg.AppBaseURL, cfg.CookieSecure)

	pulseService := pulse.NewService(pool, electionService, electorateService)
	pulseHandler := pulse.NewHandler(pulseService)

	brandingService := branding.NewService(pool)
	brandingHandler := branding.NewHandler(brandingService, authService)

	mux := http.NewServeMux()
	bootstrapHandler.Register(mux)
	authHandler.Register(mux)
	electionHandler.Register(mux)
	electorateHandler.Register(mux)
	voterHandler.Register(mux)
	pulseHandler.Register(mux)
	brandingHandler.Register(mux)

	handler := httpx.CORS(cfg.AppBaseURL, mux)

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
