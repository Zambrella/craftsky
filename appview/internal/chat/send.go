package chat

import (
	"context"
	"errors"
	"time"

	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
	"social.craftsky/appview/internal/ownerlifecycle"
)

var ErrRequestNotAccepted = errors.New("accept this conversation before sending")
var ErrBlocked = errors.New("messaging is unavailable for this relationship")

type EffectSessionRunner interface {
	WithCapabilityEffect(context.Context, []ownerlifecycle.ExpectedOwner, syntax.DID, string, []string, auth.OAuthSessionOperation) error
}

type Message struct {
	ID        MessageID  `json:"id"`
	Revision  string     `json:"revision"`
	Kind      string     `json:"kind"`
	SenderDID syntax.DID `json:"senderDid,omitempty"`
	Text      string     `json:"text,omitempty"`
	SentAt    string     `json:"sentAt,omitempty"`
}
type SendOutcome struct {
	OperationReceipt
	Message *Message `json:"message,omitempty"`
}

func (service *Service) expectedParticipants(ctx context.Context, owner, peer syntax.DID) ([]ownerlifecycle.ExpectedOwner, error) {
	result := make([]ownerlifecycle.ExpectedOwner, 0, 2)
	for _, did := range []syntax.DID{owner, peer} {
		authority, err := service.store.owners.Get(ctx, did)
		if errors.Is(err, pgx.ErrNoRows) && did != owner {
			result = append(result, ownerlifecycle.ExpectedOwner{Owner: did, AllowMissing: true})
			continue
		}
		if err != nil {
			return nil, err
		}
		if authority.State != ownerlifecycle.StateActive {
			return nil, ownerlifecycle.ErrOwnerNotActive
		}
		result = append(result, ownerlifecycle.ExpectedOwner{Owner: did, Generation: authority.Generation})
	}
	return result, nil
}

func (service *Service) Send(ctx context.Context, info auth.AuthInfo, id uuid.UUID, conversationID ConversationID, text string) (SendOutcome, error) {
	if err := ValidateMessageText(text); err != nil {
		return SendOutcome{}, err
	}
	if id == uuid.Nil || !validOpaqueID(string(conversationID)) {
		return SendOutcome{}, ErrInaccessibleConversation
	}
	if service.operations == nil {
		return SendOutcome{}, ErrOperationUnavailable
	}
	effects, ok := service.sessions.(EffectSessionRunner)
	if !ok {
		return SendOutcome{}, ErrOperationUnavailable
	}
	initial, err := service.store.ConnectionForChild(ctx, info)
	if err != nil {
		return SendOutcome{}, err
	}
	if !initial.Enabled {
		return SendOutcome{}, ErrDisabled
	}
	var peer syntax.DID
	err = service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo"}, func(ctx context.Context, transport *Transport) error {
		conversation, err := currentConversation(ctx, transport, info.DID, conversationID)
		if err != nil {
			return err
		}
		if *conversation.Status != "accepted" {
			return ErrRequestNotAccepted
		}
		peer, err = directPeer(info.DID, conversation)
		return err
	})
	if err != nil {
		return SendOutcome{}, err
	}
	expected, err := service.expectedParticipants(ctx, info.DID, peer)
	if err != nil {
		return SendOutcome{}, err
	}
	var outcome SendOutcome
	err = effects.WithCapabilityEffect(ctx, expected, info.DID, info.SessionID, rpcScopes("chat.bsky.actor.getStatus", "chat.bsky.convo.getConvo", "chat.bsky.convo.sendMessage"), func(ctx context.Context, session *oauth.ClientSession) error {
		connection, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		if !connection.Enabled || connection.Generation != initial.Generation {
			return ErrConnectionChanged
		}
		transport, err := service.transport(session)
		if err != nil {
			return err
		}
		var status struct {
			ChatDisabled *bool `json:"chatDisabled"`
		}
		if err := transport.Query(ctx, "chat.bsky.actor.getStatus", nil, &status); err != nil {
			return err
		}
		if status.ChatDisabled == nil {
			return ErrProviderResponse
		}
		if *status.ChatDisabled {
			return ErrRestricted
		}
		conversation, err := currentConversation(ctx, transport, info.DID, conversationID)
		if err != nil {
			return err
		}
		currentPeer, err := directPeer(info.DID, conversation)
		if err != nil {
			return err
		}
		if currentPeer != peer {
			return ErrInaccessibleConversation
		}
		if *conversation.Status != "accepted" {
			return ErrRequestNotAccepted
		}
		if err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			blocked, err := chatdb.New(tx).ChatBlocked(ctx, chatdb.ChatBlockedParams{Viewer: info.DID.String(), Peer: peer.String()})
			if err != nil {
				return err
			}
			if !blocked.Valid || blocked.Bool {
				return ErrBlocked
			}
			return nil
		}); err != nil {
			return err
		}
		body := upstream.ConvoSendMessage_Input{ConvoId: string(conversationID), Message: &upstream.ConvoDefs_MessageInput{Text: text}}
		outcome.OperationReceipt, err = service.operations.Execute(ctx, info.DID, id, "send", body, func(ctx context.Context) (OperationResult, error) {
			var message upstream.ConvoDefs_MessageView
			if err := transport.Procedure(ctx, "chat.bsky.convo.sendMessage", body, &message); err != nil {
				return OperationResult{}, err
			}
			if !validOpaqueID(message.Id) || message.Sender == nil || message.Sender.Did != info.DID.String() || message.Text != text {
				return OperationResult{}, ErrProviderResponse
			}
			if _, err := time.Parse(time.RFC3339Nano, message.SentAt); err != nil {
				return OperationResult{}, ErrProviderResponse
			}
			outcome.Message = &Message{ID: MessageID(message.Id), Revision: message.Rev, Kind: "text", SenderDID: info.DID, Text: message.Text, SentAt: message.SentAt}
			return OperationResult{ID: message.Id, Revision: message.Rev}, nil
		})
		if outcome.State != "confirmed" {
			outcome.Message = nil
		}
		return err
	})
	return outcome, err
}
