ALTER TABLE candidates DROP COLUMN IF EXISTS photo_url;
ALTER TABLE candidates ADD COLUMN manifesto TEXT;
