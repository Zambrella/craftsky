CREATE TABLE test_posts (
    uri         TEXT PRIMARY KEY,
    cid         TEXT NOT NULL,
    did         TEXT NOT NULL,
    text        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX test_posts_created_at_idx ON test_posts (created_at DESC);
