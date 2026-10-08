-- Retention jobs expose bounded operational state only. Error categories and
-- completion events deliberately contain no object keys, DIDs, or free-form errors.

CREATE TABLE safety_retention_jobs (
    id                   UUID        NOT NULL PRIMARY KEY,
    data_class           TEXT        NOT NULL,
    item_reference       UUID        NOT NULL,
    state                TEXT        NOT NULL CHECK (state IN ('queued', 'running', 'retry', 'completed', 'deadLetter')),
    attempt_count        SMALLINT    NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 10),
    max_attempts         SMALLINT    NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 10),
    next_attempt_at      TIMESTAMPTZ,
    lease_owner          TEXT,
    lease_expires_at     TIMESTAMPTZ,
    safe_error_category  TEXT CHECK (safe_error_category IN ('objectDelete', 'databaseWrite', 'dependencyUnavailable', 'unknown')),
    created_at           TIMESTAMPTZ NOT NULL,
    updated_at           TIMESTAMPTZ NOT NULL,
    UNIQUE (data_class, item_reference),
    CONSTRAINT safety_retention_jobs_values_check CHECK (
        btrim(data_class) <> '' AND char_length(data_class) <= 64
        AND attempt_count <= max_attempts
        AND updated_at >= created_at
        AND (lease_owner IS NULL OR (btrim(lease_owner) <> '' AND char_length(lease_owner) <= 128))
    ),
    CONSTRAINT safety_retention_jobs_lifecycle_check CHECK (
        (state = 'queued' AND attempt_count = 0 AND next_attempt_at IS NULL
            AND lease_owner IS NULL AND lease_expires_at IS NULL AND safe_error_category IS NULL)
        OR
        (state = 'running' AND attempt_count > 0 AND next_attempt_at IS NULL
            AND lease_owner IS NOT NULL AND lease_expires_at > updated_at AND safe_error_category IS NULL)
        OR
        (state = 'retry' AND attempt_count > 0 AND attempt_count < max_attempts
            AND next_attempt_at IS NOT NULL AND lease_owner IS NULL AND lease_expires_at IS NULL
            AND safe_error_category IS NOT NULL)
        OR
        (state = 'completed' AND attempt_count > 0 AND next_attempt_at IS NULL
            AND lease_owner IS NULL AND lease_expires_at IS NULL AND safe_error_category IS NULL)
        OR
        (state = 'deadLetter' AND attempt_count = max_attempts AND next_attempt_at IS NULL
            AND lease_owner IS NULL AND lease_expires_at IS NULL AND safe_error_category IS NOT NULL)
    )
);

CREATE INDEX safety_retention_jobs_claim_idx
    ON safety_retention_jobs(state, next_attempt_at, created_at, id)
    WHERE state IN ('queued', 'retry');

CREATE INDEX safety_retention_jobs_lease_idx
    ON safety_retention_jobs(lease_expires_at, id) WHERE state = 'running';

CREATE INDEX safety_retention_jobs_dead_letter_idx
    ON safety_retention_jobs(updated_at, id) WHERE state = 'deadLetter';

CREATE TABLE safety_retention_attempts (
    id                   UUID        NOT NULL PRIMARY KEY,
    job_id               UUID        NOT NULL REFERENCES safety_retention_jobs(id) ON DELETE RESTRICT,
    attempt_number       SMALLINT    NOT NULL CHECK (attempt_number BETWEEN 1 AND 10),
    outcome              TEXT        NOT NULL CHECK (outcome IN ('succeeded', 'failed')),
    safe_error_category  TEXT CHECK (safe_error_category IN ('objectDelete', 'databaseWrite', 'dependencyUnavailable', 'unknown')),
    started_at           TIMESTAMPTZ NOT NULL,
    completed_at         TIMESTAMPTZ NOT NULL,
    UNIQUE (job_id, attempt_number),
    CONSTRAINT safety_retention_attempts_values_check CHECK (
        completed_at >= started_at
        AND ((outcome = 'succeeded' AND safe_error_category IS NULL)
            OR (outcome = 'failed' AND safe_error_category IS NOT NULL))
    )
);

CREATE INDEX safety_retention_attempts_job_idx
    ON safety_retention_attempts(job_id, attempt_number);

CREATE FUNCTION reject_safety_retention_event_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'safety retention completion events are append-only';
END;
$$;

CREATE TRIGGER safety_retention_events_append_only
BEFORE UPDATE OR DELETE OR TRUNCATE ON safety_retention_events
FOR EACH STATEMENT EXECUTE FUNCTION reject_safety_retention_event_mutation();
