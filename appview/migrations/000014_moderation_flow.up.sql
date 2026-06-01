-- appview/migrations/000014_moderation_flow.up.sql
-- Private moderation-report intake and synthetic/Ozone-like moderation outputs.
-- See docs/changes/2026-05-30-moderation-flow-mvp/.

CREATE TABLE moderation_reports (
    id                         TEXT        NOT NULL PRIMARY KEY,
    reporter_did               TEXT        NOT NULL,
    subject_type               TEXT        NOT NULL CHECK (subject_type IN ('post', 'account')),
    subject_did                TEXT        NOT NULL,
    subject_collection         TEXT,
    subject_rkey               TEXT,
    subject_uri                TEXT,
    subject_cid_snapshot       TEXT,
    submitted_handle_snapshot  TEXT,
    reason_type                TEXT        NOT NULL CHECK (reason_type IN (
        'harassment',
        'hate',
        'spam',
        'misleading',
        'suspected_ai_generated',
        'adult_or_graphic',
        'impersonation',
        'off_topic',
        'intellectual_property',
        'other'
    )),
    details                    TEXT,
    device_id                  TEXT,
    forwarding_status          TEXT        NOT NULL CHECK (forwarding_status IN ('prepared_not_submitted')),
    forwarding_schema_version  TEXT,
    forwarding_prepared_at     TIMESTAMPTZ NOT NULL,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (
        (subject_type = 'post' AND subject_collection IS NOT NULL AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL)
        OR
        (subject_type = 'account' AND subject_collection IS NULL AND subject_rkey IS NULL AND subject_uri IS NULL AND subject_cid_snapshot IS NULL)
    ),
    CHECK (details IS NULL OR char_length(details) <= 1000)
);

CREATE INDEX moderation_reports_reporter_created_desc_idx
    ON moderation_reports (reporter_did, created_at DESC);
CREATE INDEX moderation_reports_subject_created_desc_idx
    ON moderation_reports (subject_type, subject_did, created_at DESC);
CREATE INDEX moderation_reports_subject_uri_created_desc_idx
    ON moderation_reports (subject_uri, created_at DESC)
    WHERE subject_uri IS NOT NULL;

CREATE TABLE moderation_outputs (
    id                  TEXT        NOT NULL PRIMARY KEY,
    source_did          TEXT        NOT NULL,
    subject_type        TEXT        NOT NULL CHECK (subject_type IN ('post', 'account')),
    subject_did         TEXT        NOT NULL,
    subject_collection  TEXT,
    subject_rkey        TEXT,
    subject_uri         TEXT,
    value               TEXT        NOT NULL CHECK (value IN ('hide', 'takedown', 'warn')),
    action              TEXT        NOT NULL CHECK (action IN ('apply', 'negate')),
    internal_reason     TEXT,
    expires_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    indexed_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (
        (subject_type = 'post' AND subject_collection IS NOT NULL AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL)
        OR
        (subject_type = 'account' AND subject_collection IS NULL AND subject_rkey IS NULL AND subject_uri IS NULL)
    )
);

CREATE INDEX moderation_outputs_subject_value_indexed_desc_idx
    ON moderation_outputs (subject_type, subject_did, value, indexed_at DESC);
CREATE INDEX moderation_outputs_subject_uri_value_indexed_desc_idx
    ON moderation_outputs (subject_uri, value, indexed_at DESC)
    WHERE subject_uri IS NOT NULL;
CREATE INDEX moderation_outputs_source_subject_value_indexed_desc_idx
    ON moderation_outputs (source_did, subject_type, subject_did, value, indexed_at DESC);
CREATE INDEX moderation_outputs_expires_at_idx
    ON moderation_outputs (expires_at)
    WHERE expires_at IS NOT NULL;
