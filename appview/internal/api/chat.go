package api

import (
	"context"
	"encoding/json"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chat"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

type ChatStatusReader interface {
	Status(context.Context, auth.AuthInfo) (chat.Status, error)
}

func ChatStatusHandler(service ChatStatusReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Query()) != 0 {
			envelope.WriteError(w, 400, "invalid_request", "unexpected messaging parameters", middleware.GetRunID(r.Context()), nil)
			return
		}
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, 401, "unauthorized", "authentication required", middleware.GetRunID(r.Context()), nil)
			return
		}
		parent, _ := middleware.GetOAuthSessionID(r.Context())
		if service == nil {
			envelope.WriteError(w, 503, "chat_unavailable", "messaging unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		hash, _ := middleware.GetChildTokenHash(r.Context())
		status, err := service.Status(r.Context(), auth.AuthInfo{DID: did, SessionID: parent, ChildTokenHash: hash})
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.ChatStatusHandler", "handler")
			envelope.WriteError(w, 503, "chat_unavailable", "messaging unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(status)
	})
}
