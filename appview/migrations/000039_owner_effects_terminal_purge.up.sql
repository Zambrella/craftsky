-- One authoritative terminal predicate is shared by serving, projection, and
-- effect SQL. It deliberately treats NULL and unknown DIDs as non-terminal;
-- callers that require a known lifecycle must enforce that separately.
CREATE OR REPLACE FUNCTION appview_owner_is_terminal(candidate_did TEXT)
RETURNS BOOLEAN
LANGUAGE SQL
STABLE
PARALLEL SAFE
AS $$
    SELECT COALESCE((
        SELECT lifecycle.state = 'terminal'
        FROM owner_lifecycles AS lifecycle
        WHERE lifecycle.owner_did = candidate_did
    ), false)
$$;

-- Current membership is deliberately stricter than "profile row exists".
-- Departed, deletion-pending, deleting, terminal, and unknown owners are all
-- denied even if a delayed projector has retained or recreated a profile row.
CREATE OR REPLACE FUNCTION appview_owner_is_active(candidate_did TEXT)
RETURNS BOOLEAN
LANGUAGE SQL
STABLE
PARALLEL SAFE
AS $$
    SELECT COALESCE((
        SELECT lifecycle.state = 'active'
        FROM owner_lifecycles AS lifecycle
        WHERE lifecycle.owner_did = candidate_did
    ), false)
$$;

CREATE TABLE owner_purge_components (
    owner_did          TEXT        NOT NULL,
    owner_generation   BIGINT      NOT NULL CHECK (owner_generation > 0),
    component          TEXT        NOT NULL CHECK (btrim(component) <> '' AND char_length(component) <= 128),
    did_role           TEXT        NOT NULL CHECK (btrim(did_role) <> '' AND char_length(did_role) <= 64),
    state              TEXT        NOT NULL DEFAULT 'pending'
        CHECK (state IN ('pending', 'running', 'complete')),
    attempts           INTEGER     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    lease_owner        TEXT,
    lease_token        UUID,
    lease_expires_at   TIMESTAMPTZ,
    last_error_category TEXT,
    completed_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_did, owner_generation, component, did_role),
    FOREIGN KEY (owner_did) REFERENCES owner_lifecycles(owner_did) ON DELETE RESTRICT,
    CONSTRAINT owner_purge_components_lease_check CHECK (
        (state = 'pending'
            AND lease_owner IS NULL AND lease_token IS NULL
            AND lease_expires_at IS NULL AND completed_at IS NULL)
        OR
        (state = 'running'
            AND btrim(lease_owner) <> '' AND lease_token IS NOT NULL
            AND lease_expires_at IS NOT NULL AND completed_at IS NULL)
        OR
        (state = 'complete'
            AND lease_owner IS NULL AND lease_token IS NULL
            AND lease_expires_at IS NULL AND completed_at IS NOT NULL)
    ),
    CONSTRAINT owner_purge_components_timestamp_order_check
        CHECK (updated_at >= created_at AND (completed_at IS NULL OR completed_at >= created_at))
);

CREATE INDEX owner_purge_components_claim_idx
    ON owner_purge_components (next_attempt_at, owner_did, owner_generation, component, did_role)
    WHERE state <> 'complete';

CREATE TABLE owner_effect_attempts (
    operation_id           TEXT        NOT NULL PRIMARY KEY,
    owner_did              TEXT        NOT NULL,
    owner_generation       BIGINT      NOT NULL CHECK (owner_generation > 0),
    effect_kind            TEXT        NOT NULL
        CHECK (effect_kind IN ('pds_record', 'object_put', 'object_delete')),
    deterministic_key      TEXT        NOT NULL,
    request_fingerprint    BYTEA       NOT NULL,
    expected_cid           TEXT,
    result_cid             TEXT,
    remote_outcome         TEXT        NOT NULL DEFAULT 'prepared'
        CHECK (remote_outcome IN (
            'prepared', 'dispatched', 'accepted', 'rejected',
            'abandoned_pre_transition', 'outcome_unknown_pre_transition',
            'reconciled_accepted', 'reconciled_not_accepted', 'reconciliation_mismatch'
        )),
    projection_disposition TEXT        NOT NULL DEFAULT 'pending'
        CHECK (projection_disposition IN (
            'pending', 'hidden_non_active', 'eligible_current',
            'denied_terminal', 'not_applicable'
        )),
    repeat_forbidden       BOOLEAN     NOT NULL DEFAULT false,
    remote_deadline        TIMESTAMPTZ NOT NULL,
    dispatched_at          TIMESTAMPTZ,
    completed_at           TIMESTAMPTZ,
    reconciled_at          TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (owner_did) REFERENCES owner_lifecycles(owner_did) ON DELETE RESTRICT,
    CONSTRAINT owner_effect_attempts_operation_id_check
        CHECK (btrim(operation_id) <> '' AND char_length(operation_id) <= 512),
    CONSTRAINT owner_effect_attempts_deterministic_key_check
        CHECK (btrim(deterministic_key) <> '' AND char_length(deterministic_key) <= 2048),
    CONSTRAINT owner_effect_attempts_fingerprint_check
        CHECK (octet_length(request_fingerprint) = 32),
    CONSTRAINT owner_effect_attempts_cid_check CHECK (
        (expected_cid IS NULL OR btrim(expected_cid) <> '')
        AND (result_cid IS NULL OR btrim(result_cid) <> '')
    ),
    CONSTRAINT owner_effect_attempts_deadline_check
        CHECK (remote_deadline > created_at),
    CONSTRAINT owner_effect_attempts_boundary_check CHECK (
        (remote_outcome = 'prepared'
            AND dispatched_at IS NULL AND completed_at IS NULL
            AND reconciled_at IS NULL AND repeat_forbidden = false)
        OR
        (remote_outcome = 'dispatched'
            AND dispatched_at IS NOT NULL AND completed_at IS NULL
            AND reconciled_at IS NULL AND repeat_forbidden = true)
        OR
        (remote_outcome IN ('accepted', 'rejected')
            AND dispatched_at IS NOT NULL AND completed_at IS NOT NULL
            AND reconciled_at IS NULL AND repeat_forbidden = true)
        OR
        (remote_outcome = 'abandoned_pre_transition'
            AND dispatched_at IS NULL AND completed_at IS NOT NULL
            AND reconciled_at IS NULL AND repeat_forbidden = true)
        OR
        (remote_outcome = 'outcome_unknown_pre_transition'
            AND dispatched_at IS NOT NULL AND completed_at IS NULL
            AND reconciled_at IS NULL AND repeat_forbidden = true)
        OR
        (remote_outcome IN (
                'reconciled_accepted', 'reconciled_not_accepted', 'reconciliation_mismatch'
            )
            AND dispatched_at IS NOT NULL AND completed_at IS NOT NULL
            AND reconciled_at IS NOT NULL AND repeat_forbidden = true)
    ),
    CONSTRAINT owner_effect_attempts_timestamp_order_check CHECK (
        updated_at >= created_at
        AND (dispatched_at IS NULL OR dispatched_at >= created_at)
        AND (completed_at IS NULL OR completed_at >= created_at)
        AND (reconciled_at IS NULL OR reconciled_at >= created_at)
    ),
    CONSTRAINT owner_effect_attempts_remote_identity_key
        UNIQUE (owner_did, owner_generation, effect_kind, deterministic_key)
);

CREATE INDEX owner_effect_attempts_owner_generation_idx
    ON owner_effect_attempts (owner_did, owner_generation, operation_id);

CREATE INDEX owner_effect_attempts_remote_key_idx
    ON owner_effect_attempts (effect_kind, deterministic_key, owner_generation);

CREATE INDEX owner_effect_attempts_unresolved_idx
    ON owner_effect_attempts (owner_did, owner_generation, updated_at, operation_id)
    WHERE remote_outcome IN ('dispatched', 'outcome_unknown_pre_transition');
