package auth

import (
	"net/url"
	"slices"
	"strings"
)

const (
	ChatServiceAudience = "did:web:api.bsky.chat#bsky_chat"
	ChatReportAudience  = "did:plc:ar7c4by46qjdydhdevvrndac#atproto_labeler"
)

// OptionalChatScopes contains only implemented v1 operations. Consumer login
// and registration request these alongside the independently required base.
// The report destination is deliberately distinct from the chat service.
func OptionalChatScopes() []string {
	methods := []string{
		"chat.bsky.actor.getStatus",
		"chat.bsky.convo.listConvos", "chat.bsky.convo.listConvoRequests",
		"chat.bsky.convo.getConvo", "chat.bsky.convo.getMessages",
		"chat.bsky.convo.getConvoAvailability", "chat.bsky.convo.getConvoForMembers",
		"chat.bsky.convo.sendMessage",
		"chat.bsky.convo.deleteMessageForSelf",
		"chat.bsky.convo.updateRead", "chat.bsky.convo.acceptConvo",
		"chat.bsky.convo.muteConvo", "chat.bsky.convo.unmuteConvo",
		"chat.bsky.convo.leaveConvo", "chat.bsky.convo.getLog",
	}
	scopes := make([]string, 0, len(methods)+2)
	for _, method := range methods {
		scopes = append(scopes, "rpc:"+method+"?aud=did:web:api.bsky.chat%23bsky_chat")
	}
	return append(scopes,
		"rpc:com.atproto.moderation.createReport?aud=did:plc:ar7c4by46qjdydhdevvrndac%23atproto_labeler",
		"repo:chat.bsky.actor.declaration?action=create&action=update",
	)
}

func advertisedConsumerScopes(base []string) []string {
	advertised := slices.Clone(base)
	for _, optional := range OptionalChatScopes() {
		if !slices.Contains(advertised, optional) {
			advertised = append(advertised, optional)
		}
	}
	return advertised
}

// DefaultScopes grants only the collections and media operations CraftSky uses.
// Reads of public records require no additional authority. Video processing uses
// method-restricted service tokens, including uploads to the user's own PDS.
func DefaultScopes() []string {
	return []string{
		"atproto",
		"include:social.craftsky.authFull",
		"repo:app.bsky.actor.profile?action=create&action=update",
		"repo:app.bsky.graph.follow",
		"repo:app.bsky.graph.block",
		"blob?accept=image/jpeg&accept=image/png&accept=image/webp&accept=video/mp4&accept=text/vtt",
		"rpc:com.atproto.repo.uploadBlob?aud=*",
		"rpc:app.bsky.video.getUploadLimits?aud=*",
		"repo:app.bsky.actor.link",
	}
}

// usesBroadOAuthAuthority recognizes both positional and parameter spellings.
func usesBroadOAuthAuthority(scope string) bool {
	if strings.HasPrefix(scope, "transition:") {
		return true
	}
	resource, rawQuery, _ := strings.Cut(scope, "?")
	name, positional, _ := strings.Cut(resource, ":")
	if name != "repo" {
		return false
	}
	collection, _ := url.QueryUnescape(positional)
	if collection == "*" {
		return true
	}
	query, _ := url.ParseQuery(rawQuery)
	for _, value := range query["collection"] {
		if value == "*" {
			return true
		}
	}
	return false
}
