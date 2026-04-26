// appview/internal/index/bluesky_backfiller.go
package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/tap"
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

// blueskyBackfiller implements BlueskyBackfiller.
type blueskyBackfiller struct {
	reader  auth.PDSClient
	indexer *BlueskyProfile
}

// NewBlueskyBackfiller wires an anonymous PDS reader to the Bluesky
// indexer. Callers pass their existing *BlueskyProfile; the backfiller
// dispatches synthesised events through it so parse, gate, and upsert
// logic stay in one place.
func NewBlueskyBackfiller(reader auth.PDSClient, indexer *BlueskyProfile) BlueskyBackfiller {
	return &blueskyBackfiller{reader: reader, indexer: indexer}
}

// Backfill fetches the user's app.bsky.actor.profile/self record from
// their PDS and feeds it to BlueskyProfile.Handle as a synthesised
// tap.Event{Action:"create"}. A missing Bluesky record (ErrRecordNotFound)
// is a no-op and returns nil — many users don't have one.
func (b *blueskyBackfiller) Backfill(ctx context.Context, did syntax.DID) error {
	var rec map[string]any
	cid, err := b.reader.GetRecord(ctx, did, "app.bsky.actor.profile", "self", &rec)
	if errors.Is(err, auth.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("backfill fetch %s: %w", did, err)
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("backfill marshal %s: %w", did, err)
	}
	ev := tap.Event{
		URI:        syntax.ATURI("at://" + did.String() + "/app.bsky.actor.profile/self"),
		CID:        syntax.CID(cid),
		DID:        did,
		Collection: "app.bsky.actor.profile",
		Rkey:       "self",
		Action:     "create",
		Record:     raw,
	}
	return b.indexer.Handle(ctx, ev)
}
