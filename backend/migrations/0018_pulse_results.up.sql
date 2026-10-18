-- A frozen copy of the final tally, written once by an admin's explicit
-- "publish results" action - never a live view onto votes. Its mere
-- existence is what makes the public pulse page show candidate standings
-- at all; see internal/pulse for why that distinction matters.
CREATE TABLE election_pulse_results (
    election_id UUID PRIMARY KEY REFERENCES elections(id) ON DELETE CASCADE,
    portfolios JSONB NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
