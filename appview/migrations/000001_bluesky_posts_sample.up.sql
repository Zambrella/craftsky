-- SAMPLE TABLE — throwaway. Delete this migration (up + down) and every
-- reference to it when the first social.craftsky.* indexer lands.
-- See docs/superpowers/specs/2026-04-17-tap-integration-design.md.

CREATE TABLE bluesky_posts_sample (
    uri        TEXT PRIMARY KEY,
    cid        TEXT NOT NULL,
    did        TEXT NOT NULL,
    rkey       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    record     JSONB NOT NULL
);
CREATE INDEX bluesky_posts_sample_did_idx ON bluesky_posts_sample (did);
