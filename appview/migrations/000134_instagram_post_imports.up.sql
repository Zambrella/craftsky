-- Private, content-free history of confirmed Instagram post publications.
-- Kept independently of short-lived command replay bodies and public indexing.
CREATE TABLE instagram_post_imports (
    owner_did TEXT NOT NULL,
    source_key TEXT NOT NULL CHECK (source_key ~ '^[0-9a-f]{32}$'),
    selected_rkey TEXT NOT NULL,
    post_uri TEXT,
    operation_key UUID,
    kind TEXT CHECK (kind IN ('standard', 'project')),
    CHECK ((post_uri IS NULL AND operation_key IS NULL AND kind IS NULL)
        OR (post_uri IS NOT NULL AND operation_key IS NOT NULL AND kind IS NOT NULL)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_did, source_key)
);
