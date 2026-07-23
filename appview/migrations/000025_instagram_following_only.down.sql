DROP INDEX IF EXISTS instagram_graph_handles_match_idx;

ALTER TABLE instagram_graph_handles
    DROP CONSTRAINT IF EXISTS instagram_graph_handles_import_username_key,
    DROP CONSTRAINT IF EXISTS instagram_graph_handles_import_id_username_normalized_key,
    ADD COLUMN direction TEXT NOT NULL DEFAULT 'following'
        CHECK (direction IN ('following', 'follower'));

ALTER TABLE instagram_graph_handles
    ALTER COLUMN direction DROP DEFAULT,
    ADD CONSTRAINT instagram_graph_handles_import_username_direction_key
        UNIQUE (import_id, username_normalized, direction);

CREATE INDEX instagram_graph_handles_match_idx
    ON instagram_graph_handles (username_normalized, direction, import_id);

ALTER TABLE instagram_graph_imports
    ADD COLUMN follower_count INTEGER NOT NULL DEFAULT 0
        CHECK (follower_count >= 0 AND follower_count <= 10000);

ALTER TABLE instagram_graph_imports
    ALTER COLUMN follower_count DROP DEFAULT;
