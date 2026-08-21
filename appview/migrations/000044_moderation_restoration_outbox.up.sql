-- Synthetic moderation rows predate the required HTTP idempotency contract.
-- This is a pre-production reset: do not invent replayable keys for legacy
-- development rows whose original request identity is unknowable.
DELETE FROM moderation_outputs;

CREATE TABLE moderation_restoration_outbox (
    moderation_output_id   TEXT        NOT NULL PRIMARY KEY
        REFERENCES moderation_outputs(id) ON DELETE RESTRICT,
    target_did             TEXT        NOT NULL CHECK (target_did <> ''),
    status                 TEXT        NOT NULL CHECK (status IN (
                                'pending',
                                'queued',
                                'no_work',
                                'cancelled_target_terminal'
                            )),
    reconciliation_job_id  UUID
        REFERENCES instagram_reconciliation_jobs(id) ON DELETE SET NULL,
    created_at             TIMESTAMPTZ NOT NULL,
    processed_at           TIMESTAMPTZ,

    CONSTRAINT moderation_restoration_outbox_state_check CHECK (
        (status = 'pending'
            AND reconciliation_job_id IS NULL
            AND processed_at IS NULL)
        OR
        (status = 'queued' AND processed_at IS NOT NULL)
        OR
        (status IN ('no_work', 'cancelled_target_terminal')
            AND reconciliation_job_id IS NULL
            AND processed_at IS NOT NULL)
    ),
    CONSTRAINT moderation_restoration_outbox_time_check CHECK (
        processed_at IS NULL OR processed_at >= created_at
    )
);

CREATE INDEX moderation_restoration_outbox_pending_idx
    ON moderation_restoration_outbox (created_at, moderation_output_id)
    WHERE status = 'pending';

CREATE INDEX moderation_restoration_outbox_reconciliation_job_id_idx
    ON moderation_restoration_outbox (reconciliation_job_id)
    WHERE reconciliation_job_id IS NOT NULL;

CREATE INDEX moderation_restoration_outbox_retention_idx
    ON moderation_restoration_outbox (processed_at, moderation_output_id)
    WHERE status IN ('queued', 'no_work', 'cancelled_target_terminal');

CREATE TABLE moderation_restoration_history (
    moderation_output_id  TEXT        NOT NULL PRIMARY KEY,
    outcome               TEXT        NOT NULL CHECK (outcome IN (
                              'queued',
                              'no_work',
                              'cancelled_target_terminal'
                          )),
    processed_at          TIMESTAMPTZ NOT NULL,
    archived_at           TIMESTAMPTZ NOT NULL,

    CONSTRAINT moderation_restoration_history_time_check CHECK (
        archived_at >= processed_at
    )
);

CREATE INDEX moderation_restoration_history_retention_idx
    ON moderation_restoration_history (archived_at, moderation_output_id);

CREATE TABLE moderation_idempotency_receipts (
    request_key_hash     BYTEA       NOT NULL PRIMARY KEY
        CHECK (octet_length(request_key_hash) = 32),
    request_fingerprint  BYTEA       NOT NULL
        CHECK (octet_length(request_fingerprint) = 32),
    output_id            TEXT        NOT NULL,
    output_status        TEXT        NOT NULL CHECK (output_status = 'indexed'),
    created_at           TIMESTAMPTZ NOT NULL,
    expires_at           TIMESTAMPTZ NOT NULL,

    CONSTRAINT moderation_idempotency_receipts_expiry_check CHECK (
        expires_at > created_at
    )
);

CREATE INDEX moderation_idempotency_receipts_expires_at_idx
    ON moderation_idempotency_receipts (expires_at, request_key_hash);

CREATE INDEX moderation_idempotency_receipts_output_id_idx
    ON moderation_idempotency_receipts (output_id, request_key_hash);
