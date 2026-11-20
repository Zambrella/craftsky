DROP TRIGGER notification_seen_state_legacy_ack_trigger ON notification_seen_state;
DROP FUNCTION apply_legacy_notification_seen_marker();

-- The old watermark cannot represent gaps. Preserve the seen prefix without
-- suppressing any active notification that was still new.
TRUNCATE notification_seen_state;
WITH oldest_unseen AS (
    SELECT recipient_did, min(newness_revision) AS revision
    FROM notification_events
    WHERE state = 'active' AND is_new
    GROUP BY recipient_did
)
INSERT INTO notification_seen_state (account_did, last_seen_revision)
SELECT event.recipient_did, max(event.newness_revision)
FROM notification_events AS event
LEFT JOIN oldest_unseen AS unseen USING (recipient_did)
WHERE NOT event.is_new
  AND (unseen.revision IS NULL OR event.newness_revision < unseen.revision)
GROUP BY event.recipient_did;

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

DROP INDEX notification_events_unseen_idx;
ALTER TABLE notification_events DROP COLUMN is_new;
