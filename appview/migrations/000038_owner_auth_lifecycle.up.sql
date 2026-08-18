CREATE TABLE owner_lifecycles (
    owner_did            TEXT        NOT NULL PRIMARY KEY,
    state                TEXT        NOT NULL,
    generation           BIGINT      NOT NULL,
    auth_epoch           BIGINT      NOT NULL,
    transition_reason    TEXT        NOT NULL,
    transitioned_at      TIMESTAMPTZ NOT NULL,
    terminal_at          TIMESTAMPTZ,
    purge_completed_at   TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT owner_lifecycles_owner_did_check
        CHECK (btrim(owner_did) <> ''),
    CONSTRAINT owner_lifecycles_state_check
        CHECK (state IN ('active', 'departed', 'deletion_pending', 'deleting', 'terminal')),
    CONSTRAINT owner_lifecycles_generation_check
        CHECK (generation > 0),
    CONSTRAINT owner_lifecycles_auth_epoch_check
        CHECK (auth_epoch > 0),
    CONSTRAINT owner_lifecycles_transition_reason_check
        CHECK (btrim(transition_reason) <> '' AND char_length(transition_reason) <= 256),
    CONSTRAINT owner_lifecycles_terminal_timestamp_check
        CHECK (
            (state = 'terminal' AND terminal_at IS NOT NULL)
            OR (state <> 'terminal' AND terminal_at IS NULL)
        ),
    CONSTRAINT owner_lifecycles_purge_completion_check
        CHECK (
            purge_completed_at IS NULL
            OR (state = 'terminal' AND purge_completed_at >= terminal_at)
        ),
    CONSTRAINT owner_lifecycles_timestamp_order_check
        CHECK (transitioned_at >= created_at AND updated_at >= transitioned_at)
);

-- The trigger is a defence-in-depth invariant for every caller, including
-- future workers which update purge completion directly. State changes always
-- advance exactly one generation; auth epochs never move backwards; terminal
-- identity state can never be reversed by replay.
CREATE FUNCTION enforce_owner_lifecycle_monotonicity()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.owner_did <> OLD.owner_did THEN
        RAISE EXCEPTION 'owner lifecycle owner is immutable'
            USING ERRCODE = '23514';
    END IF;
    IF OLD.state = 'terminal' AND NEW.state <> 'terminal' THEN
        RAISE EXCEPTION 'terminal owner lifecycle is irreversible'
            USING ERRCODE = '23514';
    END IF;
    IF OLD.state = 'terminal' AND NEW.terminal_at IS DISTINCT FROM OLD.terminal_at THEN
        RAISE EXCEPTION 'terminal owner lifecycle timestamp is immutable'
            USING ERRCODE = '23514';
    END IF;
    IF NEW.generation < OLD.generation THEN
        RAISE EXCEPTION 'owner lifecycle generation cannot decrease'
            USING ERRCODE = '23514';
    END IF;
    IF NEW.auth_epoch < OLD.auth_epoch THEN
        RAISE EXCEPTION 'owner lifecycle auth epoch cannot decrease'
            USING ERRCODE = '23514';
    END IF;
    IF NEW.state <> OLD.state AND NEW.generation <> OLD.generation + 1 THEN
        RAISE EXCEPTION 'owner lifecycle state change must advance one generation'
            USING ERRCODE = '23514';
    END IF;
    IF NEW.state = OLD.state AND NEW.generation <> OLD.generation THEN
        RAISE EXCEPTION 'owner lifecycle generation changes only with state'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER owner_lifecycles_monotonicity
BEFORE UPDATE ON owner_lifecycles
FOR EACH ROW
EXECUTE FUNCTION enforce_owner_lifecycle_monotonicity();

CREATE INDEX owner_lifecycles_terminal_pending_idx
    ON owner_lifecycles (owner_did)
    WHERE state = 'terminal' AND purge_completed_at IS NULL;

-- The repository is pre-production, so current projected members are the
-- authoritative seed for the new explicit lifecycle. Missing rows must never
-- be interpreted as implicit active authority after this migration.
INSERT INTO owner_lifecycles(
    owner_did,state,generation,auth_epoch,transition_reason,
    transitioned_at,created_at,updated_at
)
SELECT did,'active',1,1,'migrationBackfill',created_at,created_at,created_at
FROM craftsky_profiles
ON CONFLICT (owner_did) DO NOTHING;

-- Pre-production reset: the old rows have no trustworthy lifecycle, epoch,
-- expiry, or handoff-confirmation state. Requiring a fresh sign-in is safer
-- than manufacturing authority for opaque development credentials.
DELETE FROM oauth_auth_requests;
DELETE FROM craftsky_sessions;
DELETE FROM account_deletion_operations;
DELETE FROM oauth_sessions;

ALTER TABLE account_deletion_operations
    ADD COLUMN deletion_credential_generation BIGINT,
    ADD CONSTRAINT account_deletion_operations_credential_shape_check
        CHECK (
            (
                reauth_oauth_session_id IS NULL
                AND deletion_oauth_session_id IS NULL
                AND deletion_credential_generation IS NULL
            )
            OR
            (
                deletion_credential_generation > 0
                AND (
                    (reauth_oauth_session_id IS NOT NULL AND deletion_oauth_session_id IS NULL)
                    OR
                    (reauth_oauth_session_id IS NULL AND deletion_oauth_session_id IS NOT NULL)
                )
            )
        ),
    DROP CONSTRAINT IF EXISTS account_deletion_operations_state_check,
    ADD CONSTRAINT account_deletion_operations_state_check
        CHECK (state IN ('intent', 'active', 'retrying', 'reauth_required'));

ALTER TABLE oauth_sessions
    ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'pending_handoff',
    ADD COLUMN owner_generation BIGINT NOT NULL DEFAULT 1,
    ADD COLUMN auth_epoch BIGINT NOT NULL DEFAULT 1,
    ADD COLUMN row_version BIGINT NOT NULL DEFAULT 1,
    ADD COLUMN absolute_expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '180 days'),
    ADD COLUMN deletion_operation_id UUID,
    ADD COLUMN deletion_credential_generation BIGINT,
    ADD COLUMN revocation_requested_at TIMESTAMPTZ,
    ADD COLUMN cleanup_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN cleanup_next_attempt_at TIMESTAMPTZ,
    ADD COLUMN cleanup_lease_token UUID,
    ADD COLUMN cleanup_lease_expires_at TIMESTAMPTZ,
    ADD COLUMN cleanup_last_category TEXT,
    ADD CONSTRAINT oauth_sessions_owner_fkey
        FOREIGN KEY (account_did) REFERENCES owner_lifecycles(owner_did),
    ADD CONSTRAINT oauth_sessions_lifecycle_check
        CHECK (lifecycle_state IN ('pending_handoff', 'active', 'deletion_only', 'revocation_pending')),
    ADD CONSTRAINT oauth_sessions_authority_check
        CHECK (owner_generation > 0 AND auth_epoch > 0 AND row_version > 0),
    ADD CONSTRAINT oauth_sessions_expiry_check
        CHECK (absolute_expires_at > created_at),
    ADD CONSTRAINT oauth_sessions_deletion_shape_check
        CHECK (
            (lifecycle_state = 'deletion_only'
                AND deletion_operation_id IS NOT NULL
                AND deletion_credential_generation > 0)
            OR
            (lifecycle_state <> 'deletion_only'
                AND deletion_operation_id IS NULL
                AND deletion_credential_generation IS NULL)
        ),
    ADD CONSTRAINT oauth_sessions_revocation_shape_check
        CHECK (
            cleanup_attempts >= 0
            AND (cleanup_last_category IS NULL OR btrim(cleanup_last_category) <> '')
            AND (
                (cleanup_lease_token IS NULL AND cleanup_lease_expires_at IS NULL)
                OR
                (cleanup_lease_token IS NOT NULL AND cleanup_lease_expires_at IS NOT NULL)
            )
            AND (lifecycle_state <> 'revocation_pending' OR revocation_requested_at IS NOT NULL)
        );

CREATE INDEX oauth_sessions_owner_epoch_idx
    ON oauth_sessions(account_did, auth_epoch, lifecycle_state, session_id);
CREATE INDEX oauth_sessions_revocation_claim_idx
    ON oauth_sessions(cleanup_next_attempt_at, cleanup_lease_expires_at, account_did, session_id)
    WHERE lifecycle_state = 'revocation_pending';
CREATE INDEX oauth_sessions_expiry_idx
    ON oauth_sessions(absolute_expires_at, account_did, session_id)
    WHERE lifecycle_state IN ('pending_handoff', 'active', 'deletion_only');

ALTER TABLE oauth_auth_requests
    ALTER COLUMN handoff_mode DROP DEFAULT,
    ADD COLUMN owner_did TEXT NOT NULL,
    ADD COLUMN owner_generation BIGINT NOT NULL,
    ADD COLUMN auth_epoch BIGINT NOT NULL,
    ADD COLUMN request_uri TEXT NOT NULL,
    ADD COLUMN request_state TEXT NOT NULL DEFAULT 'ready',
    ADD COLUMN exchange_attempt_id UUID,
    ADD COLUMN exchange_started_at TIMESTAMPTZ,
    ADD COLUMN exchange_finished_at TIMESTAMPTZ,
    ADD COLUMN consumed_at TIMESTAMPTZ,
    ADD CONSTRAINT oauth_auth_requests_owner_fkey
        FOREIGN KEY (owner_did) REFERENCES owner_lifecycles(owner_did),
    ADD CONSTRAINT oauth_auth_requests_authority_check
        CHECK (owner_generation > 0 AND auth_epoch > 0),
    ADD CONSTRAINT oauth_auth_requests_request_uri_check
        CHECK (btrim(request_uri) <> ''),
    ADD CONSTRAINT oauth_auth_requests_handoff_check
        CHECK (
            handoff_mode IN ('verified_link', 'loopback')
            AND btrim(device_id) <> ''
            AND (
                (handoff_mode = 'verified_link' AND loopback_redirect_uri IS NULL)
                OR
                (handoff_mode = 'loopback' AND loopback_redirect_uri IS NOT NULL)
            )
        ),
    ADD CONSTRAINT oauth_auth_requests_state_check
        CHECK (request_state IN ('ready', 'exchange_started', 'exchange_failed', 'exchange_ambiguous', 'consumed', 'revoked')),
    ADD CONSTRAINT oauth_auth_requests_attempt_shape_check
        CHECK (
            (request_state = 'ready'
                AND exchange_attempt_id IS NULL
                AND exchange_started_at IS NULL
                AND exchange_finished_at IS NULL
                AND consumed_at IS NULL)
            OR
            (request_state = 'exchange_started'
                AND exchange_attempt_id IS NOT NULL
                AND exchange_started_at IS NOT NULL
                AND exchange_finished_at IS NULL
                AND consumed_at IS NOT NULL)
            OR
            (request_state IN ('exchange_failed', 'exchange_ambiguous')
                AND exchange_attempt_id IS NOT NULL
                AND exchange_started_at IS NOT NULL
                AND exchange_finished_at IS NOT NULL
                AND consumed_at IS NOT NULL)
            OR
            (request_state IN ('consumed', 'revoked') AND consumed_at IS NOT NULL)
        );

CREATE UNIQUE INDEX oauth_auth_requests_request_uri_idx
    ON oauth_auth_requests(request_uri);
CREATE INDEX oauth_auth_requests_owner_epoch_idx
    ON oauth_auth_requests(owner_did, auth_epoch, request_state, created_at);
CREATE INDEX oauth_auth_requests_state_created_idx
    ON oauth_auth_requests(request_state, created_at, state);
CREATE INDEX oauth_auth_requests_ambiguous_idx
    ON oauth_auth_requests(exchange_finished_at, owner_did)
    WHERE request_state = 'exchange_ambiguous';

ALTER TABLE craftsky_sessions
    ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'pending_confirmation',
    ADD COLUMN auth_epoch BIGINT NOT NULL DEFAULT 1,
    ADD COLUMN idle_expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '30 days'),
    ADD COLUMN last_device_seen_at TIMESTAMPTZ,
    ADD CONSTRAINT craftsky_sessions_lifecycle_check
        CHECK (lifecycle_state IN ('pending_confirmation', 'active', 'revoked')),
    ADD CONSTRAINT craftsky_sessions_authority_check
        CHECK (auth_epoch > 0),
    ADD CONSTRAINT craftsky_sessions_expiry_check
        CHECK (idle_expires_at >= last_seen_at),
    ADD CONSTRAINT craftsky_sessions_revocation_shape_check
        CHECK (
            (lifecycle_state = 'revoked' AND revoked_at IS NOT NULL)
            OR (lifecycle_state <> 'revoked' AND revoked_at IS NULL)
        );

CREATE INDEX craftsky_sessions_parent_epoch_idx
    ON craftsky_sessions(account_did, oauth_session_id, auth_epoch, lifecycle_state);
CREATE INDEX craftsky_sessions_expiry_idx
    ON craftsky_sessions(idle_expires_at, account_did, oauth_session_id)
    WHERE lifecycle_state IN ('pending_confirmation', 'active');

CREATE TABLE oauth_handoff_exchanges (
    id                UUID        NOT NULL PRIMARY KEY,
    code_hash         BYTEA       UNIQUE,
    owner_did         TEXT        NOT NULL,
    owner_generation  BIGINT      NOT NULL CHECK (owner_generation > 0),
    auth_epoch        BIGINT      NOT NULL CHECK (auth_epoch > 0),
    oauth_session_id  TEXT        NOT NULL,
    device_id         TEXT        NOT NULL CHECK (btrim(device_id) <> ''),
    canonical_handle  TEXT        NOT NULL CHECK (btrim(canonical_handle) <> ''),
    state             TEXT        NOT NULL CHECK (state IN ('ready', 'redeemed', 'confirmed', 'expired', 'revoked')),
    expires_at        TIMESTAMPTZ NOT NULL,
    consumed_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT oauth_handoff_exchanges_parent_fkey
        FOREIGN KEY (owner_did, oauth_session_id)
        REFERENCES oauth_sessions(account_did, session_id) ON DELETE CASCADE,
    CONSTRAINT oauth_handoff_exchanges_code_hash_check
        CHECK (code_hash IS NULL OR octet_length(code_hash) = 32),
    CONSTRAINT oauth_handoff_exchanges_time_check
        CHECK (expires_at > created_at),
    CONSTRAINT oauth_handoff_exchanges_state_shape_check
        CHECK (
            (state IN ('ready', 'redeemed') AND code_hash IS NOT NULL AND consumed_at IS NULL)
            OR (state = 'confirmed' AND code_hash IS NULL AND consumed_at IS NOT NULL)
            OR (state IN ('expired', 'revoked') AND consumed_at IS NOT NULL)
        )
);

CREATE INDEX oauth_handoff_exchanges_owner_epoch_idx
    ON oauth_handoff_exchanges(owner_did, auth_epoch, state, id);
CREATE INDEX oauth_handoff_exchanges_expiry_idx
    ON oauth_handoff_exchanges(expires_at, id)
    WHERE state IN ('ready', 'redeemed');

CREATE TABLE oauth_handoff_receipts (
    id                UUID        NOT NULL PRIMARY KEY,
    exchange_id       UUID        NOT NULL UNIQUE,
    child_token_hash  BYTEA       NOT NULL UNIQUE,
    ciphertext        BYTEA,
    nonce             BYTEA,
    key_version       INTEGER     NOT NULL CHECK (key_version > 0),
    state             TEXT        NOT NULL CHECK (state IN ('pending', 'confirmed', 'expired', 'revoked')),
    confirm_by        TIMESTAMPTZ NOT NULL,
    consumed_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT oauth_handoff_receipts_exchange_fkey
        FOREIGN KEY (exchange_id) REFERENCES oauth_handoff_exchanges(id) ON DELETE CASCADE,
    CONSTRAINT oauth_handoff_receipts_child_fkey
        FOREIGN KEY (child_token_hash) REFERENCES craftsky_sessions(token_hash) ON DELETE CASCADE,
    CONSTRAINT oauth_handoff_receipts_time_check
        CHECK (confirm_by > created_at),
    CONSTRAINT oauth_handoff_receipts_secret_shape_check
        CHECK (
            (state = 'pending' AND ciphertext IS NOT NULL AND nonce IS NOT NULL AND consumed_at IS NULL)
            OR
            (state IN ('confirmed', 'expired', 'revoked') AND ciphertext IS NULL AND nonce IS NULL AND consumed_at IS NOT NULL)
        )
);

CREATE INDEX oauth_handoff_receipts_expiry_idx
    ON oauth_handoff_receipts(confirm_by, id)
    WHERE state = 'pending';

CREATE TABLE auth_auxiliary_cleanup_jobs (
    id                UUID        NOT NULL PRIMARY KEY,
    owner_did         TEXT        NOT NULL,
    auth_epoch        BIGINT      NOT NULL CHECK (auth_epoch > 0),
    kind              TEXT        NOT NULL CHECK (kind IN ('installation_push', 'account_push')),
    installation_id   TEXT,
    state             TEXT        NOT NULL CHECK (state IN ('pending', 'leased', 'complete', 'exhausted')),
    attempt_count     INTEGER     NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at   TIMESTAMPTZ NOT NULL,
    lease_token       UUID,
    lease_expires_at  TIMESTAMPTZ,
    last_category     TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT auth_auxiliary_cleanup_jobs_owner_fkey
        FOREIGN KEY (owner_did) REFERENCES owner_lifecycles(owner_did) ON DELETE CASCADE,
    CONSTRAINT auth_auxiliary_cleanup_jobs_target_check
        CHECK (
            (kind = 'installation_push' AND btrim(installation_id) <> '')
            OR (kind = 'account_push' AND installation_id IS NULL)
        ),
    CONSTRAINT auth_auxiliary_cleanup_jobs_lease_check
        CHECK (
            (state = 'leased' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL)
            OR (state <> 'leased' AND lease_token IS NULL AND lease_expires_at IS NULL)
        )
);

CREATE INDEX auth_auxiliary_cleanup_claim_idx
    ON auth_auxiliary_cleanup_jobs(next_attempt_at, lease_expires_at, id)
    WHERE state IN ('pending', 'leased');
