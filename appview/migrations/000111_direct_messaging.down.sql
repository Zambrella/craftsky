-- Consent requests cannot survive removal of their binding state.
DELETE FROM oauth_auth_requests WHERE purpose='chat';
ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT oauth_auth_requests_purpose_check,
    DROP CONSTRAINT oauth_auth_requests_authority_check,
    DROP CONSTRAINT oauth_auth_requests_account_deletion_metadata_check,
    DROP CONSTRAINT oauth_auth_requests_state_check,
    DROP CONSTRAINT oauth_auth_requests_callback_authority_check,
    DROP CONSTRAINT oauth_auth_requests_chat_binding_check;
ALTER TABLE oauth_auth_requests
    DROP COLUMN chat_parent_session_id,
    DROP COLUMN chat_child_token_hash,
    DROP COLUMN chat_generation,
    DROP COLUMN chat_push_enabled;
ALTER TABLE oauth_auth_requests ADD CONSTRAINT oauth_auth_requests_purpose_check
        CHECK (purpose IN ('login', 'accountDeletion', 'registration'));
ALTER TABLE oauth_auth_requests ADD CONSTRAINT oauth_auth_requests_authority_check
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
        );
ALTER TABLE oauth_auth_requests ADD CONSTRAINT oauth_auth_requests_account_deletion_metadata_check
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
ALTER TABLE oauth_auth_requests ADD CONSTRAINT oauth_auth_requests_state_check
        CHECK (
            request_state IN (
                'ready', 'exchange_started', 'exchange_failed', 'exchange_ambiguous',
                'cleanup_pending', 'consumed', 'revoked'
            )
            AND (request_state <> 'cleanup_pending' OR purpose IN ('login', 'registration'))
        );
ALTER TABLE oauth_auth_requests ADD CONSTRAINT oauth_auth_requests_callback_authority_check
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
DROP TRIGGER clear_revoked_chat_view_activity ON craftsky_sessions;
DROP FUNCTION clear_revoked_chat_view_activity();
DROP TABLE chat_push_deliveries;
DROP TABLE chat_notifications;
DROP TABLE chat_view_activity;
DROP TABLE chat_message_dedupe;
DROP TABLE chat_conversation_state;
DROP TABLE chat_sync_state;
DROP TABLE chat_continuations;
DROP TABLE chat_operations;
DROP TABLE chat_accounts;
