-- Recorded by the periodic integrity check (see internal/audit.RunIntegrityCheck),
-- never by a request handler - this is what makes "automatic" verification
-- actually mean something, instead of only ever running when an admin
-- remembers to click the Verify button.
CREATE TABLE audit_integrity_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind TEXT NOT NULL,
    detail TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID REFERENCES users(id)
);

-- The one query this table serves on every dashboard page load: is there
-- anything an admin hasn't seen yet. Partial index keeps that cheap
-- forever, regardless of how much acknowledged history piles up.
CREATE INDEX idx_audit_integrity_alerts_unacknowledged
    ON audit_integrity_alerts(created_at) WHERE acknowledged_at IS NULL;
