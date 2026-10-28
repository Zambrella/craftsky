-- Restore the previous storage shape; historical inferred times are unrecoverable.
ALTER TABLE craftsky_posts ADD COLUMN edit_time_source TEXT CHECK (edit_time_source IN ('confirmed','reported','observed'));
UPDATE craftsky_posts SET edit_time_source='reported' WHERE edited_at IS NOT NULL;
ALTER TABLE craftsky_posts ADD CONSTRAINT craftsky_posts_edit_time_pair CHECK ((edited_at IS NULL) = (edit_time_source IS NULL));
CREATE TABLE craftsky_post_edit_state (
 uri TEXT PRIMARY KEY REFERENCES tap_source_records(uri) ON DELETE CASCADE,
 source_cid TEXT NOT NULL, source_revision TEXT COLLATE "C" NOT NULL,
 content_digest BYTEA NOT NULL CHECK (octet_length(content_digest)=32),
 edited_at TIMESTAMPTZ, edit_time_source TEXT CHECK (edit_time_source IN ('confirmed','reported','observed')),
 CONSTRAINT craftsky_post_edit_time_pair CHECK ((edited_at IS NULL) = (edit_time_source IS NULL))
);
