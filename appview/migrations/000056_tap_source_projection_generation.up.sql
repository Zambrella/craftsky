ALTER TABLE tap_source_records
    DROP CONSTRAINT tap_source_records_effect_attempt_fk,
    DROP CONSTRAINT tap_source_records_effect_origin_check;

ALTER TABLE owner_effect_attempts
    ADD CONSTRAINT owner_effect_attempts_source_owner_key UNIQUE (
        operation_id, owner_did
    );

-- owner_generation is the current lifecycle generation that authorizes the
-- serving projection. The effect attempt retains its own historical mutation
-- generation; those values legitimately differ after an owner rejoins.
ALTER TABLE tap_source_records
    ADD CONSTRAINT tap_source_records_effect_origin_check CHECK (
        effect_operation_id IS NULL
        OR (owner_generation IS NOT NULL AND action IN ('create', 'update'))
    ),
    ADD CONSTRAINT tap_source_records_effect_attempt_fk FOREIGN KEY (
        effect_operation_id, did
    ) REFERENCES owner_effect_attempts (
        operation_id, owner_did
    ) ON DELETE RESTRICT;

UPDATE tap_source_records AS source
SET owner_generation = lifecycle.generation,
    updated_at = now()
FROM owner_effect_attempts AS attempt,
     owner_lifecycles AS lifecycle
WHERE source.effect_operation_id = attempt.operation_id
  AND source.did = attempt.owner_did
  AND lifecycle.owner_did = source.did
  AND lifecycle.state = 'active'
  AND attempt.projection_disposition = 'eligible_current'
  AND source.projection_disposition = 'eligible'
  AND source.ordering_status = 'authoritative'
  AND source.action IN ('create', 'update')
  AND source.owner_generation IS DISTINCT FROM lifecycle.generation;

UPDATE tap_projection_jobs AS job
SET state = 'pending',
    dependency_kind = NULL,
    dependency_key = NULL,
    next_attempt_at = now(),
    lease_owner = NULL,
    lease_token = NULL,
    lease_expires_at = NULL,
    last_reason_code = NULL,
    updated_at = now()
FROM tap_source_records AS source,
     owner_lifecycles AS lifecycle
WHERE job.source_uri = source.uri
  AND lifecycle.owner_did = source.did
  AND lifecycle.state = 'active'
  AND source.owner_generation = lifecycle.generation
  AND source.projection_disposition = 'eligible'
  AND source.ordering_status = 'authoritative'
  AND job.state = 'blocked'
  AND job.last_reason_code = 'source_order_uncertain';
