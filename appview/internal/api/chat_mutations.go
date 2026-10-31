package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chat"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
)

var errChatInvalidRequest = errors.New("invalid messaging request")

type ChatSender interface {
	Send(context.Context, auth.AuthInfo, uuid.UUID, chat.ConversationID, string) (chat.SendOutcome, error)
}

func chatAuthority(r *http.Request) (auth.AuthInfo, bool) {
	did, didOK := middleware.GetDID(r.Context())
	parent, parentOK := middleware.GetOAuthSessionID(r.Context())
	hash, hashOK := middleware.GetChildTokenHash(r.Context())
	return auth.AuthInfo{DID: did, SessionID: parent, ChildTokenHash: hash}, didOK && parentOK && parent != "" && hashOK && hash != ([32]byte{})
}

func decodeChatJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if len(r.URL.Query()) != 0 {
		return errChatInvalidRequest
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		return errors.Join(errChatInvalidRequest, err)
	}
	if !chat.ValidJSONUnicode(raw) {
		return errChatInvalidRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.Join(errChatInvalidRequest, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.Join(errChatInvalidRequest, err)
	}
	return nil
}

func writeChatError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := 503, "chat_unavailable", "messaging unavailable"
	var invalidText chat.TextValidationError
	var large *http.MaxBytesError
	var provider *atclient.APIError
	switch {
	case errors.As(err, &large):
		status, code, message = 413, "body_too_large", "messaging request too large"
	case errors.Is(err, errChatInvalidRequest), errors.Is(err, chat.ErrInvalidIdentifier), errors.Is(err, chat.ErrInvalidCursor), errors.As(err, &invalidText), errors.Is(err, chat.ErrUnsupportedReportReason):
		status, code, message = 400, "invalid_request", "invalid messaging request"
	case errors.Is(err, auth.ErrAuthTokenInvalid), errors.Is(err, auth.ErrOAuthSessionNotFound), errors.Is(err, auth.ErrPDSSessionExpired), errors.Is(err, ownerlifecycle.ErrOwnerNotActive), errors.Is(err, ownerlifecycle.ErrTerminalOwner):
		status, code, message = 401, "unauthorized", "authentication required"
	case errors.Is(err, auth.ErrPDSPermissionsRequired):
		status, code, message = 403, "chat_authorization_required", "messaging authorization required"
	case errors.Is(err, chat.ErrDisabled):
		status, code, message = 409, "chat_disconnected", "messaging is disconnected"
	case errors.Is(err, chat.ErrConnectionChanged), errors.Is(err, ownerlifecycle.ErrGenerationChanged):
		status, code, message = 409, "chat_connection_changed", "messaging connection changed"
	case errors.Is(err, chat.ErrRestricted):
		status, code, message = 403, "chat_restricted", "messaging is restricted"
	case errors.Is(err, chat.ErrBlocked):
		status, code, message = 403, "chat_relationship_unavailable", "messaging unavailable for this relationship"
	case errors.Is(err, chat.ErrRequestNotAccepted):
		status, code, message = 409, "chat_request_not_accepted", "accept this conversation before sending"
	case errors.Is(err, chat.ErrOperationConflict):
		status, code, message = 409, "operation_conflict", "operation ID belongs to a different request"
	case errors.Is(err, chat.ErrInaccessibleConversation):
		status, code, message = 404, "chat_conversation_unavailable", "conversation or message unavailable"
	case errors.As(err, &provider) && provider.StatusCode == 429:
		status, code, message = 429, "rate_limited", "messaging is temporarily rate limited"
	case errors.As(err, &provider) && provider.StatusCode >= 400 && provider.StatusCode < 500:
		status, code, message = 400, "chat_operation_rejected", "messaging operation was not accepted"
	default:
		observability.ReportRequestFailure(r.Context(), err, "api.chat", "handler")
	}
	var limited *auth.RateLimitError
	if status == http.StatusTooManyRequests && errors.As(err, &limited) {
		w.Header().Set("Retry-After", strconv.Itoa(limited.RetrySeconds()))
	}
	envelope.WriteError(w, status, code, message, middleware.GetRunID(r.Context()), nil)
}

func ChatSendHandler(service ChatSender) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		info, ok := chatAuthority(r)
		if !ok {
			writeChatError(w, r, auth.ErrAuthTokenInvalid)
			return
		}
		conversation, err := chat.ParseConversationID(r.PathValue("id"))
		if err != nil {
			writeChatError(w, r, err)
			return
		}
		var request struct {
			OperationID uuid.UUID `json:"operationId"`
			Text        string    `json:"text"`
		}
		if err := decodeChatJSON(w, r, &request); err != nil {
			writeChatError(w, r, err)
			return
		}
		if request.OperationID == uuid.Nil {
			writeChatError(w, r, errChatInvalidRequest)
			return
		}
		if err := chat.ValidateMessageText(request.Text); err != nil {
			writeChatError(w, r, err)
			return
		}
		if service == nil {
			writeChatError(w, r, chat.ErrOperationUnavailable)
			return
		}
		outcome, err := service.Send(r.Context(), info, request.OperationID, conversation, request.Text)
		switch outcome.State {
		case "failed":
			// An authoritative operation receipt is a successful protocol result,
			// including repeated reads of a previously rejected operation.
			envelope.WriteJSON(w, 200, outcome)
		case "confirmed":
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.ChatSendHandler", "finalization")
			}
			envelope.WriteJSON(w, 200, outcome)
		case "uncertain", "prepared", "attempting":
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.ChatSendHandler", "send")
			}
			envelope.WriteJSON(w, 202, outcome)
		default:
			if err == nil {
				err = chat.ErrOperationUnavailable
			}
			writeChatError(w, r, err)
		}
	})
}
