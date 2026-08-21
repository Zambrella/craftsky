-- Durable retry state for bounded background refresh of the searchable handle
-- index. No resolved identity document or error text is retained here.
CREATE TABLE atproto_identity_refresh_state (
    did             TEXT        NOT NULL PRIMARY KEY,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    attempt_count   INTEGER     NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    last_result     TEXT        NOT NULL CHECK (last_result IN ('retry')),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX atproto_identity_refresh_state_due_idx
    ON atproto_identity_refresh_state (next_attempt_at, did);
