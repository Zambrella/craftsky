package auth

import (
	_ "embed"
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
	return strings.Fields(optionalChatScopes)
}

// Shared with scripts/oauth-permissions-check so exact-release verification
// expects the same optional authority as consumer metadata and authorization.
//
//go:embed optional_chat_scopes.txt
var optionalChatScopes string

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
