-- jsonb reformats its stored text (e.g. adds a space after ':' and ',')
-- on every read, so a hash computed from the exact bytes json.Marshal
-- wrote at insert time never matches the bytes VerifyChain reads back
-- later. json stores the input text verbatim, so round-tripping a row
-- always returns the identical bytes that were hashed.
ALTER TABLE audit_log ALTER COLUMN metadata TYPE JSON USING metadata::text::json;
ALTER TABLE audit_log ALTER COLUMN metadata SET DEFAULT '{}';
