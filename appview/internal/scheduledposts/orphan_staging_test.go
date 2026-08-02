package scheduledposts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestStoreExpiresOnlyTrulyUnclaimedStagingAfterCreateRetry(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	for suffix := byte(1); suffix <= 3; suffix++ {
		if _, err := store.Create(ctx, capacityCreateParams(owner, suffix, now.Add(time.Hour))); err != nil {
			t.Fatalf("fill capacity: %v", err)
		}
	}
	claimedMedia := uuid.MustParse("00000000-0000-4000-8000-000000000401")
	orphanMedia := uuid.MustParse("00000000-0000-4000-8000-000000000402")
	for _, mediaID := range []uuid.UUID{claimedMedia, orphanMedia} {
		if _, err := store.pool.Exec(ctx, `
			INSERT INTO scheduled_post_media (
				id, owner_did, object_key, state, mime_type, size_bytes,
				sha256, blob_cid, unclaimed_expires_at
			) VALUES ($1, $2, $3, 'ready', 'image/jpeg', 4,
				decode(repeat('03', 32), 'hex'), $4, $5)
		`, mediaID, owner, "scheduled-media/"+mediaID.String(),
			"bafk-"+mediaID.String(), now.Add(24*time.Hour)); err != nil {
			t.Fatalf("seed ready media: %v", err)
		}
	}

	params := capacityCreateParams(owner, 4, now.Add(2*time.Hour))
	params.MediaIDs = []uuid.UUID{claimedMedia}
	if _, err := store.Create(ctx, params); !errors.Is(err, ErrCapacityReached) {
		t.Fatalf("full-capacity create error=%v, want %v", err, ErrCapacityReached)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE id=$1`, 0, params.ID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1 AND schedule_id IS NULL`, 1, claimedMedia)

	if err := store.Delete(ctx, owner, capacityCreateParams(owner, 1, now).ID, now); err != nil {
		t.Fatalf("release capacity: %v", err)
	}
	created, err := store.Create(ctx, params)
	if err != nil {
		t.Fatalf("retry create: %v", err)
	}
	repeated, err := store.Create(ctx, params)
	if err != nil {
		t.Fatalf("repeat idempotent create: %v", err)
	}
	if repeated.ID != created.ID {
		t.Fatalf("repeat created %s, want %s", repeated.ID, created.ID)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE owner_did=$1`, 3, owner)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1 AND schedule_id=$2 AND ordinal=0`, 1, claimedMedia, created.ID)

	if err := store.SweepExpiredLifecycle(ctx, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("expire unclaimed staging: %v", err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 1, claimedMedia)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 0, orphanMedia)
	claims, err := store.ClaimCleanup(ctx, 10, now.Add(24*time.Hour), time.Minute)
	if err != nil {
		t.Fatalf("claim orphan cleanup: %v", err)
	}
	if len(claims) != 1 || claims[0].ObjectKey != "scheduled-media/"+orphanMedia.String() {
		t.Fatalf("orphan cleanup claims=%#v", claims)
	}
}
