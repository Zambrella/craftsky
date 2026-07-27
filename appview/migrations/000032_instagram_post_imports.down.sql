DROP INDEX IF EXISTS craftsky_posts_profile_comments_sort_idx;
DROP INDEX IF EXISTS craftsky_posts_profile_projects_sort_idx;
DROP INDEX IF EXISTS craftsky_posts_profile_posts_sort_idx;

ALTER TABLE craftsky_posts
    DROP COLUMN IF EXISTS profile_sort_at,
    DROP COLUMN IF EXISTS external_import_source;
