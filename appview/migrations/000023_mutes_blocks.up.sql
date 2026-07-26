-- Version 22 tied public PDS projections to current CraftSky membership.
-- Public records remain valid after an account leaves, so remove only the
-- author-membership constraints; subject-post integrity remains unchanged.
ALTER TABLE IF EXISTS craftsky_posts
    DROP CONSTRAINT IF EXISTS craftsky_posts_did_fkey;
ALTER TABLE IF EXISTS craftsky_likes
    DROP CONSTRAINT IF EXISTS craftsky_likes_did_fkey;
ALTER TABLE IF EXISTS craftsky_reposts
    DROP CONSTRAINT IF EXISTS craftsky_reposts_did_fkey;

CREATE TABLE actor_mutes (
    owner_did   TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    subject_did TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (owner_did, subject_did)
);

CREATE INDEX actor_mutes_owner_list_idx
    ON actor_mutes (owner_did, created_at DESC, subject_did);

CREATE TABLE atproto_blocks (
    uri         TEXT        NOT NULL PRIMARY KEY,
    blocker_did TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (blocker_did, rkey)
);

CREATE INDEX atproto_blocks_blocker_subject_idx
    ON atproto_blocks (blocker_did, subject_did);
CREATE INDEX atproto_blocks_subject_blocker_idx
    ON atproto_blocks (subject_did, blocker_did);
CREATE INDEX atproto_blocks_owner_list_idx
    ON atproto_blocks (blocker_did, created_at DESC, subject_did, uri);
