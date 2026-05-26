-- appview/migrations/000012_atproto_follows.up.sql
CREATE TABLE atproto_follows (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (did, rkey),
    UNIQUE (did, subject_did)
);

CREATE INDEX atproto_follows_did_idx
    ON atproto_follows (did);
CREATE INDEX atproto_follows_subject_did_idx
    ON atproto_follows (subject_did);
CREATE INDEX atproto_follows_did_subject_did_idx
    ON atproto_follows (did, subject_did);
