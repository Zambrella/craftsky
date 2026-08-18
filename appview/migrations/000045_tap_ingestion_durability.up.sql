CREATE TABLE tap_ingestion_receipts (
    event_fingerprint BYTEA       NOT NULL PRIMARY KEY,
    tap_event_id      BIGINT      NOT NULL CHECK (tap_event_id >= 0),
    event_type        TEXT        NOT NULL
        CHECK (event_type IN ('record', 'identity', 'quarantine')),
    outcome           TEXT        NOT NULL
        CHECK (outcome IN ('applied', 'blocked', 'permanent_invalid')),
    source_uri        TEXT,
    reason_code       TEXT        NOT NULL,
    received_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tap_ingestion_receipts_fingerprint_check
        CHECK (octet_length(event_fingerprint) = 32),
    CONSTRAINT tap_ingestion_receipts_source_uri_check
        CHECK (source_uri IS NULL OR (btrim(source_uri) <> '' AND char_length(source_uri) <= 2048)),
    CONSTRAINT tap_ingestion_receipts_reason_check
        CHECK (btrim(reason_code) <> '' AND char_length(reason_code) <= 64)
);

CREATE INDEX tap_ingestion_receipts_event_id_idx
    ON tap_ingestion_receipts (tap_event_id, received_at);

CREATE TABLE tap_source_records (
    uri                    TEXT        NOT NULL PRIMARY KEY,
    did                    TEXT        NOT NULL,
    collection             TEXT        NOT NULL,
    rkey                   TEXT        NOT NULL,
    source_event_id        BIGINT      NOT NULL CHECK (source_event_id > 0),
    source_fingerprint     BYTEA       NOT NULL,
    revision               TEXT        NOT NULL,
    cid                    TEXT,
    action                 TEXT        NOT NULL
        CHECK (action IN ('create', 'update', 'delete')),
    record                 JSONB,
    record_bytes           INTEGER     NOT NULL,
    live                   BOOLEAN     NOT NULL,
    ordering_status        TEXT        NOT NULL
        CHECK (ordering_status IN ('authoritative', 'uncertain')),
    projection_disposition TEXT        NOT NULL
        CHECK (projection_disposition IN (
            'pending', 'eligible', 'blocked_departed',
            'denied_terminal', 'not_accepted'
        )),
    owner_generation       BIGINT,
    effect_operation_id    TEXT,
    projection_version     INTEGER     NOT NULL DEFAULT 1
        CHECK (projection_version > 0),
    observed_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tap_source_records_uri_check
        CHECK (btrim(uri) <> '' AND char_length(uri) <= 2048),
    CONSTRAINT tap_source_records_did_check
        CHECK (btrim(did) <> '' AND char_length(did) <= 512),
    CONSTRAINT tap_source_records_collection_check
        CHECK (btrim(collection) <> '' AND char_length(collection) <= 317),
    CONSTRAINT tap_source_records_rkey_check
        CHECK (btrim(rkey) <> '' AND char_length(rkey) <= 512),
    CONSTRAINT tap_source_records_fingerprint_check
        CHECK (octet_length(source_fingerprint) = 32),
    CONSTRAINT tap_source_records_revision_check
        CHECK (btrim(revision) <> '' AND char_length(revision) <= 128),
    CONSTRAINT tap_source_records_content_check CHECK (
        (action IN ('create', 'update')
            AND cid IS NOT NULL AND btrim(cid) <> ''
            AND record IS NOT NULL AND record_bytes BETWEEN 1 AND 1048576)
        OR
        (action = 'delete' AND record IS NULL AND record_bytes = 0)
    ),
    CONSTRAINT tap_source_records_owner_generation_check
        CHECK (owner_generation IS NULL OR owner_generation > 0),
    CONSTRAINT tap_source_records_effect_operation_check
        CHECK (effect_operation_id IS NULL OR (
            btrim(effect_operation_id) <> '' AND char_length(effect_operation_id) <= 512
        )),
    CONSTRAINT tap_source_records_timestamp_check
        CHECK (updated_at >= observed_at)
);

CREATE INDEX tap_source_records_owner_idx
    ON tap_source_records (did, collection, uri);

CREATE INDEX tap_source_records_projection_idx
    ON tap_source_records (projection_disposition, collection, updated_at, uri)
    WHERE projection_disposition <> 'denied_terminal';

CREATE TABLE tap_projection_jobs (
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_uri         TEXT        NOT NULL
        REFERENCES tap_source_records(uri) ON DELETE CASCADE,
    projection_kind    TEXT        NOT NULL,
    source_event_id    BIGINT      NOT NULL CHECK (source_event_id > 0),
    state              TEXT        NOT NULL
        CHECK (state IN ('pending', 'blocked', 'processing', 'complete', 'permanent_denied')),
    dependency_kind    TEXT,
    dependency_key     TEXT,
    attempts           INTEGER     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    lease_owner        TEXT,
    lease_token        UUID,
    lease_expires_at   TIMESTAMPTZ,
    last_reason_code   TEXT,
    completed_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_uri, projection_kind),
    CONSTRAINT tap_projection_jobs_projection_kind_check
        CHECK (btrim(projection_kind) <> '' AND char_length(projection_kind) <= 128),
    CONSTRAINT tap_projection_jobs_dependency_check CHECK (
        (state = 'blocked'
            AND dependency_kind IS NOT NULL
            AND dependency_kind IN ('member_did', 'subject_uri', 'repository_did')
            AND dependency_key IS NOT NULL
            AND btrim(dependency_key) <> '' AND char_length(dependency_key) <= 2048)
        OR
        (state <> 'blocked' AND dependency_kind IS NULL AND dependency_key IS NULL)
    ),
    CONSTRAINT tap_projection_jobs_lease_check CHECK (
        (state = 'processing'
            AND btrim(lease_owner) <> '' AND char_length(lease_owner) <= 256
            AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL
            AND completed_at IS NULL)
        OR
        (state IN ('pending', 'blocked')
            AND lease_owner IS NULL AND lease_token IS NULL
            AND lease_expires_at IS NULL AND completed_at IS NULL)
        OR
        (state IN ('complete', 'permanent_denied')
            AND lease_owner IS NULL AND lease_token IS NULL
            AND lease_expires_at IS NULL AND completed_at IS NOT NULL)
    ),
    CONSTRAINT tap_projection_jobs_reason_check
        CHECK (last_reason_code IS NULL OR (
            btrim(last_reason_code) <> '' AND char_length(last_reason_code) <= 64
        )),
    CONSTRAINT tap_projection_jobs_timestamp_check
        CHECK (updated_at >= created_at AND (completed_at IS NULL OR completed_at >= created_at))
);

CREATE INDEX tap_projection_jobs_claim_idx
    ON tap_projection_jobs (next_attempt_at, id)
    WHERE state = 'pending' OR state = 'processing';

CREATE INDEX tap_projection_jobs_dependency_idx
    ON tap_projection_jobs (dependency_kind, dependency_key, id)
    WHERE state = 'blocked';

CREATE TABLE tap_quarantined_events (
    event_fingerprint BYTEA       NOT NULL PRIMARY KEY,
    tap_event_id      BIGINT      NOT NULL CHECK (tap_event_id >= 0),
    event_type        TEXT        NOT NULL,
    reason_code       TEXT        NOT NULL
        CHECK (reason_code IN (
            'invalid_envelope', 'missing_record', 'invalid_did',
            'invalid_collection', 'invalid_record_key', 'unsupported_action',
            'malformed_record', 'record_too_large', 'unsupported_event_type', 'invalid_identity',
            'unsupported_identity_status'
        )),
    envelope          JSONB       NOT NULL,
    envelope_bytes    INTEGER     NOT NULL CHECK (envelope_bytes BETWEEN 2 AND 65536),
    occurrence_count  BIGINT      NOT NULL DEFAULT 1 CHECK (occurrence_count > 0),
    replay_state      TEXT        NOT NULL DEFAULT 'quarantined'
        CHECK (replay_state IN ('quarantined', 'pending', 'processing', 'resolved')),
    lease_owner       TEXT,
    lease_token       UUID,
    lease_expires_at  TIMESTAMPTZ,
    first_seen_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at       TIMESTAMPTZ,
    CONSTRAINT tap_quarantined_events_fingerprint_check
        CHECK (octet_length(event_fingerprint) = 32),
    CONSTRAINT tap_quarantined_events_type_check
        CHECK (btrim(event_type) <> '' AND char_length(event_type) <= 64),
    CONSTRAINT tap_quarantined_events_lease_check CHECK (
        (replay_state = 'processing'
            AND btrim(lease_owner) <> '' AND char_length(lease_owner) <= 256
            AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL
            AND resolved_at IS NULL)
        OR
        (replay_state IN ('quarantined', 'pending')
            AND lease_owner IS NULL AND lease_token IS NULL
            AND lease_expires_at IS NULL AND resolved_at IS NULL)
        OR
        (replay_state = 'resolved'
            AND lease_owner IS NULL AND lease_token IS NULL
            AND lease_expires_at IS NULL AND resolved_at IS NOT NULL)
    ),
    CONSTRAINT tap_quarantined_events_timestamp_check
        CHECK (last_seen_at >= first_seen_at AND (resolved_at IS NULL OR resolved_at >= first_seen_at))
);

CREATE INDEX tap_quarantined_events_claim_idx
    ON tap_quarantined_events (last_seen_at, event_fingerprint)
    WHERE replay_state = 'pending' OR replay_state = 'processing';

CREATE TABLE tap_repository_jobs (
    id                    UUID        NOT NULL PRIMARY KEY,
    did                   TEXT        NOT NULL,
    job_kind              TEXT        NOT NULL
        CHECK (job_kind IN ('tap_add_repo', 'pds_reconcile')),
    state                 TEXT        NOT NULL DEFAULT 'pending'
        CHECK (state IN ('pending', 'processing', 'complete')),
    attempts              INTEGER     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    lease_owner           TEXT,
    lease_token           UUID,
    lease_expires_at      TIMESTAMPTZ,
    last_reason_code      TEXT,
    authoritative_revision TEXT,
    last_successful_at    TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, job_kind),
    CONSTRAINT tap_repository_jobs_did_check
        CHECK (btrim(did) <> '' AND char_length(did) <= 512),
    CONSTRAINT tap_repository_jobs_lease_check CHECK (
        (state = 'processing'
            AND btrim(lease_owner) <> '' AND char_length(lease_owner) <= 256
            AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL
            AND last_successful_at IS NULL)
        OR
        (state = 'pending'
            AND lease_owner IS NULL AND lease_token IS NULL
            AND lease_expires_at IS NULL AND last_successful_at IS NULL)
        OR
        (state = 'complete'
            AND lease_owner IS NULL AND lease_token IS NULL
            AND lease_expires_at IS NULL AND last_successful_at IS NOT NULL)
    ),
    CONSTRAINT tap_repository_jobs_reason_check
        CHECK (last_reason_code IS NULL OR (
            btrim(last_reason_code) <> '' AND char_length(last_reason_code) <= 64
        )),
    CONSTRAINT tap_repository_jobs_revision_check
        CHECK (authoritative_revision IS NULL OR (
            btrim(authoritative_revision) <> '' AND char_length(authoritative_revision) <= 128
        )),
    CONSTRAINT tap_repository_jobs_timestamp_check
        CHECK (updated_at >= created_at AND (
            last_successful_at IS NULL OR last_successful_at >= created_at
        ))
);

CREATE INDEX tap_repository_jobs_claim_idx
    ON tap_repository_jobs (next_attempt_at, id)
    WHERE state = 'pending' OR state = 'processing';
