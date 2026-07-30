DROP TABLE IF EXISTS account_language_preferences;
DROP INDEX IF EXISTS craftsky_posts_langs_gin;

ALTER TABLE craftsky_posts
    DROP COLUMN IF EXISTS langs;
