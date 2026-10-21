package chat

import (
	"context"
	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

func (service *Service) Accept(ctx context.Context, info auth.AuthInfo, id ConversationID) (Conversation, error) {
	var result Conversation
	if !validOpaqueID(string(id)) {
		return result, ErrInvalidIdentifier
	}
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo", "chat.bsky.convo.acceptConvo"}, func(ctx context.Context, transport *Transport) error {
		current, err := service.supportedConversation(ctx, transport, info.DID, id)
		if err != nil {
			return err
		}
		if *current.Status != "accepted" {
			var response upstream.ConvoAcceptConvo_Output
			if err := transport.Procedure(ctx, "chat.bsky.convo.acceptConvo", upstream.ConvoAcceptConvo_Input{ConvoId: string(id)}, &response); err != nil {
				return err
			}
			current, err = service.supportedConversation(ctx, transport, info.DID, id)
			if err != nil {
				return err
			}
		}
		if *current.Status != "accepted" {
			return ErrProviderResponse
		}
		result, err = conversationFromView(info.DID, current)
		return err
	})
	return result, err
}

func (service *Service) SetMuted(ctx context.Context, info auth.AuthInfo, id ConversationID, muted bool) (Conversation, error) {
	var result Conversation
	if !validOpaqueID(string(id)) {
		return result, ErrInvalidIdentifier
	}
	method := syntax.NSID("chat.bsky.convo.unmuteConvo")
	if muted {
		method = "chat.bsky.convo.muteConvo"
	}
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo", method}, func(ctx context.Context, transport *Transport) error {
		if _, err := service.supportedConversation(ctx, transport, info.DID, id); err != nil {
			return err
		}
		var response upstream.ConvoMuteConvo_Output
		if err := transport.Procedure(ctx, method, upstream.ConvoMuteConvo_Input{ConvoId: string(id)}, &response); err != nil {
			return err
		}
		if response.Convo == nil || response.Convo.Id != string(id) || response.Convo.Muted != muted {
			return ErrProviderResponse
		}
		var err error
		result, err = conversationFromView(info.DID, response.Convo)
		return err
	})
	return result, err
}

func (service *Service) Leave(ctx context.Context, info auth.AuthInfo, id ConversationID) error {
	if !validOpaqueID(string(id)) {
		return ErrInvalidIdentifier
	}
	return service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo", "chat.bsky.convo.leaveConvo"}, func(ctx context.Context, transport *Transport) error {
		if _, err := service.supportedConversation(ctx, transport, info.DID, id); err != nil {
			return err
		}
		var response upstream.ConvoLeaveConvo_Output
		if err := transport.Procedure(ctx, "chat.bsky.convo.leaveConvo", upstream.ConvoLeaveConvo_Input{ConvoId: string(id)}, &response); err != nil {
			return err
		}
		if response.ConvoId != string(id) || response.Rev == "" || len(response.Rev) > 512 {
			return ErrProviderResponse
		}
		return service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			q := chatdb.New(tx)
			if err := q.MarkChatConversationLeft(ctx, chatdb.MarkChatConversationLeftParams{OwnerDid: info.DID.String(), ConversationID: string(id)}); err != nil {
				return err
			}
			return q.CancelChatConversationNotifications(ctx, chatdb.CancelChatConversationNotificationsParams{OwnerDid: info.DID.String(), ConversationID: string(id)})
		})
	})
}
