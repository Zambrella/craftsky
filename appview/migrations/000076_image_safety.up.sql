-- Private, provider-neutral image-safety state. No image bytes or raw scanner
-- payloads are stored in these tables.

ALTER TABLE tap_projection_jobs
    DROP CONSTRAINT tap_projection_jobs_dependency_check,
    ADD CONSTRAINT tap_projection_jobs_dependency_check CHECK (
        (state = 'blocked'
            AND dependency_kind IS NOT NULL
            AND dependency_kind IN (
                'member_did', 'subject_uri', 'repository_did', 'image_subject_uri'
            )
            AND dependency_key IS NOT NULL
            AND btrim(dependency_key) <> '' AND char_length(dependency_key) <= 2048)
        OR
        (state <> 'blocked' AND dependency_kind IS NULL AND dependency_key IS NULL)
    );

CREATE TABLE image_scan_results (
    id                            UUID        NOT NULL PRIMARY KEY,
    blob_cid                      TEXT        NOT NULL,
    scanner_id                    TEXT        NOT NULL,
    policy_version                TEXT        NOT NULL,
    corpus_version                TEXT        NOT NULL,
    state                         TEXT        NOT NULL,
    provider_reference            TEXT,
    integrity_metadata_reference  TEXT,
    completed_at                  TIMESTAMPTZ,
    current_version               BIGINT      NOT NULL DEFAULT 1,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT image_scan_results_state_check
        CHECK (state IN ('pending', 'clear', 'match', 'unavailable', 'error')),
    CONSTRAINT image_scan_results_key_check CHECK (
        btrim(blob_cid) <> ''
        AND btrim(scanner_id) <> ''
        AND btrim(policy_version) <> ''
        AND btrim(corpus_version) <> ''
    ),
    CONSTRAINT image_scan_results_reference_check CHECK (
        provider_reference IS NULL OR (
            btrim(provider_reference) <> '' AND char_length(provider_reference) <= 512
        )
    ),
    CONSTRAINT image_scan_results_integrity_reference_check CHECK (
        integrity_metadata_reference IS NULL OR (
            btrim(integrity_metadata_reference) <> ''
            AND char_length(integrity_metadata_reference) <= 512
        )
    ),
    CONSTRAINT image_scan_results_version_check CHECK (current_version > 0),
    CONSTRAINT image_scan_results_timestamp_check CHECK (
        updated_at >= created_at
        AND (completed_at IS NULL OR completed_at >= created_at)
    )
);

CREATE UNIQUE INDEX image_scan_results_cache_key_idx
    ON image_scan_results(blob_cid, scanner_id, policy_version, corpus_version);

CREATE TABLE image_scan_jobs (
    id                   UUID        NOT NULL PRIMARY KEY,
    scan_result_id       UUID        NOT NULL UNIQUE
        REFERENCES image_scan_results(id) ON DELETE CASCADE,
    state                TEXT        NOT NULL DEFAULT 'queued',
    scan_version         BIGINT      NOT NULL DEFAULT 1,
    attempts             INTEGER     NOT NULL DEFAULT 0,
    next_attempt_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    lease_owner          TEXT,
    lease_token          UUID,
    lease_expires_at     TIMESTAMPTZ,
    safe_error_category  TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT image_scan_jobs_state_check
        CHECK (state IN ('queued', 'leased', 'dead_letter')),
    CONSTRAINT image_scan_jobs_attempts_check CHECK (attempts >= 0),
    CONSTRAINT image_scan_jobs_version_check CHECK (scan_version > 0),
    CONSTRAINT image_scan_jobs_lease_check CHECK (
        (state = 'leased'
            AND lease_owner IS NOT NULL AND btrim(lease_owner) <> ''
            AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL)
        OR
        (state <> 'leased'
            AND lease_owner IS NULL AND lease_token IS NULL AND lease_expires_at IS NULL)
    ),
    CONSTRAINT image_scan_jobs_error_check CHECK (
        safe_error_category IS NULL OR (
            btrim(safe_error_category) <> '' AND char_length(safe_error_category) <= 64
        )
    ),
    CONSTRAINT image_scan_jobs_timestamp_check CHECK (updated_at >= created_at)
);

CREATE INDEX image_scan_jobs_claim_idx
    ON image_scan_jobs(next_attempt_at, id)
    WHERE state = 'queued';
CREATE INDEX image_scan_jobs_dead_letter_idx
    ON image_scan_jobs(updated_at, id)
    WHERE state = 'dead_letter';

CREATE TABLE image_blob_sources (
    blob_cid       TEXT        NOT NULL,
    source_did     TEXT        NOT NULL,
    source_uri     TEXT        NOT NULL,
    source_cid     TEXT        NOT NULL,
    declared_mime  TEXT        NOT NULL,
    declared_size  BIGINT      NOT NULL,
    observed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (blob_cid, source_uri, source_cid),
    CONSTRAINT image_blob_sources_values_check CHECK (
        btrim(blob_cid) <> ''
        AND btrim(source_did) <> ''
        AND btrim(source_uri) <> ''
        AND btrim(source_cid) <> ''
        AND btrim(declared_mime) <> ''
        AND declared_size >= 0
    )
);

CREATE INDEX image_blob_sources_blob_idx
    ON image_blob_sources(blob_cid, observed_at DESC, source_uri);
CREATE INDEX image_blob_sources_source_idx
    ON image_blob_sources(source_did, blob_cid, source_uri, source_cid);

CREATE TABLE image_subject_states (
    subject_uri       TEXT        NOT NULL PRIMARY KEY,
    subject_kind      TEXT        NOT NULL,
    source_cid        TEXT        NOT NULL,
    visibility_state  TEXT        NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT image_subject_states_visibility_state_check
        CHECK (visibility_state IN ('blocked', 'clear')),
    CONSTRAINT image_subject_states_values_check CHECK (
        btrim(subject_uri) <> ''
        AND btrim(subject_kind) <> ''
        AND btrim(source_cid) <> ''
    ),
    UNIQUE (subject_uri, source_cid)
);

CREATE INDEX image_subject_states_visibility_idx
    ON image_subject_states(visibility_state, updated_at, subject_uri);

CREATE OR REPLACE FUNCTION appview_image_subject_is_clear(candidate_uri TEXT, candidate_cid TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1
        FROM image_subject_states
        WHERE subject_uri = candidate_uri
          AND source_cid = candidate_cid
          AND visibility_state = 'clear'
    );
END;
$$;

CREATE TABLE image_subject_requirements (
    subject_uri     TEXT        NOT NULL,
    source_cid      TEXT        NOT NULL,
    subject_kind    TEXT        NOT NULL,
    image_slot      TEXT        NOT NULL,
    blob_cid        TEXT        NOT NULL,
    scan_result_id  UUID        NOT NULL REFERENCES image_scan_results(id) ON DELETE RESTRICT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (subject_uri, image_slot),
    FOREIGN KEY (subject_uri, source_cid)
        REFERENCES image_subject_states(subject_uri, source_cid) ON DELETE CASCADE,
    CONSTRAINT image_subject_requirements_values_check CHECK (
        btrim(subject_kind) <> ''
        AND btrim(image_slot) <> ''
        AND btrim(blob_cid) <> ''
    )
);

CREATE INDEX image_subject_requirements_result_idx
    ON image_subject_requirements(scan_result_id, subject_uri, image_slot);

CREATE TABLE profile_image_candidates (
    profile_did         TEXT        NOT NULL,
    slot                TEXT        NOT NULL,
    source_cid          TEXT        NOT NULL,
    candidate_blob_cid  TEXT        NOT NULL,
    declared_mime       TEXT        NOT NULL,
    scan_result_id      UUID        NOT NULL REFERENCES image_scan_results(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (profile_did, slot),
    CONSTRAINT profile_image_candidates_slot_check CHECK (slot IN ('avatar', 'banner')),
    CONSTRAINT profile_image_candidates_values_check CHECK (
        btrim(profile_did) <> ''
        AND btrim(source_cid) <> ''
        AND btrim(candidate_blob_cid) <> ''
        AND btrim(declared_mime) <> ''
    )
);

CREATE INDEX profile_image_candidates_result_idx
    ON profile_image_candidates(scan_result_id, profile_did, slot);

CREATE TABLE image_scan_events (
    id              UUID        NOT NULL PRIMARY KEY,
    scan_result_id  UUID        NOT NULL REFERENCES image_scan_results(id) ON DELETE CASCADE,
    scan_job_id     UUID        REFERENCES image_scan_jobs(id) ON DELETE SET NULL,
    event_type      TEXT        NOT NULL,
    from_state      TEXT,
    to_state        TEXT,
    attempt         INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT image_scan_events_event_type_check CHECK (event_type IN (
        'queued', 'leased', 'retry_scheduled', 'dead_lettered', 'completed', 'manual_retry'
    )),
    CONSTRAINT image_scan_events_from_state_check CHECK (
        from_state IS NULL OR from_state IN ('pending', 'clear', 'match', 'unavailable', 'error')
    ),
    CONSTRAINT image_scan_events_to_state_check CHECK (
        to_state IS NULL OR to_state IN ('pending', 'clear', 'match', 'unavailable', 'error')
    ),
    CONSTRAINT image_scan_events_attempt_check CHECK (attempt IS NULL OR attempt >= 0)
);

CREATE INDEX image_scan_events_result_created_idx
    ON image_scan_events(scan_result_id, created_at, id);
