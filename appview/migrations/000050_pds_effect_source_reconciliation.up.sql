ALTER TABLE owner_effect_attempts
    ADD COLUMN record_fingerprint BYTEA,
    ADD COLUMN mutation_sequence BIGINT GENERATED ALWAYS AS IDENTITY;

-- Pre-production compatibility only: unconditional legacy Put attempts used
-- the same request/content fingerprint. Conditional legacy Puts cannot be
-- reconstructed from SQL, so retaining their request fingerprint here makes
-- them fail closed during content matching rather than guessing provenance.
UPDATE owner_effect_attempts
SET record_fingerprint = request_fingerprint
WHERE effect_kind = 'pds_record' AND effect_action = 'put_record';

ALTER TABLE owner_effect_attempts
    ADD CONSTRAINT owner_effect_attempts_record_fingerprint_check CHECK (
        (effect_kind = 'pds_record' AND effect_action = 'put_record'
            AND record_fingerprint IS NOT NULL
            AND octet_length(record_fingerprint) = 32)
        OR
        (NOT (effect_kind = 'pds_record' AND effect_action = 'put_record')
            AND record_fingerprint IS NULL)
    ),
    ADD CONSTRAINT owner_effect_attempts_source_identity_key UNIQUE (
        operation_id, owner_did, owner_generation
    );

CREATE INDEX owner_effect_attempts_record_source_match_idx
    ON owner_effect_attempts (
        owner_did,
        deterministic_key,
        effect_kind,
        effect_action,
        record_fingerprint,
        mutation_sequence DESC
    )
    INCLUDE (result_cid, remote_outcome, projection_disposition);

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
