-- Reconstruct the migration-74 schema for a rollback through migrations 73,
-- 60, 46, 43, 23, 13, 12, and 11. Historical physical rows are not restored.
CREATE TABLE craftsky_likes (
    uri TEXT NOT NULL PRIMARY KEY, did TEXT NOT NULL, rkey TEXT NOT NULL,
    cid TEXT NOT NULL, subject_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT NOT NULL, record JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ,
    UNIQUE (did, rkey)
);
CREATE UNIQUE INDEX craftsky_likes_did_subject_uri_active_unique
    ON craftsky_likes(did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_likes_active_subject_uri
    ON craftsky_likes(subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_likes_indexed_at_desc ON craftsky_likes(indexed_at DESC);
CREATE INDEX craftsky_likes_subject_uri_idx ON craftsky_likes(subject_uri);
CREATE INDEX terminal_purge_craftsky_likes_actor_idx ON craftsky_likes(did, uri);

CREATE TABLE craftsky_reposts (
    uri TEXT NOT NULL PRIMARY KEY, did TEXT NOT NULL, rkey TEXT NOT NULL,
    cid TEXT NOT NULL, subject_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT NOT NULL, record JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ,
    UNIQUE (did, rkey)
);
CREATE UNIQUE INDEX craftsky_reposts_did_subject_uri_active_unique
    ON craftsky_reposts(did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_active_subject_uri
    ON craftsky_reposts(subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_indexed_at_desc ON craftsky_reposts(indexed_at DESC);
CREATE INDEX craftsky_reposts_subject_uri_idx ON craftsky_reposts(subject_uri);
CREATE INDEX terminal_purge_craftsky_reposts_actor_idx ON craftsky_reposts(did, uri);

CREATE TABLE atproto_follows (
    uri TEXT NOT NULL PRIMARY KEY, did TEXT NOT NULL, rkey TEXT NOT NULL,
    cid TEXT NOT NULL, subject_did TEXT NOT NULL, record JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL, indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey), UNIQUE (did, subject_did)
);
CREATE INDEX atproto_follows_did_idx ON atproto_follows(did);
CREATE INDEX atproto_follows_subject_did_idx ON atproto_follows(subject_did);
CREATE INDEX atproto_follows_subject_created_uri_desc_idx
    ON atproto_follows(subject_did, created_at DESC, uri DESC);
CREATE INDEX atproto_follows_did_created_uri_desc_idx
    ON atproto_follows(did, created_at DESC, uri DESC);
CREATE INDEX terminal_purge_atproto_follows_actor_idx ON atproto_follows(did, uri);
CREATE INDEX terminal_purge_atproto_follows_subject_idx ON atproto_follows(subject_did, uri);

CREATE TABLE atproto_blocks (
    uri TEXT NOT NULL PRIMARY KEY, blocker_did TEXT NOT NULL, rkey TEXT NOT NULL,
    cid TEXT NOT NULL, subject_did TEXT NOT NULL, record JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL, indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (blocker_did, rkey)
);
CREATE INDEX atproto_blocks_blocker_subject_idx ON atproto_blocks(blocker_did, subject_did);
CREATE INDEX atproto_blocks_subject_blocker_idx ON atproto_blocks(subject_did, blocker_did);
CREATE INDEX atproto_blocks_owner_list_idx
    ON atproto_blocks(blocker_did, created_at DESC, subject_did, uri);
CREATE INDEX terminal_purge_atproto_blocks_actor_idx ON atproto_blocks(blocker_did, uri);
CREATE INDEX terminal_purge_atproto_blocks_subject_idx ON atproto_blocks(subject_did, uri);
