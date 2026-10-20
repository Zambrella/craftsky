package chat

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

type MessagePage struct {
	Items  []Message    `json:"items"`
	Cursor Continuation `json:"cursor,omitempty"`
}

func decodeMessage(raw json.RawMessage, owner, peer syntax.DID) (Message, error) {
	var envelope struct {
		Type   string    `json:"$type"`
		ID     MessageID `json:"id"`
		Rev    string    `json:"rev"`
		SentAt string    `json:"sentAt"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Message{}, errorsProviderCause(err)
	}
	if envelope.ID == "" || len(envelope.Rev) > 512 {
		return Message{}, ErrProviderResponse
	}
	sentAt, err := time.Parse(time.RFC3339Nano, envelope.SentAt)
	if err != nil {
		return Message{}, ErrProviderResponse
	}
	message := Message{ID: envelope.ID, Revision: envelope.Rev, Kind: "unsupported", SentAt: sentAt.UTC().Format(time.RFC3339Nano)}
	switch envelope.Type {
	case "chat.bsky.convo.defs#messageView":
		var view upstream.ConvoDefs_MessageView
		if err := json.Unmarshal(raw, &view); err != nil {
			return Message{}, errorsProviderCause(err)
		}
		mapped := messageFromView(&view, nil, nil)
		if mapped.SenderDID == owner || mapped.SenderDID == peer {
			message.Kind, message.SenderDID, message.Text = mapped.Kind, mapped.SenderDID, mapped.Text
		}
	case "chat.bsky.convo.defs#deletedMessageView":
		message.Kind = "deleted"
	}
	return message, nil
}

func (service *Service) History(ctx context.Context, info auth.AuthInfo, id ConversationID, cursor Continuation, limit int) (MessagePage, error) {
	result := MessagePage{Items: make([]Message, 0)}
	if !validOpaqueID(string(id)) {
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
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo", "chat.bsky.convo.getMessages"}, func(ctx context.Context, transport *Transport) error {
		connection, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		conversation, err := currentConversation(ctx, transport, info.DID, id)
		if err != nil {
			return err
		}
		peer, err := directPeer(info.DID, conversation)
		if err != nil {
			return err
		}
		rawCursor, err := service.resolveCursor(ctx, info.DID, connection.Generation, "history:"+string(id), cursor)
		if err != nil {
			return err
		}
		params := map[string]any{"convoId": string(id), "limit": int64(limit)}
		if rawCursor != "" {
			params["cursor"] = rawCursor
		}
		var page struct {
			Messages []json.RawMessage `json:"messages"`
			Cursor   *string           `json:"cursor"`
		}
		if err := transport.Query(ctx, "chat.bsky.convo.getMessages", params, &page); err != nil {
			return err
		}
		if page.Messages == nil || len(page.Messages) > limit {
			return ErrProviderResponse
		}
		for _, raw := range page.Messages {
			message, err := decodeMessage(raw, info.DID, peer)
			if err != nil {
				return err
			}
			result.Items = append(result.Items, message)
		}
		if cursor != "" {
			ids := make([]string, 0, len(result.Items))
			for _, message := range result.Items {
				ids = append(ids, string(message.ID))
			}
			if err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
				return chatdb.New(tx).RecordChatPageMessages(ctx, chatdb.RecordChatPageMessagesParams{OwnerDid: info.DID.String(), ID: pgtype.UUID{Bytes: uuid.MustParse(string(cursor)), Valid: true}, Scope: "history:" + string(id), Generation: connection.Generation, MessageIds: ids})
			}); err != nil {
				return err
			}
		}
		sort.SliceStable(result.Items, func(i, j int) bool {
			a, _ := time.Parse(time.RFC3339Nano, result.Items[i].SentAt)
			b, _ := time.Parse(time.RFC3339Nano, result.Items[j].SentAt)
			if a.Equal(b) {
				return result.Items[i].ID < result.Items[j].ID
			}
			return a.Before(b)
		})
		if page.Cursor != nil && *page.Cursor != "" {
			if *page.Cursor == rawCursor {
				return ErrProviderResponse
			}
			result.Cursor, err = service.retainCursor(ctx, info.DID, connection.Generation, "history:"+string(id), *page.Cursor)
		}
		return err
	})
	return result, err
}

func errorsProviderCause(cause error) error { return errors.Join(ErrProviderResponse, cause) }
