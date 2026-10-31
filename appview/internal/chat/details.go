package chat

import (
	"context"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

// ConversationDetails is a transient private response, never a stored archive.
type ConversationDetails struct {
	Conversation Conversation `json:"conversation"`
	Messages     MessagePage  `json:"messages"`
}

func (ConversationDetails) String() string   { return "ConversationDetails([REDACTED])" }
func (ConversationDetails) GoString() string { return "ConversationDetails([REDACTED])" }

func (service *Service) Details(ctx context.Context, info auth.AuthInfo, id ConversationID) (ConversationDetails, error) {
	var result ConversationDetails
	if !validOpaqueID(string(id)) {
		return result, ErrInvalidIdentifier
	}
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo", "chat.bsky.convo.getMessages"}, func(ctx context.Context, transport *Transport) error {
		connection, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		current, err := service.supportedConversation(ctx, transport, info.DID, id)
		if err != nil {
			return err
		}
		conversation, err := conversationFromView(info.DID, current)
		if err != nil {
			return err
		}
		blocked, err := service.blocked(ctx, info.DID, conversation.Peer.DID)
		if err != nil {
			return err
		}
		conversation.CanSend = conversation.Status == "accepted" && !blocked
		messages, err := service.historyPage(ctx, transport, info.DID, connection.Generation, id, conversation.Peer.DID, "", 50)
		if err != nil {
			return err
		}
		result = ConversationDetails{Conversation: conversation, Messages: messages}
		return nil
	})
	return result, err
}
