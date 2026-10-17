CREATE TABLE electorate_databases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    columns TEXT[] NOT NULL DEFAULT '{}',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE electorate_voters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    electorate_database_id UUID NOT NULL REFERENCES electorate_databases(id) ON DELETE CASCADE,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_electorate_voters_database_id ON electorate_voters(electorate_database_id);
CREATE INDEX idx_electorate_voters_data_gin ON electorate_voters USING GIN (data);
CREATE UNIQUE INDEX idx_electorate_voters_unique_email
    ON electorate_voters (electorate_database_id, (lower(data ->> 'email')));

ALTER TABLE elections ADD COLUMN electorate_database_id UUID REFERENCES electorate_databases(id);
ALTER TABLE elections ADD COLUMN electorate_filter JSONB;

CREATE INDEX idx_elections_electorate_database_id ON elections(electorate_database_id);
