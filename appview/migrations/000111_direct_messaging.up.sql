-- Private operational metadata only. No message, draft or report content.
-- Keep the disabled owner row until permanent CraftSky private-state cleanup;
-- generation fences stale consent and background work across disconnects.
CREATE TABLE chat_accounts (
    owner_did TEXT PRIMARY KEY REFERENCES owner_lifecycles(owner_did) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    generation BIGINT NOT NULL DEFAULT 1 CHECK (generation > 0),
    push_enabled BOOLEAN NOT NULL DEFAULT true
);

-- Optional consent is bound to the exact initiating child, device, owner
-- authority and messaging generation. It does not replace ordinary login.
ALTER TABLE oauth_auth_requests
    ADD COLUMN chat_parent_session_id TEXT,
    ADD COLUMN chat_child_token_hash BYTEA,
    ADD COLUMN chat_generation BIGINT,
    ADD COLUMN chat_push_enabled BOOLEAN,
    DROP CONSTRAINT oauth_auth_requests_purpose_check,
    DROP CONSTRAINT oauth_auth_requests_authority_check,
    DROP CONSTRAINT oauth_auth_requests_account_deletion_metadata_check,
    DROP CONSTRAINT oauth_auth_requests_state_check,
    DROP CONSTRAINT oauth_auth_requests_callback_authority_check,
    ADD CONSTRAINT oauth_auth_requests_purpose_check
        CHECK (purpose IN ('login','accountDeletion','registration','chat')),
    ADD CONSTRAINT oauth_auth_requests_authority_check CHECK (
        (purpose='registration' AND owner_did IS NULL AND owner_generation IS NULL AND auth_epoch IS NULL)
        OR ((purpose IN ('login','accountDeletion','chat') OR request_state<>'ready')
            AND owner_did IS NOT NULL AND owner_generation IS NOT NULL AND auth_epoch IS NOT NULL AND owner_generation>0 AND auth_epoch>0)),
    ADD CONSTRAINT oauth_auth_requests_account_deletion_metadata_check CHECK (
        (purpose IN ('login','chat') AND owner_did IS NOT NULL
            AND account_deletion_owner_did IS NULL AND account_deletion_job_id IS NULL
            AND registration_provider_origin IS NULL AND registration_issuer IS NULL)
        OR (purpose='accountDeletion' AND owner_did IS NOT NULL
            AND account_deletion_owner_did IS NOT NULL AND account_deletion_job_id IS NOT NULL
            AND registration_provider_origin IS NULL AND registration_issuer IS NULL)
        OR (purpose='registration' AND account_deletion_owner_did IS NULL AND account_deletion_job_id IS NULL
            AND registration_provider_origin IS NOT NULL AND btrim(registration_provider_origin)<>''
            AND registration_issuer IS NOT NULL AND btrim(registration_issuer)<>'')),
    ADD CONSTRAINT oauth_auth_requests_state_check CHECK (
        request_state IN ('ready','exchange_started','exchange_failed','exchange_ambiguous','cleanup_pending','consumed','revoked')
        AND (request_state<>'cleanup_pending' OR purpose IN ('login','registration','chat'))),
    ADD CONSTRAINT oauth_auth_requests_callback_authority_check CHECK (
        (purpose IN ('login','chat') AND resource_server_origin IS NOT NULL AND btrim(resource_server_origin)<>''
            AND authorization_server_issuer IS NOT NULL AND btrim(authorization_server_issuer)<>'')
        OR (purpose NOT IN ('login','chat') AND resource_server_origin IS NULL AND authorization_server_issuer IS NULL)),
    ADD CONSTRAINT oauth_auth_requests_chat_binding_check CHECK (
        (purpose='chat' AND chat_parent_session_id IS NOT NULL AND btrim(chat_parent_session_id)<>''
            AND chat_child_token_hash IS NOT NULL AND octet_length(chat_child_token_hash)=32
            AND chat_generation IS NOT NULL AND chat_generation>0)
        OR (purpose<>'chat' AND chat_parent_session_id IS NULL AND chat_child_token_hash IS NULL
            AND chat_generation IS NULL AND chat_push_enabled IS NULL));

-- Lifetime idempotency receipts contain only keyed request digests and bounded
-- operational outcomes. Message/report prose is never a database value.
CREATE TABLE chat_operations (
    owner_did TEXT NOT NULL REFERENCES chat_accounts(owner_did) ON DELETE CASCADE,
    operation_id UUID NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('send','report')),
    key_version INTEGER NOT NULL CHECK (key_version>0),
    request_digest BYTEA NOT NULL CHECK (octet_length(request_digest)=32),
    messaging_generation BIGINT NOT NULL DEFAULT 1 CHECK (messaging_generation>0),
    state TEXT NOT NULL CHECK (state IN ('prepared','attempting','confirmed','failed','uncertain')),
    upstream_id TEXT NOT NULL DEFAULT '',
    upstream_revision TEXT NOT NULL DEFAULT '',
    error_code TEXT NOT NULL DEFAULT '' CHECK (error_code IN ('','rejected','unavailable','uncertain','notSubmitted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(owner_did,operation_id)
);

CREATE TABLE chat_continuations (
    owner_did TEXT NOT NULL REFERENCES chat_accounts(owner_did) ON DELETE CASCADE,
    id UUID NOT NULL,
    scope TEXT NOT NULL CHECK (char_length(scope) BETWEEN 1 AND 1024),
    generation BIGINT NOT NULL CHECK (generation>0),
    upstream_cursor TEXT NOT NULL CHECK (octet_length(upstream_cursor) BETWEEN 1 AND 8192),
    message_ids TEXT[] NOT NULL DEFAULT '{}' CHECK (cardinality(message_ids)<=100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT now()+interval '30 minutes',
    PRIMARY KEY(owner_did,id)
);
CREATE INDEX chat_continuations_expiry_idx ON chat_continuations(expires_at);

CREATE TABLE chat_sync_state (
    owner_did TEXT PRIMARY KEY REFERENCES chat_accounts(owner_did) ON DELETE CASCADE,
    phase TEXT NOT NULL DEFAULT 'baselineHead' CHECK (phase IN ('baselineHead','baselineAccepted','baselineRequests','ready','recoveryHead','recoveryAccepted','recoveryRequests')),
    cursor TEXT NOT NULL DEFAULT '' CHECK (octet_length(cursor)<=8192),
    seed_cursor TEXT NOT NULL DEFAULT '' CHECK (octet_length(seed_cursor)<=8192),
    page_cursor TEXT NOT NULL DEFAULT '' CHECK (octet_length(page_cursor)<=8192),
    version BIGINT NOT NULL DEFAULT 0 CHECK (version>=0),
    recovery_id UUID,
    claim_token UUID,
    claim_expires_at TIMESTAMPTZ,
    next_due_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    failure_count INTEGER NOT NULL DEFAULT 0 CHECK (failure_count>=0),
    safe_state TEXT NOT NULL DEFAULT 'syncing' CHECK (safe_state IN ('syncing','ready','authorizationNeeded','restricted','unavailable','rateLimited')),
    last_success_at TIMESTAMPTZ,
    CHECK ((claim_token IS NULL)=(claim_expires_at IS NULL))
);
CREATE INDEX chat_sync_due_idx ON chat_sync_state(next_due_at,owner_did);

CREATE TABLE chat_view_activity (
    owner_did TEXT NOT NULL REFERENCES chat_accounts(owner_did) ON DELETE CASCADE,
    child_token_hash BYTEA NOT NULL REFERENCES craftsky_sessions(token_hash) ON DELETE CASCADE,
    oauth_session_id TEXT NOT NULL,
    view TEXT NOT NULL CHECK (view IN ('inbox','conversation')),
    conversation_id TEXT NOT NULL DEFAULT '' CHECK (octet_length(conversation_id)<=512),
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(owner_did,child_token_hash),
    UNIQUE(child_token_hash),
    CHECK ((view='conversation')=(conversation_id<>''))
);
CREATE INDEX chat_view_activity_expiry_idx ON chat_view_activity(expires_at);

-- One activity row per child makes this cleanup bounded on every existing
-- logout/revocation path, without coupling those paths to optional chat code.
CREATE FUNCTION clear_revoked_chat_view_activity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM chat_view_activity WHERE child_token_hash=NEW.token_hash;
    RETURN NEW;
END;
$$;
CREATE TRIGGER clear_revoked_chat_view_activity
AFTER UPDATE OF lifecycle_state ON craftsky_sessions
FOR EACH ROW WHEN (NEW.lifecycle_state<>'active')
EXECUTE FUNCTION clear_revoked_chat_view_activity();

CREATE TABLE chat_conversation_state (
    owner_did TEXT NOT NULL REFERENCES chat_accounts(owner_did) ON DELETE CASCADE,
    conversation_id TEXT NOT NULL CHECK (octet_length(conversation_id) BETWEEN 1 AND 512),
    status TEXT NOT NULL CHECK (status IN ('accepted','request')),
    request_generation BIGINT NOT NULL DEFAULT 1 CHECK (request_generation>0),
    request_notified BOOLEAN NOT NULL DEFAULT false,
    left_by_owner BOOLEAN NOT NULL DEFAULT false,
    last_message_id TEXT NOT NULL DEFAULT '' CHECK (octet_length(last_message_id)<=512),
    PRIMARY KEY(owner_did,conversation_id)
);

CREATE TABLE chat_message_dedupe (
    owner_did TEXT NOT NULL REFERENCES chat_accounts(owner_did) ON DELETE CASCADE,
    conversation_id TEXT NOT NULL CHECK (octet_length(conversation_id) BETWEEN 1 AND 512),
    message_id TEXT NOT NULL CHECK (octet_length(message_id) BETWEEN 1 AND 512),
    disposition TEXT NOT NULL CHECK (disposition IN ('baseline','notified','suppressed','deleted')),
    observed_revision TEXT NOT NULL CHECK (octet_length(observed_revision)<=512),
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(owner_did,conversation_id,message_id)
);

CREATE TABLE chat_notifications (
    owner_did TEXT NOT NULL REFERENCES chat_accounts(owner_did) ON DELETE CASCADE,
    id UUID NOT NULL,
    logical_key TEXT NOT NULL CHECK (octet_length(logical_key) BETWEEN 1 AND 2048),
    kind TEXT NOT NULL CHECK (kind IN ('normal','request','catchUp')),
    conversation_id TEXT NOT NULL CHECK (octet_length(conversation_id)<=512),
    message_id TEXT NOT NULL DEFAULT '' CHECK (octet_length(message_id)<=512),
    messaging_generation BIGINT NOT NULL CHECK (messaging_generation>0),
    open_reference UUID,
    state TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','complete','cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    terminal_at TIMESTAMPTZ,
    PRIMARY KEY(owner_did,id),
    UNIQUE(owner_did,logical_key),
    UNIQUE(owner_did,open_reference),
    CHECK (state<>'pending' OR (conversation_id<>'' AND open_reference IS NOT NULL))
);

CREATE TABLE chat_push_deliveries (
    owner_did TEXT NOT NULL REFERENCES chat_accounts(owner_did) ON DELETE CASCADE,
    id UUID NOT NULL,
    notification_id UUID NOT NULL,
    account_subscription_id UUID NOT NULL REFERENCES push_account_subscriptions(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','leased','retry','succeeded','permanent_failure','expired','cancelled')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts>=0),
    lease_token UUID,
    lease_expires_at TIMESTAMPTZ,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deadline_at TIMESTAMPTZ NOT NULL,
    provider_result_class TEXT NOT NULL DEFAULT '' CHECK (provider_result_class IN ('','success','retryable','invalidToken','permanentFailure','cancelled','expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    terminal_at TIMESTAMPTZ,
    PRIMARY KEY(owner_did,id),
    UNIQUE(owner_did,notification_id,account_subscription_id),
    FOREIGN KEY(owner_did,notification_id) REFERENCES chat_notifications(owner_did,id) ON DELETE CASCADE,
    CHECK ((lease_token IS NULL)=(lease_expires_at IS NULL))
);
CREATE INDEX chat_push_due_idx ON chat_push_deliveries(next_attempt_at,owner_did,id) WHERE status IN ('pending','retry','leased');
CREATE INDEX chat_push_subscription_idx ON chat_push_deliveries(account_subscription_id,owner_did,id);
CREATE INDEX chat_push_deadline_idx ON chat_push_deliveries(deadline_at,owner_did,id) WHERE status IN('pending','retry','leased');
CREATE INDEX chat_push_terminal_idx ON chat_push_deliveries(terminal_at,owner_did,id) WHERE terminal_at IS NOT NULL;
CREATE INDEX chat_notifications_pending_idx ON chat_notifications(created_at,owner_did,id) WHERE state='pending';
CREATE INDEX chat_notifications_retention_idx ON chat_notifications(terminal_at,owner_did,id)
WHERE state<>'pending' AND (conversation_id<>'' OR message_id<>'' OR open_reference IS NOT NULL OR logical_key NOT LIKE 'sha256:%');
