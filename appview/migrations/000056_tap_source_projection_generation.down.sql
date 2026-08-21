ALTER TABLE tap_source_records
    DROP CONSTRAINT tap_source_records_effect_attempt_fk,
    DROP CONSTRAINT tap_source_records_effect_origin_check;

UPDATE tap_source_records AS source
SET owner_generation = attempt.owner_generation
FROM owner_effect_attempts AS attempt
WHERE source.effect_operation_id = attempt.operation_id
  AND source.did = attempt.owner_did;

ALTER TABLE owner_effect_attempts
    DROP CONSTRAINT owner_effect_attempts_source_owner_key;

ALTER TABLE tap_source_records
    ADD CONSTRAINT tap_source_records_effect_origin_check CHECK (
        effect_operation_id IS NULL
        OR (owner_generation IS NOT NULL AND action IN ('create', 'update'))
    ),
    ADD CONSTRAINT tap_source_records_effect_attempt_fk FOREIGN KEY (
        effect_operation_id, did, owner_generation
    ) REFERENCES owner_effect_attempts (
        operation_id, owner_did, owner_generation
    ) ON DELETE RESTRICT;
