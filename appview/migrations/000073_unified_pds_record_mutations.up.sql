ALTER TABLE tap_source_records
    ADD COLUMN validation_version INTEGER NOT NULL DEFAULT 1
        CHECK (validation_version > 0),
    ADD COLUMN structural_validation_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (structural_validation_status IN ('pending', 'valid', 'invalid')),
    ADD COLUMN semantic_validation_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (semantic_validation_status IN ('pending', 'valid', 'invalid')),
    ADD COLUMN validation_reason TEXT
        CHECK (validation_reason IS NULL OR (
            btrim(validation_reason) <> '' AND char_length(validation_reason) <= 128
        ));

ALTER TABLE tap_source_records
    DROP CONSTRAINT tap_source_records_effect_attempt_fk,
    DROP CONSTRAINT tap_source_records_effect_origin_check;

UPDATE tap_source_records
SET effect_operation_id = NULL,
    projection_generation = NULL,
    updated_at = now()
WHERE effect_operation_id IS NOT NULL OR projection_generation IS NOT NULL;

DROP TABLE pds_follow_operations;

ALTER TABLE owner_effect_attempts
    DROP CONSTRAINT owner_effect_attempts_effect_kind_check,
    DROP CONSTRAINT owner_effect_attempts_action_kind_check,
    DROP CONSTRAINT owner_effect_attempts_record_fingerprint_check,
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

CREATE FUNCTION reject_new_pds_record_effect_attempts()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.effect_kind = 'pds_record' THEN
        RAISE EXCEPTION 'new pds_record owner effect attempts are not supported'
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER owner_effect_attempts_reject_new_pds_record
BEFORE INSERT ON owner_effect_attempts
FOR EACH ROW
EXECUTE FUNCTION reject_new_pds_record_effect_attempts();

CREATE TABLE pds_set_sources (
    source_uri          TEXT        NOT NULL PRIMARY KEY
        REFERENCES tap_source_records(uri) ON DELETE CASCADE,
    kind                TEXT        NOT NULL
        CHECK (kind IN ('follow', 'block', 'like', 'repost')),
    actor_did           TEXT        NOT NULL,
    scope_key           TEXT        NOT NULL,
    subject_did         TEXT,
    subject_uri         TEXT,
    subject_cid         TEXT,
    activity_at         TIMESTAMPTZ NOT NULL,
    eligible            BOOLEAN     NOT NULL,
    ineligibility_reason TEXT,
    dependency_kind     TEXT,
    dependency_key      TEXT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pds_set_sources_representative_key
        UNIQUE (source_uri, kind, actor_did, scope_key),
    CONSTRAINT pds_set_sources_actor_check
        CHECK (btrim(actor_did) <> '' AND char_length(actor_did) <= 512),
    CONSTRAINT pds_set_sources_scope_check
        CHECK (btrim(scope_key) <> '' AND char_length(scope_key) <= 2048),
    CONSTRAINT pds_set_sources_subject_check CHECK (
        (kind IN ('follow', 'block')
            AND subject_did IS NOT NULL AND btrim(subject_did) <> ''
            AND subject_uri IS NULL AND subject_cid IS NULL)
        OR
        (kind IN ('like', 'repost')
            AND subject_did IS NULL
            AND subject_uri IS NOT NULL AND btrim(subject_uri) <> '')
    ),
    CONSTRAINT pds_set_sources_eligibility_check CHECK (
        (eligible
            AND ineligibility_reason IS NULL
            AND dependency_kind IS NULL AND dependency_key IS NULL)
        OR
        (NOT eligible
            AND ineligibility_reason IS NOT NULL
            AND btrim(ineligibility_reason) <> ''
            AND char_length(ineligibility_reason) <= 128)
    ),
    CONSTRAINT pds_set_sources_dependency_check CHECK (
        (dependency_kind IS NULL AND dependency_key IS NULL)
        OR
        (dependency_kind IN ('member_did', 'subject_uri', 'repository_did')
            AND dependency_key IS NOT NULL AND btrim(dependency_key) <> ''
            AND char_length(dependency_key) <= 2048)
    )
);

CREATE INDEX pds_set_sources_scope_idx
    ON pds_set_sources (kind, actor_did, scope_key, eligible, activity_at, source_uri);
CREATE INDEX pds_set_sources_dependency_idx
    ON pds_set_sources (dependency_kind, dependency_key, source_uri)
    WHERE dependency_kind IS NOT NULL;
CREATE INDEX pds_set_sources_actor_purge_idx
    ON pds_set_sources (actor_did, source_uri);
CREATE INDEX pds_set_sources_subject_did_purge_idx
    ON pds_set_sources (subject_did, source_uri)
    WHERE subject_did IS NOT NULL;

CREATE TABLE pds_set_aggregates (
    kind                      TEXT        NOT NULL
        CHECK (kind IN ('follow', 'block', 'like', 'repost')),
    actor_did                 TEXT        NOT NULL,
    scope_key                 TEXT        NOT NULL,
    subject_did               TEXT,
    subject_uri               TEXT,
    eligible_source_count     INTEGER     NOT NULL CHECK (eligible_source_count > 0),
    representative_source_uri TEXT        NOT NULL,
    activated_at              TIMESTAMPTZ NOT NULL,
    representative_metadata   JSONB       NOT NULL DEFAULT '{}'::jsonb,
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, actor_did, scope_key),
    CONSTRAINT pds_set_aggregates_actor_check
        CHECK (btrim(actor_did) <> '' AND char_length(actor_did) <= 512),
    CONSTRAINT pds_set_aggregates_scope_check
        CHECK (btrim(scope_key) <> '' AND char_length(scope_key) <= 2048),
    CONSTRAINT pds_set_aggregates_subject_check CHECK (
        (kind IN ('follow', 'block')
            AND subject_did IS NOT NULL AND btrim(subject_did) <> ''
            AND subject_uri IS NULL)
        OR
        (kind IN ('like', 'repost')
            AND subject_did IS NULL
            AND subject_uri IS NOT NULL AND btrim(subject_uri) <> '')
    ),
    CONSTRAINT pds_set_aggregates_representative_check
        CHECK (btrim(representative_source_uri) <> ''),
    CONSTRAINT pds_set_aggregates_representative_fkey FOREIGN KEY (
        representative_source_uri, kind, actor_did, scope_key
    ) REFERENCES pds_set_sources (
        source_uri, kind, actor_did, scope_key
    ) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX pds_set_aggregates_actor_purge_idx
    ON pds_set_aggregates (actor_did, kind, scope_key);
CREATE INDEX pds_set_aggregates_subject_did_purge_idx
    ON pds_set_aggregates (subject_did, kind, actor_did, scope_key)
    WHERE subject_did IS NOT NULL;
CREATE INDEX pds_set_aggregates_subject_uri_idx
    ON pds_set_aggregates (subject_uri, kind, actor_did)
    WHERE subject_uri IS NOT NULL;
CREATE INDEX pds_set_aggregates_follow_subject_pagination_idx
    ON pds_set_aggregates (subject_did, activated_at DESC, representative_source_uri DESC)
    WHERE kind = 'follow';
CREATE INDEX pds_set_aggregates_follow_actor_pagination_idx
    ON pds_set_aggregates (actor_did, activated_at DESC, representative_source_uri DESC)
    WHERE kind = 'follow';
CREATE INDEX pds_set_aggregates_block_actor_pagination_idx
    ON pds_set_aggregates (actor_did, activated_at DESC, subject_did DESC)
    WHERE kind = 'block';

CREATE OR REPLACE VIEW craftsky_profile_follower_counts AS
SELECT
    profile.did AS profile_did,
    COUNT(follower.did)::BIGINT AS follower_count
FROM craftsky_profiles profile
LEFT JOIN pds_set_aggregates follow
    ON follow.kind = 'follow'
    AND follow.subject_did = profile.did
    AND NOT appview_owner_is_terminal(follow.actor_did)
    AND NOT appview_owner_is_terminal(follow.subject_did)
LEFT JOIN craftsky_profiles follower
    ON follower.did = follow.actor_did
    AND NOT appview_owner_is_terminal(follower.did)
WHERE NOT appview_owner_is_terminal(profile.did)
GROUP BY profile.did;

CREATE TABLE pds_commands (
    id                          UUID        NOT NULL PRIMARY KEY,
    owner_did                   TEXT        NOT NULL
        REFERENCES owner_lifecycles(owner_did) ON DELETE RESTRICT,
    owner_generation            BIGINT      NOT NULL CHECK (owner_generation > 0),
    operation_kind              TEXT        NOT NULL,
    operation_key               UUID        NOT NULL,
    request_fingerprint_version INTEGER     NOT NULL CHECK (request_fingerprint_version > 0),
    request_fingerprint         BYTEA       NOT NULL CHECK (octet_length(request_fingerprint) = 32),
    immutable_request           JSONB       NOT NULL,
    selected_uri                TEXT,
    selected_rkey               TEXT,
    expected_owner_did          TEXT,
    expected_owner_generation   BIGINT,
    expected_target_did         TEXT,
    active_plan_version         INTEGER     NOT NULL DEFAULT 0 CHECK (active_plan_version >= 0),
    state                       TEXT        NOT NULL DEFAULT 'prepared'
        CHECK (state IN ('prepared', 'dispatching', 'accepted', 'ambiguous', 'rejected')),
    terminal_http_status        INTEGER,
    terminal_response_body      JSONB,
    terminal_response_headers   JSONB,
    retry_after_seconds         INTEGER CHECK (retry_after_seconds BETWEEN 1 AND 5),
    replay_expires_at           TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    terminal_at                 TIMESTAMPTZ,
    UNIQUE (owner_did, operation_kind, operation_key),
    CONSTRAINT pds_commands_kind_check
        CHECK (btrim(operation_kind) <> '' AND char_length(operation_kind) <= 128),
    CONSTRAINT pds_commands_terminal_check CHECK (
        (state IN ('accepted', 'rejected')
            AND terminal_http_status IS NOT NULL
            AND replay_expires_at IS NOT NULL
            AND terminal_at IS NOT NULL)
        OR
        (state NOT IN ('accepted', 'rejected')
            AND terminal_http_status IS NULL
            AND terminal_response_body IS NULL
            AND terminal_response_headers IS NULL
            AND replay_expires_at IS NULL
            AND terminal_at IS NULL)
    ),
    CONSTRAINT pds_commands_ambiguity_check CHECK (
        (state = 'ambiguous' AND retry_after_seconds IS NOT NULL)
        OR (state <> 'ambiguous' AND retry_after_seconds IS NULL)
    ),
    CONSTRAINT pds_commands_timestamp_check
        CHECK (updated_at >= created_at AND (terminal_at IS NULL OR terminal_at >= created_at))
);

CREATE INDEX pds_commands_owner_purge_idx
    ON pds_commands (owner_did, id);
CREATE INDEX pds_commands_expected_owner_purge_idx
    ON pds_commands (expected_owner_did, id)
    WHERE expected_owner_did IS NOT NULL;
CREATE INDEX pds_commands_expected_target_purge_idx
    ON pds_commands (expected_target_did, id)
    WHERE expected_target_did IS NOT NULL;
CREATE INDEX pds_commands_recovery_idx
    ON pds_commands (updated_at, id)
    WHERE state IN ('prepared', 'dispatching', 'ambiguous');
CREATE INDEX pds_commands_retention_idx
    ON pds_commands (replay_expires_at, id)
    WHERE state IN ('accepted', 'rejected');

CREATE TABLE pds_command_steps (
    command_id       UUID    NOT NULL REFERENCES pds_commands(id) ON DELETE CASCADE,
    plan_version     INTEGER NOT NULL CHECK (plan_version > 0),
    ordinal          INTEGER NOT NULL CHECK (ordinal >= 0),
    action           TEXT    NOT NULL CHECK (action IN ('create', 'update', 'delete')),
    repo_did         TEXT    NOT NULL,
    collection       TEXT    NOT NULL,
    rkey             TEXT    NOT NULL,
    record           JSON,
    expected_cid     TEXT,
    selected_uri     TEXT    NOT NULL,
    generated_values JSONB   NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (command_id, plan_version, ordinal),
    CONSTRAINT pds_command_steps_identity_check CHECK (
        btrim(repo_did) <> '' AND btrim(collection) <> ''
        AND btrim(rkey) <> '' AND btrim(selected_uri) <> ''
    ),
    CONSTRAINT pds_command_steps_record_check CHECK (
        (action IN ('create', 'update') AND record IS NOT NULL)
        OR (action = 'delete' AND record IS NULL)
    )
);

CREATE INDEX pds_command_steps_uri_idx
    ON pds_command_steps (selected_uri, command_id, plan_version, ordinal);
CREATE INDEX pds_command_steps_repo_purge_idx
    ON pds_command_steps (repo_did, command_id, plan_version, ordinal);

CREATE TABLE pds_command_dispatches (
    id                           UUID        NOT NULL PRIMARY KEY,
    command_id                   UUID        NOT NULL REFERENCES pds_commands(id) ON DELETE CASCADE,
    attempt_ordinal              INTEGER     NOT NULL CHECK (attempt_ordinal > 0),
    plan_version                 INTEGER     NOT NULL CHECK (plan_version > 0),
    dispatch_fingerprint_version INTEGER     NOT NULL CHECK (dispatch_fingerprint_version > 0),
    dispatch_fingerprint         BYTEA       NOT NULL CHECK (octet_length(dispatch_fingerprint) = 32),
    repository_cid               TEXT,
    repository_revision          TEXT,
    remote_deadline              TIMESTAMPTZ NOT NULL,
    outcome                      TEXT        NOT NULL DEFAULT 'dispatching'
        CHECK (outcome IN ('dispatching', 'accepted', 'ambiguous', 'rejected', 'invalid_swap')),
    error_class                  TEXT,
    result_commit_cid            TEXT,
    result_records               JSONB,
    started_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at                 TIMESTAMPTZ,
    UNIQUE (command_id, attempt_ordinal),
    CONSTRAINT pds_command_dispatches_completion_check CHECK (
        (outcome = 'dispatching' AND completed_at IS NULL)
        OR (outcome <> 'dispatching' AND completed_at IS NOT NULL)
    )
);

CREATE INDEX pds_command_dispatches_command_idx
    ON pds_command_dispatches (command_id, attempt_ordinal);

CREATE TABLE pds_command_tombstones (
    owner_did       TEXT        NOT NULL
        REFERENCES owner_lifecycles(owner_did) ON DELETE RESTRICT,
    operation_kind  TEXT        NOT NULL,
    scoped_key_hash BYTEA       NOT NULL CHECK (octet_length(scoped_key_hash) = 32),
    compacted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_did, operation_kind, scoped_key_hash),
    CONSTRAINT pds_command_tombstones_kind_check
        CHECK (btrim(operation_kind) <> '' AND char_length(operation_kind) <= 128)
);

CREATE INDEX pds_command_tombstones_owner_purge_idx
    ON pds_command_tombstones (owner_did, operation_kind, scoped_key_hash);
