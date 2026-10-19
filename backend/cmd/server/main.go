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

const auditCheckpointInterval = 5 * time.Minute

// runAuditCheckpoints anchors the audit chain's head outside Postgres on
// a timer - see internal/audit's package doc for why VerifyChain alone,
// with no anchor, can't catch a sufficiently thorough database rewrite.
// Runs for the life of the process; errors are logged, never fatal - a
// missed checkpoint just means a slightly longer gap an attacker's
// rewrite could hide inside, not a reason to take the server down.
func runAuditCheckpoints(ctx context.Context, pool *pgxpool.Pool, path string) {
	if cp, err := audit.WriteCheckpoint(ctx, pool, path); err != nil {
		log.Printf("audit checkpoint failed: %v", err)
	} else if cp != nil {
		log.Printf("audit checkpoint: seq=%d", cp.Seq)
	}

	ticker := time.NewTicker(auditCheckpointInterval)
	defer ticker.Stop()
	for range ticker.C {
		if cp, err := audit.WriteCheckpoint(ctx, pool, path); err != nil {
			log.Printf("audit checkpoint failed: %v", err)
		} else if cp != nil {
			log.Printf("audit checkpoint: seq=%d", cp.Seq)
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
	go runAuditCheckpoints(ctx, pool, cfg.AuditCheckpointPath)

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
