-- appview/migrations/000011_craftsky_interactions.up.sql
-- See .sisyphus/plans/backend-likes-reposts-replies.md.
CREATE TABLE craftsky_likes (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,

    subject_uri TEXT        NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT        NOT NULL,

    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,

    UNIQUE (did, rkey)
);

CREATE UNIQUE INDEX craftsky_likes_did_subject_uri_active_unique
    ON craftsky_likes (did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_likes_active_subject_uri
    ON craftsky_likes (subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_likes_active_did_subject_uri
    ON craftsky_likes (did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_likes_indexed_at_desc
    ON craftsky_likes (indexed_at DESC);

CREATE TABLE craftsky_reposts (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,

    subject_uri TEXT        NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT        NOT NULL,

    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,

    UNIQUE (did, rkey)
);

CREATE UNIQUE INDEX craftsky_reposts_did_subject_uri_active_unique
    ON craftsky_reposts (did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_active_subject_uri
    ON craftsky_reposts (subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_active_did_subject_uri
    ON craftsky_reposts (did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_indexed_at_desc
    ON craftsky_reposts (indexed_at DESC);
