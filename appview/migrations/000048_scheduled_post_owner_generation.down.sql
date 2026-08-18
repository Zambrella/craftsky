-- A down migration cannot honestly collapse generation-scoped work into the
-- historical generation-free contract. Cancel it with the same durable
-- object cleanup handoff used by the up migration.
INSERT INTO scheduled_post_cleanup_jobs (
    id, object_key, owner_did, owner_generation, upload_generation,
    source_attempt_id, outcome_uncertain, settlement_not_before,
    next_attempt_at, created_at, updated_at
)
SELECT gen_random_uuid(), attempts.object_key, attempts.owner_did,
       attempts.owner_generation, attempts.upload_generation,
       attempts.upload_attempt_id,
       attempts.remote_outcome = 'dispatched',
       attempts.settlement_not_before,
       now(), now(), now()
FROM scheduled_post_media AS media
JOIN scheduled_post_object_attempts AS attempts
  ON attempts.upload_attempt_id = media.upload_attempt_id
WHERE media.schedule_id IS NOT NULL
ON CONFLICT (object_key) DO NOTHING;

DELETE FROM scheduled_posts;
DELETE FROM scheduled_post_publication_tombstones;

ALTER TABLE scheduled_post_media
    DROP CONSTRAINT scheduled_post_media_schedule_owner_generation_fkey,
    ADD CONSTRAINT scheduled_post_media_schedule_owner_fkey
        FOREIGN KEY (owner_did, schedule_id)
        REFERENCES scheduled_posts (owner_did, id)
        ON DELETE CASCADE;

DROP INDEX scheduled_posts_generation_due_claim_idx;
DROP INDEX scheduled_posts_owner_generation_scheduled_idx;

CREATE INDEX scheduled_posts_owner_scheduled_idx
    ON scheduled_posts (owner_did, scheduled_at, id);

ALTER TABLE scheduled_post_publication_tombstones
    DROP CONSTRAINT scheduled_post_publication_tombstones_owner_generation_operation_key,
    DROP CONSTRAINT scheduled_post_publication_tombstones_generation_check,
    DROP COLUMN owner_generation,
    ADD CONSTRAINT scheduled_post_publication_tombstones_owner_operation_key
        UNIQUE (owner_did, operation_id);

ALTER TABLE scheduled_posts
    DROP CONSTRAINT scheduled_posts_owner_generation_operation_key,
    DROP CONSTRAINT scheduled_posts_owner_generation_id_key,
    DROP CONSTRAINT scheduled_posts_owner_generation_check,
    DROP COLUMN owner_generation,
    ADD CONSTRAINT scheduled_posts_owner_operation_key
        UNIQUE (owner_did, operation_id);
