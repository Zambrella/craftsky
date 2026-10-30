ALTER TABLE craftsky_posts
    DROP CONSTRAINT craftsky_posts_edit_time_pair,
    DROP COLUMN record_revision,
    DROP COLUMN edit_time_source,
    DROP COLUMN edited_at;
