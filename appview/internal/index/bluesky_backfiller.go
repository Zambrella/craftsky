// appview/internal/index/bluesky_backfiller.go
package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	reader   auth.PDSClient
	indexer  *BlueskyProfile
	tracker  tap.RepositoryTracker
	observer RelationshipObserver
}

// NewBlueskyBackfiller wires an anonymous PDS reader to the Bluesky
// indexer. Callers pass their existing *BlueskyProfile; the backfiller
// dispatches synthesised events through it so parse, gate, and upsert
// logic stay in one place.
func NewBlueskyBackfiller(reader auth.PDSClient, indexer *BlueskyProfile, trackers ...tap.RepositoryTracker) BlueskyBackfiller {
	var tracker tap.RepositoryTracker
	if len(trackers) > 0 {
		tracker = trackers[0]
	}
	return &blueskyBackfiller{reader: reader, indexer: indexer, tracker: tracker}
}

// NewObservedBlueskyBackfiller adds bounded relationship backfill telemetry to
// the ordinary repository-tracking path without changing its behavior.
func NewObservedBlueskyBackfiller(reader auth.PDSClient, indexer *BlueskyProfile, tracker tap.RepositoryTracker, observer RelationshipObserver) BlueskyBackfiller {
	return &blueskyBackfiller{reader: reader, indexer: indexer, tracker: tracker, observer: observer}
}

// Backfill fetches the user's app.bsky.actor.profile/self record from
// their PDS and feeds it to BlueskyProfile.Handle as a synthesised
// tap.Event{Action:"create"}. A missing Bluesky record (ErrRecordNotFound)
// is a no-op and returns nil — many users don't have one.
func (b *blueskyBackfiller) Backfill(ctx context.Context, did syntax.DID) (err error) {
	started := time.Now()
	defer func() {
		if b.observer == nil {
			return
		}
		result := "success"
		if err != nil {
			result = "error"
		}
		b.observer.ObserveRelationship("backfill", result, time.Since(started))
	}()
	var trackingErr error
	if b.tracker != nil {
		if err := b.tracker.AddRepo(ctx, did); err != nil {
			trackingErr = fmt.Errorf("request Tap repository tracking: %w", err)
		}
	}
	var rec map[string]any
	cid, err := b.reader.GetRecord(ctx, did, "app.bsky.actor.profile", "self", &rec)
	if errors.Is(err, auth.ErrRecordNotFound) {
		return trackingErr
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
	if err := b.indexer.Handle(ctx, ev); err != nil {
		return err
	}
	return trackingErr
}
