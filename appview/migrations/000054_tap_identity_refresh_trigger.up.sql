ALTER TABLE atproto_identity_refresh_state
    DROP CONSTRAINT atproto_identity_refresh_state_last_result_check;

ALTER TABLE atproto_identity_refresh_state
    ADD CONSTRAINT atproto_identity_refresh_state_last_result_check
    CHECK (last_result IN ('pending', 'retry'));

ALTER TABLE atproto_identity_refresh_state
    ADD COLUMN tap_event_id BIGINT
    CHECK (tap_event_id > 0);

