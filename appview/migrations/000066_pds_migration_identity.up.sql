ALTER TABLE account_deletion_operations
    RENAME COLUMN confirmation_handle_hash TO confirmation_did_hash;

ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT oauth_auth_requests_state_check,
    ADD COLUMN resource_server_origin TEXT,
    ADD COLUMN authorization_server_issuer TEXT;

UPDATE oauth_auth_requests request
SET authorization_server_issuer = request.data->>'authserver_url',
    resource_server_origin = session.data->>'host_url'
FROM oauth_sessions session
WHERE request.purpose = 'login'
  AND session.session_id = request.state
  AND session.account_did = request.owner_did;

DELETE FROM oauth_auth_requests
WHERE purpose = 'login'
  AND (
      NULLIF(btrim(resource_server_origin), '') IS NULL
      OR NULLIF(btrim(authorization_server_issuer), '') IS NULL
  );

ALTER TABLE oauth_auth_requests
    ADD CONSTRAINT oauth_auth_requests_state_check
        CHECK (
            request_state IN (
                'ready', 'exchange_started', 'exchange_failed', 'exchange_ambiguous',
                'cleanup_pending', 'consumed', 'revoked'
            )
            AND (request_state <> 'cleanup_pending' OR purpose IN ('login', 'registration'))
        ),
    ADD CONSTRAINT oauth_auth_requests_callback_authority_check
        CHECK (
            (
                purpose = 'login'
                AND resource_server_origin IS NOT NULL
                AND authorization_server_issuer IS NOT NULL
                AND btrim(resource_server_origin) <> ''
                AND btrim(authorization_server_issuer) <> ''
            )
            OR
            (
                purpose <> 'login'
                AND resource_server_origin IS NULL
                AND authorization_server_issuer IS NULL
            )
        );
