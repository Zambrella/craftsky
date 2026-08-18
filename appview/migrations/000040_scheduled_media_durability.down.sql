-- This migration deliberately reset legacy scheduled state on the way up;
-- down is equally destructive because the durable attempt identity cannot be
-- represented by the historical schema.
TRUNCATE TABLE
    scheduled_post_cleanup_jobs,
    scheduled_post_publication_tombstones,
    scheduled_post_media,
    scheduled_posts;

DROP INDEX IF EXISTS scheduled_post_cleanup_jobs_claim_idx;

CREATE INDEX scheduled_post_cleanup_jobs_pending_idx
    ON scheduled_post_cleanup_jobs (next_attempt_at, id)
    WHERE state = 'pending';

ALTER TABLE scheduled_post_cleanup_jobs
    DROP CONSTRAINT IF EXISTS scheduled_post_cleanup_jobs_absence_check,
    DROP CONSTRAINT IF EXISTS scheduled_post_cleanup_jobs_settlement_check,
    DROP CONSTRAINT IF EXISTS scheduled_post_cleanup_jobs_attempt_fkey,
    DROP CONSTRAINT IF EXISTS scheduled_post_cleanup_jobs_owner_generation_check,
    DROP COLUMN IF EXISTS last_absence_at,
    DROP COLUMN IF EXISTS settlement_not_before,
    DROP COLUMN IF EXISTS outcome_uncertain,
    DROP COLUMN IF EXISTS source_attempt_id,
    DROP COLUMN IF EXISTS upload_generation,
    DROP COLUMN IF EXISTS owner_generation,
    DROP COLUMN IF EXISTS owner_did;

ALTER TABLE scheduled_post_media
    DROP CONSTRAINT IF EXISTS scheduled_post_media_object_attempt_fkey,
    DROP CONSTRAINT IF EXISTS scheduled_post_media_owner_lifecycle_fkey,
    DROP CONSTRAINT IF EXISTS scheduled_post_media_owner_generation_check,
    DROP COLUMN IF EXISTS upload_attempt_id,
    DROP COLUMN IF EXISTS upload_generation,
    DROP COLUMN IF EXISTS owner_generation,
    ADD CONSTRAINT scheduled_post_media_owner_did_fkey
        FOREIGN KEY (owner_did) REFERENCES craftsky_profiles(did)
        ON DELETE CASCADE;

DROP TABLE IF EXISTS scheduled_post_object_attempts;

ALTER TABLE scheduled_post_publication_tombstones
    DROP CONSTRAINT IF EXISTS scheduled_post_publication_tombstones_owner_lifecycle_fkey,
    ADD CONSTRAINT scheduled_post_publication_tombstones_owner_did_fkey
        FOREIGN KEY (owner_did) REFERENCES craftsky_profiles(did)
        ON DELETE CASCADE;

ALTER TABLE scheduled_posts
    DROP CONSTRAINT IF EXISTS scheduled_posts_owner_lifecycle_fkey,
    ADD CONSTRAINT scheduled_posts_owner_did_fkey
        FOREIGN KEY (owner_did) REFERENCES craftsky_profiles(did)
        ON DELETE CASCADE;
