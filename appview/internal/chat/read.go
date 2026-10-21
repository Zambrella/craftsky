package chat

import (
	"context"
	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
	"time"
)

type UnreadCounts struct {
	Accepted int64 `json:"accepted"`
	Requests int64 `json:"requests"`
}

func (service *Service) UnreadCounts(ctx context.Context, info auth.AuthInfo) (UnreadCounts, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var result UnreadCounts
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.listConvos", "chat.bsky.convo.listConvoRequests"}, func(ctx context.Context, transport *Transport) error {
		var err error
		result.Accepted, err = service.countConversations(ctx, transport, info.DID, "accepted")
		if err != nil {
			return err
		}
		result.Requests, err = service.countConversations(ctx, transport, info.DID, "request")
		return err
	})
	if err != nil {
		return UnreadCounts{}, err
	}
	return result, err
}

// Counts belong to the filtered CraftSky inbox, not the shared global inbox.
// Never return a partial traversal as a complete total. Both work and totals
// are bounded; the existing client retains previous counts on failure.
func (service *Service) countConversations(ctx context.Context, transport *Transport, owner syntax.DID, bucket string) (int64, error) {
	var count int64
	cursor := ""
	seenCursors, seenIDs := map[string]bool{}, map[string]bool{}
	for range 20 {
		seenCursors[cursor] = true
		views, next, err := syncListPage(ctx, transport, bucket, cursor)
		if err != nil {
			return 0, err
		}
		views, err = service.profileConversations(ctx, owner, views)
		if err != nil {
			return 0, err
		}
		for _, view := range views {
			if *view.Status != bucket || seenIDs[view.Id] {
				continue
			}
			seenIDs[view.Id] = true
			if view.UnreadCount < 0 {
				return 0, ErrProviderResponse
			}
			if bucket == "accepted" {
				if view.UnreadCount == 0 || view.Muted || inactiveUnreadMember(view) {
					continue
				}
				peer, _ := directPeer(owner, view)
				blocked, err := service.blocked(ctx, owner, peer)
				if err != nil {
					return 0, err
				}
				if blocked {
					continue
				}
			}
			count++
			if count == 100 {
				return count, nil
			}
		}
		if next == "" {
			return count, nil
		}
		if seenCursors[next] {
			return 0, ErrProviderResponse
		}
		cursor = next
	}
	return 0, ErrOperationUnavailable
}

func inactiveUnreadMember(view *upstream.ConvoDefs_ConvoView) bool {
	for _, member := range view.Members {
		if member.ChatDisabled != nil && *member.ChatDisabled {
			return true
		}
		if viewer := member.Viewer; viewer != nil && ((viewer.BlockedBy != nil && *viewer.BlockedBy) || viewer.Blocking != nil || viewer.BlockingByList != nil) {
			return true
		}
	}
	return false
}

func (service *Service) UpdateRead(ctx context.Context, info auth.AuthInfo, id ConversationID, message MessageID) (Conversation, error) {
	var result Conversation
	if !validOpaqueID(string(id)) || !validOpaqueID(string(message)) {
		return result, ErrInvalidIdentifier
	}
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo", "chat.bsky.convo.updateRead"}, func(ctx context.Context, transport *Transport) error {
		if _, err := service.supportedConversation(ctx, transport, info.DID, id); err != nil {
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
