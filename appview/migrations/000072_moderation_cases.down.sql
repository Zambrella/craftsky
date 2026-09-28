DELETE FROM notification_preferences WHERE category = 'moderation';
DELETE FROM notification_events WHERE category = 'moderation';

ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_fixed_scope_check,
    DROP CONSTRAINT notification_preferences_category_check,
    ADD CONSTRAINT notification_preferences_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch'
    )),
    ADD CONSTRAINT notification_preferences_instagram_match_scope_check CHECK (
        category <> 'instagramMatch' OR scope = 'everyone'
    );

DROP INDEX IF EXISTS notification_events_moderation_event_unique;

ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_type_payload_check,
    DROP CONSTRAINT notification_events_category_check,
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
    ),
	DROP COLUMN moderation_event_id,
	DROP COLUMN moderation_case_reference,
	ALTER COLUMN actor_did SET NOT NULL;

DROP TABLE IF EXISTS moderation_appeals;
DROP TABLE IF EXISTS moderation_appeal_correspondence;
DROP TABLE IF EXISTS moderation_case_strikes;
DROP TABLE IF EXISTS moderation_active_case_effects;
DROP TABLE IF EXISTS moderation_effect_events;
DROP TABLE IF EXISTS moderation_decisions;
DROP TABLE IF EXISTS moderation_case_events;
DROP TABLE IF EXISTS moderation_account_standings;
DROP TABLE IF EXISTS moderation_case_reports;
DROP TABLE IF EXISTS moderation_cases;
