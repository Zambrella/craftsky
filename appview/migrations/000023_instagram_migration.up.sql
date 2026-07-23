CREATE TABLE instagram_verification_attempts (
    id                          UUID        NOT NULL PRIMARY KEY,
    owner_did                   TEXT        NOT NULL,
    state                       TEXT        NOT NULL CHECK (state IN (
                                    'pendingDm', 'processing', 'pendingConfirmation',
                                    'confirmed', 'expired', 'cancelled', 'superseded',
                                    'rejected', 'conflicted'
                                )),
    challenge_digest_version    SMALLINT,
    challenge_digest            BYTEA,
    candidate_igsid             TEXT,
    candidate_username          TEXT,
    retry_code                  TEXT CHECK (retry_code IS NULL OR retry_code IN (
                                    'profileLookupUnavailable',
                                    'invalidProfileResponse',
                                    'membershipInactive'
                                )),
    expires_at                  TIMESTAMPTZ NOT NULL,
    processing_started_at       TIMESTAMPTZ,
    terminal_at                 TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT instagram_verification_attempts_challenge_shape_check CHECK (
        (challenge_digest IS NULL AND challenge_digest_version IS NULL) OR
        (challenge_digest IS NOT NULL AND challenge_digest_version IS NOT NULL AND octet_length(challenge_digest) = 32)
    ),
    CONSTRAINT instagram_verification_attempts_sensitive_state_check CHECK (
        (state = 'pendingDm' AND challenge_digest IS NOT NULL AND candidate_igsid IS NULL AND candidate_username IS NULL) OR
        (state = 'processing' AND challenge_digest IS NULL AND candidate_igsid IS NOT NULL) OR
        (state = 'pendingConfirmation' AND challenge_digest IS NULL AND candidate_igsid IS NOT NULL AND candidate_username IS NOT NULL) OR
        (state IN ('confirmed', 'expired', 'cancelled', 'superseded', 'rejected', 'conflicted') AND challenge_digest IS NULL)
    )
);

CREATE UNIQUE INDEX instagram_verification_attempts_owner_active_unique
    ON instagram_verification_attempts (owner_did)
    WHERE state IN ('pendingDm', 'processing', 'pendingConfirmation');
CREATE UNIQUE INDEX instagram_verification_attempts_challenge_unique
    ON instagram_verification_attempts (challenge_digest_version, challenge_digest)
    WHERE challenge_digest IS NOT NULL;
CREATE INDEX instagram_verification_attempts_owner_page_idx
    ON instagram_verification_attempts (owner_did, created_at DESC, id DESC);
CREATE INDEX instagram_verification_attempts_expiry_idx
    ON instagram_verification_attempts (expires_at, id)
    WHERE state IN ('pendingDm', 'processing', 'pendingConfirmation');

CREATE TABLE instagram_account_links (
    id                          UUID        NOT NULL PRIMARY KEY,
    owner_did                   TEXT        NOT NULL,
    state                       TEXT        NOT NULL CHECK (state IN (
                                    'active', 'membershipInactive', 'revoked',
                                    'superseded', 'disputed'
                                )),
    igsid                       TEXT,
    igsid_digest_version        SMALLINT    NOT NULL,
    igsid_digest                BYTEA       NOT NULL CHECK (octet_length(igsid_digest) = 32),
    username                    TEXT,
    username_normalized         TEXT,
    discoverable                BOOLEAN     NOT NULL DEFAULT false,
    conflict_pending            BOOLEAN     NOT NULL DEFAULT false,
    verified_at                 TIMESTAMPTZ NOT NULL,
    membership_inactive_at      TIMESTAMPTZ,
    revoked_at                  TIMESTAMPTZ,
    superseded_at               TIMESTAMPTZ,
    raw_identity_purge_at       TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT instagram_account_links_identity_shape_check CHECK (
        (state IN ('active', 'membershipInactive', 'disputed') AND igsid IS NOT NULL AND username IS NOT NULL AND username_normalized IS NOT NULL) OR
        (state IN ('revoked', 'superseded'))
    ),
    CONSTRAINT instagram_account_links_discoverable_state_check CHECK (
        NOT discoverable OR (state = 'active' AND NOT conflict_pending)
    )
);

CREATE UNIQUE INDEX instagram_account_links_owner_current_unique
    ON instagram_account_links (owner_did)
    WHERE state IN ('active', 'membershipInactive', 'disputed');
CREATE UNIQUE INDEX instagram_account_links_username_current_unique
    ON instagram_account_links (username_normalized)
    WHERE state IN ('active', 'membershipInactive', 'disputed');
CREATE INDEX instagram_account_links_username_discovery_idx
    ON instagram_account_links (username_normalized, owner_did)
    WHERE state = 'active' AND discoverable AND NOT conflict_pending;
CREATE INDEX instagram_account_links_igsid_digest_idx
    ON instagram_account_links (igsid_digest_version, igsid_digest);
CREATE INDEX instagram_account_links_retention_idx
    ON instagram_account_links (raw_identity_purge_at, id)
    WHERE raw_identity_purge_at IS NOT NULL;

CREATE TABLE instagram_identity_claims (
    id                      UUID        NOT NULL PRIMARY KEY,
    link_id                 UUID        REFERENCES instagram_account_links(id) ON DELETE CASCADE,
    owner_did               TEXT        NOT NULL,
    state                   TEXT        NOT NULL CHECK (state IN ('active', 'revoked', 'disputed')),
    igsid_digest_version    SMALLINT    NOT NULL,
    igsid_digest            BYTEA       NOT NULL CHECK (octet_length(igsid_digest) = 32),
    claimed_at              TIMESTAMPTZ NOT NULL,
    released_at             TIMESTAMPTZ,
    anonymize_at            TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (link_id)
);

CREATE UNIQUE INDEX instagram_identity_claims_active_igsid_unique
    ON instagram_identity_claims (igsid_digest_version, igsid_digest)
    WHERE state = 'active';
CREATE INDEX instagram_identity_claims_owner_idx
    ON instagram_identity_claims (owner_did, created_at DESC, id DESC);
CREATE INDEX instagram_identity_claims_anonymize_idx
    ON instagram_identity_claims (anonymize_at, id)
    WHERE anonymize_at IS NOT NULL;

CREATE TABLE instagram_link_conflicts (
    id                          UUID        NOT NULL PRIMARY KEY,
    state                       TEXT        NOT NULL CHECK (state IN (
                                    'open', 'resolvedKeepExisting',
                                    'resolvedRevokeExisting', 'expired'
                                )),
    existing_link_id            UUID        REFERENCES instagram_account_links(id) ON DELETE SET NULL,
    claimant_attempt_id         UUID        REFERENCES instagram_verification_attempts(id) ON DELETE SET NULL,
    claimant_link_id            UUID        REFERENCES instagram_account_links(id) ON DELETE SET NULL,
    igsid_digest_version        SMALLINT,
    igsid_digest                BYTEA,
    opened_at                   TIMESTAMPTZ NOT NULL,
    resolved_at                 TIMESTAMPTZ,
    expires_at                  TIMESTAMPTZ NOT NULL,
    resolution_note_digest      BYTEA,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT instagram_link_conflicts_digest_shape_check CHECK (
        (igsid_digest IS NULL AND igsid_digest_version IS NULL) OR
        (igsid_digest IS NOT NULL AND igsid_digest_version IS NOT NULL AND octet_length(igsid_digest) = 32)
    )
);

CREATE INDEX instagram_link_conflicts_open_idx
    ON instagram_link_conflicts (opened_at, id) WHERE state = 'open';
CREATE INDEX instagram_link_conflicts_expiry_idx
    ON instagram_link_conflicts (expires_at, id) WHERE state = 'open';

CREATE TABLE instagram_webhook_work (
    id                          UUID        NOT NULL PRIMARY KEY,
    verification_attempt_id     UUID        REFERENCES instagram_verification_attempts(id),
    message_digest_version      SMALLINT    NOT NULL,
    message_digest              BYTEA       NOT NULL CHECK (octet_length(message_digest) = 32),
    sender_igsid                TEXT,
    official_account_id         TEXT,
    challenge_digest_version    SMALLINT,
    challenge_digest            BYTEA,
    event_at                    TIMESTAMPTZ NOT NULL,
    status                      TEXT        NOT NULL CHECK (status IN (
                                    'queued', 'processing', 'retryable',
                                    'completed', 'ignored', 'failed'
                                )),
    attempts                    INTEGER     NOT NULL DEFAULT 0 CHECK (attempts >= 0 AND attempts <= 5),
    next_attempt_at             TIMESTAMPTZ NOT NULL,
    processing_started_at       TIMESTAMPTZ,
    lease_token                 UUID,
    lease_expires_at            TIMESTAMPTZ,
    terminal_at                 TIMESTAMPTZ,
    terminal_reason             TEXT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT instagram_webhook_work_message_digest_key UNIQUE (message_digest_version, message_digest),
    CONSTRAINT instagram_webhook_work_challenge_shape_check CHECK (
        (challenge_digest IS NULL AND challenge_digest_version IS NULL) OR
        (challenge_digest IS NOT NULL AND challenge_digest_version IS NOT NULL AND octet_length(challenge_digest) = 32)
    ),
    CONSTRAINT instagram_webhook_work_terminal_clear_check CHECK (
        status IN ('queued', 'processing', 'retryable') OR
        (sender_igsid IS NULL AND challenge_digest IS NULL AND challenge_digest_version IS NULL)
    )
);

CREATE INDEX instagram_webhook_work_claim_idx
    ON instagram_webhook_work (next_attempt_at, id)
    WHERE status IN ('queued', 'retryable');
CREATE UNIQUE INDEX instagram_webhook_work_attempt_unique
    ON instagram_webhook_work (verification_attempt_id)
    WHERE verification_attempt_id IS NOT NULL;
CREATE INDEX instagram_webhook_work_expired_lease_idx
    ON instagram_webhook_work (lease_expires_at, id)
    WHERE status = 'processing';
CREATE INDEX instagram_webhook_work_retention_idx
    ON instagram_webhook_work (terminal_at, id)
    WHERE status IN ('completed', 'ignored', 'failed');

CREATE TABLE instagram_graph_imports (
    id                      UUID        NOT NULL PRIMARY KEY,
    owner_did               TEXT        NOT NULL,
    state                   TEXT        NOT NULL CHECK (state IN ('active', 'membershipInactive')),
    source_type             TEXT        NOT NULL CHECK (source_type IN ('manual', 'instagramJson')),
    membership_inactive_at  TIMESTAMPTZ,
    following_count         INTEGER     NOT NULL CHECK (following_count >= 0 AND following_count <= 10000),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX instagram_graph_imports_owner_page_idx
    ON instagram_graph_imports (owner_did, created_at DESC, id DESC);

CREATE TABLE instagram_graph_handles (
    id                  BIGSERIAL   NOT NULL PRIMARY KEY,
    import_id           UUID        NOT NULL REFERENCES instagram_graph_imports(id) ON DELETE CASCADE,
    username_normalized TEXT        NOT NULL,
    matched             BOOLEAN     NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (import_id, username_normalized)
);

CREATE INDEX instagram_graph_handles_match_idx
    ON instagram_graph_handles (username_normalized, import_id);

CREATE TABLE instagram_follow_suggestions (
    id                  UUID        NOT NULL PRIMARY KEY,
    importer_did        TEXT        NOT NULL,
    target_did          TEXT        NOT NULL,
    state               TEXT        NOT NULL CHECK (state IN (
                                'pending', 'accepting', 'accepted',
                                'alreadyFollowing', 'dismissed', 'invalidated'
                            )),
    reason              TEXT        NOT NULL CHECK (reason = 'verifiedInstagramFollow'),
    accepting_since     TIMESTAMPTZ,
    terminal_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT instagram_follow_suggestions_not_self_check CHECK (importer_did <> target_did),
    UNIQUE (importer_did, target_did, reason)
);

CREATE INDEX instagram_follow_suggestions_owner_page_idx
    ON instagram_follow_suggestions (importer_did, created_at DESC, id DESC)
    WHERE state = 'pending';
CREATE INDEX instagram_follow_suggestions_target_idx
    ON instagram_follow_suggestions (target_did, state, id);
CREATE INDEX instagram_follow_suggestions_terminal_retention_idx
    ON instagram_follow_suggestions (terminal_at, id)
    WHERE terminal_at IS NOT NULL;

CREATE TABLE instagram_suggestion_sources (
    suggestion_id   UUID        NOT NULL REFERENCES instagram_follow_suggestions(id) ON DELETE CASCADE,
    import_id       UUID        NOT NULL REFERENCES instagram_graph_imports(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (suggestion_id, import_id)
);

CREATE INDEX instagram_suggestion_sources_import_idx
    ON instagram_suggestion_sources (import_id, suggestion_id);

CREATE TABLE instagram_reconciliation_jobs (
    id                  UUID        NOT NULL PRIMARY KEY,
    owner_did           TEXT        NOT NULL,
    target_did          TEXT,
    link_id             UUID,
    import_id           UUID,
    reason              TEXT        NOT NULL,
    status              TEXT        NOT NULL CHECK (status IN (
                                'queued', 'processing', 'retryable',
                                'completed', 'ignored', 'failed'
                            )),
    attempts            INTEGER     NOT NULL DEFAULT 0 CHECK (attempts >= 0 AND attempts <= 5),
    next_attempt_at     TIMESTAMPTZ NOT NULL,
    lease_token         UUID,
    lease_expires_at    TIMESTAMPTZ,
    terminal_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX instagram_reconciliation_jobs_claim_idx
    ON instagram_reconciliation_jobs (next_attempt_at, id)
    WHERE status IN ('queued', 'retryable');
CREATE INDEX instagram_reconciliation_jobs_owner_idx
    ON instagram_reconciliation_jobs (owner_did, created_at DESC, id DESC);
CREATE INDEX instagram_reconciliation_jobs_expired_lease_idx
    ON instagram_reconciliation_jobs (lease_expires_at, id)
    WHERE status = 'processing';

CREATE TABLE pds_follow_operations (
    id              UUID        NOT NULL PRIMARY KEY,
    suggestion_id   UUID        NOT NULL UNIQUE,
    owner_did       TEXT        NOT NULL,
    target_did      TEXT        NOT NULL,
    rkey            TEXT        NOT NULL,
    status          TEXT        NOT NULL CHECK (status IN (
                        'pending', 'writing', 'succeeded', 'alreadyFollowing', 'failed'
                    )),
    record_uri      TEXT,
    record_cid      TEXT,
    attempt_count   INTEGER     NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    last_error_code TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    CONSTRAINT pds_follow_operations_not_self_check CHECK (owner_did <> target_did)
);

CREATE UNIQUE INDEX pds_follow_operations_owner_rkey_unique
    ON pds_follow_operations (owner_did, rkey);
CREATE INDEX pds_follow_operations_recovery_idx
    ON pds_follow_operations (updated_at, id)
    WHERE status IN ('pending', 'writing');

CREATE TABLE instagram_rate_limit_buckets (
    bucket_scope    TEXT        NOT NULL,
    key_version     SMALLINT    NOT NULL,
    key_digest      BYTEA       NOT NULL CHECK (octet_length(key_digest) = 32),
    window_start    TIMESTAMPTZ NOT NULL,
    window_end      TIMESTAMPTZ NOT NULL,
    count           INTEGER     NOT NULL CHECK (count >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (bucket_scope, key_version, key_digest, window_start),
    CONSTRAINT instagram_rate_limit_buckets_window_check CHECK (window_end > window_start)
);

CREATE INDEX instagram_rate_limit_buckets_expiry_idx
    ON instagram_rate_limit_buckets (window_end, bucket_scope);

CREATE TABLE instagram_audit_events (
    id              BIGSERIAL   NOT NULL PRIMARY KEY,
    owner_did       TEXT,
    action          TEXT        NOT NULL,
    subject_kind    TEXT        NOT NULL,
    subject_id      TEXT,
    outcome         TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX instagram_audit_events_owner_idx
    ON instagram_audit_events (owner_did, created_at DESC, id DESC)
    WHERE owner_did IS NOT NULL;
CREATE INDEX instagram_audit_events_retention_idx
    ON instagram_audit_events (created_at, id);
