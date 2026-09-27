DROP TABLE IF EXISTS pds_command_tombstones;
DROP TABLE IF EXISTS pds_command_dispatches;
DROP TABLE IF EXISTS pds_command_steps;
DROP TABLE IF EXISTS pds_commands;
DROP TRIGGER IF EXISTS owner_effect_attempts_reject_new_pds_record ON owner_effect_attempts;
DROP FUNCTION IF EXISTS reject_new_pds_record_effect_attempts();
DROP VIEW IF EXISTS craftsky_profile_follower_counts;
DROP TABLE IF EXISTS pds_set_aggregates;
DROP TABLE IF EXISTS pds_set_sources;

CREATE VIEW craftsky_profile_follower_counts AS
SELECT
    profile.did AS profile_did,
    COUNT(follower.did)::BIGINT AS follower_count
FROM craftsky_profiles profile
LEFT JOIN atproto_follows follow
    ON follow.subject_did = profile.did
    AND NOT appview_owner_is_terminal(follow.did)
    AND NOT appview_owner_is_terminal(follow.subject_did)
LEFT JOIN craftsky_profiles follower
    ON follower.did = follow.did
    AND NOT appview_owner_is_terminal(follower.did)
WHERE NOT appview_owner_is_terminal(profile.did)
GROUP BY profile.did;

CREATE TABLE pds_follow_operations (
    id                  UUID        NOT NULL PRIMARY KEY,
    automatic_follow_id UUID        NOT NULL UNIQUE,
    owner_did           TEXT        NOT NULL,
    target_did          TEXT        NOT NULL,
    rkey                TEXT        NOT NULL,
    status              TEXT        NOT NULL CHECK (status IN (
                            'pending', 'writing', 'followed',
                            'alreadyFollowing', 'invalidated'
                        )),
    record_uri          TEXT,
    record_cid          TEXT,
    attempt_count       INTEGER     NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    last_error_code     TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    lease_token         UUID,
    lease_expires_at    TIMESTAMPTZ,
    next_attempt_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pds_follow_operations_not_self_check CHECK (owner_did <> target_did),
    CONSTRAINT pds_follow_operations_lease_shape_check CHECK (
        (status = 'writing' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL)
        OR
        (status <> 'writing' AND lease_token IS NULL AND lease_expires_at IS NULL)
    )
);

CREATE UNIQUE INDEX pds_follow_operations_owner_rkey_unique
    ON pds_follow_operations (owner_did, rkey);
CREATE UNIQUE INDEX pds_follow_operations_owner_target_unique
    ON pds_follow_operations (owner_did, target_did);
CREATE INDEX pds_follow_operations_claim_idx
    ON pds_follow_operations (next_attempt_at, id)
    WHERE status = 'pending';
CREATE INDEX pds_follow_operations_expired_lease_idx
    ON pds_follow_operations (lease_expires_at, id)
    WHERE status = 'writing';
CREATE INDEX terminal_purge_follow_operations_owner_idx
    ON pds_follow_operations(owner_did, id);
CREATE INDEX terminal_purge_follow_operations_target_idx
    ON pds_follow_operations(target_did, id);

ALTER TABLE owner_effect_attempts
    DROP CONSTRAINT owner_effect_attempts_record_fingerprint_check,
    DROP CONSTRAINT owner_effect_attempts_action_kind_check,
    DROP CONSTRAINT owner_effect_attempts_effect_kind_check,
    ADD CONSTRAINT owner_effect_attempts_effect_kind_check
        CHECK (effect_kind IN ('pds_record', 'object_put', 'object_delete')),
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
    ADD CONSTRAINT owner_effect_attempts_record_fingerprint_check CHECK (
        (effect_kind = 'pds_record' AND effect_action = 'put_record'
            AND record_fingerprint IS NOT NULL
            AND octet_length(record_fingerprint) = 32)
        OR
        (NOT (effect_kind = 'pds_record' AND effect_action = 'put_record')
            AND record_fingerprint IS NULL)
    );

ALTER TABLE tap_source_records
    DROP COLUMN IF EXISTS validation_reason,
    DROP COLUMN IF EXISTS semantic_validation_status,
    DROP COLUMN IF EXISTS structural_validation_status,
    DROP COLUMN IF EXISTS validation_version,
    ADD CONSTRAINT tap_source_records_effect_origin_check CHECK (
        effect_operation_id IS NULL
        OR (projection_generation IS NOT NULL AND action IN ('create', 'update'))
    ),
    ADD CONSTRAINT tap_source_records_effect_attempt_fk FOREIGN KEY (
        effect_operation_id, did
    ) REFERENCES owner_effect_attempts (
        operation_id, owner_did
    ) ON DELETE RESTRICT;
