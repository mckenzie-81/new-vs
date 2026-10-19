-- Running-mate requirement moves from a per-candidate free-form list to a
-- per-portfolio toggle: when set, every candidate under this portfolio is
-- a fixed two-person ticket (one running mate), not an optional 0-4.
ALTER TABLE portfolios ADD COLUMN running_mate_required BOOLEAN NOT NULL DEFAULT false;

-- Distinguishes a deliberate "No" (on an uncontested portfolio's Yes/No
-- ballot) from simply not voting on that portfolio at all. candidate_id
-- stays NULL for both - this is the only thing that tells them apart.
-- Never set for a contested portfolio's votes.
ALTER TABLE votes ADD COLUMN is_explicit_no BOOLEAN NOT NULL DEFAULT false;
