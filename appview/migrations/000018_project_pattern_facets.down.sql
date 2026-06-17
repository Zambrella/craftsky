ALTER TABLE craftsky_project_posts
    DROP COLUMN IF EXISTS pattern_publisher_facets,
    DROP COLUMN IF EXISTS pattern_designer_facets,
    DROP COLUMN IF EXISTS pattern_name_facets;
