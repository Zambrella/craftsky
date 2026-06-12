ALTER TABLE craftsky_project_posts
    ADD COLUMN pattern_name_facets JSONB,
    ADD COLUMN pattern_designer_facets JSONB,
    ADD COLUMN pattern_publisher_facets JSONB;
