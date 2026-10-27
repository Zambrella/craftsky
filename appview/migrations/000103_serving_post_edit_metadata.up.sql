-- Keep metadata with the version that passed serving/media visibility checks.
ALTER TABLE craftsky_posts
    ADD COLUMN edited_at TIMESTAMPTZ,
    ADD COLUMN edit_time_source TEXT CHECK (edit_time_source IN ('confirmed', 'reported', 'observed')),
    ADD COLUMN record_revision TEXT COLLATE "C",
    ADD CONSTRAINT craftsky_posts_edit_time_pair CHECK (
        (edited_at IS NULL AND edit_time_source IS NULL)
        OR (edited_at IS NOT NULL AND edit_time_source IS NOT NULL)
    );
