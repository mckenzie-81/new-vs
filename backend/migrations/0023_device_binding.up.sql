-- device_id is an opaque, non-secret identifier set in a long-lived
-- cookie the first time a browser authenticates any voter - it grants no
-- authority and never blocks anything, so it's fine to live on
-- voter_credentials (which already knows who a voter is) without any of
-- the anonymity concerns that keep it out of votes/credential_hash.
ALTER TABLE voter_credentials ADD COLUMN device_id TEXT;

-- Powers "how many distinct voters has this device authenticated as" -
-- the one device, many voters pattern that's the actual vote-buying /
-- coercion signal, not a per-request check.
CREATE INDEX idx_voter_credentials_election_device
    ON voter_credentials(election_id, device_id) WHERE device_id IS NOT NULL;
