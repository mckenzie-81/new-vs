// Package audit writes and verifies the platform's hash-chained audit
// log. Every entry's hash folds in the previous entry's hash (entry_hash
// = SHA256(previous_hash | actor | action | metadata | ip | user_agent |
// timestamp)), so altering, reordering, or deleting a past entry breaks
// every hash after it - detectable even by someone with direct database
// write access, not just through this application. VerifyChain re-walks
// the whole table and reports the first point, if any, where that no
// longer holds.
//
// VerifyChain alone has a real limit, though: it only checks that the
// chain is internally self-consistent. Someone with full database write
// access, who understands this scheme, could rewrite a past entry *and*
// recompute every hash after it to keep the chain consistent -
// VerifyChain would report that forged history as perfectly valid,
// because by its own rules it is. A hash chain only proves "nothing
// changed since X" if X is anchored somewhere that attacker doesn't also
// control. checkpoint.go is that anchor: it periodically appends the
// current chain head to a local file outside Postgres entirely, and
// VerifyAgainstCheckpoints cross-checks old checkpoints against what the
// database says today - a later rewrite changes the database's answer
// but can't reach back and silently edit a line already written to that
// file.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Entry struct {
	ActorUserID *string
	Action      string
	Metadata    map[string]any
	IPAddress   string
	UserAgent   string
}

// Log writes one chained audit entry as part of tx - the "read the latest
// hash, then insert" pair must be one atomic unit, or two concurrent
// entries could both chain from the same predecessor and silently fork
// the chain. Callers already inside their own transaction (e.g. casting a
// vote) pass that tx directly; everyone else goes through LogStandalone.
func Log(ctx context.Context, tx pgx.Tx, e Entry) error {
	metadataJSON, createdAt, err := prepare(e)
	if err != nil {
		return err
	}

	// FOR UPDATE on the tail row serializes concurrent appends: a second
	// transaction reading the same tail blocks here until the first
	// commits, then (under READ COMMITTED) re-reads and sees the new
	// tail - so chains never fork even under concurrent writes.
	var previousHash *string
	err = tx.QueryRow(ctx, `
		SELECT entry_hash FROM audit_log ORDER BY seq DESC LIMIT 1 FOR UPDATE
	`).Scan(&previousHash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("locking latest audit entry: %w", err)
	}

	entryHash := computeEntryHash(previousHash, e.ActorUserID, e.Action, metadataJSON, e.IPAddress, e.UserAgent, createdAt)

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, metadata, ip_address, user_agent, created_at, previous_hash, entry_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, e.ActorUserID, e.Action, metadataJSON, e.IPAddress, e.UserAgent, createdAt, previousHash, entryHash); err != nil {
		return fmt.Errorf("inserting audit log entry: %w", err)
	}

	return nil
}

// LogStandalone is Log for callers with no transaction of their own - it
// opens one just to make the read-latest-then-insert pair atomic, then
// commits.
func LogStandalone(ctx context.Context, pool *pgxpool.Pool, e Entry) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning audit transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := Log(ctx, tx, e); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func prepare(e Entry) ([]byte, time.Time, error) {
	metadata := e.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	// encoding/json sorts map keys, so this is deterministic for a given
	// metadata content regardless of how the map was built - required
	// for the hash to be reproducible by VerifyChain later.
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("marshaling audit metadata: %w", err)
	}
	// Generated here, not left to the database's DEFAULT now(), so the
	// exact timestamp hashed is the exact timestamp stored - truncated to
	// microseconds up front because TIMESTAMPTZ only stores that much
	// precision; hashing the untruncated Go value would make the hash
	// computed at write time permanently unreproducible after a round
	// trip through the database.
	return metadataJSON, time.Now().UTC().Truncate(time.Microsecond), nil
}

func computeEntryHash(previousHash *string, actorUserID *string, action string, metadataJSON []byte, ip, userAgent string, createdAt time.Time) string {
	prev := ""
	if previousHash != nil {
		prev = *previousHash
	}
	actor := ""
	if actorUserID != nil {
		actor = *actorUserID
	}

	h := sha256.New()
	h.Write([]byte(prev))
	h.Write([]byte{'|'})
	h.Write([]byte(actor))
	h.Write([]byte{'|'})
	h.Write([]byte(action))
	h.Write([]byte{'|'})
	h.Write(metadataJSON)
	h.Write([]byte{'|'})
	h.Write([]byte(ip))
	h.Write([]byte{'|'})
	h.Write([]byte(userAgent))
	h.Write([]byte{'|'})
	// A raw Unix microsecond count, not a formatted string - immune to a
	// time.Time's Location differing between the value hashed at write
	// time (Go's time.UTC) and the value pgx hands back on a later read
	// (whatever Location the driver happens to attach), which would
	// otherwise format to a different string for the exact same instant
	// and silently break every hash computed from a re-read value.
	h.Write([]byte(strconv.FormatInt(createdAt.UTC().UnixMicro(), 10)))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyChain re-walks the entire audit log in append order and
// recomputes every entry's hash from scratch, confirming each one both
// matches its stored entry_hash and correctly chains from the one
// before it. Returns the seq of the first entry where that breaks, if
// any - a clean result proves the log hasn't been altered since it was
// written, not just that it looks plausible.
func VerifyChain(ctx context.Context, pool *pgxpool.Pool) (ok bool, brokenAtSeq *int64, entries int, err error) {
	rows, err := pool.Query(ctx, `
		SELECT seq, actor_user_id, action, metadata, ip_address, user_agent, created_at, previous_hash, entry_hash
		FROM audit_log
		ORDER BY seq ASC
	`)
	if err != nil {
		return false, nil, 0, fmt.Errorf("reading audit log: %w", err)
	}
	defer rows.Close()

	var expectedPrevious *string
	count := 0
	for rows.Next() {
		var seq int64
		var actorUserID *string
		var action string
		var metadataJSON []byte
		var ip, userAgent *string
		var createdAt time.Time
		var previousHash *string
		var entryHash string
		if err := rows.Scan(&seq, &actorUserID, &action, &metadataJSON, &ip, &userAgent, &createdAt, &previousHash, &entryHash); err != nil {
			return false, nil, 0, fmt.Errorf("scanning audit entry: %w", err)
		}
		count++

		if !sameHash(previousHash, expectedPrevious) {
			return false, &seq, count, nil
		}

		ipVal, uaVal := "", ""
		if ip != nil {
			ipVal = *ip
		}
		if userAgent != nil {
			uaVal = *userAgent
		}
		recomputed := computeEntryHash(previousHash, actorUserID, action, metadataJSON, ipVal, uaVal, createdAt)
		if recomputed != entryHash {
			return false, &seq, count, nil
		}

		expectedPrevious = &entryHash
	}
	if err := rows.Err(); err != nil {
		return false, nil, 0, fmt.Errorf("iterating audit log: %w", err)
	}

	return true, nil, count, nil
}

func sameHash(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// BackfillChain computes previous_hash/entry_hash for every existing row
// that doesn't have one yet (entry_hash IS NULL) - a one-time migration
// step for an audit log that predates chaining, run via
// cmd/backfill-audit-chain, never called from request-serving code. Walks
// in seq order under a single transaction so it chains correctly even if
// new entries are somehow being written concurrently.
func BackfillChain(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("beginning backfill transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT seq, actor_user_id, action, metadata, ip_address, user_agent, created_at
		FROM audit_log
		WHERE entry_hash IS NULL
		ORDER BY seq ASC
	`)
	if err != nil {
		return 0, fmt.Errorf("reading unbackfilled entries: %w", err)
	}

	type row struct {
		seq           int64
		actorUserID   *string
		action        string
		metadataJSON  []byte
		ip, userAgent *string
		createdAt     time.Time
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.seq, &r.actorUserID, &r.action, &r.metadataJSON, &r.ip, &r.userAgent, &r.createdAt); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scanning entry to backfill: %w", err)
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterating entries to backfill: %w", err)
	}
	rows.Close()

	var previousHash *string
	if err := tx.QueryRow(ctx, `
		SELECT entry_hash FROM audit_log WHERE entry_hash IS NOT NULL ORDER BY seq DESC LIMIT 1
	`).Scan(&previousHash); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("finding last already-chained entry: %w", err)
	}

	for _, r := range pending {
		ipVal, uaVal := "", ""
		if r.ip != nil {
			ipVal = *r.ip
		}
		if r.userAgent != nil {
			uaVal = *r.userAgent
		}
		entryHash := computeEntryHash(previousHash, r.actorUserID, r.action, r.metadataJSON, ipVal, uaVal, r.createdAt)
		if _, err := tx.Exec(ctx, `
			UPDATE audit_log SET previous_hash = $1, entry_hash = $2 WHERE seq = $3
		`, previousHash, entryHash, r.seq); err != nil {
			return 0, fmt.Errorf("updating entry seq %d: %w", r.seq, err)
		}
		previousHash = &entryHash
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("committing backfill: %w", err)
	}

	return len(pending), nil
}
