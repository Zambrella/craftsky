package scheduledposts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

func TestClaimedGenerationCannotPublishAfterDepartureAndRejoin(t *testing.T) {
	pool := newScheduledPostStoreTestPool(t)
	store := NewStore(pool)
	fencer := newScheduledTestOwnerFencer(t, pool)
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	deletion := NewAccountDeletion(pool, time.Now, fencer)
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	attachedMediaID := uuid.New()
	attachedObjectKey := insertReadyPrivateMediaFixture(
		t, store, owner, attachedMediaID, nil, nil, "bafk-attached-generation-one", now.Add(time.Hour),
	)
	unclaimedMediaID := uuid.New()
	insertReadyPrivateMediaFixture(
		t, store, owner, unclaimedMediaID, nil, nil, "bafk-unclaimed-generation-one", now.Add(time.Hour),
	)
	createParams := capacityCreateParams(owner, 90, now)
	createParams.MediaIDs = []uuid.UUID{attachedMediaID}
	created, err := store.Create(ctx, createParams)
	if err != nil {
		t.Fatal(err)
	}
	if created.OwnerGeneration != 1 {
		t.Fatalf("created generation=%d, want 1", created.OwnerGeneration)
	}
	claims, err := store.ClaimDue(ctx, 1, now, time.Minute)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim scheduled generation: claims=%d err=%v", len(claims), err)
	}
	stale := claims[0]
	if stale.OwnerGeneration != 1 {
		t.Fatalf("claim generation=%d, want 1", stale.OwnerGeneration)
	}

	departed, err := lifecycles.TransitionWith(ctx, ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: 1, To: ownerlifecycle.StateDeparted,
		Reason: "profile_departed",
	}, deletion.DepartureParticipant())
	if err != nil {
		t.Fatal(err)
	}
	if departed.Generation != 2 {
		t.Fatalf("departed generation=%d, want 2", departed.Generation)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE owner_did=$1`, 0, owner)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 0, attachedMediaID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 1, attachedObjectKey)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 1, unclaimedMediaID)
	active, err := lifecycles.Transition(ctx, ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: 2, To: ownerlifecycle.StateActive,
		Reason: "profile_rejoined",
	})
	if err != nil {
		t.Fatal(err)
	}
	if active.Generation != 3 {
		t.Fatalf("rejoined generation=%d, want 3", active.Generation)
	}
	media := NewPrivateMediaService(store, newMemoryPrivateObjectStore())
	if _, err := media.Open(ctx, owner, unclaimedMediaID); !errors.Is(err, ErrScheduledMediaNotFound) {
		t.Fatalf("rejoined owner opened generation-one private media: %v", err)
	}

	pdsFactoryCalls := 0
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			sessionID: "stale-session",
		},
		NewEffects: func(context.Context, syntax.DID, string) (pdseffects.GuardedEffectCoordinator, error) {
			pdsFactoryCalls++
			return nil, errors.New("stale work reached PDS factory")
		},
		Objects: &memoryPrivateObjectStore{objects: make(map[string][]byte)},
		Now:     func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(ctx, WorkItem{
		ID: stale.ID, OwnerDID: stale.OwnerDID, OwnerGeneration: stale.OwnerGeneration,
		LeaseToken: stale.LeaseToken, PayloadVersion: stale.PayloadVersion,
		Rkey: stale.Rkey, CreatedAt: stale.CreatedAt,
	}); err != nil {
		t.Fatalf("stale generation worker should converge without effect: %v", err)
	}
	if pdsFactoryCalls != 0 {
		t.Fatalf("stale generation reached PDS factory %d times", pdsFactoryCalls)
	}

	recreated, err := store.Create(ctx, capacityCreateParams(owner, 91, now.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	if recreated.OwnerGeneration != 3 {
		t.Fatalf("rejoined schedule generation=%d, want 3", recreated.OwnerGeneration)
	}
}
