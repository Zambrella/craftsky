CREATE TABLE scheduled_posts (
    id                       UUID        NOT NULL,
    owner_did                TEXT        NOT NULL,
    operation_id             UUID        NOT NULL,
    request_hash             BYTEA       NOT NULL,
    status                   TEXT        NOT NULL,
    scheduled_at             TIMESTAMPTZ NOT NULL,
    next_attempt_at          TIMESTAMPTZ NOT NULL,
    attempt_count            INTEGER     NOT NULL DEFAULT 0,
    last_error_code          TEXT,
    payload_bytes            BYTEA       NOT NULL,
    payload_hash             BYTEA       NOT NULL,
    payload_version          BIGINT      NOT NULL DEFAULT 1,
    lease_token              UUID,
    lease_expires_at         TIMESTAMPTZ,
    publication_rkey         TEXT,
    publication_created_at   TIMESTAMPTZ,
    publication_record_bytes BYTEA,
    publication_record_hash  BYTEA,
    needs_attention_at       TIMESTAMPTZ,
    needs_attention_expires_at TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT scheduled_posts_pkey PRIMARY KEY (id),
    CONSTRAINT scheduled_posts_owner_did_fkey
        FOREIGN KEY (owner_did) REFERENCES craftsky_profiles (did) ON DELETE CASCADE,
    CONSTRAINT scheduled_posts_owner_id_key UNIQUE (owner_did, id),
    CONSTRAINT scheduled_posts_owner_operation_key UNIQUE (owner_did, operation_id),
    CONSTRAINT scheduled_posts_status_check
        CHECK (status IN ('scheduled', 'publishing', 'retrying', 'needs_attention')),
    CONSTRAINT scheduled_posts_request_hash_check CHECK (octet_length(request_hash) = 32),
    CONSTRAINT scheduled_posts_payload_check
        CHECK (octet_length(payload_bytes) > 0 AND octet_length(payload_hash) = 32),
    CONSTRAINT scheduled_posts_attempt_count_check CHECK (attempt_count BETWEEN 0 AND 6),
    CONSTRAINT scheduled_posts_payload_version_check CHECK (payload_version > 0),
    CONSTRAINT scheduled_posts_lease_shape_check CHECK (
        (status = 'publishing' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL)
        OR
        (status <> 'publishing' AND lease_token IS NULL AND lease_expires_at IS NULL)
    ),
    CONSTRAINT scheduled_posts_publication_identity_check CHECK (
        (publication_rkey IS NULL AND publication_created_at IS NULL)
        OR
        (publication_rkey IS NOT NULL AND publication_created_at IS NOT NULL)
    ),
    CONSTRAINT scheduled_posts_publication_record_check CHECK (
        (publication_record_bytes IS NULL AND publication_record_hash IS NULL)
        OR
        (
            publication_record_bytes IS NOT NULL
            AND publication_record_hash IS NOT NULL
            AND octet_length(publication_record_bytes) > 0
            AND octet_length(publication_record_hash) = 32
            AND publication_rkey IS NOT NULL
            AND publication_created_at IS NOT NULL
        )
    ),
    CONSTRAINT scheduled_posts_publishing_identity_check CHECK (
        status <> 'publishing'
        OR (publication_rkey IS NOT NULL AND publication_created_at IS NOT NULL)
    ),
    CONSTRAINT scheduled_posts_needs_attention_shape_check CHECK (
        (
            status = 'needs_attention'
            AND needs_attention_at IS NOT NULL
            AND needs_attention_expires_at IS NOT NULL
        )
        OR
        (
            status <> 'needs_attention'
            AND needs_attention_at IS NULL
            AND needs_attention_expires_at IS NULL
        )
    )
);

CREATE INDEX scheduled_posts_due_claim_idx
    ON scheduled_posts (next_attempt_at, id)
    WHERE status IN ('scheduled', 'retrying');

CREATE INDEX scheduled_posts_expired_lease_idx
    ON scheduled_posts (lease_expires_at, id)
    WHERE status = 'publishing';

CREATE INDEX scheduled_posts_owner_scheduled_idx
    ON scheduled_posts (owner_did, scheduled_at, id);

CREATE INDEX scheduled_posts_needs_attention_expiry_idx
    ON scheduled_posts (needs_attention_expires_at, id)
    WHERE status = 'needs_attention';

CREATE UNIQUE INDEX scheduled_posts_owner_publication_rkey_unique
    ON scheduled_posts (owner_did, publication_rkey)
    WHERE publication_rkey IS NOT NULL;

CREATE TABLE scheduled_post_media (
    id                   UUID        NOT NULL,
    owner_did            TEXT        NOT NULL,
    object_key           TEXT        NOT NULL,
    state                TEXT        NOT NULL,
    schedule_id          UUID,
    ordinal              INTEGER,
    mime_type            TEXT        NOT NULL,
    size_bytes           BIGINT      NOT NULL,
    sha256               BYTEA       NOT NULL,
    blob_cid             TEXT,
    unclaimed_expires_at TIMESTAMPTZ NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT scheduled_post_media_pkey PRIMARY KEY (id),
    CONSTRAINT scheduled_post_media_owner_id_key UNIQUE (owner_did, id),
    CONSTRAINT scheduled_post_media_object_key_key UNIQUE (object_key),
    CONSTRAINT scheduled_post_media_owner_did_fkey
        FOREIGN KEY (owner_did) REFERENCES craftsky_profiles (did) ON DELETE CASCADE,
    CONSTRAINT scheduled_post_media_schedule_owner_fkey
        FOREIGN KEY (owner_did, schedule_id)
        REFERENCES scheduled_posts (owner_did, id) ON DELETE CASCADE,
    CONSTRAINT scheduled_post_media_state_check CHECK (state IN ('uploading', 'ready')),
    CONSTRAINT scheduled_post_media_content_check CHECK (
        object_key <> ''
        AND mime_type <> ''
        AND size_bytes > 0
        AND octet_length(sha256) = 32
    ),
    CONSTRAINT scheduled_post_media_attachment_check CHECK (
        (schedule_id IS NULL AND ordinal IS NULL)
        OR
        (schedule_id IS NOT NULL AND ordinal IS NOT NULL AND ordinal BETWEEN 0 AND 3)
    ),
    CONSTRAINT scheduled_post_media_lifecycle_check CHECK (
        (
            state = 'uploading'
            AND schedule_id IS NULL
            AND ordinal IS NULL
            AND blob_cid IS NULL
        )
        OR
        (state = 'ready' AND blob_cid IS NOT NULL AND blob_cid <> '')
    ),
    CONSTRAINT scheduled_post_media_schedule_ordinal_key UNIQUE (schedule_id, ordinal)
);

CREATE INDEX scheduled_post_media_unclaimed_cleanup_idx
    ON scheduled_post_media (unclaimed_expires_at, id)
    WHERE schedule_id IS NULL;

CREATE TABLE scheduled_post_publication_tombstones (
    schedule_id    UUID        NOT NULL,
    owner_did      TEXT        NOT NULL,
    operation_id   UUID        NOT NULL,
    request_hash   BYTEA       NOT NULL,
    publication_uri TEXT       NOT NULL,
    publication_cid TEXT       NOT NULL,
    published_at   TIMESTAMPTZ NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT scheduled_post_publication_tombstones_pkey PRIMARY KEY (schedule_id),
    CONSTRAINT scheduled_post_publication_tombstones_owner_did_fkey
        FOREIGN KEY (owner_did) REFERENCES craftsky_profiles (did) ON DELETE CASCADE,
    CONSTRAINT scheduled_post_publication_tombstones_owner_operation_key
        UNIQUE (owner_did, operation_id),
    CONSTRAINT scheduled_post_publication_tombstones_request_hash_check
        CHECK (octet_length(request_hash) = 32),
    CONSTRAINT scheduled_post_publication_tombstones_identity_check
        CHECK (publication_uri <> '' AND publication_cid <> '' AND expires_at > published_at)
);

CREATE INDEX scheduled_post_publication_tombstones_expiry_idx
    ON scheduled_post_publication_tombstones (expires_at, schedule_id);

CREATE TABLE scheduled_post_cleanup_jobs (
    id                UUID        NOT NULL,
    object_key        TEXT        NOT NULL,
    state             TEXT        NOT NULL DEFAULT 'pending',
    lease_token       UUID,
    lease_expires_at  TIMESTAMPTZ,
    attempt_count     INTEGER     NOT NULL DEFAULT 0,
    next_attempt_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error_code   TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT scheduled_post_cleanup_jobs_pkey PRIMARY KEY (id),
    CONSTRAINT scheduled_post_cleanup_jobs_object_key_key UNIQUE (object_key),
    CONSTRAINT scheduled_post_cleanup_jobs_state_check CHECK (state IN ('pending', 'deleting')),
    CONSTRAINT scheduled_post_cleanup_jobs_attempt_count_check CHECK (attempt_count >= 0),
    CONSTRAINT scheduled_post_cleanup_jobs_lease_shape_check CHECK (
        (state = 'deleting' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL)
        OR
        (state = 'pending' AND lease_token IS NULL AND lease_expires_at IS NULL)
    )
);

CREATE INDEX scheduled_post_cleanup_jobs_pending_idx
    ON scheduled_post_cleanup_jobs (next_attempt_at, id)
    WHERE state = 'pending';

CREATE INDEX scheduled_post_cleanup_jobs_expired_lease_idx
    ON scheduled_post_cleanup_jobs (lease_expires_at, id)
    WHERE state = 'deleting';
