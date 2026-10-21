package chat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"math/rand/v2"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
	"strings"
	"time"
)

var ErrPollNotDue = errors.New("messaging synchronization is already claimed or not due")

type BackgroundSessions interface {
	SelectForCapability(context.Context, syntax.DID, []string) (string, error)
}
type SyncFailureObserver interface {
	ObservePrivateFailure(context.Context, error, syntax.DID, string, string, string, string, int)
}
type SyncOptions struct {
	Now      func() time.Time
	Pulses   <-chan time.Time
	Observer SyncFailureObserver
	Jitter   func() float64
}
type Synchronizer struct {
	service  *Service
	selector BackgroundSessions
	now      func() time.Time
	pulses   <-chan time.Time
	observer SyncFailureObserver
	jitter   func() float64
}

func NewSynchronizer(service *Service, selector BackgroundSessions, options SyncOptions) (*Synchronizer, error) {
	if service == nil || selector == nil {
		return nil, ErrOperationUnavailable
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Jitter == nil {
		options.Jitter = rand.Float64
	}
	return &Synchronizer{service: service, selector: selector, now: options.Now, pulses: options.Pulses, observer: options.Observer, jitter: options.Jitter}, nil
}
func dbTime(value time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: value, Valid: true} }
func dbUUID(value uuid.UUID) pgtype.UUID        { return pgtype.UUID{Bytes: value, Valid: true} }

// This private server-owned path selects a live child for one parent. It is
// never exposed as a consumer authorization alternative or an activity touch.
func (service *Service) backgroundConnection(ctx context.Context, owner syntax.DID, parent string) (Connection, error) {
	var result Connection
	err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		q := chatdb.New(tx)
		live, err := q.ChatLiveBackgroundParent(ctx, chatdb.ChatLiveBackgroundParentParams{AccountDid: owner.String(), OauthSessionID: parent})
		if err != nil {
			return err
		}
		if !live {
			return auth.ErrNoUsableBackgroundSession
		}
		row, err := q.ChatConnection(ctx, owner.String())
		result = Connection{Enabled: row.Enabled, Generation: row.Generation, PushEnabled: row.PushEnabled}
		return err
	})
	return result, err
}

type syncObservation struct {
	event        logEvent
	conversation *upstream.ConvoDefs_ConvoView
	eligible     bool
}

func (worker *Synchronizer) Poll(ctx context.Context, owner syntax.DID) (pollErr error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	now := worker.now()
	token := dbUUID(uuid.New())
	var state chatdb.ChatSyncState
	var connection Connection
	// Claim before session selection/discovery. Every external preflight then
	// shares the same account/global capacity and durable failure schedule.
	err := worker.service.store.withAccount(ctx, owner, func(ctx context.Context, q *chatdb.Queries) error {
		row, err := q.ChatConnection(ctx, owner.String())
		if err != nil {
			return err
		}
		connection = Connection{Enabled: row.Enabled, Generation: row.Generation, PushEnabled: row.PushEnabled}
		if !connection.Enabled {
			return ErrDisabled
		}
		if err := q.EnsureChatSync(ctx, owner.String()); err != nil {
			return err
		}
		if err := q.LockChatPollCapacity(ctx); err != nil {
			return err
		}
		state, err = q.ClaimChatSync(ctx, chatdb.ClaimChatSyncParams{OwnerDid: owner.String(), ClaimToken: token, ClaimExpiresAt: dbTime(now.Add(time.Minute)), NextDueAt: dbTime(now)})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPollNotDue
		}
		return err
	})
	if err != nil {
		return err
	}
	defer func() {
		if pollErr == nil {
			return
		}
		finalCtx, finalCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer finalCancel()
		delay := time.Second * time.Duration(1<<min(state.FailureCount, 6))
		delay = min(60*time.Second, time.Duration(float64(delay)*(1+0.2*min(1, max(0, worker.jitter())))))
		safeState := "unavailable"
		var limited *auth.RateLimitError
		if errors.As(pollErr, &limited) {
			safeState = "rateLimited"
			delay = max(delay, limited.RetryAfter)
		}
		if errors.Is(pollErr, ErrRestricted) {
			safeState = "restricted"
			delay = 5 * time.Minute
		}
		if errors.Is(pollErr, auth.ErrNoUsableBackgroundSession) || errors.Is(pollErr, auth.ErrPDSPermissionsRequired) || errors.Is(pollErr, auth.ErrPDSSessionExpired) || errors.Is(pollErr, auth.ErrOAuthSessionNotFound) {
			safeState = "authorizationNeeded"
			delay = 5 * time.Minute
		}
		finishErr := worker.service.store.withAccount(finalCtx, owner, func(ctx context.Context, q *chatdb.Queries) error {
			// Claim-token CAS cannot clear a replacement/disconnect claim.
			return q.FailChatSync(ctx, chatdb.FailChatSyncParams{OwnerDid: owner.String(), ClaimToken: token, NextDueAt: dbTime(worker.now().Add(delay)), SafeState: safeState})
		})
		pollErr = errors.Join(pollErr, finishErr)
	}()
	required := rpcScopes("chat.bsky.actor.getStatus", "chat.bsky.convo.getLog", "chat.bsky.convo.getConvo", "chat.bsky.convo.listConvos", "chat.bsky.convo.listConvoRequests")
	parent, err := worker.selector.SelectForCapability(ctx, owner, required)
	if err != nil {
		return err
	}
	return worker.service.sessions.WithCapability(ctx, owner, parent, required, func(ctx context.Context, session *oauth.ClientSession) error {
		current, err := worker.service.backgroundConnection(ctx, owner, parent)
		if err != nil {
			return err
		}
		if !current.Enabled || current.Generation != connection.Generation {
			return ErrConnectionChanged
		}
		return worker.pollClaim(ctx, owner, current, state, session)
	})
}

func (worker *Synchronizer) pollClaim(ctx context.Context, owner syntax.DID, connection Connection, state chatdb.ChatSyncState, session *oauth.ClientSession) error {
	transport, err := worker.service.transport(session)
	if err != nil {
		return err
	}
	var eligibility struct {
		ChatDisabled *bool `json:"chatDisabled"`
	}
	if err := transport.Query(ctx, "chat.bsky.actor.getStatus", nil, &eligibility); err != nil {
		return err
	}
	if eligibility.ChatDisabled == nil {
		return ErrProviderResponse
	}
	if *eligibility.ChatDisabled {
		return ErrRestricted
	}
	phase, cursor, seed, pageCursor := state.Phase, state.Cursor, state.SeedCursor, state.PageCursor
	recoveryID := state.RecoveryID
	observations := make([]syncObservation, 0)
	baseline := strings.HasPrefix(phase, "baseline")
	switch phase {
	case "baselineHead", "recoveryHead":
		var raw json.RawMessage
		if err := transport.Query(ctx, "chat.bsky.convo.getLog", nil, &raw); err != nil {
			return err
		}
		page, err := decodeLogPage(raw)
		if err != nil {
			return err
		}
		if page.Cursor == "" {
			return ErrProviderResponse
		}
		seed = page.Cursor
		if baseline {
			phase = "baselineAccepted"
		} else {
			phase = "recoveryAccepted"
		}
	case "baselineAccepted", "baselineRequests", "recoveryAccepted", "recoveryRequests":
		bucket := "accepted"
		if strings.HasSuffix(phase, "Requests") {
			bucket = "request"
		}
		views, next, err := syncListPage(ctx, transport, bucket, pageCursor)
		if err != nil {
			return err
		}
		// Baseline suppression retains only the existing opaque dedupe IDs and
		// request-lifetime flags, including hidden peers. Otherwise a peer
		// joining CraftSky later could replay pre-enable history as new push.
		// No profile identity or message body is stored or exposed here.
		if !baseline {
			views, err = worker.service.profileConversations(ctx, owner, views)
			if err != nil {
				return err
			}
		}
		for _, view := range views {
			peer, err := directPeer(owner, view)
			if err != nil {
				continue
			}
			if *view.Status != bucket {
				continue
			}
			observation := syncObservation{conversation: view}
			if !baseline && view.LastMessage != nil {
				message := messageFromView(view.LastMessage.ConvoDefs_MessageView, view.LastMessage.ConvoDefs_DeletedMessageView, view.LastMessage.ConvoDefs_SystemMessageView)
				if message.Kind == "text" && message.ID != "" {
					blocked, err := worker.service.blocked(ctx, owner, peer)
					if err != nil {
						return err
					}
					observation.event = logEvent{Kind: "message", Revision: view.Rev, ConversationID: ConversationID(view.Id), MessageID: message.ID, SenderDID: message.SenderDID}
					observation.eligible = !blocked && !view.Muted && view.UnreadCount > 0 && message.SenderDID == peer && connection.PushEnabled
				}
			}
			observations = append(observations, observation)
		}
		if next == pageCursor && next != "" {
			return ErrProviderResponse
		}
		pageCursor = next
		if next == "" {
			if strings.HasSuffix(phase, "Accepted") {
				if baseline {
					phase = "baselineRequests"
				} else {
					phase = "recoveryRequests"
				}
			} else {
				phase = "ready"
				cursor = seed
			}
		}
	case "ready":
		var raw json.RawMessage
		if err := transport.Query(ctx, "chat.bsky.convo.getLog", map[string]any{"cursor": cursor}, &raw); err != nil {
			var apiError *atclient.APIError
			if errors.As(err, &apiError) && apiError.StatusCode == 400 && (apiError.Name == "InvalidCursor" || apiError.Name == "CursorExpired") {
				phase = "recoveryHead"
				if !recoveryID.Valid {
					recoveryID = dbUUID(uuid.New())
				}
				pageCursor = ""
				break
			}
			return err
		}
		page, err := decodeLogPage(raw)
		if err != nil {
			return err
		}
		processed := 0
		for _, event := range page.Events {
			if event.Revision <= cursor {
				continue
			}
			if processed == 10 {
				break
			}
			processed++
			observation := syncObservation{event: event}
			if event.Kind != "unknown" && event.Kind != "leave" {
				view, err := worker.service.supportedConversation(ctx, transport, owner, event.ConversationID)
				var apiError *atclient.APIError
				switch {
				case errors.Is(err, ErrInaccessibleConversation), errors.As(err, &apiError) && apiError.StatusCode == 404:
				case err != nil:
					return err
				default:
					peer, _ := directPeer(owner, view)
					blocked, err := worker.service.blocked(ctx, owner, peer)
					if err != nil {
						return err
					}
					observation.conversation = view
					observation.eligible = !blocked && !view.Muted && view.UnreadCount > 0 && event.SenderDID == peer && connection.PushEnabled
				}
			}
			observations = append(observations, observation)
			cursor = event.Revision
		}
		if processed == 0 {
			recoveryID = pgtype.UUID{}
		}
	default:
		return ErrProviderResponse
	}
	return worker.service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		q := chatdb.New(tx)
		current, err := q.ChatConnection(ctx, owner.String())
		if err != nil {
			return err
		}
		if !current.Enabled || current.Generation != connection.Generation {
			return ErrConnectionChanged
		}
		for _, observation := range observations {
			if err := worker.recordObservation(ctx, q, owner, connection.Generation, observation, baseline, state.RecoveryID); err != nil {
				return err
			}
		}
		due := worker.now()
		if phase == "ready" {
			foreground, err := q.HasChatForegroundView(ctx, chatdb.HasChatForegroundViewParams{OwnerDid: owner.String(), ExpiresAt: dbTime(due)})
			if err != nil {
				return err
			}
			interval := 30 * time.Second
			if foreground {
				interval = 4 * time.Second
			}
			due = due.Add(interval)
		}
		changed, err := q.FinishChatSync(ctx, chatdb.FinishChatSyncParams{OwnerDid: owner.String(), ClaimToken: state.ClaimToken, Version: state.Version, Phase: phase, Cursor: cursor, SeedCursor: seed, PageCursor: pageCursor, NextDueAt: dbTime(due), LastSuccessAt: dbTime(worker.now()), RecoveryID: recoveryID})
		if err != nil {
			return err
		}
		if changed != 1 {
			return ErrConnectionChanged
		}
		return nil
	})
}

func syncListPage(ctx context.Context, transport *Transport, bucket, cursor string) ([]*upstream.ConvoDefs_ConvoView, string, error) {
	params := map[string]any{"limit": int64(100)}
	if cursor != "" {
		params["cursor"] = cursor
	}
	next := ""
	var views []*upstream.ConvoDefs_ConvoView
	if bucket == "accepted" {
		params["kind"], params["status"] = "direct", "accepted"
		var response upstream.ConvoListConvos_Output
		if err := transport.Query(ctx, "chat.bsky.convo.listConvos", params, &response); err != nil {
			return nil, "", err
		}
		if response.Convos == nil || len(response.Convos) > 100 {
			return nil, "", ErrProviderResponse
		}
		views = response.Convos
		if response.Cursor != nil {
			next = *response.Cursor
		}
	} else {
		var response upstream.ConvoListConvoRequests_Output
		if err := transport.Query(ctx, "chat.bsky.convo.listConvoRequests", params, &response); err != nil {
			return nil, "", err
		}
		if response.Requests == nil || len(response.Requests) > 100 {
			return nil, "", ErrProviderResponse
		}
		for _, item := range response.Requests {
			if item != nil && item.ConvoDefs_ConvoView != nil {
				views = append(views, item.ConvoDefs_ConvoView)
			}
		}
		if response.Cursor != nil {
			next = *response.Cursor
		}
	}
	if len(next) > 8192 {
		return nil, "", ErrProviderResponse
	}
	return views, next, nil
}

func (worker *Synchronizer) recordObservation(ctx context.Context, q *chatdb.Queries, owner syntax.DID, generation int64, observation syncObservation, baseline bool, recoveryID pgtype.UUID) error {
	view := observation.conversation
	if view == nil {
		if observation.event.Kind == "leave" {
			if err := q.MarkChatConversationLeft(ctx, chatdb.MarkChatConversationLeftParams{OwnerDid: owner.String(), ConversationID: string(observation.event.ConversationID)}); err != nil {
				return err
			}
			return q.CancelChatConversationNotifications(ctx, chatdb.CancelChatConversationNotificationsParams{OwnerDid: owner.String(), ConversationID: string(observation.event.ConversationID)})
		}
		return nil
	}
	lastID := ""
	if view.LastMessage != nil {
		mapped := messageFromView(view.LastMessage.ConvoDefs_MessageView, view.LastMessage.ConvoDefs_DeletedMessageView, view.LastMessage.ConvoDefs_SystemMessageView)
		lastID = string(mapped.ID)
	}
	conversation, err := q.RecordChatConversation(ctx, chatdb.RecordChatConversationParams{OwnerDid: owner.String(), ConversationID: view.Id, Status: *view.Status, LastMessageID: lastID, RequestNotified: baseline})
	if err != nil {
		return err
	}
	if observation.event.Kind == "accept" && *view.Status == "accepted" {
		if err := q.MarkChatConversationAccepted(ctx, chatdb.MarkChatConversationAcceptedParams{OwnerDid: owner.String(), ConversationID: view.Id}); err != nil {
			return err
		}
		conversation.LeftByOwner = false
	}
	messageID := string(observation.event.MessageID)
	disposition := "suppressed"
	if baseline {
		messageID = lastID
		disposition = "baseline"
	}
	if messageID == "" {
		return nil
	}
	if observation.event.Kind == "delete" {
		disposition = "deleted"
	}
	eligible := !baseline && observation.eligible && !conversation.LeftByOwner && observation.event.Kind == "message"
	if eligible {
		disposition = "notified"
	}
	inserted, err := q.RecordChatMessage(ctx, chatdb.RecordChatMessageParams{OwnerDid: owner.String(), ConversationID: view.Id, MessageID: messageID, Disposition: disposition, ObservedRevision: observation.event.Revision})
	if err != nil {
		return err
	}
	if observation.event.Kind == "delete" {
		return q.CancelChatMessageNotifications(ctx, chatdb.CancelChatMessageNotificationsParams{OwnerDid: owner.String(), ConversationID: view.Id, MessageID: messageID})
	}
	if inserted == 0 || !eligible {
		return nil
	}
	kind, key := "normal", messageID
	if recoveryID.Valid {
		kind = "catchUp"
		key = uuid.UUID(recoveryID.Bytes).String()
	}
	if *view.Status == "request" {
		if conversation.RequestNotified {
			return nil
		}
		kind = "request"
		encoded, _ := json.Marshal(conversation.RequestGeneration)
		key = string(encoded)
		if err := q.MarkChatRequestNotified(ctx, chatdb.MarkChatRequestNotifiedParams{OwnerDid: owner.String(), ConversationID: view.Id}); err != nil {
			return err
		}
	}
	logical, _ := json.Marshal([]string{kind, view.Id, key})
	// These are operational IDs, never message/report prose. A stable digest
	// preserves lifetime uniqueness after full routing detail expires.
	digest := sha256.Sum256(logical)
	return q.InsertChatNotification(ctx, chatdb.InsertChatNotificationParams{OwnerDid: owner.String(), ID: dbUUID(uuid.New()), LogicalKey: "sha256:" + hex.EncodeToString(digest[:]), Kind: kind, ConversationID: view.Id, MessageID: messageID, MessagingGeneration: generation, OpenReference: dbUUID(uuid.New())})
}
