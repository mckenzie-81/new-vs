-- Hash-chains the audit log: each entry's hash folds in the previous
-- entry's hash, so altering, inserting, or deleting a past entry breaks
-- every hash after it - detectable even by someone with direct DB write
-- access, not just through this application.
--
-- seq gives an unambiguous append order (audit_log's own id is a random
-- UUID, unusable for chaining) - backfilled here from created_at for
-- existing rows, then driven by a sequence for everything after.
ALTER TABLE audit_log ADD COLUMN seq BIGINT;

UPDATE audit_log SET seq = sub.rn
FROM (
    SELECT id, row_number() OVER (ORDER BY created_at ASC, id ASC) AS rn
    FROM audit_log
) sub
WHERE audit_log.id = sub.id;

ALTER TABLE audit_log ALTER COLUMN seq SET NOT NULL;

CREATE SEQUENCE audit_log_seq_seq OWNED BY audit_log.seq;
SELECT setval('audit_log_seq_seq', COALESCE((SELECT max(seq) FROM audit_log), 0));
ALTER TABLE audit_log ALTER COLUMN seq SET DEFAULT nextval('audit_log_seq_seq');

CREATE UNIQUE INDEX idx_audit_log_seq ON audit_log(seq);

ALTER TABLE audit_log ADD COLUMN previous_hash TEXT;
ALTER TABLE audit_log ADD COLUMN entry_hash TEXT;
-- entry_hash becomes NOT NULL once a one-off backfill (run manually, see
-- cmd/backfill-audit-chain) has computed it for every existing row.
