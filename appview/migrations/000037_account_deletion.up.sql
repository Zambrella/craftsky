ALTER TABLE oauth_auth_requests
    ADD COLUMN purpose TEXT NOT NULL DEFAULT 'login',
    ADD COLUMN device_id TEXT,
    ADD COLUMN account_deletion_owner_did TEXT,
    ADD COLUMN account_deletion_job_id UUID,
    ADD CONSTRAINT oauth_auth_requests_purpose_check
        CHECK (purpose IN ('login', 'accountDeletion')),
    ADD CONSTRAINT oauth_auth_requests_account_deletion_metadata_check
        CHECK (
            (purpose = 'login' AND account_deletion_owner_did IS NULL AND account_deletion_job_id IS NULL)
            OR
            (purpose = 'accountDeletion' AND account_deletion_owner_did IS NOT NULL AND account_deletion_job_id IS NOT NULL)
        );

CREATE TABLE account_deletion_operations (
    id                         UUID        NOT NULL PRIMARY KEY,
    owner_did                  TEXT        NOT NULL UNIQUE,
    state                      TEXT        NOT NULL CHECK (state IN ('intent', 'active', 'retrying')),
    accepted_at                TIMESTAMPTZ,
    reauth_oauth_session_id    TEXT,
    deletion_oauth_session_id  TEXT,
    attempt_count              INTEGER     NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at            TIMESTAMPTZ,
    error_category             TEXT,
    intent_proof_hash          BYTEA,
    confirmation_handle_hash   BYTEA,
    intent_expires_at          TIMESTAMPTZ,
    lease_owner                TEXT,
    lease_token                UUID,
    lease_expires_at           TIMESTAMPTZ,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT account_deletion_operations_id_owner_key UNIQUE (id, owner_did),
    CONSTRAINT account_deletion_operations_oauth_fkey
        FOREIGN KEY (owner_did, deletion_oauth_session_id)
        REFERENCES oauth_sessions(account_did, session_id),
    CONSTRAINT account_deletion_operations_reauth_oauth_fkey
        FOREIGN KEY (owner_did, reauth_oauth_session_id)
        REFERENCES oauth_sessions(account_did, session_id)
);

CREATE INDEX account_deletion_operations_worker_idx
    ON account_deletion_operations(state, next_attempt_at, lease_expires_at);
