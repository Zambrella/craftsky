DROP INDEX IF EXISTS owner_effect_attempts_action_remote_key_idx;

ALTER TABLE owner_effect_attempts
    DROP CONSTRAINT owner_effect_attempts_remote_identity_key,
    DROP CONSTRAINT owner_effect_attempts_action_kind_check,
    DROP CONSTRAINT owner_effect_attempts_mutation_key_check,
    DROP COLUMN effect_action,
    DROP COLUMN mutation_key,
    ADD CONSTRAINT owner_effect_attempts_remote_identity_key UNIQUE (
        owner_did,
        owner_generation,
        effect_kind,
        deterministic_key
    );
