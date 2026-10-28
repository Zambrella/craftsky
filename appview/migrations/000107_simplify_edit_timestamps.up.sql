-- Current public declaration is the entire edit disclosure contract (ADR 022).
ALTER TABLE craftsky_posts DROP CONSTRAINT craftsky_posts_edit_time_pair;
ALTER TABLE craftsky_posts DROP COLUMN edit_time_source;
UPDATE craftsky_posts SET edited_at = (record->>'editedAt')::timestamptz;
DROP TABLE craftsky_post_edit_state;
