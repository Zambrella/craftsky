ALTER TABLE moderation_reports
    DROP CONSTRAINT moderation_reports_subject_type_check,
    DROP CONSTRAINT moderation_reports_check;

ALTER TABLE moderation_reports
    ADD CONSTRAINT moderation_reports_subject_type_check
        CHECK (subject_type IN ('post', 'account', 'event')),
    ADD CONSTRAINT moderation_reports_subject_shape_check CHECK (
        (subject_type = 'post'
            AND subject_collection IS NOT NULL
            AND subject_rkey IS NOT NULL
            AND subject_uri IS NOT NULL)
        OR
        (subject_type = 'event'
            AND subject_collection = 'social.craftsky.business.event'
            AND subject_rkey IS NOT NULL
            AND subject_uri IS NOT NULL
            AND subject_cid_snapshot IS NOT NULL)
        OR
        (subject_type = 'account'
            AND subject_collection IS NULL
            AND subject_rkey IS NULL
            AND subject_uri IS NULL
            AND subject_cid_snapshot IS NULL)
    );

ALTER TABLE moderation_outputs
    DROP CONSTRAINT moderation_outputs_subject_type_check,
    DROP CONSTRAINT moderation_outputs_check;

ALTER TABLE moderation_outputs
    ADD CONSTRAINT moderation_outputs_subject_type_check
        CHECK (subject_type IN ('post', 'account', 'event')),
    ADD CONSTRAINT moderation_outputs_subject_shape_check CHECK (
        (subject_type = 'post'
            AND subject_collection IS NOT NULL
            AND subject_rkey IS NOT NULL
            AND subject_uri IS NOT NULL)
        OR
        (subject_type = 'event'
            AND subject_collection = 'social.craftsky.business.event'
            AND subject_rkey IS NOT NULL
            AND subject_uri IS NOT NULL)
        OR
        (subject_type = 'account'
            AND subject_collection IS NULL
            AND subject_rkey IS NULL
            AND subject_uri IS NULL)
    );
