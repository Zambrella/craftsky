DELETE FROM notification_preferences WHERE category = 'instagramMatch';
DELETE FROM notification_events WHERE category = 'instagramMatch';

DROP INDEX IF EXISTS notification_events_instagram_suggestion_unique;

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
