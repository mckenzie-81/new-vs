ALTER TABLE electorate_databases ADD COLUMN sensitive_columns TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE elections ADD COLUMN voter_verification_columns TEXT[] NOT NULL DEFAULT '{}';
CREATE INDEX idx_elections_end_time ON elections(end_time);

CREATE TYPE voter_credential_status AS ENUM ('ISSUED', 'AUTHENTICATED', 'VOTED', 'EXPIRED', 'LOCKED');

CREATE TABLE voter_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    election_id UUID NOT NULL REFERENCES elections(id) ON DELETE CASCADE,
    electorate_voter_id UUID NOT NULL REFERENCES electorate_voters(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    status voter_credential_status NOT NULL DEFAULT 'ISSUED',
    failed_attempts INT NOT NULL DEFAULT 0,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    authenticated_at TIMESTAMPTZ,
    voted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (election_id, electorate_voter_id)
);

CREATE INDEX idx_voter_credentials_election_id ON voter_credentials(election_id);

-- No FK on credential_hash anywhere - that's load-bearing, not an oversight.
-- A FK would force a verifiable join path back to voter_credentials, exactly
-- what the two-hash split (see internal/voter) exists to prevent. Portfolio/
-- candidate/election validity is enforced in Go inside CastVote's
-- transaction before insert, not by the schema.
CREATE TABLE votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    election_id UUID NOT NULL REFERENCES elections(id) ON DELETE CASCADE,
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    candidate_id UUID REFERENCES candidates(id),
    credential_hash TEXT NOT NULL,
    cast_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (credential_hash, portfolio_id)
);

CREATE INDEX idx_votes_election_id ON votes(election_id);
CREATE INDEX idx_votes_portfolio_candidate ON votes(portfolio_id, candidate_id);
