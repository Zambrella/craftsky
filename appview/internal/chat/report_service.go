package chat

import (
	"context"
	"errors"
	"strconv"

	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

func currentConversation(ctx context.Context, transport *Transport, owner syntax.DID, id ConversationID) (*upstream.ConvoDefs_ConvoView, error) {
	if !validOpaqueID(string(id)) {
		return nil, ErrInaccessibleConversation
	}
	var response upstream.ConvoGetConvo_Output
	if err := transport.Query(ctx, "chat.bsky.convo.getConvo", map[string]any{"convoId": string(id)}, &response); err != nil {
		var provider *atclient.APIError
		if errors.As(err, &provider) && (provider.StatusCode == 403 || provider.StatusCode == 404 || provider.StatusCode == 410) {
			return nil, errors.Join(ErrInaccessibleConversation, err)
		}
		return nil, err
	}
	if _, err := directPeer(owner, response.Convo); err != nil {
		return nil, err
	}
	if response.Convo.Id != string(id) {
		return nil, ErrInaccessibleConversation
	}
	return response.Convo, nil
}

func (service *Service) supportedConversation(ctx context.Context, transport *Transport, owner syntax.DID, id ConversationID) (*upstream.ConvoDefs_ConvoView, error) {
	view, err := currentConversation(ctx, transport, owner, id)
	if err != nil {
		return nil, err
	}
	peer, _ := directPeer(owner, view)
	if err := service.requireProfiles(ctx, owner, peer); err != nil {
		return nil, err
	}
	return view, nil
}

func currentMessage(ctx context.Context, transport *Transport, conversation ConversationID, messageID MessageID, pageHint string) (*upstream.ConvoDefs_MessageView, error) {
	if !validOpaqueID(string(messageID)) {
		return nil, ErrInaccessibleConversation
	}
	cursor := pageHint
	budget := 5
	if pageHint != "" {
		budget = 1
	}
	for range budget {
		params := map[string]any{"convoId": string(conversation), "limit": int64(100)}
		if cursor != "" {
			params["cursor"] = cursor
		}
		var page upstream.ConvoGetMessages_Output
		if err := transport.Query(ctx, "chat.bsky.convo.getMessages", params, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Messages {
			if item != nil && item.ConvoDefs_MessageView != nil && item.ConvoDefs_MessageView.Id == string(messageID) {
				return item.ConvoDefs_MessageView, nil
			}
		}
		if page.Cursor == nil || *page.Cursor == "" || *page.Cursor == cursor {
			break
		}
		cursor = *page.Cursor
	}
	return nil, ErrInaccessibleConversation
}

func (service *Service) Report(ctx context.Context, info auth.AuthInfo, id uuid.UUID, conversationID ConversationID, messageID MessageID, reasonType string, reason *string) (OperationReceipt, error) {
	if err := ValidateReportInput(reasonType, reason); err != nil {
		return OperationReceipt{}, err
	}
	if id == uuid.Nil || !validOpaqueID(string(conversationID)) || !validOpaqueID(string(messageID)) {
		return OperationReceipt{}, ErrInaccessibleConversation
	}
	if service.operations == nil {
		return OperationReceipt{}, ErrOperationUnavailable
	}
	var receipt OperationReceipt
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo", "chat.bsky.convo.getMessages", "com.atproto.moderation.createReport"}, func(ctx context.Context, transport *Transport) error {
		connection, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		conversation, err := service.supportedConversation(ctx, transport, info.DID, conversationID)
		if err != nil {
			return err
		}
		var hint string
		if err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			var err error
			hint, err = chatdb.New(tx).FindChatMessagePage(ctx, chatdb.FindChatMessagePageParams{OwnerDid: info.DID.String(), Scope: "history:" + string(conversationID), Generation: connection.Generation, Column4: string(messageID)})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}); err != nil {
			return err
		}
		message, err := currentMessage(ctx, transport, conversationID, messageID, hint)
		if err != nil {
			return err
		}
		evidence, err := BuildReportEvidence(info.DID, conversation, message)
		if err != nil {
			return err
		}
		body := struct {
			ReasonType string                         `json:"reasonType"`
			Reason     *string                        `json:"reason,omitempty"`
			Subject    *upstream.ConvoDefs_MessageRef `json:"subject"`
		}{reasonType, reason, evidence}
		receipt, err = service.operations.Execute(ctx, info.DID, id, "report", body, func(ctx context.Context) (OperationResult, error) {
			var response struct {
				ID int64 `json:"id"`
			}
			if err := transport.Procedure(ctx, "com.atproto.moderation.createReport", body, &response); err != nil {
				return OperationResult{}, err
			}
			if response.ID <= 0 {
				return OperationResult{}, ErrProviderResponse
			}
			return OperationResult{ID: strconv.FormatInt(response.ID, 10)}, nil
		})
		return err
	})
	return receipt, err
}
