package chat

import (
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/url"
	"social.craftsky/appview/internal/auth"
	"strings"
)

type Capabilities struct {
	CanRead      bool `json:"canRead"`
	CanSend      bool `json:"canSend"`
	CanReport    bool `json:"canReport"`
	CanSetPolicy bool `json:"canSetPolicy"`
}

func coreChatScopes() []string {
	var required []string
	for _, scope := range auth.OptionalChatScopes() {
		if strings.HasPrefix(scope, "rpc:chat.bsky.") {
			required = append(required, scope)
		}
	}
	return required
}

func rpcScopes(methods ...syntax.NSID) []string {
	result := make([]string, 0, len(methods))
	for _, method := range methods {
		audience := auth.ChatServiceAudience
		if method == "com.atproto.moderation.createReport" {
			audience = auth.ChatReportAudience
		}
		result = append(result, "rpc:"+method.String()+"?aud="+url.QueryEscape(audience))
	}
	return result
}

func EvaluateCapabilities(scopes []string) Capabilities {
	return Capabilities{
		CanRead:      auth.HasEffectiveScopes(scopes, rpcScopes("chat.bsky.actor.getStatus", "chat.bsky.convo.listConvos", "chat.bsky.convo.listConvoRequests", "chat.bsky.convo.getConvo", "chat.bsky.convo.getMessages", "chat.bsky.convo.getLog")),
		CanSend:      auth.HasEffectiveScopes(scopes, rpcScopes("chat.bsky.convo.getConvoAvailability", "chat.bsky.convo.getConvoForMembers", "chat.bsky.convo.sendMessage")),
		CanReport:    auth.HasEffectiveScopes(scopes, rpcScopes("com.atproto.moderation.createReport")),
		CanSetPolicy: auth.HasEffectiveScopes(scopes, []string{"repo:chat.bsky.actor.declaration?action=create&action=update"}),
	}
}
