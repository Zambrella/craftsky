UPDATE instagram_follow_suggestions
SET state = CASE state
        WHEN 'accepted' THEN 'followed'
        WHEN 'accepting' THEN 'pending'
        WHEN 'dismissed' THEN 'invalidated'
        ELSE state
    END,
    accepting_since = NULL;

ALTER TABLE instagram_follow_suggestions
    DROP CONSTRAINT instagram_follow_suggestions_state_check,
    ADD CONSTRAINT instagram_follow_suggestions_state_check CHECK (state IN (
        'pending', 'writing', 'followed', 'alreadyFollowing', 'invalidated'
    ));

UPDATE pds_follow_operations operation
SET status = CASE
        WHEN operation.status = 'succeeded' THEN 'followed'
        WHEN suggestion.state = 'followed' THEN 'followed'
        WHEN suggestion.state = 'alreadyFollowing' THEN 'alreadyFollowing'
        WHEN suggestion.state = 'invalidated' THEN 'invalidated'
        ELSE 'pending'
    END
FROM instagram_follow_suggestions suggestion
WHERE suggestion.id = operation.suggestion_id;

ALTER TABLE pds_follow_operations
    DROP CONSTRAINT pds_follow_operations_status_check,
    ADD CONSTRAINT pds_follow_operations_status_check CHECK (status IN (
        'pending', 'writing', 'followed', 'alreadyFollowing', 'invalidated'
    )),
    ADD COLUMN lease_token UUID,
    ADD COLUMN lease_expires_at TIMESTAMPTZ,
    ADD COLUMN next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE pds_follow_operations
    ADD CONSTRAINT pds_follow_operations_lease_shape_check CHECK (
        (
            status = 'writing'
            AND lease_token IS NOT NULL
            AND lease_expires_at IS NOT NULL
        ) OR (
            status <> 'writing'
            AND lease_token IS NULL
            AND lease_expires_at IS NULL
        )
    );

DROP INDEX pds_follow_operations_recovery_idx;

CREATE INDEX pds_follow_operations_claim_idx
    ON pds_follow_operations (next_attempt_at, id)
    WHERE status = 'pending';

CREATE INDEX pds_follow_operations_expired_lease_idx
    ON pds_follow_operations (lease_expires_at, id)
    WHERE status = 'writing';

CREATE UNIQUE INDEX pds_follow_operations_owner_target_unique
    ON pds_follow_operations (owner_did, target_did);

DELETE FROM notification_events WHERE category = 'instagramMatch';

DROP TABLE instagram_notification_suggestions;
DROP INDEX notification_events_system_close_idx;
DROP INDEX notification_events_system_group_unique;

ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_type_payload_check,
    DROP COLUMN system_push_released_at,
    DROP COLUMN coalesce_until,
    DROP COLUMN system_group_key,
    DROP COLUMN system_count_capped,
    DROP COLUMN system_count;

CREATE UNIQUE INDEX notification_events_instagram_operation_unique
    ON notification_events (recipient_did, category, subject_key)
    WHERE category = 'instagramMatch';

ALTER TABLE notification_events
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
