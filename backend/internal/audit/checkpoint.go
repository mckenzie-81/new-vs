package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Checkpoint is a snapshot of the chain's head at some point in time -
// one line in the checkpoint file.
type Checkpoint struct {
	Seq            int64     `json:"seq"`
	EntryHash      string    `json:"entryHash"`
	CreatedAt      time.Time `json:"createdAt"`
	CheckpointedAt time.Time `json:"checkpointedAt"`
}

// WriteCheckpoint appends the chain's current head to path. Returns nil,
// nil if the chain is empty - nothing to anchor yet. Opened in append
// mode every call, not held open, since this only runs once every few
// minutes from a background ticker (see cmd/server) and an admin-
// triggered retry; there's no concurrent-writer story to design for.
func WriteCheckpoint(ctx context.Context, pool *pgxpool.Pool, path string) (*Checkpoint, error) {
	var cp Checkpoint
	err := pool.QueryRow(ctx, `
		SELECT seq, entry_hash, created_at FROM audit_log ORDER BY seq DESC LIMIT 1
	`).Scan(&cp.Seq, &cp.EntryHash, &cp.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading chain head: %w", err)
	}
	cp.CheckpointedAt = time.Now().UTC()

	line, err := json.Marshal(cp)
	if err != nil {
		return nil, fmt.Errorf("encoding checkpoint: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening checkpoint file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return nil, fmt.Errorf("writing checkpoint: %w", err)
	}

	return &cp, nil
}

// ReadCheckpoints returns every checkpoint ever recorded, oldest first.
// A missing file (nothing checkpointed yet) is not an error.
func ReadCheckpoints(path string) ([]Checkpoint, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Checkpoint{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading checkpoint file: %w", err)
	}

	checkpoints := []Checkpoint{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var cp Checkpoint
		if err := json.Unmarshal([]byte(line), &cp); err != nil {
			return nil, fmt.Errorf("parsing checkpoint line: %w", err)
		}
		checkpoints = append(checkpoints, cp)
	}
	return checkpoints, nil
}

// VerifyAgainstCheckpoints is what actually gives the checkpoint file
// teeth: for every checkpoint ever recorded, it looks up what the
// database says that seq's entry_hash is *today* and compares. A chain
// that was rewritten after a checkpoint was taken - even a rewrite
// clever enough to keep every hash internally consistent, which
// VerifyChain alone cannot catch - shows up here as a mismatch, because
// the file still holds the hash that seq had before the rewrite.
func VerifyAgainstCheckpoints(ctx context.Context, pool *pgxpool.Pool, path string) (ok bool, mismatchAtSeq *int64, checked int, err error) {
	checkpoints, err := ReadCheckpoints(path)
	if err != nil {
		return false, nil, 0, err
	}

	for _, cp := range checkpoints {
		var currentHash string
		err := pool.QueryRow(ctx, `SELECT entry_hash FROM audit_log WHERE seq = $1`, cp.Seq).Scan(&currentHash)
		if errors.Is(err, pgx.ErrNoRows) {
			seq := cp.Seq
			return false, &seq, checked, nil
		}
		if err != nil {
			return false, nil, checked, fmt.Errorf("looking up seq %d: %w", cp.Seq, err)
		}
		checked++
		if currentHash != cp.EntryHash {
			seq := cp.Seq
			return false, &seq, checked, nil
		}
	}

	return true, nil, checked, nil
}
