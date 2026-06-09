-- appview/migrations/000016_project_posts.up.sql
-- AppView project post materialization. Historical backfill is intentionally
-- out of scope; new/updated Tap events populate craftsky_project_posts.

ALTER TABLE craftsky_posts
    ADD COLUMN is_project BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN project_craft_type TEXT;

CREATE TABLE craftsky_project_posts (
    uri TEXT PRIMARY KEY REFERENCES craftsky_posts(uri) ON DELETE CASCADE,

    raw_project JSONB NOT NULL,
    common_craft_type TEXT NOT NULL,
    common_status TEXT,
    common_title TEXT,
    common_duration TEXT,
    pattern_url TEXT,
    pattern_name TEXT,
    pattern_difficulty TEXT,
    pattern_designer TEXT,
    pattern_publisher TEXT,
    materials TEXT[] NOT NULL DEFAULT '{}',
    colors TEXT[] NOT NULL DEFAULT '{}',
    design_tags TEXT[] NOT NULL DEFAULT '{}',
    project_tags TEXT[] NOT NULL DEFAULT '{}',
    details_type TEXT,
    raw_details JSONB,

    knitting_project_type TEXT,
    knitting_project_subtype TEXT,
    knitting_yarn_weight TEXT,
    knitting_needle_size_mm TEXT,
    knitting_gauge JSONB,
    knitting_finished_size TEXT,

    crochet_project_type TEXT,
    crochet_project_subtype TEXT,
    crochet_yarn_weight TEXT,
    crochet_hook_size_mm TEXT,
    crochet_gauge JSONB,
    crochet_finished_size TEXT,

    quilting_project_type TEXT,
    quilting_project_subtype TEXT,
    quilting_piecing_technique TEXT,
    quilting_quilting_method TEXT,
    quilting_size TEXT,

    sewing_project_type TEXT,
    sewing_project_subtype TEXT,
    sewing_size_made TEXT,
    sewing_fit_notes TEXT,

    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX craftsky_posts_profile_projects_idx
    ON craftsky_posts (did, indexed_at DESC, uri DESC)
    WHERE is_project = true AND reply_root_uri IS NULL AND reply_parent_uri IS NULL AND quote_uri IS NULL;

CREATE INDEX craftsky_posts_project_craft_type_idx
    ON craftsky_posts (project_craft_type)
    WHERE project_craft_type IS NOT NULL;

CREATE INDEX craftsky_project_posts_common_craft_type_idx
    ON craftsky_project_posts (common_craft_type);
CREATE INDEX craftsky_project_posts_common_status_idx
    ON craftsky_project_posts (common_status)
    WHERE common_status IS NOT NULL;
CREATE INDEX craftsky_project_posts_pattern_difficulty_idx
    ON craftsky_project_posts (pattern_difficulty)
    WHERE pattern_difficulty IS NOT NULL;

CREATE INDEX craftsky_project_posts_materials_gin
    ON craftsky_project_posts USING GIN (materials);
CREATE INDEX craftsky_project_posts_colors_gin
    ON craftsky_project_posts USING GIN (colors);
CREATE INDEX craftsky_project_posts_design_tags_gin
    ON craftsky_project_posts USING GIN (design_tags);
CREATE INDEX craftsky_project_posts_project_tags_gin
    ON craftsky_project_posts USING GIN (project_tags);
