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
    owner_did                  TEXT        NOT NULL,
    state                      TEXT        NOT NULL,
    phase                      TEXT,
    accepted_at                TIMESTAMPTZ,
    reauth_oauth_session_id    TEXT,
    deletion_oauth_session_id  TEXT,
    attempt_count              INTEGER     NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at            TIMESTAMPTZ,
    error_category             TEXT,
    intent_proof_hash          BYTEA,
    confirmation_handle_hash   BYTEA,
    intent_expires_at          TIMESTAMPTZ,
    final_rescan_at            TIMESTAMPTZ,
    lease_owner                TEXT,
    lease_token                UUID,
    lease_expires_at           TIMESTAMPTZ,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT account_deletion_operations_owner_key UNIQUE (owner_did),
    CONSTRAINT account_deletion_operations_id_owner_key UNIQUE (id, owner_did),
    CONSTRAINT account_deletion_operations_state_check
        CHECK (state IN ('intent', 'active', 'retrying', 'needsAttention', 'canceled')),
    CONSTRAINT account_deletion_operations_phase_check
        CHECK (phase IS NULL OR phase IN (
            'queued', 'removingPrivateData', 'removingCraftskyRecords',
            'waitingForIndexerConvergence', 'finalizing'
        )),
    CONSTRAINT account_deletion_operations_oauth_fkey
        FOREIGN KEY (owner_did, deletion_oauth_session_id)
        REFERENCES oauth_sessions(account_did, session_id),
    CONSTRAINT account_deletion_operations_reauth_oauth_fkey
        FOREIGN KEY (owner_did, reauth_oauth_session_id)
        REFERENCES oauth_sessions(account_did, session_id)
);

CREATE INDEX account_deletion_operations_worker_idx
    ON account_deletion_operations(state, next_attempt_at, lease_expires_at);

CREATE TABLE account_deletion_status_credentials (
    token_hash   BYTEA       NOT NULL PRIMARY KEY,
    job_id       UUID        NOT NULL,
    owner_did    TEXT        NOT NULL,
    device_id    TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,
    CONSTRAINT account_deletion_status_credentials_job_owner_fkey
        FOREIGN KEY (job_id, owner_did)
        REFERENCES account_deletion_operations(id, owner_did)
        ON DELETE CASCADE
);
CREATE INDEX account_deletion_status_credentials_job_idx
    ON account_deletion_status_credentials(job_id);

CREATE TABLE account_deletion_recovery_credentials (
    token_hash  BYTEA       NOT NULL PRIMARY KEY,
    job_id      UUID        NOT NULL,
    owner_did   TEXT        NOT NULL,
    device_id   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at     TIMESTAMPTZ,
    CONSTRAINT account_deletion_recovery_credentials_job_owner_fkey
        FOREIGN KEY (job_id, owner_did)
        REFERENCES account_deletion_operations(id, owner_did)
        ON DELETE CASCADE
);
CREATE INDEX account_deletion_recovery_credentials_job_idx
    ON account_deletion_recovery_credentials(job_id);

CREATE TABLE account_deletion_expected_records (
    job_id                UUID        NOT NULL REFERENCES account_deletion_operations(id) ON DELETE CASCADE,
    uri                   TEXT        NOT NULL,
    collection            TEXT        NOT NULL,
    registered_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    delete_requested_at   TIMESTAMPTZ,
    PRIMARY KEY (job_id, uri)
);

CREATE TABLE account_deletion_index_receipts (
    job_id         UUID        NOT NULL REFERENCES account_deletion_operations(id) ON DELETE CASCADE,
    uri            TEXT        NOT NULL,
    collection     TEXT        NOT NULL,
    tap_event_id   BIGINT      NOT NULL,
    repo_revision  TEXT        NOT NULL,
    handled_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (job_id, uri)
);
CREATE UNIQUE INDEX account_deletion_index_receipts_event_idx
    ON account_deletion_index_receipts(job_id, tap_event_id, repo_revision);

CREATE TABLE account_deletion_cleanup_steps (
    job_id       UUID        NOT NULL REFERENCES account_deletion_operations(id) ON DELETE CASCADE,
    component    TEXT        NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (job_id, component)
);

CREATE TABLE account_deletion_cleanup_artifacts (
    job_id      UUID        NOT NULL REFERENCES account_deletion_operations(id) ON DELETE CASCADE,
    component   TEXT        NOT NULL,
    artifact_id UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (job_id, component, artifact_id)
);

CREATE TABLE account_deletion_audits (
    job_id      UUID        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    accepted_at TIMESTAMPTZ NOT NULL,
    terminal_at TIMESTAMPTZ NOT NULL,
    outcome     TEXT        NOT NULL CHECK (outcome IN ('deleted')),
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX account_deletion_audits_expires_at_idx
    ON account_deletion_audits(expires_at);
