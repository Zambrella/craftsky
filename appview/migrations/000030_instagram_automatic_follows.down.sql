DROP INDEX IF EXISTS pds_follow_operations_expired_lease_idx;
DROP INDEX IF EXISTS pds_follow_operations_claim_idx;
DROP INDEX IF EXISTS pds_follow_operations_owner_target_unique;

DELETE FROM notification_events WHERE category = 'instagramMatch';

ALTER TABLE notification_events
    DROP CONSTRAINT notification_events_type_payload_check;

DROP INDEX IF EXISTS notification_events_instagram_operation_unique;

ALTER TABLE notification_events
    ADD COLUMN system_count INTEGER,
    ADD COLUMN system_count_capped BOOLEAN,
    ADD COLUMN system_group_key TEXT,
    ADD COLUMN coalesce_until TIMESTAMPTZ,
    ADD COLUMN system_push_released_at TIMESTAMPTZ;

CREATE UNIQUE INDEX notification_events_system_group_unique
    ON notification_events (recipient_did, category, system_group_key)
    WHERE category = 'instagramMatch';

CREATE INDEX notification_events_system_close_idx
    ON notification_events (coalesce_until, id)
    WHERE category = 'instagramMatch'
      AND state = 'active'
      AND system_push_released_at IS NULL;

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
            AND system_group_key IS NOT NULL
            AND coalesce_until IS NOT NULL
        )
    );

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
            NEW.category = 'instagramMatch'
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

UPDATE instagram_follow_suggestions
SET state = CASE state
        WHEN 'followed' THEN 'accepted'
        WHEN 'writing' THEN 'accepting'
        ELSE state
    END;

ALTER TABLE instagram_follow_suggestions
    DROP CONSTRAINT instagram_follow_suggestions_state_check,
    ADD CONSTRAINT instagram_follow_suggestions_state_check CHECK (state IN (
        'pending', 'accepting', 'accepted',
        'alreadyFollowing', 'dismissed', 'invalidated'
    ));

UPDATE pds_follow_operations operation
SET status = CASE
        WHEN operation.status = 'followed' THEN 'succeeded'
        WHEN operation.status = 'invalidated' THEN 'failed'
        ELSE operation.status
    END;

ALTER TABLE pds_follow_operations
    DROP CONSTRAINT pds_follow_operations_lease_shape_check,
    DROP CONSTRAINT pds_follow_operations_status_check,
    ADD CONSTRAINT pds_follow_operations_status_check CHECK (status IN (
        'pending', 'writing', 'succeeded', 'alreadyFollowing', 'failed'
    )),
    DROP COLUMN next_attempt_at,
    DROP COLUMN lease_expires_at,
    DROP COLUMN lease_token;

CREATE INDEX pds_follow_operations_recovery_idx
    ON pds_follow_operations (updated_at, id)
    WHERE status IN ('pending', 'writing');
