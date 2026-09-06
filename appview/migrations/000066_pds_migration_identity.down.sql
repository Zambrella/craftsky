ALTER TABLE account_deletion_operations
    RENAME COLUMN confirmation_did_hash TO confirmation_handle_hash;

DELETE FROM oauth_auth_requests
WHERE purpose = 'login' AND request_state = 'cleanup_pending';

ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT oauth_auth_requests_callback_authority_check,
    DROP CONSTRAINT oauth_auth_requests_state_check,
    DROP COLUMN authorization_server_issuer,
    DROP COLUMN resource_server_origin,
    ADD CONSTRAINT oauth_auth_requests_state_check
        CHECK (
            request_state IN (
                'ready', 'exchange_started', 'exchange_failed', 'exchange_ambiguous',
                'cleanup_pending', 'consumed', 'revoked'
            )
            AND (request_state <> 'cleanup_pending' OR purpose = 'registration')
        );
