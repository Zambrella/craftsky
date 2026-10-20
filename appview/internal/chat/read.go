package chat

import (
	"context"
	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

type UnreadCounts struct {
	Accepted int64 `json:"accepted"`
	Requests int64 `json:"requests"`
}

func (service *Service) UnreadCounts(ctx context.Context, info auth.AuthInfo) (UnreadCounts, error) {
	var result UnreadCounts
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getUnreadCounts"}, func(ctx context.Context, transport *Transport) error {
		var response struct {
			Accepted *int64 `json:"unreadAcceptedConvos"`
			Requests *int64 `json:"unreadRequestConvos"`
		}
		if err := transport.Query(ctx, "chat.bsky.convo.getUnreadCounts", map[string]any{"includeGroupChats": false}, &response); err != nil {
			return err
		}
		if response.Accepted == nil || response.Requests == nil || *response.Accepted < 0 || *response.Accepted > 100 || *response.Requests < 0 || *response.Requests > 100 {
			return ErrProviderResponse
		}
		result = UnreadCounts{Accepted: *response.Accepted, Requests: *response.Requests}
		return nil
	})
	return result, err
}

func (service *Service) UpdateRead(ctx context.Context, info auth.AuthInfo, id ConversationID, message MessageID) (Conversation, error) {
	var result Conversation
	if !validOpaqueID(string(id)) || !validOpaqueID(string(message)) {
		return result, ErrInvalidIdentifier
	}
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo", "chat.bsky.convo.updateRead"}, func(ctx context.Context, transport *Transport) error {
		if _, err := currentConversation(ctx, transport, info.DID, id); err != nil {
			return err
		}
		watermark := string(message)
		var response upstream.ConvoUpdateRead_Output
		if err := transport.Procedure(ctx, "chat.bsky.convo.updateRead", upstream.ConvoUpdateRead_Input{ConvoId: string(id), MessageId: &watermark}, &response); err != nil {
			return err
		}
		if response.Convo == nil || response.Convo.Id != string(id) {
			return ErrProviderResponse
		}
		var err error
		result, err = conversationFromView(info.DID, response.Convo)
		return err
	})
	return result, err
}
