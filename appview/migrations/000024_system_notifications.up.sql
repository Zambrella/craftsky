ALTER TABLE notification_events
    ADD COLUMN kind TEXT NOT NULL DEFAULT 'social' CHECK (kind IN ('social', 'system')),
    ADD COLUMN system_count INTEGER,
    ADD COLUMN system_count_capped BOOLEAN,
    ADD COLUMN system_destination TEXT,
    ADD COLUMN system_group_key TEXT,
    ADD COLUMN coalesce_until TIMESTAMPTZ,
    ADD COLUMN system_push_released_at TIMESTAMPTZ;

ALTER TABLE notification_events
    ALTER COLUMN actor_did DROP NOT NULL,
    ALTER COLUMN source_uri DROP NOT NULL,
    ALTER COLUMN source_cid DROP NOT NULL,
    ALTER COLUMN source_rkey DROP NOT NULL;

ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_category_check,
    ADD CONSTRAINT notification_events_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch'
    ));

ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_category_check,
    ADD CONSTRAINT notification_preferences_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch'
    )),
    ADD CONSTRAINT notification_preferences_instagram_match_scope_check CHECK (
        category <> 'instagramMatch' OR scope = 'everyone'
    );

ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_recipient_actor_category_subject_key_key;

CREATE UNIQUE INDEX notification_events_social_semantic_unique
    ON notification_events (recipient_did, actor_did, category, subject_key)
    WHERE kind = 'social';

CREATE UNIQUE INDEX notification_events_system_group_unique
    ON notification_events (recipient_did, category, system_group_key)
    WHERE kind = 'system';

ALTER TABLE notification_events
    ADD CONSTRAINT notification_events_kind_payload_check CHECK (
        (
            kind = 'social'
            AND category <> 'instagramMatch'
            AND actor_did IS NOT NULL
            AND source_uri IS NOT NULL
            AND source_cid IS NOT NULL
            AND source_rkey IS NOT NULL
            AND system_count IS NULL
            AND system_count_capped IS NULL
            AND system_destination IS NULL
            AND system_group_key IS NULL
            AND coalesce_until IS NULL
            AND system_push_released_at IS NULL
        ) OR (
            kind = 'system'
            AND category = 'instagramMatch'
            AND actor_did IS NULL
            AND source_uri IS NULL
            AND source_cid IS NULL
            AND source_rkey IS NULL
            AND subject_uri IS NULL
            AND subject_cid IS NULL
            AND parent_uri IS NULL
            AND parent_cid IS NULL
            AND root_uri IS NULL
            AND root_cid IS NULL
            AND quoted_uri IS NULL
            AND quoted_cid IS NULL
            AND eligibility_scope = 'everyone'
            AND NOT recipient_followed_actor
            AND system_count BETWEEN 1 AND 99
            AND system_count_capped IS NOT NULL
            AND system_destination = 'instagramMigration'
            AND system_group_key IS NOT NULL
            AND coalesce_until IS NOT NULL
        )
    );

CREATE INDEX notification_events_system_close_idx
    ON notification_events (coalesce_until, id)
    WHERE kind = 'system'
      AND state = 'active'
      AND system_push_released_at IS NULL;

CREATE TABLE instagram_notification_suggestions (
    notification_id UUID        NOT NULL REFERENCES notification_events(id) ON DELETE CASCADE,
    suggestion_id   UUID        NOT NULL REFERENCES instagram_follow_suggestions(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (notification_id, suggestion_id)
);

CREATE UNIQUE INDEX instagram_notification_suggestions_suggestion_idx
    ON instagram_notification_suggestions (suggestion_id);

CREATE OR REPLACE FUNCTION set_notification_newness_revision()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.state = 'active' AND (
        OLD.state = 'retracted'
        OR OLD.source_uri IS DISTINCT FROM NEW.source_uri
        OR OLD.source_cid IS DISTINCT FROM NEW.source_cid
        OR (
            NEW.kind = 'system'
            AND (
                COALESCE(NEW.system_count, 0) > COALESCE(OLD.system_count, 0)
                OR (
                    COALESCE(NEW.system_count_capped, false)
                    AND NOT COALESCE(OLD.system_count_capped, false)
                )
                OR NEW.activity_at > OLD.activity_at
            )
        )
    ) THEN
        NEW.newness_revision := nextval('notification_newness_revision_seq');
    END IF;
    RETURN NEW;
END;
$$;
