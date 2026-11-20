ALTER TABLE notification_events ADD COLUMN is_new BOOLEAN NOT NULL DEFAULT true;

UPDATE notification_events AS event
SET is_new = false
FROM notification_seen_state AS seen
WHERE seen.account_did = event.recipient_did
  AND event.newness_revision <= seen.last_seen_revision;

CREATE INDEX notification_events_unseen_idx ON notification_events (recipient_did)
    WHERE state = 'active' AND is_new;

-- Pre-deploy migrations can run while the previous AppView still serves
-- requests. Import its acknowledgements until the new AppView takes over.
CREATE FUNCTION apply_legacy_notification_seen_marker()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE notification_events
    SET is_new = false
    WHERE recipient_did = NEW.account_did AND state = 'active' AND is_new
      AND newness_revision <= NEW.last_seen_revision;
    RETURN NEW;
END;
$$;

CREATE TRIGGER notification_seen_state_legacy_ack_trigger
    AFTER INSERT OR UPDATE OF last_seen_revision ON notification_seen_state
    FOR EACH ROW
    EXECUTE FUNCTION apply_legacy_notification_seen_marker();

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
        NEW.is_new := true;
    END IF;
    RETURN NEW;
END;
$$;
