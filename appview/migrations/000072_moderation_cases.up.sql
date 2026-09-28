-- Private moderation cases, append-only adjudication history, and current
-- account-standing projections. Existing reports and outputs are intentionally
-- not backfilled: only post-migration intake participates in case grouping.

CREATE TABLE moderation_cases (
    id                  UUID        NOT NULL PRIMARY KEY,
    subject_key         TEXT        NOT NULL,
    subject_type        TEXT        NOT NULL CHECK (subject_type IN ('post', 'account', 'event')),
    subject_did         TEXT        NOT NULL,
    subject_collection  TEXT,
    subject_rkey        TEXT,
    subject_uri         TEXT,
    subject_cid_snapshot TEXT,
    owner_did           TEXT        NOT NULL,
    safe_snapshot       JSONB       NOT NULL,
    state               TEXT        NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'resolved')),
    revision            BIGINT      NOT NULL DEFAULT 0 CHECK (revision >= 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at         TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (substring(id::text from 15 for 1) = '4'),
    CHECK (substring(id::text from 20 for 1) IN ('8', '9', 'a', 'b')),
    CHECK (char_length(subject_key) > 0),
    CHECK (char_length(owner_did) > 0),
    CHECK (
        (subject_type = 'account'
            AND subject_collection IS NULL
            AND subject_rkey IS NULL
            AND subject_uri IS NULL)
        OR
        (subject_type = 'post'
            AND subject_collection IS NOT NULL
            AND subject_rkey IS NOT NULL
            AND subject_uri IS NOT NULL)
        OR
        (subject_type = 'event'
            AND subject_collection = 'social.craftsky.business.event'
            AND subject_rkey IS NOT NULL
            AND subject_uri IS NOT NULL)
    ),
    CHECK (
        (state = 'open' AND resolved_at IS NULL)
        OR (state = 'resolved' AND resolved_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX moderation_cases_one_open_subject_idx
    ON moderation_cases(subject_key)
    WHERE state = 'open';
CREATE INDEX moderation_cases_queue_idx
    ON moderation_cases(state, created_at DESC, id DESC);
CREATE INDEX moderation_cases_owner_idx
    ON moderation_cases(owner_did, created_at DESC, id DESC);
CREATE INDEX moderation_cases_owner_purge_idx
    ON moderation_cases(owner_did, id);
CREATE INDEX moderation_cases_subject_purge_idx
    ON moderation_cases(subject_did, id);

CREATE TABLE moderation_case_reports (
    case_id      UUID        NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
    report_id    TEXT        NOT NULL UNIQUE REFERENCES moderation_reports(id) ON DELETE CASCADE,
    attached_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (case_id, report_id)
);

CREATE INDEX moderation_case_reports_case_attached_idx
    ON moderation_case_reports(case_id, attached_at, report_id);

CREATE TABLE moderation_account_standings (
    owner_did              TEXT        NOT NULL PRIMARY KEY,
    active_strike_count    INTEGER     NOT NULL DEFAULT 0 CHECK (active_strike_count >= 0),
    threshold_suspended    BOOLEAN     NOT NULL DEFAULT false,
    severe_suspended       BOOLEAN     NOT NULL DEFAULT false,
    effective_suspended    BOOLEAN     GENERATED ALWAYS AS
        (threshold_suspended OR severe_suspended) STORED,
    revision               BIGINT      NOT NULL DEFAULT 0 CHECK (revision >= 0),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (threshold_suspended = (active_strike_count >= 3))
);

CREATE INDEX moderation_account_standings_suspended_idx
    ON moderation_account_standings(owner_did)
    WHERE effective_suspended;

CREATE TABLE moderation_case_events (
    id                   UUID        NOT NULL PRIMARY KEY,
    case_id              UUID        NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
    event_type           TEXT        NOT NULL CHECK (event_type IN (
        'decision', 'appealConfirmed', 'appealResolved', 'effectsChanged',
        'strikeExpired', 'severeRestored'
    )),
    actor_id             TEXT        NOT NULL,
    source_system        TEXT        NOT NULL,
    replay_id            TEXT        NOT NULL,
    request_fingerprint  BYTEA       NOT NULL CHECK (octet_length(request_fingerprint) = 32),
    expected_revision    BIGINT      NOT NULL CHECK (expected_revision >= 0),
    result_revision      BIGINT      NOT NULL CHECK (result_revision = expected_revision + 1),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (substring(id::text from 15 for 1) = '4'),
    CHECK (substring(id::text from 20 for 1) IN ('8', '9', 'a', 'b')),
    CHECK (char_length(actor_id) > 0),
    CHECK (char_length(source_system) > 0),
    CHECK (char_length(replay_id) > 0),
    UNIQUE (case_id, result_revision)
);

CREATE UNIQUE INDEX moderation_case_events_replay_idx
    ON moderation_case_events(source_system, replay_id);
CREATE INDEX moderation_case_events_case_created_idx
    ON moderation_case_events(case_id, created_at, id);

CREATE TABLE moderation_decisions (
    id                       UUID        NOT NULL PRIMARY KEY,
    case_id                  UUID        NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
    case_event_id            UUID        NOT NULL UNIQUE REFERENCES moderation_case_events(id) ON DELETE RESTRICT,
    disposition              TEXT        NOT NULL CHECK (disposition IN ('violation', 'noAction')),
    reason                   TEXT CHECK (reason IN (
        'harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated',
        'adult_or_graphic', 'impersonation', 'off_topic',
        'intellectual_property', 'other'
    )),
    internal_evidence_notes  TEXT,
    user_safe_detail         TEXT,
    severity_rationale       TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (substring(id::text from 15 for 1) = '4'),
    CHECK (substring(id::text from 20 for 1) IN ('8', '9', 'a', 'b')),
    CHECK (
        (disposition = 'violation'
            AND reason IS NOT NULL
            AND internal_evidence_notes IS NOT NULL
            AND char_length(btrim(internal_evidence_notes)) > 0)
        OR
        (disposition = 'noAction'
            AND reason IS NULL
            AND user_safe_detail IS NULL
            AND severity_rationale IS NULL)
    )
);

CREATE INDEX moderation_decisions_case_created_idx
    ON moderation_decisions(case_id, created_at, id);

CREATE TABLE moderation_effect_events (
    id                    UUID        NOT NULL PRIMARY KEY,
    case_id               UUID        NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
    case_event_id         UUID        NOT NULL REFERENCES moderation_case_events(id) ON DELETE RESTRICT,
    logical_effect_id     UUID        NOT NULL,
    effect_type           TEXT        NOT NULL CHECK (effect_type IN (
        'formalWarning', 'visibilityWarn', 'visibilityHide',
        'visibilityTakedown', 'strike', 'severeSuspension'
    )),
    action                TEXT        NOT NULL CHECK (action IN ('apply', 'negate', 'expire', 'restore')),
    moderation_output_id  TEXT REFERENCES moderation_outputs(id) ON DELETE RESTRICT,
    rationale             TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (substring(id::text from 15 for 1) = '4'),
    CHECK (substring(id::text from 20 for 1) IN ('8', '9', 'a', 'b')),
    CHECK (substring(logical_effect_id::text from 15 for 1) = '4'),
    CHECK (substring(logical_effect_id::text from 20 for 1) IN ('8', '9', 'a', 'b')),
    CHECK (
        (effect_type IN ('visibilityWarn', 'visibilityHide', 'visibilityTakedown')
            AND moderation_output_id IS NOT NULL)
        OR
        (effect_type NOT IN ('visibilityWarn', 'visibilityHide', 'visibilityTakedown')
            AND moderation_output_id IS NULL)
    )
);

CREATE INDEX moderation_effect_events_case_created_idx
    ON moderation_effect_events(case_id, created_at, id);
CREATE INDEX moderation_effect_events_logical_idx
    ON moderation_effect_events(logical_effect_id, created_at, id);

CREATE TABLE moderation_active_case_effects (
    case_id               UUID        NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
    effect_type           TEXT        NOT NULL CHECK (effect_type IN (
        'formalWarning', 'visibilityWarn', 'visibilityHide',
        'visibilityTakedown', 'strike', 'severeSuspension'
    )),
    logical_effect_id     UUID        NOT NULL UNIQUE,
    applied_event_id      UUID        NOT NULL REFERENCES moderation_case_events(id) ON DELETE RESTRICT,
    moderation_output_id  TEXT REFERENCES moderation_outputs(id) ON DELETE RESTRICT,
    applied_at            TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (case_id, effect_type),
    CHECK (substring(logical_effect_id::text from 15 for 1) = '4'),
    CHECK (substring(logical_effect_id::text from 20 for 1) IN ('8', '9', 'a', 'b')),
    CHECK (
        (effect_type IN ('visibilityWarn', 'visibilityHide', 'visibilityTakedown')
            AND moderation_output_id IS NOT NULL)
        OR
        (effect_type NOT IN ('visibilityWarn', 'visibilityHide', 'visibilityTakedown')
            AND moderation_output_id IS NULL)
    )
);

CREATE TABLE moderation_case_strikes (
    case_id             UUID        NOT NULL PRIMARY KEY REFERENCES moderation_cases(id) ON DELETE CASCADE,
    logical_effect_id   UUID        NOT NULL UNIQUE,
    owner_did           TEXT        NOT NULL,
    issued_at           TIMESTAMPTZ NOT NULL,
    due_at              TIMESTAMPTZ NOT NULL,
    expired_at          TIMESTAMPTZ,
    overturned_at       TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (substring(logical_effect_id::text from 15 for 1) = '4'),
    CHECK (substring(logical_effect_id::text from 20 for 1) IN ('8', '9', 'a', 'b')),
    CHECK (due_at > issued_at),
    CHECK (expired_at IS NULL OR overturned_at IS NULL)
);

CREATE INDEX moderation_case_strikes_due_idx
    ON moderation_case_strikes(due_at, case_id)
    WHERE expired_at IS NULL AND overturned_at IS NULL;
CREATE INDEX moderation_case_strikes_owner_idx
    ON moderation_case_strikes(owner_did, issued_at, case_id);
CREATE INDEX moderation_case_strikes_owner_purge_idx
    ON moderation_case_strikes(owner_did, case_id);

CREATE TABLE moderation_appeal_correspondence (
    id                     UUID        NOT NULL PRIMARY KEY,
    case_id                UUID        NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
    source_system          TEXT        NOT NULL,
    replay_id              TEXT        NOT NULL,
    sender_reference_hash  BYTEA,
    received_at            TIMESTAMPTZ NOT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (substring(id::text from 15 for 1) = '4'),
    CHECK (substring(id::text from 20 for 1) IN ('8', '9', 'a', 'b')),
    CHECK (char_length(source_system) > 0),
    CHECK (char_length(replay_id) > 0),
    UNIQUE (source_system, replay_id)
);

CREATE INDEX moderation_appeal_correspondence_case_received_idx
    ON moderation_appeal_correspondence(case_id, received_at, id);

CREATE TABLE moderation_appeals (
    case_id             UUID        NOT NULL PRIMARY KEY REFERENCES moderation_cases(id) ON DELETE CASCADE,
    confirmed_event_id  UUID        NOT NULL UNIQUE REFERENCES moderation_case_events(id) ON DELETE RESTRICT,
    resolved_event_id   UUID        UNIQUE REFERENCES moderation_case_events(id) ON DELETE RESTRICT,
    status              TEXT        NOT NULL CHECK (status IN ('pending', 'upheld', 'changed')),
    confirmed_at        TIMESTAMPTZ NOT NULL,
    resolved_at         TIMESTAMPTZ,
    CHECK (
        (status = 'pending' AND resolved_event_id IS NULL AND resolved_at IS NULL)
        OR
        (status IN ('upheld', 'changed') AND resolved_event_id IS NOT NULL AND resolved_at IS NOT NULL)
    )
);

ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_type_payload_check,
    DROP CONSTRAINT notification_events_category_check,
	ALTER COLUMN actor_did DROP NOT NULL,
	ADD COLUMN moderation_case_reference TEXT,
	ADD COLUMN moderation_event_id UUID,
    ADD CONSTRAINT notification_events_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch', 'moderation'
    )),
    ADD CONSTRAINT notification_events_type_payload_check CHECK (
		(category = 'moderation'
			AND actor_did IS NULL
			AND moderation_case_reference ~ '^MOD-[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
			AND moderation_event_id IS NOT NULL
            AND source_uri IS NULL
            AND source_cid IS NULL
            AND source_rkey IS NULL
            AND eligibility_scope = 'everyone')
        OR
		(category = 'instagramMatch'
			AND actor_did IS NOT NULL
			AND moderation_case_reference IS NULL
			AND moderation_event_id IS NULL
            AND source_uri IS NULL
            AND source_cid IS NULL
            AND source_rkey IS NULL)
        OR
		(category NOT IN ('moderation', 'instagramMatch')
			AND actor_did IS NOT NULL
			AND moderation_case_reference IS NULL
			AND moderation_event_id IS NULL
            AND source_uri IS NOT NULL
            AND source_cid IS NOT NULL
            AND source_rkey IS NOT NULL)
    );

CREATE UNIQUE INDEX notification_events_moderation_event_unique
	ON notification_events(moderation_event_id)
	WHERE category = 'moderation';

ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_instagram_match_scope_check,
    DROP CONSTRAINT notification_preferences_category_check,
    ADD CONSTRAINT notification_preferences_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch', 'moderation'
    )),
    ADD CONSTRAINT notification_preferences_fixed_scope_check CHECK (
        category NOT IN ('instagramMatch', 'moderation') OR scope = 'everyone'
    );
