package auth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"log/slog"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

type ChatAuthorizationStarter interface {
	StartChatAuthorization(context.Context, AuthInfo, string, HandoffMode, string, string, *bool) (string, error)
}

func ChatAuthorizationHandler(starter ChatAuthorizationStarter, allowDev bool, loggers ...*slog.Logger) http.Handler {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail := func(status int, code, message string) {
			envelope.WriteError(w, status, code, message, ctxkeys.GetRunID(r.Context()), nil)
		}
		if len(r.URL.Query()) != 0 {
			fail(400, "invalid_request", "unexpected messaging parameters")
			return
		}
		var request struct {
			HandoffMode HandoffMode `json:"handoffMode"`
			Loopback    string      `json:"loopbackRedirectUri,omitempty"`
			PushEnabled *bool       `json:"pushEnabled,omitempty"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			var large *http.MaxBytesError
			if errors.As(err, &large) {
				fail(413, "body_too_large", "messaging request too large")
			} else {
				fail(400, "invalid_request", "invalid messaging authorization")
			}
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			fail(400, "invalid_request", "invalid messaging authorization")
			return
		}
		if request.HandoffMode != HandoffVerifiedLink && request.HandoffMode != HandoffLoopback && !(request.HandoffMode == HandoffDevScheme && allowDev) {
			fail(400, "invalid_handoff_mode", "handoff mode unavailable")
			return
		}
		if request.HandoffMode == HandoffLoopback {
			if _, err := exactLoopbackOrigin(request.Loopback); err != nil {
				fail(400, "invalid_handoff_mode", "invalid loopback handoff")
				return
			}
		} else if request.Loopback != "" {
			fail(400, "invalid_handoff_mode", "unexpected loopback handoff")
			return
		}
		owner, ownerOK := ctxkeys.GetDID(r.Context())
		parent, parentOK := ctxkeys.GetOAuthSessionID(r.Context())
		device, deviceOK := ctxkeys.GetDeviceID(r.Context())
		hash, hashOK := ctxkeys.GetChildTokenHash(r.Context())
		bearer := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if !ownerOK || !parentOK || !deviceOK || !hashOK || bearer == "" || sha256.Sum256([]byte(bearer)) != hash {
			fail(401, "unauthorized", "authentication required")
			return
		}
		if starter == nil {
			fail(503, "chat_unavailable", "messaging authorization unavailable")
			return
		}
		redirect, err := starter.StartChatAuthorization(r.Context(), AuthInfo{DID: owner, SessionID: parent, ChildTokenHash: hash}, bearer, request.HandoffMode, request.Loopback, device, request.PushEnabled)
		if err != nil {
			if errors.Is(err, ErrOAuthFlowInvalid) {
				fail(401, "unauthorized", "messaging authorization expired")
				return
			}
			logger.Warn("messaging authorization failed", authLogErrorAttrs(r.Context(), ctxkeys.GetRunID(r.Context()), "chat.authorization", "authorization", err)...)
			fail(503, "chat_unavailable", "messaging authorization unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			AuthURL string `json:"authUrl"`
		}{redirect})
	})
}
