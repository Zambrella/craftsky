package chat

import (
	"context"
	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

type Availability struct {
	CanChat        bool           `json:"canChat"`
	ConversationID ConversationID `json:"conversationId,omitempty"`
}

func (service *Service) blocked(ctx context.Context, owner, peer syntax.DID) (bool, error) {
	var blocked bool
	err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		value, err := chatdb.New(tx).ChatBlocked(ctx, chatdb.ChatBlockedParams{Viewer: owner.String(), Peer: peer.String()})
		blocked = !value.Valid || value.Bool
		return err
	})
	return blocked, err
}

func (service *Service) Availability(ctx context.Context, info auth.AuthInfo, peer syntax.DID) (Availability, error) {
	var result Availability
	if peer == "" || peer == info.DID {
		return result, ErrInvalidIdentifier
	}
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvoAvailability"}, func(ctx context.Context, transport *Transport) error {
		blocked, err := service.blocked(ctx, info.DID, peer)
		if err != nil || blocked {
			return err
		}
		var response struct {
			CanChat *bool                         `json:"canChat"`
			Convo   *upstream.ConvoDefs_ConvoView `json:"convo"`
		}
		if err := transport.Query(ctx, "chat.bsky.convo.getConvoAvailability", map[string]any{"members": []string{peer.String()}}, &response); err != nil {
			return err
		}
		if response.CanChat == nil {
			return ErrProviderResponse
		}
		result.CanChat = *response.CanChat
		if response.Convo != nil {
			actual, err := directPeer(info.DID, response.Convo)
			if err != nil || actual != peer {
				return ErrInaccessibleConversation
			}
			result.ConversationID = ConversationID(response.Convo.Id)
		}
		return nil
	})
	return result, err
}

func (service *Service) StartConversation(ctx context.Context, info auth.AuthInfo, peer syntax.DID) (Conversation, error) {
	var result Conversation
	if peer == "" || peer == info.DID {
		return result, ErrInvalidIdentifier
	}
	initial, err := service.store.ConnectionForChild(ctx, info)
	if err != nil {
		return result, err
	}
	if !initial.Enabled {
		return result, ErrDisabled
	}
	expected, err := service.expectedParticipants(ctx, info.DID, peer)
	if err != nil {
		return result, err
	}
	effects, ok := service.sessions.(EffectSessionRunner)
	if !ok {
		return result, ErrOperationUnavailable
	}
	err = effects.WithCapabilityEffect(ctx, expected, info.DID, info.SessionID, rpcScopes("chat.bsky.actor.getStatus", "chat.bsky.convo.getConvoAvailability", "chat.bsky.convo.getConvoForMembers"), func(ctx context.Context, session *oauth.ClientSession) error {
		current, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		if !current.Enabled || current.Generation != initial.Generation {
			return ErrConnectionChanged
		}
		blocked, err := service.blocked(ctx, info.DID, peer)
		if err != nil {
			return err
		}
		if blocked {
			return ErrBlocked
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
		params := map[string]any{"members": []string{peer.String()}}
		var availability struct {
			CanChat *bool `json:"canChat"`
		}
		if err := transport.Query(ctx, "chat.bsky.convo.getConvoAvailability", params, &availability); err != nil {
			return err
		}
		if availability.CanChat == nil {
			return ErrProviderResponse
		}
		if !*availability.CanChat {
			return ErrBlocked
		}
		var response upstream.ConvoGetConvoForMembers_Output
		if err := transport.Query(ctx, "chat.bsky.convo.getConvoForMembers", params, &response); err != nil {
			return err
		}
		actual, err := directPeer(info.DID, response.Convo)
		if err != nil || actual != peer {
			return ErrInaccessibleConversation
		}
		result, err = conversationFromView(info.DID, response.Convo)
		return err
	})
	return result, err
}
