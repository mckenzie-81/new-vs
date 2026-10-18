CREATE TABLE election_schedule_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    election_id UUID NOT NULL REFERENCES elections(id) ON DELETE CASCADE,
    actor_user_id UUID NOT NULL REFERENCES users(id),
    previous_end_time TIMESTAMPTZ NOT NULL,
    new_end_time TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_election_schedule_changes_election_id ON election_schedule_changes(election_id);
