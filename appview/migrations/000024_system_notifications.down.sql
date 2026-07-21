DELETE FROM notification_preferences WHERE category = 'instagramMatch';
DELETE FROM notification_events WHERE kind = 'system';

DROP TABLE IF EXISTS instagram_notification_suggestions;
DROP INDEX IF EXISTS notification_events_system_close_idx;

CREATE OR REPLACE FUNCTION set_notification_newness_revision()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.state = 'active' AND (
        OLD.state = 'retracted'
        OR OLD.source_uri IS DISTINCT FROM NEW.source_uri
        OR OLD.source_cid IS DISTINCT FROM NEW.source_cid
    ) THEN
        NEW.newness_revision := nextval('notification_newness_revision_seq');
    END IF;
    RETURN NEW;
END;
$$;

ALTER TABLE notification_events
    DROP CONSTRAINT IF EXISTS notification_events_kind_payload_check;

DROP INDEX IF EXISTS notification_events_system_group_unique;
DROP INDEX IF EXISTS notification_events_social_semantic_unique;

ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_category_check,
    ADD CONSTRAINT notification_events_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost', 'everythingElse'
    )),
    ADD CONSTRAINT notification_events_recipient_actor_category_subject_key_key
        UNIQUE (recipient_did, actor_did, category, subject_key),
    ALTER COLUMN actor_did SET NOT NULL,
    ALTER COLUMN source_uri SET NOT NULL,
    ALTER COLUMN source_cid SET NOT NULL,
    ALTER COLUMN source_rkey SET NOT NULL;

ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_instagram_match_scope_check,
    DROP CONSTRAINT notification_preferences_category_check,
    ADD CONSTRAINT notification_preferences_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost', 'everythingElse'
    ));

ALTER TABLE notification_events
    DROP COLUMN system_push_released_at,
    DROP COLUMN coalesce_until,
    DROP COLUMN system_group_key,
    DROP COLUMN system_destination,
    DROP COLUMN system_count_capped,
    DROP COLUMN system_count,
    DROP COLUMN kind;
