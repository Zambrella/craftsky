-- DELETE ME: part of the disposable test pipeline. Drop this table
-- (with a follow-up drop migration) when the real social.craftsky.feed.post
-- indexer lands. See appview/internal/testpipeline/ and
-- docs/superpowers/specs/2026-04-19-test-pipeline-design.md.
CREATE TABLE test_posts (
    uri         TEXT PRIMARY KEY,
    cid         TEXT NOT NULL,
    did         TEXT NOT NULL,
    text        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX test_posts_created_at_idx ON test_posts (created_at DESC);
