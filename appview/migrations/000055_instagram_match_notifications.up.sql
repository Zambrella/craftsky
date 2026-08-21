-- Restore private Instagram match notifications without restoring automatic
-- follows. Match events are actor-backed but have no AT Protocol source.

ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_type_payload_check,
    DROP CONSTRAINT notification_events_category_check,
    ALTER COLUMN source_uri DROP NOT NULL,
    ALTER COLUMN source_cid DROP NOT NULL,
    ALTER COLUMN source_rkey DROP NOT NULL,
    ADD CONSTRAINT notification_events_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch'
    )),
    ADD CONSTRAINT notification_events_type_payload_check CHECK (
        actor_did IS NOT NULL
        AND (
            (category = 'instagramMatch'
                AND source_uri IS NULL
                AND source_cid IS NULL
                AND source_rkey IS NULL)
            OR
            (category <> 'instagramMatch'
                AND source_uri IS NOT NULL
                AND source_cid IS NOT NULL
                AND source_rkey IS NOT NULL)
        )
    );

CREATE UNIQUE INDEX notification_events_instagram_suggestion_unique
    ON notification_events(recipient_did, category, subject_key)
    WHERE category = 'instagramMatch';

ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_category_check,
    ADD CONSTRAINT notification_preferences_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch'
    )),
    ADD CONSTRAINT notification_preferences_instagram_match_scope_check CHECK (
        category <> 'instagramMatch' OR scope = 'everyone'
    );
