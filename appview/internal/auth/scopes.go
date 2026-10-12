package auth

// DefaultScopes grants only the collections and media operations CraftSky uses.
// Reads of public records require no additional authority. Video processing uses
// method-restricted service tokens, including uploads to the user's own PDS.
func DefaultScopes() []string {
	return []string{
		"atproto",
		"repo:social.craftsky.actor.profile",
		"repo:social.craftsky.feed.post",
		"repo:social.craftsky.feed.like",
		"repo:social.craftsky.feed.repost",
		"repo:social.craftsky.business.profile",
		"repo:social.craftsky.business.event",
		"repo:app.bsky.actor.profile?action=create&action=update",
		"repo:app.bsky.graph.follow",
		"repo:app.bsky.graph.block",
		"blob?accept=image/jpeg&accept=image/png&accept=image/webp&accept=video/mp4&accept=text/vtt",
		"rpc:com.atproto.repo.uploadBlob?aud=*",
		"rpc:app.bsky.video.getUploadLimits?aud=*",
	}
}
