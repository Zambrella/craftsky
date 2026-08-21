ALTER TABLE atproto_identity_refresh_state
    ADD COLUMN refresh_version BIGINT NOT NULL DEFAULT 1
    CHECK (refresh_version > 0);

COMMENT ON COLUMN atproto_identity_refresh_state.refresh_version IS
    'Database-owned CAS version; Tap event IDs are diagnostic and may reset.';
