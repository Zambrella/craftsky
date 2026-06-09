-- appview/migrations/000016_project_posts.down.sql

DROP INDEX IF EXISTS craftsky_project_posts_project_tags_gin;
DROP INDEX IF EXISTS craftsky_project_posts_design_tags_gin;
DROP INDEX IF EXISTS craftsky_project_posts_colors_gin;
DROP INDEX IF EXISTS craftsky_project_posts_materials_gin;
DROP INDEX IF EXISTS craftsky_project_posts_pattern_difficulty_idx;
DROP INDEX IF EXISTS craftsky_project_posts_common_status_idx;
DROP INDEX IF EXISTS craftsky_project_posts_common_craft_type_idx;
DROP INDEX IF EXISTS craftsky_posts_project_craft_type_idx;
DROP INDEX IF EXISTS craftsky_posts_profile_projects_idx;

DROP TABLE IF EXISTS craftsky_project_posts;

ALTER TABLE craftsky_posts
    DROP COLUMN IF EXISTS project_craft_type,
    DROP COLUMN IF EXISTS is_project;
