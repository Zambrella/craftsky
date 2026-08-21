ALTER TABLE account_deletion_operations
    ADD COLUMN owner_generation BIGINT;

UPDATE account_deletion_operations operation
SET owner_generation = lifecycle.generation
FROM owner_lifecycles lifecycle
WHERE lifecycle.owner_did = operation.owner_did;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM account_deletion_operations
        WHERE owner_generation IS NULL
    ) THEN
        RAISE EXCEPTION 'account deletion operation has no owner lifecycle generation';
    END IF;
END;
$$;

ALTER TABLE account_deletion_operations
    ALTER COLUMN owner_generation SET NOT NULL,
    ADD CONSTRAINT account_deletion_operations_owner_generation_check
        CHECK (owner_generation > 0),
    ADD CONSTRAINT account_deletion_operations_id_owner_generation_key
        UNIQUE (id, owner_did, owner_generation);

CREATE TABLE account_deletion_safety_tombstones (
    id                    UUID        NOT NULL PRIMARY KEY,
    operation_id          UUID        NOT NULL,
    owner_did             TEXT        NOT NULL,
    owner_generation      BIGINT      NOT NULL CHECK (owner_generation > 0),
    kind                  TEXT        NOT NULL
        CHECK (kind IN ('pds_record', 'scheduled_object')),
    exact_key             TEXT        NOT NULL,
    upload_generation     BIGINT,
    source_attempt_id     TEXT        NOT NULL,
    state                 TEXT        NOT NULL DEFAULT 'pending'
        CHECK (state IN ('pending', 'reconciling', 'settled')),
    remote_deadline       TIMESTAMPTZ,
    settlement_not_before TIMESTAMPTZ,
    attempts              INTEGER     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at       TIMESTAMPTZ,
    lease_token           UUID,
    lease_expires_at      TIMESTAMPTZ,
    last_result_category  TEXT,
    settled_at            TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    FOREIGN KEY (operation_id, owner_did, owner_generation)
        REFERENCES account_deletion_operations(id, owner_did, owner_generation)
        ON DELETE CASCADE,
    FOREIGN KEY (owner_did) REFERENCES owner_lifecycles(owner_did)
        ON DELETE RESTRICT,
    CONSTRAINT account_deletion_safety_tombstones_identity_key
        UNIQUE NULLS NOT DISTINCT (
            operation_id, kind, exact_key, upload_generation
        ),
    CONSTRAINT account_deletion_safety_tombstones_attempt_id_check
        CHECK (
            btrim(source_attempt_id) <> ''
            AND char_length(source_attempt_id) <= 512
        ),
    CONSTRAINT account_deletion_safety_tombstones_key_size_check
        CHECK (btrim(exact_key) <> '' AND char_length(exact_key) <= 2048),
    CONSTRAINT account_deletion_safety_tombstones_exact_key_check CHECK (
        (
            kind = 'pds_record'
            AND upload_generation IS NULL
            AND split_part(exact_key, '/', 1) = 'at:'
            AND split_part(exact_key, '/', 2) = ''
            AND split_part(exact_key, '/', 3) = owner_did
            AND split_part(exact_key, '/', 4) IN (
                'social.craftsky.feed.post',
                'social.craftsky.feed.like',
                'social.craftsky.feed.repost',
                'social.craftsky.actor.profile'
            )
            AND split_part(exact_key, '/', 5) <> ''
            AND split_part(exact_key, '/', 6) = ''
        )
        OR (
            kind = 'scheduled_object'
            AND upload_generation > 0
			AND upload_generation = owner_generation
            AND exact_key ~ '^scheduled-media/v2/[1-9][0-9]*/[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
            AND split_part(exact_key, '/', 3) = upload_generation::text
			AND split_part(exact_key, '/', 4) = source_attempt_id
            AND split_part(exact_key, '/', 5) = ''
        )
    ),
    CONSTRAINT account_deletion_safety_tombstones_deadline_check CHECK (
        settlement_not_before IS NULL
        OR (
            remote_deadline IS NOT NULL
            AND settlement_not_before > remote_deadline
        )
    ),
    CONSTRAINT account_deletion_safety_tombstones_lease_check CHECK (
        (
            state = 'pending'
            AND lease_token IS NULL
            AND lease_expires_at IS NULL
            AND settled_at IS NULL
            AND next_attempt_at IS NOT NULL
        )
        OR (
            state = 'reconciling'
            AND lease_token IS NOT NULL
            AND lease_expires_at IS NOT NULL
            AND settled_at IS NULL
        )
        OR (
            state = 'settled'
            AND lease_token IS NULL
            AND lease_expires_at IS NULL
            AND next_attempt_at IS NULL
            AND settled_at IS NOT NULL
        )
    ),
    CONSTRAINT account_deletion_safety_tombstones_result_check
        CHECK (
            last_result_category IS NULL
            OR (
                btrim(last_result_category) <> ''
                AND char_length(last_result_category) <= 64
            )
        ),
    CONSTRAINT account_deletion_safety_tombstones_timestamp_check CHECK (
        updated_at >= created_at
        AND (lease_expires_at IS NULL OR lease_expires_at > updated_at)
        AND (settled_at IS NULL OR settled_at >= created_at)
    )
);

CREATE INDEX account_deletion_safety_tombstones_claim_idx
    ON account_deletion_safety_tombstones (
        next_attempt_at, operation_id, owner_did, id
    )
    WHERE state = 'pending';

CREATE INDEX account_deletion_safety_tombstones_operation_idx
    ON account_deletion_safety_tombstones (
        operation_id, state, kind, id
    );
