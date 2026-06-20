CREATE TABLE craftsky_recent_searches (
    id TEXT PRIMARY KEY,
    viewer_did TEXT NOT NULL,
    search_type TEXT NOT NULL CHECK (search_type IN ('hashtag', 'profile', 'post', 'project')),
    display_label TEXT NOT NULL,
    normalized_payload JSONB NOT NULL,
    normalized_payload_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (viewer_did, search_type, normalized_payload_hash)
);

CREATE INDEX craftsky_recent_searches_viewer_updated_idx
    ON craftsky_recent_searches (viewer_did, updated_at DESC, id DESC);

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX craftsky_posts_search_vector_idx
    ON craftsky_posts USING GIN (to_tsvector('simple', coalesce(text, '')));

CREATE INDEX craftsky_posts_root_created_uri_idx
    ON craftsky_posts (created_at DESC, uri DESC)
    WHERE reply_root_uri IS NULL AND reply_parent_uri IS NULL;

CREATE INDEX craftsky_project_posts_search_vector_idx
    ON craftsky_project_posts USING GIN (to_tsvector('simple',
        coalesce(common_title, '') || ' ' ||
        coalesce(pattern_name, '') || ' ' ||
        coalesce(array_to_string(materials, ' '), '') || ' ' ||
        coalesce(array_to_string(project_tags, ' '), '') || ' ' ||
        coalesce(array_to_string(design_tags, ' '), '')
    ));

CREATE INDEX craftsky_project_posts_lower_craft_type_idx
    ON craftsky_project_posts (lower(common_craft_type));
CREATE INDEX craftsky_project_posts_lower_pattern_difficulty_idx
    ON craftsky_project_posts (lower(pattern_difficulty));

CREATE INDEX atproto_identity_cache_handle_trgm_idx
    ON atproto_identity_cache USING GIN (handle_lower gin_trgm_ops);
CREATE INDEX bluesky_profiles_display_name_trgm_idx
    ON bluesky_profiles USING GIN (lower(display_name) gin_trgm_ops);
CREATE INDEX bluesky_profiles_description_trgm_idx
    ON bluesky_profiles USING GIN (lower(description) gin_trgm_ops);
