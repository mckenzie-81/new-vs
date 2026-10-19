package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"backend/internal/audit"
	"backend/internal/auditlog"
	"backend/internal/auth"
	"backend/internal/bootstrap"
	"backend/internal/branding"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/election"
	"backend/internal/electorate"
	"backend/internal/httpx"
	"backend/internal/pulse"
	"backend/internal/storage"
	"backend/internal/voter"

	"github.com/jackc/pgx/v5/pgxpool"
)

const auditIntegrityCheckInterval = 5 * time.Minute

// runAuditIntegrityChecks is what makes verification and the external
// anchor actually automatic, instead of only ever running when an admin
// remembers to click a button: every few minutes it re-verifies the
// chain, cross-checks it against every checkpoint taken so far, records
// an alert the moment either disagrees (surfaced on every dashboard page
// via audit.LatestUnacknowledgedAlert), and only then takes this run's
// own checkpoint. Runs for the life of the process; errors are logged,
// never fatal - a missed check just means a slightly longer gap a
// rewrite could hide inside, not a reason to take the server down.
func runAuditIntegrityChecks(ctx context.Context, pool *pgxpool.Pool, path string) {
	if err := audit.RunIntegrityCheck(ctx, pool, path); err != nil {
		log.Printf("audit integrity check failed: %v", err)
	}

	ticker := time.NewTicker(auditIntegrityCheckInterval)
	defer ticker.Stop()
	for range ticker.C {
		if err := audit.RunIntegrityCheck(ctx, pool, path); err != nil {
			log.Printf("audit integrity check failed: %v", err)
		}
	}
}

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

	pulseService := pulse.NewService(pool, electionService, electorateService, voterService)
	pulseHandler := pulse.NewHandler(pulseService, authService, electionService)

	brandingService := branding.NewService(pool)
	brandingHandler := branding.NewHandler(brandingService, authService)

	storageService := storage.NewService(cfg)
	storageHandler := storage.NewHandler(storageService, authService)
	if storageService == nil {
		log.Printf("uploads disabled: R2 isn't configured (set R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET, R2_PUBLIC_BASE_URL)")
	}

	auditLogHandler := auditlog.NewHandler(pool, authService, cfg.AuditCheckpointPath)
	go runAuditIntegrityChecks(ctx, pool, cfg.AuditCheckpointPath)

	mux := http.NewServeMux()
	bootstrapHandler.Register(mux)
	authHandler.Register(mux)
	electionHandler.Register(mux)
	electorateHandler.Register(mux)
	voterHandler.Register(mux)
	pulseHandler.Register(mux)
	brandingHandler.Register(mux)
	storageHandler.Register(mux)
	auditLogHandler.Register(mux)

	handler := httpx.CORS(cfg.AppBaseURL, mux)

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
