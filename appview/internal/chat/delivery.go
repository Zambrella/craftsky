package chat

import (
	"context"
	"errors"
	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/push"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var errDeliverySuppressed = errors.New("private message delivery is no longer eligible")

type DeliveryOptions struct {
	Now      func() time.Time
	Observer SyncFailureObserver
}
type DeliveryDispatcher struct {
	service  *Service
	selector BackgroundSessions
	sender   push.Sender
	now      func() time.Time
	observer SyncFailureObserver
}

func NewDeliveryDispatcher(service *Service, selector BackgroundSessions, sender push.Sender, options DeliveryOptions) (*DeliveryDispatcher, error) {
	if service == nil || selector == nil || sender == nil {
		return nil, ErrOperationUnavailable
	}
	if _, ok := service.sessions.(EffectSessionRunner); !ok {
		return nil, ErrOperationUnavailable
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &DeliveryDispatcher{service: service, selector: selector, sender: sender, now: options.Now, observer: options.Observer}, nil
}

func dispatchConversation(ctx context.Context, transport *Transport, owner syntax.DID, id ConversationID) (*upstream.ConvoDefs_ConvoView, error) {
	var status struct {
		ChatDisabled *bool `json:"chatDisabled"`
	}
	if err := transport.Query(ctx, "chat.bsky.actor.getStatus", nil, &status); err != nil {
		return nil, err
	}
	if status.ChatDisabled == nil {
		return nil, ErrProviderResponse
	}
	if *status.ChatDisabled {
		return nil, ErrRestricted
	}
	return currentConversation(ctx, transport, owner, id)
}

func (dispatcher *DeliveryDispatcher) terminate(ctx context.Context, item chatdb.ChatPushDelivery, status string) error {
	tx, err := dispatcher.service.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := chatdb.New(tx)
	if _, err := q.TerminateChatDelivery(ctx, chatdb.TerminateChatDeliveryParams{OwnerDid: item.OwnerDid, ID: item.ID, LeaseToken: item.LeaseToken, Status: status, TerminalAt: dbTime(dispatcher.now())}); err != nil {
		return err
	}
	if err := q.CompleteChatNotification(ctx, chatdb.CompleteChatNotificationParams{OwnerDid: item.OwnerDid, ID: item.NotificationID, TerminalAt: dbTime(dispatcher.now())}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (dispatcher *DeliveryDispatcher) ProcessOne(ctx context.Context) (processed bool, resultErr error) {
	diagnosticCtx := ctx
	var diagnosticOwner syntax.DID
	var diagnosticReference string
	var attempt int
	effectAttempted := false
	defer func() {
		if resultErr == nil || dispatcher.observer == nil || errors.Is(resultErr, context.Canceled) {
			return
		}
		if !effectAttempted && (errors.Is(resultErr, auth.ErrNoUsableBackgroundSession) || errors.Is(resultErr, auth.ErrPDSPermissionsRequired) || errors.Is(resultErr, auth.ErrPDSSessionExpired) || errors.Is(resultErr, ErrConnectionChanged)) {
			return
		}
		stage := "eligibility"
		if effectAttempted {
			stage = "finalization"
		}
		dispatcher.observer.ObservePrivateFailure(diagnosticCtx, resultErr, diagnosticOwner, diagnosticReference, "chat.push", stage, "retry", attempt)
	}()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	q := chatdb.New(dispatcher.service.store.pool)
	now := dispatcher.now()
	item, err := q.ClaimChatDelivery(ctx, chatdb.ClaimChatDeliveryParams{NextAttemptAt: dbTime(now), LeaseToken: dbUUID(uuid.New()), LeaseExpiresAt: dbTime(now.Add(time.Minute))})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	diagnosticReference = uuid.UUID(item.ID.Bytes).String()
	attempt = int(item.Attempts)
	if !item.DeadlineAt.Time.After(now) {
		return true, dispatcher.terminate(ctx, item, "expired")
	}
	owner, err := syntax.ParseDID(item.OwnerDid)
	if err != nil {
		return true, errorsProviderCause(err)
	}
	diagnosticOwner = owner
	metadata, err := q.ChatDeliveryMetadata(ctx, chatdb.ChatDeliveryMetadataParams{OwnerDid: item.OwnerDid, ID: item.ID, LeaseToken: item.LeaseToken})
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return true, err
	}
	if !metadata.Eligible {
		return true, dispatcher.terminate(ctx, item, "cancelled")
	}
	handled, err := dispatcher.processClaim(ctx, owner, item, metadata)
	effectAttempted = handled
	if handled {
		return true, err
	}
	var provider *atclient.APIError
	if errors.Is(err, errDeliverySuppressed) || errors.Is(err, ErrInaccessibleConversation) || errors.Is(err, ErrBlocked) || errors.Is(err, ErrDisabled) || errors.Is(err, ErrConnectionChanged) || errors.Is(err, ErrRestricted) || errors.Is(err, ownerlifecycle.ErrOwnerNotActive) || errors.Is(err, ownerlifecycle.ErrTerminalOwner) || (errors.As(err, &provider) && provider.StatusCode == 404) {
		return true, dispatcher.terminate(ctx, item, "cancelled")
	}
	// Eligibility failures defer rather than deliver stale copy. This is a
	// new leased push attempt, never a replay of a send/report operation.
	finalCtx, finalCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer finalCancel()
	delay := time.Second
	if errors.Is(err, auth.ErrNoUsableBackgroundSession) || errors.Is(err, auth.ErrPDSPermissionsRequired) || errors.Is(err, auth.ErrPDSSessionExpired) {
		delay = 5 * time.Minute
	}
	var limited *auth.RateLimitError
	if errors.As(err, &limited) {
		delay = max(delay, limited.RetryAfter)
	}
	_, finishErr := q.FinishChatDelivery(finalCtx, chatdb.FinishChatDeliveryParams{OwnerDid: item.OwnerDid, ID: item.ID, LeaseToken: item.LeaseToken, Status: "retry", ProviderResultClass: "retryable", NextAttemptAt: dbTime(dispatcher.now().Add(delay)), Column7: dbTime(dispatcher.now()), ID_2: metadata.InstallationID, FcmToken: metadata.FcmToken})
	return true, errors.Join(err, finishErr)
}

func (dispatcher *DeliveryDispatcher) processClaim(ctx context.Context, owner syntax.DID, item chatdb.ChatPushDelivery, initial chatdb.ChatDeliveryMetadataRow) (bool, error) {
	required := rpcScopes("chat.bsky.actor.getStatus", "chat.bsky.convo.getConvo")
	parent, err := dispatcher.selector.SelectForCapability(ctx, owner, required)
	if err != nil {
		return false, err
	}
	conversationID, err := ParseConversationID(initial.ConversationID)
	if err != nil {
		return false, ErrInaccessibleConversation
	}
	var peer syntax.DID
	err = dispatcher.service.sessions.WithCapability(ctx, owner, parent, required, func(ctx context.Context, session *oauth.ClientSession) error {
		connection, err := dispatcher.service.backgroundConnection(ctx, owner, parent)
		if err != nil {
			return err
		}
		if !connection.Enabled || !connection.PushEnabled || connection.Generation != initial.MessagingGeneration {
			return errDeliverySuppressed
		}
		transport, err := dispatcher.service.transport(session)
		if err != nil {
			return err
		}
		view, err := dispatchConversation(ctx, transport, owner, conversationID)
		if err != nil {
			return err
		}
		peer, err = directPeer(owner, view)
		return err
	})
	if err != nil {
		return false, err
	}
	expected, err := dispatcher.service.expectedParticipants(ctx, owner, peer)
	if err != nil {
		return false, err
	}
	effects := dispatcher.service.sessions.(EffectSessionRunner)
	handled := false
	err = effects.WithCapabilityEffect(ctx, expected, owner, parent, required, func(ctx context.Context, session *oauth.ClientSession) error {
		connection, err := dispatcher.service.backgroundConnection(ctx, owner, parent)
		if err != nil {
			return err
		}
		if !connection.Enabled || !connection.PushEnabled || connection.Generation != initial.MessagingGeneration {
			return errDeliverySuppressed
		}
		transport, err := dispatcher.service.transport(session)
		if err != nil {
			return err
		}
		view, err := dispatchConversation(ctx, transport, owner, conversationID)
		if err != nil {
			return err
		}
		currentPeer, err := directPeer(owner, view)
		if err != nil {
			return err
		}
		if currentPeer != peer {
			return ErrInaccessibleConversation
		}
		if view.Muted || view.UnreadCount <= 0 || view.LastMessage == nil {
			return errDeliverySuppressed
		}
		blocked, err := dispatcher.service.blocked(ctx, owner, peer)
		if err != nil {
			return err
		}
		if blocked {
			return ErrBlocked
		}
		var metadata chatdb.ChatDeliveryMetadataRow
		err = dispatcher.service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			var err error
			metadata, err = chatdb.New(tx).ChatDeliveryMetadata(ctx, chatdb.ChatDeliveryMetadataParams{OwnerDid: item.OwnerDid, ID: item.ID, LeaseToken: item.LeaseToken})
			return err
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return errDeliverySuppressed
		}
		if err != nil {
			return err
		}
		if !metadata.Eligible {
			return errDeliverySuppressed
		}
		if (metadata.Kind == "request") != (*view.Status == "request") {
			return errDeliverySuppressed
		}
		latest := messageFromView(view.LastMessage.ConvoDefs_MessageView, view.LastMessage.ConvoDefs_DeletedMessageView, view.LastMessage.ConvoDefs_SystemMessageView)
		if latest.Kind != "text" || latest.SenderDID != peer {
			return errDeliverySuppressed
		}
		if metadata.Kind != "request" && string(latest.ID) != metadata.MessageID {
			return errDeliverySuppressed
		}
		summary, err := conversationFromView(owner, view)
		if err != nil {
			return err
		}
		now := dispatcher.now()
		deadline := minTime(now.Add(10*time.Second), item.DeadlineAt.Time, item.LeaseExpiresAt.Time.Add(-5*time.Second))
		if !deadline.After(now) {
			return ErrOperationUnavailable
		}
		sendCtx, cancel := context.WithTimeout(ctx, deadline.Sub(now))
		result, sendErr := dispatcher.sender.Send(sendCtx, push.SendRequest{Token: metadata.FcmToken, Chat: &push.ChatRouting{OpenReference: uuid.UUID(metadata.OpenReference.Bytes).String()}, AccountSubscriptionID: uuid.UUID(metadata.RoutingID.Bytes).String(), ActorDisplayName: chatSenderName(summary.Peer), Platform: metadata.Platform, Semantics: push.DeliveryUniqueEvent, TTL: item.DeadlineAt.Time.Sub(now)})
		cancel()
		handled = true
		status, code, next := "succeeded", string(result.Class), dispatcher.now()
		switch result.Class {
		case push.ResultSuccess:
		case push.ResultInvalidToken:
			status = "permanent_failure"
		case push.ResultPermanentFailure:
			status = "permanent_failure"
		default:
			code = string(push.ResultRetryable)
			var retry bool
			next, retry = push.NextRetry(dispatcher.now(), item.DeadlineAt.Time, int(item.Attempts), 0)
			status = "retry"
			if !retry {
				status = "expired"
				next = dispatcher.now()
			}
		}
		finalCtx, finalCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer finalCancel()
		finishErr := dispatcher.service.store.owners.WithAuthTransaction(finalCtx, func(tx pgx.Tx) error {
			q := chatdb.New(tx)
			count, err := q.FinishChatDelivery(finalCtx, chatdb.FinishChatDeliveryParams{OwnerDid: item.OwnerDid, ID: item.ID, LeaseToken: item.LeaseToken, Status: status, ProviderResultClass: code, NextAttemptAt: dbTime(next), Column7: dbTime(dispatcher.now()), ID_2: metadata.InstallationID, FcmToken: metadata.FcmToken})
			if err != nil {
				return err
			}
			if count != 1 {
				return ErrConnectionChanged
			}
			if result.Class == push.ResultInvalidToken {
				if err := q.InvalidateChatDeliveryInstallation(finalCtx, chatdb.InvalidateChatDeliveryInstallationParams{ID: metadata.InstallationID, FcmToken: metadata.FcmToken, TerminalAt: dbTime(dispatcher.now())}); err != nil {
					return err
				}
			}
			return q.CompleteChatNotification(finalCtx, chatdb.CompleteChatNotificationParams{OwnerDid: item.OwnerDid, ID: item.NotificationID, TerminalAt: dbTime(dispatcher.now())})
		})
		return errors.Join(sendErr, finishErr)
	})
	return handled, err
}

func minTime(first time.Time, others ...time.Time) time.Time {
	for _, value := range others {
		if value.Before(first) {
			first = value
		}
	}
	return first
}
func chatSenderName(actor Actor) string {
	name := strings.TrimSpace(actor.DisplayName)
	valid := name != "" && len(name) <= 256 && utf8.ValidString(name)
	for _, r := range name {
		if unicode.IsControl(r) {
			valid = false
		}
	}
	if valid {
		return name
	}
	return actor.Handle.String()
}
