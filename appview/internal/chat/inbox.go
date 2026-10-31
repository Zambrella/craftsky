package chat

import (
	"context"

	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

type ConversationPage struct {
	Items  []Conversation `json:"items"`
	Cursor Continuation   `json:"cursor,omitempty"`
}

func (service *Service) ListConversations(ctx context.Context, info auth.AuthInfo, bucket string, cursor Continuation, limit int) (ConversationPage, error) {
	result := ConversationPage{Items: make([]Conversation, 0)}
	if bucket != "accepted" && bucket != "request" {
		return result, ErrInvalidIdentifier
	}
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return result, ErrInvalidIdentifier
	}
	if _, err := ParseContinuation(string(cursor)); err != nil {
		return result, err
	}
	method := syntax.NSID("chat.bsky.convo.listConvos")
	if bucket == "request" {
		method = "chat.bsky.convo.listConvoRequests"
	}
	err := service.withRPC(ctx, info, []syntax.NSID{method}, func(ctx context.Context, transport *Transport) error {
		connection, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		raw, err := service.resolveCursor(ctx, info.DID, connection.Generation, "list:"+bucket, cursor)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for range 5 {
			seen[raw] = true
			params := map[string]any{"limit": int64(limit)}
			if raw != "" {
				params["cursor"] = raw
			}
			var views []*upstream.ConvoDefs_ConvoView
			var next *string
			if bucket == "accepted" {
				params["kind"], params["status"] = "direct", "accepted"
				var page upstream.ConvoListConvos_Output
				if err := transport.Query(ctx, method, params, &page); err != nil {
					return err
				}
				if page.Convos == nil || len(page.Convos) > limit {
					return ErrProviderResponse
				}
				views, next = page.Convos, page.Cursor
			} else {
				var page upstream.ConvoListConvoRequests_Output
				if err := transport.Query(ctx, method, params, &page); err != nil {
					return err
				}
				if page.Requests == nil || len(page.Requests) > limit {
					return ErrProviderResponse
				}
				next = page.Cursor
				for _, request := range page.Requests {
					if request != nil && request.ConvoDefs_ConvoView != nil {
						views = append(views, request.ConvoDefs_ConvoView)
					}
				}
			}
			views, err = service.profileConversations(ctx, info.DID, views)
			if err != nil {
				return err
			}
			for _, view := range views {
				conversation, err := conversationFromView(info.DID, view)
				if err == nil && conversation.Status == bucket {
					result.Items = append(result.Items, conversation)
				}
			}
			raw = ""
			if next != nil {
				raw = *next
			}
			if raw != "" && seen[raw] {
				return ErrProviderResponse
			}
			if raw == "" || len(result.Items) > 0 {
				break
			}
		}
		result.Cursor, err = service.retainCursor(ctx, info.DID, connection.Generation, "list:"+bucket, raw)
		return err
	})
	return result, err
}
