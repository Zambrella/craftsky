-- AV-007 strict branch: matching produces caller-private suggestions only.
-- This repository is pre-production, so obsolete automatic-write and
-- automatic-notification state is deliberately reset instead of translated.

DELETE FROM notification_preferences WHERE category = 'instagramMatch';
DELETE FROM notification_events WHERE category = 'instagramMatch';

DROP INDEX IF EXISTS notification_events_instagram_operation_unique;

ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_type_payload_check,
    DROP CONSTRAINT notification_events_category_check,
    ADD CONSTRAINT notification_events_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse'
    )),
    ADD CONSTRAINT notification_events_type_payload_check CHECK (
        actor_did IS NOT NULL
        AND source_uri IS NOT NULL
        AND source_cid IS NOT NULL
        AND source_rkey IS NOT NULL
    ),
    ALTER COLUMN actor_did SET NOT NULL,
    ALTER COLUMN source_uri SET NOT NULL,
    ALTER COLUMN source_cid SET NOT NULL,
    ALTER COLUMN source_rkey SET NOT NULL;

ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_instagram_match_scope_check,
    DROP CONSTRAINT notification_preferences_category_check,
    ADD CONSTRAINT notification_preferences_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse'
    ));

TRUNCATE TABLE
    pds_follow_operations,
    instagram_automatic_follow_sources,
    instagram_automatic_follow_ledger;

CREATE TABLE instagram_private_suggestions (
    id                  UUID        NOT NULL PRIMARY KEY,
    importer_did        TEXT        NOT NULL,
    target_did          TEXT        NOT NULL,
    importer_generation BIGINT      NOT NULL CHECK (importer_generation > 0),
    target_generation   BIGINT      NOT NULL CHECK (target_generation > 0),
    evidence_link_id    UUID        NOT NULL REFERENCES instagram_account_links(id) ON DELETE CASCADE,
    state               TEXT        NOT NULL CHECK (state IN (
                            'pending', 'accepting', 'followed',
                            'alreadyFollowing', 'dismissed', 'invalidated'
                        )),
    reason              TEXT        NOT NULL CHECK (reason = 'verifiedInstagramFollow'),
    accepting_since     TIMESTAMPTZ,
    terminal_at         TIMESTAMPTZ,
    result_record_uri   TEXT,
    result_record_cid   TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT instagram_private_suggestions_not_self_check
        CHECK (importer_did <> target_did),
    CONSTRAINT instagram_private_suggestions_state_shape_check CHECK (
        (state = 'pending' AND accepting_since IS NULL AND terminal_at IS NULL)
        OR (state = 'accepting' AND accepting_since IS NOT NULL AND terminal_at IS NULL)
        OR (state IN ('followed','alreadyFollowing','dismissed','invalidated')
            AND accepting_since IS NULL AND terminal_at IS NOT NULL)
    ),
    CONSTRAINT instagram_private_suggestions_result_shape_check CHECK (
        (state = 'followed' AND result_record_uri IS NOT NULL)
        OR (state <> 'followed' AND result_record_uri IS NULL AND result_record_cid IS NULL)
    ),
    UNIQUE (
        importer_did,
        target_did,
        importer_generation,
        target_generation,
        evidence_link_id,
        reason
    )
);

CREATE INDEX instagram_private_suggestions_owner_page_idx
    ON instagram_private_suggestions (importer_did, created_at DESC, id DESC)
    WHERE state = 'pending';
CREATE INDEX instagram_private_suggestions_target_idx
    ON instagram_private_suggestions (target_did, state, id);
CREATE INDEX instagram_private_suggestions_terminal_retention_idx
    ON instagram_private_suggestions (terminal_at, id)
    WHERE terminal_at IS NOT NULL;
CREATE INDEX instagram_private_suggestions_link_idx
    ON instagram_private_suggestions (evidence_link_id, id);

CREATE TABLE instagram_private_suggestion_sources (
    suggestion_id UUID        NOT NULL REFERENCES instagram_private_suggestions(id) ON DELETE CASCADE,
    import_id     UUID        NOT NULL REFERENCES instagram_graph_imports(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (suggestion_id, import_id)
);

CREATE INDEX instagram_private_suggestion_sources_import_idx
    ON instagram_private_suggestion_sources (import_id, suggestion_id);

