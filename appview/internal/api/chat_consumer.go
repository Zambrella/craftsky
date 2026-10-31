package api

import (
	"context"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chat"
	"social.craftsky/appview/internal/observability"
	"strconv"
)

// ChatConsumer admits fixed domain operations, never arbitrary XRPC controls.
type ChatConsumer interface {
	SetConnection(context.Context, auth.AuthInfo, bool) (chat.Status, error)
	ListConversations(context.Context, auth.AuthInfo, string, chat.Continuation, int) (chat.ConversationPage, error)
	Availability(context.Context, auth.AuthInfo, syntax.DID) (chat.Availability, error)
	StartConversation(context.Context, auth.AuthInfo, syntax.DID) (chat.Conversation, error)
	Conversation(context.Context, auth.AuthInfo, chat.ConversationID) (chat.Conversation, error)
	Details(context.Context, auth.AuthInfo, chat.ConversationID) (chat.ConversationDetails, error)
	History(context.Context, auth.AuthInfo, chat.ConversationID, chat.Continuation, int) (chat.MessagePage, error)
	Receipt(context.Context, auth.AuthInfo, uuid.UUID) (chat.OperationReceipt, error)
	UpdateRead(context.Context, auth.AuthInfo, chat.ConversationID, chat.MessageID) (chat.Conversation, error)
	Accept(context.Context, auth.AuthInfo, chat.ConversationID) (chat.Conversation, error)
	SetMuted(context.Context, auth.AuthInfo, chat.ConversationID, bool) (chat.Conversation, error)
	Leave(context.Context, auth.AuthInfo, chat.ConversationID) error
	DeleteMessage(context.Context, auth.AuthInfo, chat.ConversationID, chat.MessageID) error
	Report(context.Context, auth.AuthInfo, uuid.UUID, chat.ConversationID, chat.MessageID, string, *string) (chat.OperationReceipt, error)
	ReportConversation(context.Context, auth.AuthInfo, uuid.UUID, chat.ConversationID, string, *string) (chat.OperationReceipt, error)
	UnreadCounts(context.Context, auth.AuthInfo) (chat.UnreadCounts, error)
	SetIncomingPolicy(context.Context, auth.AuthInfo, string) (chat.IncomingPolicy, error)
	IncomingPolicy(context.Context, auth.AuthInfo) (chat.IncomingPolicy, error)
	PushPreference(context.Context, auth.AuthInfo) (chat.PushPreference, error)
	SetPushPreference(context.Context, auth.AuthInfo, bool) (chat.PushPreference, error)
	Heartbeat(context.Context, auth.AuthInfo, chat.ViewActivity) (chat.SyncSnapshot, error)
	NotificationOpen(context.Context, auth.AuthInfo, uuid.UUID) (chat.NotificationTarget, error)
}

func chatQuery(r *http.Request, allowed ...string) error {
	query, err := urlQuery(r)
	if err != nil {
		return errChatInvalidRequest
	}
	for key, values := range query {
		found := false
		for _, name := range allowed {
			if key == name {
				found = true
				break
			}
		}
		if !found || len(values) != 1 {
			return errChatInvalidRequest
		}
	}
	return nil
}
func urlQuery(r *http.Request) (map[string][]string, error) { return url.ParseQuery(r.URL.RawQuery) }
func chatPageQuery(r *http.Request) (chat.Continuation, int, error) {
	limit := 50
	if raw, ok := r.URL.Query()["limit"]; ok {
		parsed, err := strconv.Atoi(raw[0])
		if err != nil || parsed < 1 || parsed > 100 {
			return "", 0, errChatInvalidRequest
		}
		limit = parsed
	}
	cursor, err := chat.ParseContinuation(r.URL.Query().Get("cursor"))
	return cursor, limit, err
}
func chatEmptyInput(r *http.Request) error {
	if err := chatQuery(r); err != nil {
		return err
	}
	if r.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1))
	if err != nil {
		return err
	}
	if len(raw) != 0 {
		return errChatInvalidRequest
	}
	return nil
}

// action is selected solely by the checked-in route catalogue.
func ChatConsumerHandler(service ChatConsumer, action string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		info, ok := chatAuthority(r)
		if !ok {
			writeChatError(w, r, auth.ErrAuthTokenInvalid)
			return
		}
		ctx := r.Context()
		var execute func() (any, error)
		var err error
		var id chat.ConversationID
		switch action {
		case "conversation", "details", "history", "read", "accept", "mute", "unmute", "leave", "deleteMessage":
			id, err = chat.ParseConversationID(r.PathValue("id"))
		}
		if err != nil {
			writeChatError(w, r, err)
			return
		}
		switch action {
		case "list", "history":
			keys := []string{"cursor", "limit"}
			if action == "list" {
				keys = append(keys, "status")
			}
			err = chatQuery(r, keys...)
			if err != nil {
				break
			}
			var cursor chat.Continuation
			var limit int
			cursor, limit, err = chatPageQuery(r)
			if err != nil {
				break
			}
			if action == "list" {
				status := r.URL.Query().Get("status")
				if status != "accepted" && status != "request" {
					err = errChatInvalidRequest
					break
				}
				execute = func() (any, error) { return service.ListConversations(ctx, info, status, cursor, limit) }
			} else {
				execute = func() (any, error) { return service.History(ctx, info, id, cursor, limit) }
			}
		case "availability", "start":
			var peer syntax.DID
			if action == "availability" {
				err = chatQuery(r, "recipientDid")
				if err == nil {
					peer, err = syntax.ParseDID(r.URL.Query().Get("recipientDid"))
				}
			} else {
				var request struct {
					RecipientDID syntax.DID `json:"recipientDid"`
				}
				err = decodeChatJSON(w, r, &request)
				peer = request.RecipientDID
			}
			if peer == "" || peer == info.DID {
				err = errChatInvalidRequest
			}
			if action == "availability" {
				execute = func() (any, error) { return service.Availability(ctx, info, peer) }
			} else {
				execute = func() (any, error) { return service.StartConversation(ctx, info, peer) }
			}
		case "conversation", "details", "counts", "receipt":
			err = chatEmptyInput(r)
			if err != nil {
				break
			}
			if action == "conversation" {
				execute = func() (any, error) { return service.Conversation(ctx, info, id) }
			}
			if action == "details" {
				execute = func() (any, error) { return service.Details(ctx, info, id) }
			}
			if action == "counts" {
				execute = func() (any, error) { return service.UnreadCounts(ctx, info) }
			}
			if action == "receipt" {
				var operation uuid.UUID
				operation, err = uuid.Parse(r.PathValue("operationId"))
				if operation == uuid.Nil {
					err = errChatInvalidRequest
				}
				execute = func() (any, error) { return service.Receipt(ctx, info, operation) }
			}
		case "read":
			var request struct {
				MessageID chat.MessageID `json:"messageId"`
			}
			err = decodeChatJSON(w, r, &request)
			if request.MessageID == "" {
				err = errChatInvalidRequest
			}
			execute = func() (any, error) { return service.UpdateRead(ctx, info, id, request.MessageID) }
		case "accept", "mute", "unmute", "leave":
			err = chatEmptyInput(r)
			switch action {
			case "accept":
				execute = func() (any, error) { return service.Accept(ctx, info, id) }
			case "mute", "unmute":
				execute = func() (any, error) { return service.SetMuted(ctx, info, id, action == "mute") }
			case "leave":
				execute = func() (any, error) {
					err := service.Leave(ctx, info, id)
					return struct {
						Left bool `json:"left"`
					}{err == nil}, err
				}
			}
		case "deleteMessage":
			err = chatEmptyInput(r)
			message, parseErr := chat.ParseMessageID(r.PathValue("messageId"))
			if parseErr != nil {
				err = errChatInvalidRequest
			}
			execute = func() (any, error) {
				err := service.DeleteMessage(ctx, info, id, message)
				return struct {
					Deleted bool `json:"deleted"`
				}{err == nil}, err
			}
		case "connection":
			var request struct {
				Enabled *bool `json:"enabled"`
			}
			err = decodeChatJSON(w, r, &request)
			if request.Enabled == nil {
				err = errChatInvalidRequest
			}
			execute = func() (any, error) { return service.SetConnection(ctx, info, *request.Enabled) }
		case "setPolicy":
			var request struct {
				AllowIncoming string `json:"allowIncoming"`
			}
			err = decodeChatJSON(w, r, &request)
			if request.AllowIncoming != "everyone" && request.AllowIncoming != "following" && request.AllowIncoming != "nobody" {
				err = errChatInvalidRequest
			}
			execute = func() (any, error) { return service.SetIncomingPolicy(ctx, info, request.AllowIncoming) }
		case "policy", "pushPreference":
			err = chatEmptyInput(r)
			if action == "policy" {
				execute = func() (any, error) { return service.IncomingPolicy(ctx, info) }
			} else {
				execute = func() (any, error) { return service.PushPreference(ctx, info) }
			}
		case "setPushPreference":
			var request struct {
				PushEnabled *bool `json:"pushEnabled"`
			}
			err = decodeChatJSON(w, r, &request)
			if request.PushEnabled == nil {
				err = errChatInvalidRequest
			}
			execute = func() (any, error) { return service.SetPushPreference(ctx, info, *request.PushEnabled) }
		case "sync":
			var request chat.ViewActivity
			err = decodeChatJSON(w, r, &request)
			if request.View != "conversation" && request.View != "inbox" && request.View != "none" {
				err = errChatInvalidRequest
			}
			if (request.View == "conversation") != (request.ConversationID != "") || (request.KnownRevision != nil && *request.KnownRevision < 0) {
				err = errChatInvalidRequest
			}
			execute = func() (any, error) { return service.Heartbeat(ctx, info, request) }
		case "open":
			err = chatEmptyInput(r)
			if err != nil {
				break
			}
			var reference uuid.UUID
			reference, err = uuid.Parse(r.PathValue("reference"))
			if err != nil || reference == uuid.Nil {
				err = errChatInvalidRequest
			}
			execute = func() (any, error) { return service.NotificationOpen(ctx, info, reference) }
		case "report":
			var request struct {
				OperationID    uuid.UUID           `json:"operationId"`
				ConversationID chat.ConversationID `json:"conversationId"`
				MessageID      *chat.MessageID     `json:"messageId"`
				SubjectType    string              `json:"subjectType"`
				ReasonType     string              `json:"reasonType"`
				Reason         *string             `json:"reason"`
			}
			err = decodeChatJSON(w, r, &request)
			if err == nil {
				err = chat.ValidateReportInput(request.ReasonType, request.Reason)
			}
			if request.OperationID == uuid.Nil || request.ConversationID == "" {
				err = errChatInvalidRequest
			}
			switch request.SubjectType {
			case "conversation":
				if request.MessageID != nil {
					err = errChatInvalidRequest
				}
				execute = func() (any, error) {
					return service.ReportConversation(ctx, info, request.OperationID, request.ConversationID, request.ReasonType, request.Reason)
				}
			case "", "message":
				if request.MessageID == nil || *request.MessageID == "" {
					err = errChatInvalidRequest
				}
				execute = func() (any, error) {
					return service.Report(ctx, info, request.OperationID, request.ConversationID, *request.MessageID, request.ReasonType, request.Reason)
				}
			default:
				err = errChatInvalidRequest
			}
		default:
			err = errChatInvalidRequest
		}
		if err != nil {
			writeChatError(w, r, err)
			return
		}
		if service == nil || execute == nil {
			writeChatError(w, r, chat.ErrOperationUnavailable)
			return
		}
		result, err := execute()
		status := 200
		if receipt, ok := result.(chat.OperationReceipt); ok && receipt.State != "" {
			if receipt.State == "uncertain" {
				status = 202
			}
			if err != nil && receipt.State == "uncertain" {
				observability.ReportRequestFailure(ctx, err, "api.ChatConsumerHandler", "report")
			}
		} else if err != nil {
			writeChatError(w, r, err)
			return
		}
		envelope.WriteJSON(w, status, result)
	})
}
