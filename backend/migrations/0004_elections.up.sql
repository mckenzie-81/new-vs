CREATE TYPE election_status AS ENUM (
    'DRAFT',
    'PENDING_APPROVAL',
    'APPROVED',
    'LIVE',
    'CLOSED',
    'ARCHIVED'
);

CREATE TABLE elections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT,
    status election_status NOT NULL DEFAULT 'DRAFT',
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_elections_status ON elections(status);
CREATE INDEX idx_elections_start_time ON elections(start_time);
