DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS voter_credentials;
DROP TYPE IF EXISTS voter_credential_status;
DROP INDEX IF EXISTS idx_elections_end_time;
ALTER TABLE elections DROP COLUMN IF EXISTS voter_verification_columns;
ALTER TABLE electorate_databases DROP COLUMN IF EXISTS sensitive_columns;
