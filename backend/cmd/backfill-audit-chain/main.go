// backfill-audit-chain computes previous_hash/entry_hash for every
// audit_log row written before chaining existed. Run once, after
// migration 0021 and before relying on audit.VerifyChain.
package main

import (
	"context"
	"log"

	"backend/internal/audit"
	"backend/internal/config"
	"backend/internal/db"
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

	n, err := audit.BackfillChain(ctx, pool)
	if err != nil {
		log.Fatalf("backfill failed: %v", err)
	}
	log.Printf("backfilled %d audit log entries", n)
}
