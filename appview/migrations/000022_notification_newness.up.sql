CREATE SEQUENCE notification_newness_revision_seq AS BIGINT;

ALTER TABLE notification_events
    ADD COLUMN newness_revision BIGINT;

UPDATE notification_events
SET newness_revision = nextval('notification_newness_revision_seq');

ALTER TABLE notification_events
    ALTER COLUMN newness_revision SET DEFAULT nextval('notification_newness_revision_seq'),
    ALTER COLUMN newness_revision SET NOT NULL;

CREATE INDEX notification_events_active_newness_idx
    ON notification_events (recipient_did, newness_revision DESC)
    WHERE state = 'active';

CREATE INDEX notification_events_recipient_newness_idx
    ON notification_events (recipient_did, newness_revision DESC);

CREATE TABLE notification_seen_state (
    account_did        TEXT        NOT NULL PRIMARY KEY,
    last_seen_revision BIGINT      NOT NULL DEFAULT 0 CHECK (last_seen_revision >= 0),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE FUNCTION set_notification_newness_revision()
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

CREATE TRIGGER notification_events_newness_revision_trigger
    BEFORE UPDATE ON notification_events
    FOR EACH ROW
    EXECUTE FUNCTION set_notification_newness_revision();
