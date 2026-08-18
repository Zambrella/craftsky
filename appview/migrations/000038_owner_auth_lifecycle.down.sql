DROP TABLE IF EXISTS auth_auxiliary_cleanup_jobs;
DROP TABLE IF EXISTS oauth_handoff_receipts;
DROP TABLE IF EXISTS oauth_handoff_exchanges;

ALTER TABLE account_deletion_operations
    DROP CONSTRAINT IF EXISTS account_deletion_operations_credential_shape_check,
    DROP CONSTRAINT IF EXISTS account_deletion_operations_state_check,
    DROP COLUMN IF EXISTS deletion_credential_generation;

DELETE FROM account_deletion_operations WHERE state = 'reauth_required';

ALTER TABLE account_deletion_operations
    ADD CONSTRAINT account_deletion_operations_state_check
        CHECK (state IN ('intent', 'active', 'retrying'));

ALTER TABLE craftsky_sessions
    DROP CONSTRAINT IF EXISTS craftsky_sessions_revocation_shape_check,
    DROP CONSTRAINT IF EXISTS craftsky_sessions_expiry_check,
    DROP CONSTRAINT IF EXISTS craftsky_sessions_authority_check,
    DROP CONSTRAINT IF EXISTS craftsky_sessions_lifecycle_check,
    DROP COLUMN IF EXISTS last_device_seen_at,
    DROP COLUMN IF EXISTS idle_expires_at,
    DROP COLUMN IF EXISTS auth_epoch,
    DROP COLUMN IF EXISTS lifecycle_state;

ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_attempt_shape_check,
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_state_check,
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_handoff_check,
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_request_uri_check,
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_authority_check,
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_owner_fkey,
    DROP COLUMN IF EXISTS consumed_at,
    DROP COLUMN IF EXISTS exchange_finished_at,
    DROP COLUMN IF EXISTS exchange_started_at,
    DROP COLUMN IF EXISTS exchange_attempt_id,
    DROP COLUMN IF EXISTS request_state,
    DROP COLUMN IF EXISTS request_uri,
    DROP COLUMN IF EXISTS auth_epoch,
    DROP COLUMN IF EXISTS owner_generation,
    DROP COLUMN IF EXISTS owner_did,
    ALTER COLUMN handoff_mode SET DEFAULT 'deep_link';

ALTER TABLE oauth_sessions
    DROP CONSTRAINT IF EXISTS oauth_sessions_revocation_shape_check,
    DROP CONSTRAINT IF EXISTS oauth_sessions_deletion_shape_check,
    DROP CONSTRAINT IF EXISTS oauth_sessions_expiry_check,
    DROP CONSTRAINT IF EXISTS oauth_sessions_authority_check,
    DROP CONSTRAINT IF EXISTS oauth_sessions_lifecycle_check,
    DROP CONSTRAINT IF EXISTS oauth_sessions_owner_fkey,
    DROP COLUMN IF EXISTS cleanup_last_category,
    DROP COLUMN IF EXISTS cleanup_lease_expires_at,
    DROP COLUMN IF EXISTS cleanup_lease_token,
    DROP COLUMN IF EXISTS cleanup_next_attempt_at,
    DROP COLUMN IF EXISTS cleanup_attempts,
    DROP COLUMN IF EXISTS revocation_requested_at,
    DROP COLUMN IF EXISTS deletion_credential_generation,
    DROP COLUMN IF EXISTS deletion_operation_id,
    DROP COLUMN IF EXISTS absolute_expires_at,
    DROP COLUMN IF EXISTS row_version,
    DROP COLUMN IF EXISTS auth_epoch,
    DROP COLUMN IF EXISTS owner_generation,
    DROP COLUMN IF EXISTS lifecycle_state;

DROP TABLE IF EXISTS owner_lifecycles;
DROP FUNCTION IF EXISTS enforce_owner_lifecycle_monotonicity();
