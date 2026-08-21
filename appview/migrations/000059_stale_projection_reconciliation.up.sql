-- Requeue repository reads that the pre-000059 worker incorrectly completed
-- without selecting authoritative sources blocked by a lifecycle-generation
-- change. AppView is stopped while migrations run, so clearing a retained
-- processing lease is safe and makes crash recovery deterministic.
UPDATE tap_repository_jobs AS repository_job
SET state = 'pending',
    next_attempt_at = GREATEST(repository_job.created_at, statement_timestamp()),
    lease_owner = NULL,
    lease_token = NULL,
    lease_expires_at = NULL,
    last_reason_code = NULL,
    authoritative_revision = NULL,
    last_successful_at = NULL,
    updated_at = GREATEST(repository_job.created_at, statement_timestamp())
FROM owner_lifecycles AS lifecycle
WHERE repository_job.did = lifecycle.owner_did
  AND repository_job.job_kind = 'pds_reconcile'
  AND lifecycle.state = 'active'
  AND EXISTS (
      SELECT 1
      FROM tap_projection_jobs AS projection_job
      JOIN tap_source_records AS source
        ON source.uri = projection_job.source_uri
       AND source.source_event_id = projection_job.source_event_id
      WHERE source.did = repository_job.did
        AND projection_job.state = 'blocked'
        AND projection_job.dependency_kind = 'repository_did'
        AND projection_job.dependency_key = repository_job.did
        AND (
            source.projection_generation IS NULL
            OR source.projection_generation <> lifecycle.generation
        )
  );
