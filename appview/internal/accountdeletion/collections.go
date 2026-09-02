package accountdeletion

import "github.com/bluesky-social/indigo/atproto/syntax"

var craftskyRecordCollections = [...]syntax.NSID{
	"social.craftsky.feed.post",
	"social.craftsky.feed.like",
	"social.craftsky.feed.repost",
	"social.craftsky.business.event",
	"social.craftsky.business.profile",
	"social.craftsky.actor.profile",
}

// CraftskyRecordCollections returns every CraftSky PDS record collection in
// deletion order. The membership-defining profile is deliberately last.
func CraftskyRecordCollections() []syntax.NSID {
	collections := make([]syntax.NSID, len(craftskyRecordCollections))
	copy(collections, craftskyRecordCollections[:])
	return collections
}
