-- Restore bluesky_posts_sample at its original shape.
-- Matches migration 000001_bluesky_posts_sample.up.sql.
CREATE TABLE bluesky_posts_sample (
    uri        TEXT PRIMARY KEY,
    cid        TEXT NOT NULL,
    did        TEXT NOT NULL,
    rkey       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    record     JSONB NOT NULL
);
CREATE INDEX bluesky_posts_sample_did_idx ON bluesky_posts_sample (did);
