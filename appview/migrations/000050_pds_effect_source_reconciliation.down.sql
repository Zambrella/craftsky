ALTER TABLE tap_source_records
    DROP CONSTRAINT IF EXISTS tap_source_records_effect_attempt_fk,
    DROP CONSTRAINT IF EXISTS tap_source_records_effect_origin_check;

DROP INDEX IF EXISTS owner_effect_attempts_record_source_match_idx;

ALTER TABLE owner_effect_attempts
    DROP CONSTRAINT IF EXISTS owner_effect_attempts_source_identity_key,
    DROP CONSTRAINT IF EXISTS owner_effect_attempts_record_fingerprint_check,
    DROP COLUMN IF EXISTS mutation_sequence,
    DROP COLUMN IF EXISTS record_fingerprint;
