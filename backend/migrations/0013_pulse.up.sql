CREATE TABLE election_pulse_snapshots (
    election_id UUID PRIMARY KEY REFERENCES elections(id) ON DELETE CASCADE,
    eligible_count INT NOT NULL,
    voted_count INT NOT NULL,
    turnout_series JSONB NOT NULL,
    portfolio_abstention JSONB NOT NULL,
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
