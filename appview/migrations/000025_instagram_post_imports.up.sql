-- Explicit historical-import provenance and author-profile chronology.
--
-- Existing rows remain ordinary and keep their current indexed-time profile
-- order. Exact Instagram imports are assigned their record-created time by
-- the post indexer. The public record remains the authoritative source of the
-- nullable classification.
ALTER TABLE craftsky_posts
    ADD COLUMN external_import_source TEXT
        CONSTRAINT craftsky_posts_external_import_source_check
        CHECK (external_import_source IS NULL OR external_import_source = 'instagram'),
    ADD COLUMN profile_sort_at TIMESTAMPTZ;

UPDATE craftsky_posts
SET profile_sort_at = indexed_at;

ALTER TABLE craftsky_posts
    ALTER COLUMN profile_sort_at SET NOT NULL,
    ALTER COLUMN profile_sort_at SET DEFAULT now();

CREATE INDEX craftsky_posts_profile_posts_sort_idx
    ON craftsky_posts (did, profile_sort_at DESC, uri DESC)
    WHERE is_project = false
      AND reply_root_uri IS NULL
      AND reply_parent_uri IS NULL;

CREATE INDEX craftsky_posts_profile_projects_sort_idx
    ON craftsky_posts (did, profile_sort_at DESC, uri DESC)
    WHERE is_project = true
      AND reply_root_uri IS NULL
      AND reply_parent_uri IS NULL
      AND quote_uri IS NULL;

CREATE INDEX craftsky_posts_profile_comments_sort_idx
    ON craftsky_posts (did, profile_sort_at DESC, uri DESC)
    WHERE reply_root_uri IS NOT NULL
      AND reply_parent_uri IS NOT NULL;
