-- ADR 022: one current edit state per source URI; no previous bodies or versions.
CREATE TABLE craftsky_post_edit_state (
    uri TEXT PRIMARY KEY REFERENCES tap_source_records(uri) ON DELETE CASCADE,
    source_cid TEXT NOT NULL,
    source_revision TEXT COLLATE "C" NOT NULL,
    content_digest BYTEA NOT NULL CHECK (octet_length(content_digest) = 32),
    edited_at TIMESTAMPTZ,
    edit_time_source TEXT CHECK (edit_time_source IN ('confirmed', 'reported', 'observed')),
    CONSTRAINT craftsky_post_edit_time_pair CHECK (
        (edited_at IS NULL AND edit_time_source IS NULL)
        OR (edited_at IS NOT NULL AND edit_time_source IS NOT NULL)
    )
);
