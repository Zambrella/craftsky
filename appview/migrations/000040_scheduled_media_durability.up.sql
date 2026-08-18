-- Pre-production breaking reset: old scheduled rows do not carry a lifecycle
-- generation or a durable remote-attempt identity and cannot be backfilled
-- honestly after an outcome-uncertain object write.
TRUNCATE TABLE
    scheduled_post_cleanup_jobs,
    scheduled_post_publication_tombstones,
    scheduled_post_media,
    scheduled_posts;

ALTER TABLE scheduled_posts
    DROP CONSTRAINT scheduled_posts_owner_did_fkey,
    ADD CONSTRAINT scheduled_posts_owner_lifecycle_fkey
        FOREIGN KEY (owner_did) REFERENCES owner_lifecycles(owner_did)
        ON DELETE RESTRICT;

ALTER TABLE scheduled_post_publication_tombstones
    DROP CONSTRAINT scheduled_post_publication_tombstones_owner_did_fkey,
    ADD CONSTRAINT scheduled_post_publication_tombstones_owner_lifecycle_fkey
        FOREIGN KEY (owner_did) REFERENCES owner_lifecycles(owner_did)
        ON DELETE RESTRICT;

ALTER TABLE scheduled_post_media
    DROP CONSTRAINT scheduled_post_media_owner_did_fkey;

CREATE TABLE scheduled_post_object_attempts (
    upload_attempt_id    UUID        NOT NULL PRIMARY KEY,
    media_id             UUID        NOT NULL,
    owner_did            TEXT        NOT NULL,
    owner_generation     BIGINT      NOT NULL CHECK (owner_generation > 0),
    upload_generation    BIGINT      NOT NULL CHECK (upload_generation > 0),
    object_key           TEXT        NOT NULL UNIQUE,
    request_fingerprint  BYTEA       NOT NULL,
    remote_outcome       TEXT        NOT NULL DEFAULT 'prepared'
        CHECK (remote_outcome IN ('prepared', 'dispatched', 'accepted')),
    remote_started_at    TIMESTAMPTZ NOT NULL,
    remote_deadline      TIMESTAMPTZ NOT NULL,
    settlement_not_before TIMESTAMPTZ,
    dispatched_at        TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    FOREIGN KEY (owner_did) REFERENCES owner_lifecycles(owner_did)
        ON DELETE RESTRICT,
    CONSTRAINT scheduled_post_object_attempts_identity_key
        UNIQUE (
            upload_attempt_id, owner_did, owner_generation,
            upload_generation, object_key
        ),
    CONSTRAINT scheduled_post_object_attempts_media_generation_key
        UNIQUE (owner_did, owner_generation, media_id),
    CONSTRAINT scheduled_post_object_attempts_generation_check
        CHECK (upload_generation = owner_generation),
    CONSTRAINT scheduled_post_object_attempts_object_key_check CHECK (
        object_key ~ '^scheduled-media/v2/[1-9][0-9]*/[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        AND split_part(object_key, '/', 3) = upload_generation::text
        AND split_part(object_key, '/', 5) = ''
    ),
    CONSTRAINT scheduled_post_object_attempts_fingerprint_check
        CHECK (octet_length(request_fingerprint) = 32),
    CONSTRAINT scheduled_post_object_attempts_deadline_check CHECK (
        remote_deadline > remote_started_at
        AND (
            settlement_not_before IS NULL
            OR settlement_not_before > remote_deadline
        )
    ),
    CONSTRAINT scheduled_post_object_attempts_outcome_check CHECK (
        (
            remote_outcome = 'prepared'
            AND dispatched_at IS NULL
            AND completed_at IS NULL
        )
        OR (
            remote_outcome = 'dispatched'
            AND dispatched_at IS NOT NULL
            AND completed_at IS NULL
        )
        OR (
            remote_outcome = 'accepted'
            AND dispatched_at IS NOT NULL
            AND completed_at IS NOT NULL
        )
    ),
    CONSTRAINT scheduled_post_object_attempts_timestamp_order_check CHECK (
        updated_at >= created_at
        AND remote_started_at >= created_at
        AND (dispatched_at IS NULL OR dispatched_at >= remote_started_at)
        AND (completed_at IS NULL OR completed_at >= dispatched_at)
    )
);

CREATE INDEX scheduled_post_object_attempts_unresolved_idx
    ON scheduled_post_object_attempts (
        owner_did, owner_generation, remote_deadline, upload_attempt_id
    )
    WHERE remote_outcome <> 'accepted';

ALTER TABLE scheduled_post_media
    ADD COLUMN owner_generation BIGINT NOT NULL,
    ADD COLUMN upload_generation BIGINT NOT NULL,
    ADD COLUMN upload_attempt_id UUID NOT NULL,
    ADD CONSTRAINT scheduled_post_media_owner_generation_check
        CHECK (owner_generation > 0 AND upload_generation = owner_generation),
    ADD CONSTRAINT scheduled_post_media_owner_lifecycle_fkey
        FOREIGN KEY (owner_did) REFERENCES owner_lifecycles(owner_did)
        ON DELETE RESTRICT,
    ADD CONSTRAINT scheduled_post_media_object_attempt_fkey
        FOREIGN KEY (
            upload_attempt_id, owner_did, owner_generation,
            upload_generation, object_key
        ) REFERENCES scheduled_post_object_attempts (
            upload_attempt_id, owner_did, owner_generation,
            upload_generation, object_key
        ) ON DELETE RESTRICT;

ALTER TABLE scheduled_post_cleanup_jobs
    ADD COLUMN owner_did TEXT NOT NULL,
    ADD COLUMN owner_generation BIGINT NOT NULL,
    ADD COLUMN upload_generation BIGINT NOT NULL,
    ADD COLUMN source_attempt_id UUID NOT NULL,
    ADD COLUMN outcome_uncertain BOOLEAN NOT NULL,
    ADD COLUMN settlement_not_before TIMESTAMPTZ,
    ADD COLUMN last_absence_at TIMESTAMPTZ,
    ADD CONSTRAINT scheduled_post_cleanup_jobs_owner_generation_check
        CHECK (
            owner_generation > 0
            AND upload_generation = owner_generation
        ),
    ADD CONSTRAINT scheduled_post_cleanup_jobs_attempt_fkey
        FOREIGN KEY (
            source_attempt_id, owner_did, owner_generation,
            upload_generation, object_key
        ) REFERENCES scheduled_post_object_attempts (
            upload_attempt_id, owner_did, owner_generation,
            upload_generation, object_key
        ) ON DELETE RESTRICT,
    ADD CONSTRAINT scheduled_post_cleanup_jobs_settlement_check CHECK (
        settlement_not_before IS NULL OR outcome_uncertain
    ),
    ADD CONSTRAINT scheduled_post_cleanup_jobs_absence_check CHECK (
        last_absence_at IS NULL OR last_absence_at >= created_at
    );

DROP INDEX scheduled_post_cleanup_jobs_pending_idx;

CREATE INDEX scheduled_post_cleanup_jobs_claim_idx
    ON scheduled_post_cleanup_jobs (
        next_attempt_at, owner_did, owner_generation, id
    )
    WHERE state = 'pending';
