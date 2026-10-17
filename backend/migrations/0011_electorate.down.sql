ALTER TABLE elections DROP COLUMN IF EXISTS electorate_filter;
ALTER TABLE elections DROP COLUMN IF EXISTS electorate_database_id;
DROP TABLE IF EXISTS electorate_voters;
DROP TABLE IF EXISTS electorate_databases;
