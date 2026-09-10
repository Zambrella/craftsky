DROP INDEX IF EXISTS craftsky_project_posts_self_drafted_true_idx;

ALTER TABLE craftsky_project_posts
    DROP COLUMN IF EXISTS pattern_self_drafted;
