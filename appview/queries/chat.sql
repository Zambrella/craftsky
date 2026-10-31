-- name: EnsureChatAccount :exec
INSERT INTO chat_accounts (owner_did) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: ChatProfileDIDs :many
SELECT profile.did FROM craftsky_profiles profile
LEFT JOIN owner_lifecycles owner ON owner.owner_did=profile.did
WHERE profile.did=ANY($1::text[]) AND (owner.state IS NULL OR owner.state='active');

-- name: ChatConnection :one
SELECT owner_did, enabled, generation, push_enabled
FROM chat_accounts WHERE owner_did=$1;

-- name: ConnectChat :one
UPDATE chat_accounts
SET enabled=true, generation=generation+1,
    push_enabled=COALESCE(sqlc.narg(push_enabled)::boolean,push_enabled)
WHERE owner_did=sqlc.arg(owner_did) AND generation=sqlc.arg(expected_generation)
RETURNING owner_did, enabled, generation, push_enabled;

-- name: DisconnectChat :one
WITH removed_continuations AS (DELETE FROM chat_continuations WHERE chat_continuations.owner_did=$1),
 removed_activity AS (DELETE FROM chat_view_activity WHERE chat_view_activity.owner_did=$1),
 cancelled_notifications AS (UPDATE chat_notifications SET state='cancelled',terminal_at=now() WHERE chat_notifications.owner_did=$1 AND state='pending'),
 cancelled_deliveries AS (UPDATE chat_push_deliveries SET status='cancelled',provider_result_class='cancelled',lease_token=NULL,lease_expires_at=NULL,terminal_at=now() WHERE chat_push_deliveries.owner_did=$1 AND status IN('pending','retry','leased')),
 released_sync AS (
  UPDATE chat_sync_state SET claim_token=NULL,claim_expires_at=NULL,next_due_at=now(),
  recovery_id=CASE WHEN phase='ready' THEN coalesce(recovery_id,gen_random_uuid()) ELSE recovery_id END
  WHERE chat_sync_state.owner_did=$1
 )
UPDATE chat_accounts SET enabled=false,generation=chat_accounts.generation+1
WHERE chat_accounts.owner_did=$1
RETURNING chat_accounts.owner_did, chat_accounts.enabled, chat_accounts.generation, chat_accounts.push_enabled;

-- name: PrepareChatOperation :execrows
INSERT INTO chat_operations(owner_did,operation_id,kind,key_version,request_digest,messaging_generation,state)
VALUES($1,$2,$3,$4,$5,$6,'prepared') ON CONFLICT DO NOTHING;

-- name: GetChatOperation :one
SELECT * FROM chat_operations WHERE owner_did=$1 AND operation_id=$2 FOR UPDATE;

-- name: AttemptChatOperation :execrows
UPDATE chat_operations SET state='attempting',updated_at=now()
WHERE owner_did=$1 AND operation_id=$2 AND state='prepared' AND created_at >= now()-interval '30 seconds';

-- name: FinishChatOperation :execrows
UPDATE chat_operations SET state=$3,upstream_id=$4,upstream_revision=$5,error_code=$6,updated_at=now()
WHERE owner_did=$1 AND operation_id=$2 AND state='attempting';

-- name: ExpirePreparedChatOperation :execrows
UPDATE chat_operations SET state='failed',error_code='notSubmitted',updated_at=now()
WHERE owner_did=$1 AND operation_id=$2 AND state='prepared'
  AND created_at < now()-interval '30 seconds';

-- name: ChatBlocked :one
WITH latest_commands AS (
    SELECT DISTINCT ON (command.owner_did,command.expected_target_did)
        command.operation_kind
    FROM pds_commands command
    JOIN owner_lifecycles owner ON owner.owner_did=command.owner_did
    WHERE command.state='accepted'
      AND command.operation_kind IN ('profile.block','profile.unblock')
      AND owner.state='active' AND owner.generation=command.owner_generation
      AND ((command.owner_did=sqlc.arg(viewer)::text AND command.expected_target_did=sqlc.arg(peer)::text)
        OR (command.owner_did=sqlc.arg(peer)::text AND command.expected_target_did=sqlc.arg(viewer)::text))
    ORDER BY command.owner_did,command.expected_target_did,command.terminal_at DESC,command.id DESC
)
SELECT EXISTS (
    SELECT 1 FROM pds_set_aggregates block WHERE block.kind='block'
      AND ((block.actor_did=sqlc.arg(viewer)::text AND block.subject_did=sqlc.arg(peer)::text)
        OR (block.actor_did=sqlc.arg(peer)::text AND block.subject_did=sqlc.arg(viewer)::text))
) OR EXISTS(SELECT 1 FROM latest_commands WHERE operation_kind='profile.block') AS blocked;

-- name: ChatLiveChild :one
SELECT EXISTS (
    SELECT 1 FROM craftsky_sessions child
    JOIN oauth_sessions parent ON parent.account_did=child.account_did AND parent.session_id=child.oauth_session_id
    JOIN owner_lifecycles owner ON owner.owner_did=child.account_did
    WHERE child.account_did=$1 AND child.oauth_session_id=$2 AND child.token_hash=$3
      AND child.lifecycle_state='active' AND child.idle_expires_at>now()
      AND parent.lifecycle_state='active' AND parent.absolute_expires_at>now()
      AND owner.state='active' AND owner.auth_epoch=child.auth_epoch
      AND owner.auth_epoch=parent.auth_epoch AND owner.generation=parent.owner_generation
) AS live;

-- name: LockChatAccount :one
SELECT * FROM chat_accounts WHERE owner_did=$1 FOR UPDATE;

-- name: GetChatContinuation :one
SELECT upstream_cursor FROM chat_continuations
WHERE owner_did=$1 AND id=$2 AND scope=$3 AND generation=$4 AND expires_at>now();

-- name: DeleteExpiredChatContinuations :exec
DELETE FROM chat_continuations WHERE owner_did=$1 AND expires_at<=now();

-- name: TrimChatContinuations :exec
WITH expired_budget AS (
    SELECT id FROM chat_continuations WHERE owner_did=$1 ORDER BY created_at DESC,id DESC OFFSET 255
)
DELETE FROM chat_continuations WHERE chat_continuations.owner_did=$1 AND chat_continuations.id IN (SELECT id FROM expired_budget);

-- name: CreateChatContinuation :exec
INSERT INTO chat_continuations(owner_did,id,scope,generation,upstream_cursor) VALUES($1,$2,$3,$4,$5);

-- name: RecordChatPageMessages :exec
UPDATE chat_continuations SET message_ids=$5
WHERE owner_did=$1 AND id=$2 AND scope=$3 AND generation=$4 AND expires_at>now();

-- name: FindChatMessagePage :one
SELECT upstream_cursor FROM chat_continuations
WHERE owner_did=$1 AND scope=$2 AND generation=$3 AND $4::text=ANY(message_ids) AND expires_at>now()
ORDER BY created_at DESC LIMIT 1;

-- name: SetChatPushPreference :exec
WITH cancelled_notifications AS (
 UPDATE chat_notifications notification SET state='cancelled',terminal_at=now()
 WHERE notification.owner_did=$1 AND NOT $2::boolean AND notification.state='pending'
), cancelled_deliveries AS (
 UPDATE chat_push_deliveries delivery SET status='cancelled',provider_result_class='cancelled',
 lease_token=NULL,lease_expires_at=NULL,terminal_at=now()
 WHERE delivery.owner_did=$1 AND NOT $2::boolean AND delivery.status IN('pending','retry','leased')
)
UPDATE chat_accounts SET push_enabled=$2 WHERE chat_accounts.owner_did=$1;

-- name: ChatLiveBackgroundParent :one
SELECT EXISTS (
 SELECT 1 FROM craftsky_sessions child
 JOIN oauth_sessions parent ON parent.account_did=child.account_did AND parent.session_id=child.oauth_session_id
 JOIN owner_lifecycles owner ON owner.owner_did=child.account_did
 WHERE child.account_did=$1 AND child.oauth_session_id=$2
 AND child.lifecycle_state='active' AND child.idle_expires_at>now()
 AND parent.lifecycle_state='active' AND parent.absolute_expires_at>now()
 AND owner.state='active' AND owner.auth_epoch=child.auth_epoch
 AND owner.auth_epoch=parent.auth_epoch AND owner.generation=parent.owner_generation
) AS live;

-- name: EnsureChatSync :exec
INSERT INTO chat_sync_state(owner_did) VALUES($1) ON CONFLICT DO NOTHING;

-- name: ClaimChatSync :one
UPDATE chat_sync_state SET claim_token=$2,claim_expires_at=$3
WHERE chat_sync_state.owner_did=$1 AND chat_sync_state.next_due_at<=$4 AND (chat_sync_state.claim_expires_at IS NULL OR chat_sync_state.claim_expires_at<=$4)
 AND (SELECT count(*) FROM chat_sync_state capacity WHERE capacity.claim_expires_at>$4)<2
RETURNING chat_sync_state.*;

-- name: LockChatPollCapacity :exec
SELECT pg_advisory_xact_lock(hashtextextended('craftsky.chat.poll.capacity.v1',0));

-- name: FinishChatSync :execrows
UPDATE chat_sync_state SET phase=$4,cursor=$5,seed_cursor=$6,page_cursor=$7,
 version=version+1,claim_token=NULL,claim_expires_at=NULL,next_due_at=$8,
 safe_state=CASE WHEN $4='ready' THEN 'ready' ELSE 'syncing' END,
 failure_count=0,last_success_at=$9,recovery_id=$10
WHERE owner_did=$1 AND claim_token=$2 AND version=$3;

-- name: FailChatSync :exec
UPDATE chat_sync_state SET claim_token=NULL,claim_expires_at=NULL,
 next_due_at=$3,safe_state=$4,failure_count=failure_count+1,
 recovery_id=CASE WHEN phase='ready' THEN coalesce(recovery_id,gen_random_uuid()) ELSE recovery_id END
WHERE owner_did=$1 AND claim_token=$2;

-- name: RecordChatConversation :one
INSERT INTO chat_conversation_state(owner_did,conversation_id,status,last_message_id,request_notified)
VALUES($1,$2,$3,$4,$5)
ON CONFLICT(owner_did,conversation_id) DO UPDATE SET
 status=excluded.status,last_message_id=excluded.last_message_id,
 request_generation=CASE WHEN chat_conversation_state.status='accepted' AND excluded.status='request' THEN chat_conversation_state.request_generation+1 ELSE chat_conversation_state.request_generation END,
 request_notified=CASE WHEN chat_conversation_state.status='accepted' AND excluded.status='request' THEN excluded.request_notified ELSE chat_conversation_state.request_notified END
RETURNING *;

-- name: RecordChatMessage :execrows
INSERT INTO chat_message_dedupe(owner_did,conversation_id,message_id,disposition,observed_revision)
VALUES($1,$2,$3,$4,$5) ON CONFLICT(owner_did,conversation_id,message_id)
DO UPDATE SET disposition='deleted',observed_revision=greatest(chat_message_dedupe.observed_revision,excluded.observed_revision)
WHERE excluded.disposition='deleted';

-- name: CancelChatMessageNotifications :exec
WITH cancelled AS (
 UPDATE chat_notifications notification SET state='cancelled',terminal_at=now()
 WHERE notification.owner_did=$1 AND notification.conversation_id=$2 AND notification.message_id=$3 AND notification.state='pending'
 RETURNING notification.owner_did,notification.id
)
UPDATE chat_push_deliveries delivery SET status='cancelled',provider_result_class='cancelled',
 lease_token=NULL,lease_expires_at=NULL,terminal_at=now()
FROM cancelled WHERE delivery.owner_did=cancelled.owner_did AND delivery.notification_id=cancelled.id
 AND delivery.status IN('pending','retry','leased');

-- name: InsertChatNotification :exec
WITH notification AS (
INSERT INTO chat_notifications(owner_did,id,logical_key,kind,conversation_id,message_id,messaging_generation,open_reference)
VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(owner_did,logical_key)
DO UPDATE SET message_id=excluded.message_id WHERE chat_notifications.state='pending'
RETURNING owner_did,id,created_at
)
INSERT INTO chat_push_deliveries(owner_did,id,notification_id,account_subscription_id,deadline_at)
SELECT notification.owner_did,gen_random_uuid(),notification.id,subscription.id,notification.created_at+interval '24 hours'
FROM notification
JOIN push_account_subscriptions subscription ON subscription.account_did=notification.owner_did
JOIN push_installations installation ON installation.id=subscription.installation_id
WHERE subscription.active AND installation.active
ON CONFLICT(owner_did,notification_id,account_subscription_id) DO NOTHING;

-- name: MarkChatRequestNotified :exec
UPDATE chat_conversation_state SET request_notified=true WHERE owner_did=$1 AND conversation_id=$2;

-- name: MarkChatConversationLeft :exec
INSERT INTO chat_conversation_state(owner_did,conversation_id,status,request_notified,left_by_owner)
VALUES($1,$2,'request',true,true)
ON CONFLICT(owner_did,conversation_id) DO UPDATE SET left_by_owner=true;

-- name: CancelChatConversationNotifications :exec
WITH cancelled AS (
 UPDATE chat_notifications notification SET state='cancelled',terminal_at=now()
 WHERE notification.owner_did=$1 AND notification.conversation_id=$2 AND notification.state='pending'
 RETURNING notification.owner_did,notification.id
)
UPDATE chat_push_deliveries delivery SET status='cancelled',provider_result_class='cancelled',
 lease_token=NULL,lease_expires_at=NULL,terminal_at=now()
FROM cancelled WHERE delivery.owner_did=cancelled.owner_did AND delivery.notification_id=cancelled.id
 AND delivery.status IN('pending','retry','leased');

-- name: MarkChatConversationAccepted :exec
UPDATE chat_conversation_state SET left_by_owner=false
WHERE owner_did=$1 AND conversation_id=$2 AND status='accepted';

-- name: ChatSyncSnapshot :one
SELECT version,safe_state,last_success_at FROM chat_sync_state WHERE owner_did=$1;

-- name: SetChatViewActivity :exec
INSERT INTO chat_view_activity(owner_did,child_token_hash,oauth_session_id,view,conversation_id,expires_at)
VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT(owner_did,child_token_hash) DO UPDATE SET
 oauth_session_id=excluded.oauth_session_id,view=excluded.view,
 conversation_id=excluded.conversation_id,expires_at=excluded.expires_at;

-- name: ClearChatViewActivity :exec
DELETE FROM chat_view_activity WHERE owner_did=$1 AND child_token_hash=$2;

-- name: ScheduleChatForeground :exec
UPDATE chat_sync_state SET next_due_at=least(next_due_at,$2)
WHERE owner_did=$1 AND failure_count=0 AND claim_token IS NULL;

-- name: HasChatForegroundView :one
SELECT EXISTS(
 SELECT 1 FROM chat_view_activity activity
 JOIN craftsky_sessions child ON child.token_hash=activity.child_token_hash AND child.account_did=activity.owner_did
 JOIN oauth_sessions parent ON parent.account_did=child.account_did AND parent.session_id=child.oauth_session_id
 JOIN owner_lifecycles owner ON owner.owner_did=activity.owner_did
 WHERE activity.owner_did=$1 AND activity.view='conversation' AND activity.expires_at>$2
 AND child.lifecycle_state='active' AND child.idle_expires_at>now()
 AND parent.lifecycle_state='active' AND parent.absolute_expires_at>now()
 AND owner.state='active' AND owner.auth_epoch=child.auth_epoch
 AND owner.auth_epoch=parent.auth_epoch AND owner.generation=parent.owner_generation
) AS foreground;

-- name: EnsureScheduledChatAccounts :exec
INSERT INTO chat_sync_state(owner_did)
SELECT account.owner_did FROM chat_accounts account
JOIN owner_lifecycles owner ON owner.owner_did=account.owner_did
WHERE account.enabled AND owner.state='active'
AND NOT EXISTS(SELECT 1 FROM chat_sync_state state WHERE state.owner_did=account.owner_did)
ORDER BY account.owner_did LIMIT 100 ON CONFLICT DO NOTHING;

-- name: DueChatOwners :many
SELECT state.owner_did FROM chat_sync_state state
JOIN chat_accounts account ON account.owner_did=state.owner_did
JOIN owner_lifecycles owner ON owner.owner_did=state.owner_did
WHERE account.enabled AND owner.state='active' AND state.next_due_at<=$1
 AND (state.claim_expires_at IS NULL OR state.claim_expires_at<=$1)
 AND EXISTS(
  SELECT 1 FROM craftsky_sessions child
  JOIN oauth_sessions parent ON parent.account_did=child.account_did AND parent.session_id=child.oauth_session_id
  WHERE child.account_did=state.owner_did AND child.lifecycle_state='active' AND child.idle_expires_at>now()
  AND parent.lifecycle_state='active' AND parent.absolute_expires_at>now()
  AND child.auth_epoch=owner.auth_epoch AND parent.auth_epoch=owner.auth_epoch AND parent.owner_generation=owner.generation
 )
ORDER BY state.next_due_at,state.owner_did LIMIT 100;

-- name: PauseChatSync :exec
UPDATE chat_sync_state SET next_due_at=$2,safe_state=$3,failure_count=failure_count+1,
 recovery_id=CASE WHEN phase='ready' THEN coalesce(recovery_id,gen_random_uuid()) ELSE recovery_id END
WHERE owner_did=$1 AND claim_token IS NULL;

-- name: ResumeChatSync :exec
UPDATE chat_sync_state SET next_due_at=now(),failure_count=0,
 safe_state=CASE WHEN phase='ready' THEN 'ready' ELSE 'syncing' END,
 claim_token=NULL,claim_expires_at=NULL,
 recovery_id=CASE WHEN phase='ready' THEN coalesce(recovery_id,gen_random_uuid()) ELSE recovery_id END
WHERE owner_did=$1;

-- name: ClaimChatDelivery :one
WITH due AS (
 SELECT candidate.owner_did,candidate.id FROM chat_push_deliveries candidate
 WHERE ((candidate.status IN ('pending','retry') AND candidate.next_attempt_at<=$1)
   OR (candidate.status='leased' AND candidate.lease_expires_at<=$1))
 ORDER BY candidate.next_attempt_at,candidate.owner_did,candidate.id FOR UPDATE SKIP LOCKED LIMIT 1
)
UPDATE chat_push_deliveries delivery SET status='leased',lease_token=$2,
 lease_expires_at=$3,attempts=attempts+1
FROM due WHERE delivery.owner_did=due.owner_did AND delivery.id=due.id
RETURNING delivery.*;

-- name: ChatDeliveryMetadata :one
SELECT notification.kind,notification.conversation_id,notification.message_id,
 notification.messaging_generation,notification.open_reference,notification.state,
 subscription.installation_id,subscription.routing_id,installation.fcm_token,installation.platform,
 (subscription.active AND installation.active AND account.enabled AND account.push_enabled
  AND account.generation=notification.messaging_generation AND notification.state='pending'
  AND NOT coalesce(conversation.left_by_owner,false))::boolean AS eligible
FROM chat_push_deliveries delivery
JOIN chat_notifications notification ON notification.owner_did=delivery.owner_did AND notification.id=delivery.notification_id
JOIN chat_accounts account ON account.owner_did=delivery.owner_did
JOIN push_account_subscriptions subscription ON subscription.id=delivery.account_subscription_id AND subscription.account_did=delivery.owner_did
JOIN push_installations installation ON installation.id=subscription.installation_id
LEFT JOIN chat_conversation_state conversation ON conversation.owner_did=delivery.owner_did AND conversation.conversation_id=notification.conversation_id
WHERE delivery.owner_did=$1 AND delivery.id=$2 AND delivery.status='leased' AND delivery.lease_token=$3;

-- name: TerminateChatDelivery :execrows
UPDATE chat_push_deliveries SET status=$4,provider_result_class=$4,
 lease_token=NULL,lease_expires_at=NULL,terminal_at=$5
WHERE owner_did=$1 AND id=$2 AND lease_token=$3 AND status='leased';

-- name: FinishChatDelivery :execrows
UPDATE chat_push_deliveries delivery SET status=$4,provider_result_class=$5,
 next_attempt_at=$6,lease_token=NULL,lease_expires_at=NULL,
 terminal_at=CASE WHEN $4='retry' THEN NULL ELSE $7::timestamptz END
WHERE delivery.owner_did=$1 AND delivery.id=$2 AND delivery.lease_token=$3
 AND delivery.status='leased' AND delivery.lease_expires_at>$7
 AND EXISTS(
  SELECT 1 FROM push_account_subscriptions subscription
  JOIN push_installations installation ON installation.id=subscription.installation_id
  WHERE subscription.id=delivery.account_subscription_id AND subscription.active
   AND installation.id=$8 AND installation.active AND installation.fcm_token=$9
 );

-- name: CompleteChatNotification :exec
UPDATE chat_notifications notification SET state='complete',terminal_at=$3
WHERE notification.owner_did=$1 AND notification.id=$2 AND notification.state='pending'
 AND NOT EXISTS(SELECT 1 FROM chat_push_deliveries delivery
  WHERE delivery.owner_did=notification.owner_did AND delivery.notification_id=notification.id
  AND delivery.status IN ('pending','retry','leased'));

-- name: InvalidateChatDeliveryInstallation :exec
WITH invalid_installation AS (
 UPDATE push_installations installation SET active=false,deactivated_at=$3,updated_at=$3
 WHERE installation.id=$1 AND installation.active AND installation.fcm_token=$2 RETURNING installation.id
), disabled_subscriptions AS (
 UPDATE push_account_subscriptions subscription SET active=false,deactivated_at=$3,updated_at=$3
 WHERE subscription.installation_id IN(SELECT invalid_installation.id FROM invalid_installation) RETURNING subscription.id
), cancelled_social AS (
 UPDATE push_deliveries SET status='cancelled',lease_owner=NULL,lease_expires_at=NULL,updated_at=$3
 WHERE account_subscription_id IN(SELECT disabled_subscriptions.id FROM disabled_subscriptions) AND status IN('pending','retry','leased')
)
UPDATE chat_push_deliveries SET status='cancelled',provider_result_class='cancelled',
 lease_token=NULL,lease_expires_at=NULL,terminal_at=$3
WHERE account_subscription_id IN(SELECT disabled_subscriptions.id FROM disabled_subscriptions) AND status IN('pending','retry','leased');

-- name: ResolveChatNotificationOpen :one
SELECT conversation_id,messaging_generation FROM chat_notifications
WHERE owner_did=$1 AND open_reference=$2 AND conversation_id<>''
 AND (terminal_at IS NULL OR terminal_at>now()-interval '30 days');

-- name: ExpireChatDeliveries :exec
WITH expired AS (
 SELECT owner_did,id FROM chat_push_deliveries
 WHERE status IN('pending','retry','leased') AND deadline_at<=now()
 ORDER BY deadline_at,owner_did,id LIMIT 100 FOR UPDATE SKIP LOCKED
)
UPDATE chat_push_deliveries delivery SET status='expired',provider_result_class='expired',
 lease_token=NULL,lease_expires_at=NULL,terminal_at=now()
FROM expired WHERE delivery.owner_did=expired.owner_did AND delivery.id=expired.id;

-- name: CompleteEmptyChatNotifications :exec
WITH completed AS (
 SELECT notification.owner_did,notification.id FROM chat_notifications notification
 WHERE notification.state='pending' AND NOT EXISTS(
  SELECT 1 FROM chat_push_deliveries delivery WHERE delivery.owner_did=notification.owner_did
   AND delivery.notification_id=notification.id AND delivery.status IN('pending','retry','leased')
 )
 ORDER BY notification.created_at,notification.owner_did,notification.id LIMIT 100 FOR UPDATE SKIP LOCKED
)
UPDATE chat_notifications notification SET state='complete',terminal_at=now()
FROM completed WHERE notification.owner_did=completed.owner_did AND notification.id=completed.id;

-- name: PruneChatDeliveryDetails :exec
WITH expired AS (
 SELECT owner_did,id FROM chat_push_deliveries WHERE terminal_at<=now()-interval '30 days'
 ORDER BY terminal_at,owner_did,id LIMIT 100 FOR UPDATE SKIP LOCKED
)
DELETE FROM chat_push_deliveries delivery USING expired
WHERE delivery.owner_did=expired.owner_did AND delivery.id=expired.id;

-- name: PruneChatNotificationRouting :exec
WITH expired AS (
 SELECT owner_did,id FROM chat_notifications WHERE state<>'pending'
 AND terminal_at<=now()-interval '30 days'
 AND (conversation_id<>'' OR message_id<>'' OR open_reference IS NOT NULL OR logical_key NOT LIKE 'sha256:%')
 ORDER BY terminal_at,owner_did,id LIMIT 100 FOR UPDATE SKIP LOCKED
)
UPDATE chat_notifications notification SET conversation_id='',message_id='',open_reference=NULL,
 logical_key=CASE WHEN notification.logical_key LIKE 'sha256:%' THEN notification.logical_key
 ELSE 'sha256:'||encode(sha256(convert_to(notification.logical_key,'UTF8')),'hex') END
FROM expired WHERE notification.owner_did=expired.owner_did AND notification.id=expired.id;

-- name: PruneExpiredChatViewActivity :exec
WITH expired AS (
 SELECT owner_did,child_token_hash FROM chat_view_activity WHERE expires_at<=now()
 ORDER BY expires_at,owner_did,child_token_hash LIMIT 100 FOR UPDATE SKIP LOCKED
)
DELETE FROM chat_view_activity activity USING expired
WHERE activity.owner_did=expired.owner_did AND activity.child_token_hash=expired.child_token_hash;

-- name: PruneExpiredChatContinuations :exec
WITH expired AS (
 SELECT owner_did,id FROM chat_continuations WHERE expires_at<=now()
 ORDER BY expires_at,owner_did,id LIMIT 100 FOR UPDATE SKIP LOCKED
)
DELETE FROM chat_continuations continuation USING expired
WHERE continuation.owner_did=expired.owner_did AND continuation.id=expired.id;
