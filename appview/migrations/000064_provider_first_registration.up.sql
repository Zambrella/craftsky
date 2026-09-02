DROP INDEX oauth_auth_requests_owner_epoch_idx;

ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT oauth_auth_requests_purpose_check,
    DROP CONSTRAINT oauth_auth_requests_account_deletion_metadata_check,
    DROP CONSTRAINT oauth_auth_requests_authority_check,
    DROP CONSTRAINT oauth_auth_requests_state_check,
    DROP CONSTRAINT oauth_auth_requests_attempt_shape_check,
    ALTER COLUMN owner_did DROP NOT NULL,
    ALTER COLUMN owner_generation DROP NOT NULL,
    ALTER COLUMN auth_epoch DROP NOT NULL,
    ADD COLUMN registration_provider_origin TEXT,
    ADD COLUMN registration_issuer TEXT,
    ADD CONSTRAINT oauth_auth_requests_purpose_check
        CHECK (purpose IN ('login', 'accountDeletion', 'registration')),
    ADD CONSTRAINT oauth_auth_requests_state_check
        CHECK (
            request_state IN (
                'ready', 'exchange_started', 'exchange_failed', 'exchange_ambiguous',
                'cleanup_pending', 'consumed', 'revoked'
            )
            AND (request_state <> 'cleanup_pending' OR purpose = 'registration')
        ),
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
            (request_state IN ('exchange_failed', 'exchange_ambiguous', 'cleanup_pending')
                AND exchange_attempt_id IS NOT NULL
                AND exchange_started_at IS NOT NULL
                AND exchange_finished_at IS NOT NULL
                AND consumed_at IS NOT NULL)
            OR
            (request_state IN ('consumed', 'revoked') AND consumed_at IS NOT NULL)
        ),
    ADD CONSTRAINT oauth_auth_requests_authority_check
        CHECK (
            (
                purpose = 'registration'
                AND owner_did IS NULL
                AND owner_generation IS NULL
                AND auth_epoch IS NULL
            )
            OR
            (
                (purpose IN ('login', 'accountDeletion') OR request_state <> 'ready')
                AND owner_did IS NOT NULL
                AND owner_generation IS NOT NULL
                AND auth_epoch IS NOT NULL
                AND owner_generation > 0
                AND auth_epoch > 0
            )
        ),
    ADD CONSTRAINT oauth_auth_requests_account_deletion_metadata_check
        CHECK (
            (
                purpose = 'login'
                AND owner_did IS NOT NULL
                AND account_deletion_owner_did IS NULL
                AND account_deletion_job_id IS NULL
                AND registration_provider_origin IS NULL
                AND registration_issuer IS NULL
            )
            OR
            (
                purpose = 'accountDeletion'
                AND owner_did IS NOT NULL
                AND account_deletion_owner_did IS NOT NULL
                AND account_deletion_job_id IS NOT NULL
                AND registration_provider_origin IS NULL
                AND registration_issuer IS NULL
            )
            OR
            (
                purpose = 'registration'
                AND account_deletion_owner_did IS NULL
                AND account_deletion_job_id IS NULL
                AND registration_provider_origin IS NOT NULL
                AND registration_issuer IS NOT NULL
                AND btrim(registration_provider_origin) <> ''
                AND btrim(registration_issuer) <> ''
            )
        );

CREATE INDEX oauth_auth_requests_owner_epoch_idx
    ON oauth_auth_requests(owner_did, auth_epoch, request_state, created_at)
    WHERE owner_did IS NOT NULL;

CREATE TABLE oauth_auth_request_reservations (
    id         UUID        NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT oauth_auth_request_reservations_expiry_check
        CHECK (expires_at > created_at)
);

CREATE INDEX oauth_auth_request_reservations_expiry_idx
    ON oauth_auth_request_reservations(expires_at, id);

CREATE TABLE oauth_unverified_credentials (
    request_state          TEXT        NOT NULL PRIMARY KEY,
    data                   JSONB       NOT NULL,
    status                 TEXT        NOT NULL,
    eligible_at            TIMESTAMPTZ NOT NULL,
    expires_at             TIMESTAMPTZ NOT NULL,
    cleanup_attempts       INTEGER     NOT NULL DEFAULT 0,
    cleanup_next_attempt_at TIMESTAMPTZ,
    cleanup_lease_token    UUID,
    cleanup_lease_expires_at TIMESTAMPTZ,
    cleanup_last_category  TEXT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT oauth_unverified_credentials_request_fkey
        FOREIGN KEY (request_state) REFERENCES oauth_auth_requests(state) ON DELETE CASCADE,
    CONSTRAINT oauth_unverified_credentials_status_check
        CHECK (status IN ('held', 'pending')),
    CONSTRAINT oauth_unverified_credentials_expiry_check
        CHECK (eligible_at <= expires_at AND expires_at > created_at),
    CONSTRAINT oauth_unverified_credentials_cleanup_check
        CHECK (
            cleanup_attempts >= 0
            AND (cleanup_last_category IS NULL OR btrim(cleanup_last_category) <> '')
            AND (
                (cleanup_lease_token IS NULL AND cleanup_lease_expires_at IS NULL)
                OR
                (cleanup_lease_token IS NOT NULL AND cleanup_lease_expires_at IS NOT NULL)
            )
        )
);

CREATE INDEX oauth_unverified_credentials_cleanup_idx
    ON oauth_unverified_credentials(
        status, eligible_at, cleanup_next_attempt_at, cleanup_lease_expires_at, request_state
    );
