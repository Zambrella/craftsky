DROP TABLE IF EXISTS atproto_blocks;
DROP TABLE IF EXISTS actor_mutes;

-- Retained public rows may already outlive their former membership. Restore
-- the version-22 constraints as NOT VALID so rollback succeeds without
-- deleting those records; PostgreSQL still enforces them for subsequent writes.
DO $$
BEGIN
    IF to_regclass(current_schema() || '.craftsky_posts') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM pg_constraint
           WHERE conrelid = to_regclass(current_schema() || '.craftsky_posts')
             AND conname = 'craftsky_posts_did_fkey'
       ) THEN
        ALTER TABLE craftsky_posts
            ADD CONSTRAINT craftsky_posts_did_fkey
            FOREIGN KEY (did) REFERENCES craftsky_profiles(did)
            ON DELETE CASCADE NOT VALID;
    END IF;

    IF to_regclass(current_schema() || '.craftsky_likes') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM pg_constraint
           WHERE conrelid = to_regclass(current_schema() || '.craftsky_likes')
             AND conname = 'craftsky_likes_did_fkey'
       ) THEN
        ALTER TABLE craftsky_likes
            ADD CONSTRAINT craftsky_likes_did_fkey
            FOREIGN KEY (did) REFERENCES craftsky_profiles(did)
            ON DELETE CASCADE NOT VALID;
    END IF;

    IF to_regclass(current_schema() || '.craftsky_reposts') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM pg_constraint
           WHERE conrelid = to_regclass(current_schema() || '.craftsky_reposts')
             AND conname = 'craftsky_reposts_did_fkey'
       ) THEN
        ALTER TABLE craftsky_reposts
            ADD CONSTRAINT craftsky_reposts_did_fkey
            FOREIGN KEY (did) REFERENCES craftsky_profiles(did)
            ON DELETE CASCADE NOT VALID;
    END IF;
END
$$;
