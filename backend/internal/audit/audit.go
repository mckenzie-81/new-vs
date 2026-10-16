package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

type Entry struct {
	ActorUserID *string
	Action      string
	Metadata    map[string]any
	IPAddress   string
	UserAgent   string
}

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, so audit entries
// can be written either standalone or as part of a caller's transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func Log(ctx context.Context, q Querier, e Entry) error {
	metadata := e.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshaling audit metadata: %w", err)
	}

	_, err = q.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, metadata, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5)
	`, e.ActorUserID, e.Action, metadataJSON, e.IPAddress, e.UserAgent)
	if err != nil {
		return fmt.Errorf("inserting audit log entry: %w", err)
	}

	return nil
}
