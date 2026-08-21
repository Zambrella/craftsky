ALTER TABLE owner_effect_attempts
    ADD COLUMN effect_action TEXT,
    ADD COLUMN mutation_key TEXT;

-- This is a pre-production breaking migration. Existing PDS record attempts
-- were all writes; the private-object kinds retain their matching action.
UPDATE owner_effect_attempts
SET effect_action = CASE effect_kind
        WHEN 'pds_record' THEN 'put_record'
        WHEN 'object_put' THEN 'put_object'
        WHEN 'object_delete' THEN 'delete_object'
    END,
    mutation_key = operation_id;

ALTER TABLE owner_effect_attempts
    ALTER COLUMN effect_action SET NOT NULL,
    ALTER COLUMN mutation_key SET NOT NULL,
    ADD CONSTRAINT owner_effect_attempts_mutation_key_check CHECK (
        btrim(mutation_key) <> '' AND char_length(mutation_key) <= 512
    ),
    ADD CONSTRAINT owner_effect_attempts_action_kind_check CHECK (
        (effect_kind = 'pds_record'
            AND effect_action IN ('put_record', 'delete_record'))
        OR
        (effect_kind = 'object_put'
            AND effect_action IN ('put_object', 'upload_blob'))
        OR
        (effect_kind = 'object_delete'
            AND effect_action = 'delete_object')
    ),
    DROP CONSTRAINT owner_effect_attempts_remote_identity_key,
    ADD CONSTRAINT owner_effect_attempts_remote_identity_key UNIQUE (
        owner_did,
        owner_generation,
        effect_kind,
        deterministic_key,
        effect_action,
        mutation_key
    );

CREATE INDEX owner_effect_attempts_action_remote_key_idx
    ON owner_effect_attempts (
        effect_kind,
        effect_action,
        deterministic_key,
        mutation_key,
        owner_generation
    );
