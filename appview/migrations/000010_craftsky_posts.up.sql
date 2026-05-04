-- appview/migrations/000010_craftsky_posts.up.sql
-- See docs/superpowers/specs/2026-05-04-feed-post-indexing-design.md.
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey             TEXT        NOT NULL,
    cid              TEXT        NOT NULL,

    text             TEXT        NOT NULL,
    facets           JSONB,
    images           JSONB,

    reply_root_uri   TEXT,
    reply_root_cid   TEXT,
    reply_parent_uri TEXT,
    reply_parent_cid TEXT,

    quote_uri        TEXT,
    quote_cid        TEXT,

    tags             TEXT[]      NOT NULL DEFAULT '{}',

    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (did, rkey)
);

CREATE INDEX craftsky_posts_indexed_at_desc
    ON craftsky_posts (indexed_at DESC);
CREATE INDEX craftsky_posts_did_indexed_at_desc
    ON craftsky_posts (did, indexed_at DESC);
CREATE INDEX craftsky_posts_reply_parent_uri
    ON craftsky_posts (reply_parent_uri) WHERE reply_parent_uri IS NOT NULL;
CREATE INDEX craftsky_posts_reply_root_uri
    ON craftsky_posts (reply_root_uri)   WHERE reply_root_uri   IS NOT NULL;
CREATE INDEX craftsky_posts_quote_uri
    ON craftsky_posts (quote_uri)        WHERE quote_uri        IS NOT NULL;
CREATE INDEX craftsky_posts_tags_gin
    ON craftsky_posts USING GIN (tags);
