ALTER TABLE users DROP COLUMN IF EXISTS password_reset_token_hash;
ALTER TABLE users DROP COLUMN IF EXISTS password_reset_expires_at;
