-- Mentions materialized from post text facets and faceted project metadata.
CREATE TABLE craftsky_post_mentions (
    post_uri      TEXT        NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    mentioned_did TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL,
    indexed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (post_uri, mentioned_did)
);

CREATE INDEX craftsky_post_mentions_mentioned_idx
    ON craftsky_post_mentions (mentioned_did, indexed_at DESC, post_uri DESC);
