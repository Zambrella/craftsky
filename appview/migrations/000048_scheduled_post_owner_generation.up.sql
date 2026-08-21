-- Existing schedules predate immutable lifecycle generations, so assigning
-- the current generation would revive stale work after a departure/rejoin.
-- Cancel them instead, preserving every attached private object in the
-- durable cleanup queue before the cascading media delete.
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

ALTER TABLE scheduled_posts
    ADD COLUMN owner_generation BIGINT NOT NULL,
    ADD CONSTRAINT scheduled_posts_owner_generation_check
        CHECK (owner_generation > 0),
    ADD CONSTRAINT scheduled_posts_owner_generation_id_key
        UNIQUE (owner_did, owner_generation, id),
    DROP CONSTRAINT scheduled_posts_owner_operation_key,
    ADD CONSTRAINT scheduled_posts_owner_generation_operation_key
        UNIQUE (owner_did, owner_generation, operation_id);

ALTER TABLE scheduled_post_publication_tombstones
    ADD COLUMN owner_generation BIGINT NOT NULL,
    ADD CONSTRAINT scheduled_post_publication_tombstones_generation_check
        CHECK (owner_generation > 0),
    DROP CONSTRAINT scheduled_post_publication_tombstones_owner_operation_key,
    ADD CONSTRAINT scheduled_post_publication_tombstones_owner_generation_operation_key
        UNIQUE (owner_did, owner_generation, operation_id);

ALTER TABLE scheduled_post_media
    DROP CONSTRAINT scheduled_post_media_schedule_owner_fkey,
    ADD CONSTRAINT scheduled_post_media_schedule_owner_generation_fkey
        FOREIGN KEY (owner_did, owner_generation, schedule_id)
        REFERENCES scheduled_posts (owner_did, owner_generation, id)
        ON DELETE CASCADE;

DROP INDEX scheduled_posts_owner_scheduled_idx;

CREATE INDEX scheduled_posts_owner_generation_scheduled_idx
    ON scheduled_posts (owner_did, owner_generation, scheduled_at, id);

CREATE INDEX scheduled_posts_generation_due_claim_idx
    ON scheduled_posts (next_attempt_at, owner_did, owner_generation, id)
    WHERE status IN ('scheduled', 'retrying');

