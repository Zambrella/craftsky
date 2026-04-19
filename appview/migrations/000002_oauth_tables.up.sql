-- OAuth tables for BFF v1. See:
-- docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md §2.

CREATE TABLE oauth_sessions (
    account_did  TEXT        NOT NULL,
    session_id   TEXT        NOT NULL,
    data         JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_did, session_id)
);
CREATE INDEX oauth_sessions_updated_at_idx ON oauth_sessions (updated_at);
CREATE INDEX oauth_sessions_created_at_idx ON oauth_sessions (created_at);

CREATE TABLE oauth_auth_requests (
    state       TEXT        NOT NULL PRIMARY KEY,
    data        JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX oauth_auth_requests_created_at_idx ON oauth_auth_requests (created_at);

CREATE TABLE craftsky_sessions (
    token_hash        BYTEA       NOT NULL PRIMARY KEY,
    account_did       TEXT        NOT NULL,
    oauth_session_id  TEXT        NOT NULL,
    device_label      TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at        TIMESTAMPTZ,
    FOREIGN KEY (account_did, oauth_session_id)
        REFERENCES oauth_sessions (account_did, session_id)
        ON DELETE CASCADE
);
CREATE INDEX craftsky_sessions_did_idx ON craftsky_sessions (account_did);
CREATE INDEX craftsky_sessions_last_seen_idx ON craftsky_sessions (last_seen_at);
