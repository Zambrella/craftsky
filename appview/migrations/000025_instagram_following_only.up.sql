-- Existing development databases may contain the earlier directional import
-- schema. Follower rows are intentionally discarded because CraftSky uses
-- only accounts the member chose to follow for discovery. Fresh databases
-- already receive the compact schema from 000023, so this migration is
-- deliberately conditional.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'instagram_graph_handles'
          AND column_name = 'direction'
    ) THEN
        EXECUTE 'DELETE FROM instagram_graph_handles WHERE direction = ''follower''';
        DROP INDEX IF EXISTS instagram_graph_handles_match_idx;
        ALTER TABLE instagram_graph_handles DROP COLUMN direction;
        ALTER TABLE instagram_graph_handles
            ADD CONSTRAINT instagram_graph_handles_import_username_key
            UNIQUE (import_id, username_normalized);
        CREATE INDEX instagram_graph_handles_match_idx
            ON instagram_graph_handles (username_normalized, import_id);
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'instagram_graph_imports'
          AND column_name = 'follower_count'
    ) THEN
        ALTER TABLE instagram_graph_imports DROP COLUMN follower_count;
    END IF;
END
$$;
