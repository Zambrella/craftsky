DROP TABLE oauth_unverified_credentials;
DROP TABLE oauth_auth_request_reservations;

DELETE FROM oauth_auth_requests WHERE purpose = 'registration';

DROP INDEX oauth_auth_requests_owner_epoch_idx;

ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT oauth_auth_requests_account_deletion_metadata_check,
    DROP CONSTRAINT oauth_auth_requests_authority_check,
    DROP CONSTRAINT oauth_auth_requests_purpose_check,
    DROP CONSTRAINT oauth_auth_requests_attempt_shape_check,
    DROP CONSTRAINT oauth_auth_requests_state_check,
    DROP COLUMN registration_issuer,
    DROP COLUMN registration_provider_origin,
    ALTER COLUMN auth_epoch SET NOT NULL,
    ALTER COLUMN owner_generation SET NOT NULL,
    ALTER COLUMN owner_did SET NOT NULL,
    ADD CONSTRAINT oauth_auth_requests_purpose_check
        CHECK (purpose IN ('login', 'accountDeletion')),
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
        ),
    ADD CONSTRAINT oauth_auth_requests_authority_check
        CHECK (owner_generation > 0 AND auth_epoch > 0),
    ADD CONSTRAINT oauth_auth_requests_account_deletion_metadata_check
        CHECK (
            (purpose = 'login' AND account_deletion_owner_did IS NULL AND account_deletion_job_id IS NULL)
            OR
            (purpose = 'accountDeletion' AND account_deletion_owner_did IS NOT NULL AND account_deletion_job_id IS NOT NULL)
        );

CREATE INDEX oauth_auth_requests_owner_epoch_idx
    ON oauth_auth_requests(owner_did, auth_epoch, request_state, created_at);
