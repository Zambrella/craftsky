UPDATE atproto_identity_refresh_state
SET last_result = 'retry'
WHERE last_result = 'pending';

ALTER TABLE atproto_identity_refresh_state
    DROP COLUMN tap_event_id;

ALTER TABLE atproto_identity_refresh_state
    DROP CONSTRAINT atproto_identity_refresh_state_last_result_check;

ALTER TABLE atproto_identity_refresh_state
    ADD CONSTRAINT atproto_identity_refresh_state_last_result_check
    CHECK (last_result IN ('retry'));

