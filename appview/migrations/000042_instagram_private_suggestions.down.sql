DROP TABLE IF EXISTS instagram_private_suggestion_sources;
DROP TABLE IF EXISTS instagram_private_suggestions;

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
    ALTER COLUMN actor_did DROP NOT NULL,
    ALTER COLUMN source_uri DROP NOT NULL,
    ALTER COLUMN source_cid DROP NOT NULL,
    ALTER COLUMN source_rkey DROP NOT NULL,
    DROP CONSTRAINT notification_events_type_payload_check,
    DROP CONSTRAINT notification_events_category_check,
    ADD CONSTRAINT notification_events_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch'
    )),
    ADD CONSTRAINT notification_events_type_payload_check CHECK (
        (
            category <> 'instagramMatch'
            AND actor_did IS NOT NULL
            AND source_uri IS NOT NULL
            AND source_cid IS NOT NULL
            AND source_rkey IS NOT NULL
        ) OR (
            category = 'instagramMatch'
            AND actor_did IS NOT NULL
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
        )
    );

CREATE UNIQUE INDEX notification_events_instagram_operation_unique
    ON notification_events (recipient_did, category, subject_key)
    WHERE category = 'instagramMatch';
