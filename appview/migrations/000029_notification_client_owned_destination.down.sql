ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_type_payload_check;

ALTER TABLE notification_events
    ADD COLUMN system_destination TEXT;

UPDATE notification_events
SET system_destination = 'instagramMigration'
WHERE category = 'instagramMatch';

ALTER TABLE notification_events
    ADD CONSTRAINT notification_events_type_payload_check CHECK (
        (
            category <> 'instagramMatch'
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
            category = 'instagramMatch'
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
