ALTER TABLE audit_log ALTER COLUMN metadata TYPE JSONB USING metadata::text::jsonb;
ALTER TABLE audit_log ALTER COLUMN metadata SET DEFAULT '{}';
