-- appview/migrations/000015_identity_handle_cache.up.sql
-- Separate atproto identity metadata cache for handle autocomplete.
CREATE TABLE atproto_identity_cache (
    did          TEXT        NOT NULL PRIMARY KEY,
    handle       TEXT        NOT NULL,
    handle_lower TEXT        NOT NULL UNIQUE,
    resolved_at  TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX atproto_identity_cache_handle_lower_pattern_idx
    ON atproto_identity_cache (handle_lower text_pattern_ops);

CREATE INDEX atproto_identity_cache_resolved_at_idx
    ON atproto_identity_cache (resolved_at);
