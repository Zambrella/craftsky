DROP TABLE IF EXISTS image_scan_events;
DROP TABLE IF EXISTS profile_image_candidates;
DROP TABLE IF EXISTS image_subject_requirements;
DROP FUNCTION IF EXISTS appview_image_subject_is_clear(TEXT, TEXT);
DROP TABLE IF EXISTS image_subject_states;
DROP TABLE IF EXISTS image_blob_sources;
DROP TABLE IF EXISTS image_scan_jobs;
DROP TABLE IF EXISTS image_scan_results;

UPDATE tap_projection_jobs
SET state = 'pending',
    dependency_kind = NULL,
    dependency_key = NULL
WHERE state = 'blocked'
  AND dependency_kind = 'image_subject_uri';

ALTER TABLE tap_projection_jobs
    DROP CONSTRAINT tap_projection_jobs_dependency_check,
    ADD CONSTRAINT tap_projection_jobs_dependency_check CHECK (
        (state = 'blocked'
            AND dependency_kind IS NOT NULL
            AND dependency_kind IN ('member_did', 'subject_uri', 'repository_did')
            AND dependency_key IS NOT NULL
            AND btrim(dependency_key) <> '' AND char_length(dependency_key) <= 2048)
        OR
        (state <> 'blocked' AND dependency_kind IS NULL AND dependency_key IS NULL)
    );
