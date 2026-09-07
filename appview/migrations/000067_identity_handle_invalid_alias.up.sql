ALTER TABLE atproto_identity_cache
    DROP CONSTRAINT atproto_identity_cache_handle_lower_key;

CREATE UNIQUE INDEX atproto_identity_cache_valid_handle_lower_unique
    ON atproto_identity_cache (handle_lower)
    WHERE handle_lower <> 'handle.invalid';
