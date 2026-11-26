DELETE FROM post_publication_mentions WHERE edit_activity_at IS NOT NULL;
ALTER TABLE post_publication_mentions DROP COLUMN edit_activity_at;
