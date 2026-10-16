ALTER TABLE safety_operators ADD COLUMN did TEXT UNIQUE;
ALTER TABLE safety_operators ADD CONSTRAINT safety_operators_did_check
    CHECK (did IS NULL OR (did LIKE 'did:%' AND char_length(did) <= 2048));

-- Prior human integration credentials cannot authenticate the browser console.
UPDATE safety_operator_tokens SET active=false, revoked_at=now() WHERE active;

ALTER TABLE safety_operator_events DROP CONSTRAINT safety_operator_events_action_check;
ALTER TABLE safety_operator_events ADD CONSTRAINT safety_operator_events_action_check CHECK (action IN (
    'bootstrapped','created','tokenIssued','tokensRotated','tokenRevoked','deactivated',
    'permissionGranted','permissionRevoked','assignmentGranted','assignmentReleased',
    'allowlisted','permissionSetChanged','sessionsRevoked'
));

CREATE TABLE operator_browser_sessions (
    token_digest BYTEA PRIMARY KEY CHECK (octet_length(token_digest)=32),
    operator_id UUID NOT NULL REFERENCES safety_operators(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ NOT NULL,
    CHECK (expires_at > created_at)
);
CREATE INDEX operator_browser_sessions_operator_idx ON operator_browser_sessions(operator_id);

CREATE TABLE operator_oauth_attempts (
    state TEXT PRIMARY KEY,
    browser_digest BYTEA NOT NULL CHECK (octet_length(browser_digest)=32),
    did TEXT NOT NULL,
    pds_origin TEXT NOT NULL,
    data JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX operator_oauth_attempts_expiry_idx ON operator_oauth_attempts(expires_at);
