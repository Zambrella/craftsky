// appview/internal/index/bluesky_backfiller.go
package index

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// BlueskyBackfiller eagerly populates bluesky_profiles for a newly-
// onboarded Craftsky member by fetching their app.bsky.actor.profile
// record from their PDS and feeding it back through BlueskyProfile.Handle.
//
// This exists to sidestep a race during Tap backfill: the MST emits
// records in key-sorted order, so app.bsky.actor.profile arrives before
// social.craftsky.actor.profile and is dropped by the membership gate.
// CraftskyProfile.Handle invokes Backfill only when it commits a
// genuinely new membership row (see craftsky_profile.go's xmax check).
//
// Implementations must tolerate a missing Bluesky record (many users
// won't have one) and return nil in that case.
type BlueskyBackfiller interface {
	Backfill(ctx context.Context, did syntax.DID) error
}
