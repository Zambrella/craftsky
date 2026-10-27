package auth

import (
	"net/url"
	"strings"
)

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
