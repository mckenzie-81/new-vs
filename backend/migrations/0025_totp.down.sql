DROP TABLE pending_totp_logins;
DROP INDEX idx_totp_recovery_codes_user;
DROP TABLE totp_recovery_codes;
ALTER TABLE users DROP COLUMN totp_enabled_at;
ALTER TABLE users DROP COLUMN totp_secret;
