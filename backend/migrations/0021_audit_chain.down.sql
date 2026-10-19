ALTER TABLE audit_log DROP COLUMN entry_hash;
ALTER TABLE audit_log DROP COLUMN previous_hash;
DROP INDEX idx_audit_log_seq;
ALTER TABLE audit_log ALTER COLUMN seq DROP DEFAULT;
DROP SEQUENCE audit_log_seq_seq;
ALTER TABLE audit_log DROP COLUMN seq;
