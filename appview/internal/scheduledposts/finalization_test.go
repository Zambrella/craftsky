package scheduledposts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPublicationCompletionReusesHeldEffectConnection(t *testing.T) {
	base := newScheduledPostStoreTestPool(t)
	config := base.Config().Copy()
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store := NewStore(pool)
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	created, err := store.Create(ctx, capacityCreateParams(owner, 70, now))
	if err != nil {
		t.Fatal(err)
	}
	claims, err := store.ClaimDue(ctx, 1, now, time.Minute)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim publication: claims=%d err=%v", len(claims), err)
	}
	claim := claims[0]
	guard, err := store.AcquirePublishingEffect(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	effectCtx := guard.bind(ctx)
	uri := syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/" + claim.Rkey.String())
	if _, err := store.FinalizePublication(effectCtx, FinalizePublicationParams{
		Claim: claim, PublicationURI: uri, PublicationCID: "bafk-one-connection",
		PublishedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("finalize with only held effect connection: %v", err)
	}
	if err := guard.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, owner, created.ID); !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("finalized schedule remained: %v", err)
	}
}

func TestStoreFinalizesPublicationAtomically(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	primary, err := store.Create(ctx, capacityCreateParams(owner, 1, now))
	if err != nil {
		t.Fatalf("create primary schedule: %v", err)
	}
	for suffix := byte(2); suffix <= 3; suffix++ {
		if _, err := store.Create(ctx, capacityCreateParams(owner, suffix, now.Add(time.Hour))); err != nil {
			t.Fatalf("fill capacity: %v", err)
		}
	}
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000301")
	ordinal := 0
	objectKey := insertReadyPrivateMediaFixture(
		t, store, owner, mediaID, &primary.ID, &ordinal, "bafk-final", now.Add(24*time.Hour),
	)
	claims, err := store.ClaimDue(ctx, 1, now, time.Minute)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim primary schedule=%v err=%v", claims, err)
	}
	claim := claims[0]
	publicationURI, err := syntax.ParseATURI(
		"at://did:plc:alice/social.craftsky.feed.post/" + claim.Rkey.String(),
	)
	if err != nil {
		t.Fatalf("parse publication URI: %v", err)
	}
	params := FinalizePublicationParams{
		Claim:          claim,
		PublicationURI: publicationURI,
		PublicationCID: syntax.CID("bafk-publication"),
		PublishedAt:    now.Add(time.Second),
	}
	stale := params
	stale.Claim.LeaseToken = uuid.New()
	if _, err := store.FinalizePublication(ctx, stale); !errors.Is(err, ErrWorkerLeaseLost) {
		t.Fatalf("stale finalization error=%v, want %v", err, ErrWorkerLeaseLost)
	}

	result, err := store.FinalizePublication(ctx, params)
	if err != nil {
		t.Fatalf("finalize publication: %v", err)
	}
	if result.AlreadyFinalized {
		t.Fatal("first completion reported already finalized")
	}
	repeated, err := store.FinalizePublication(ctx, params)
	if err != nil {
		t.Fatalf("repeat finalization: %v", err)
	}
	if !repeated.AlreadyFinalized || repeated.PublicationURI != publicationURI ||
		repeated.PublicationCID != params.PublicationCID {
		t.Fatalf("repeated finalization=%#v", repeated)
	}

	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE id=$1`, 0, primary.ID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 0, mediaID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 1, objectKey)
	var expiresAt time.Time
	if err := store.pool.QueryRow(ctx, `
		SELECT expires_at
		FROM scheduled_post_publication_tombstones
		WHERE schedule_id=$1 AND owner_did=$2
	`, primary.ID, owner).Scan(&expiresAt); err != nil {
		t.Fatalf("read publication tombstone: %v", err)
	}
	if !expiresAt.Equal(params.PublishedAt.Add(30 * 24 * time.Hour)) {
		t.Fatalf("tombstone expiry=%s", expiresAt)
	}
	if _, err := store.Create(ctx, capacityCreateParams(owner, 4, now.Add(2*time.Hour))); err != nil {
		t.Fatalf("capacity was not released: %v", err)
	}
}

func TestStoreCreateDedupeUsesPublicationTombstone(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	createParams := capacityCreateParams(owner, 1, now)
	created, err := store.Create(ctx, createParams)
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	claims, err := store.ClaimDue(ctx, 1, now, time.Minute)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim schedule=%v err=%v", claims, err)
	}
	publicationURI, err := syntax.ParseATURI(
		"at://did:plc:alice/social.craftsky.feed.post/" + claims[0].Rkey.String(),
	)
	if err != nil {
		t.Fatalf("parse publication URI: %v", err)
	}
	publicationCID := syntax.CID("bafk-idempotent-publication")
	publishedAt := now.Add(time.Second)
	if _, err := store.FinalizePublication(ctx, FinalizePublicationParams{
		Claim: claims[0], PublicationURI: publicationURI,
		PublicationCID: publicationCID, PublishedAt: publishedAt,
	}); err != nil {
		t.Fatalf("finalize publication: %v", err)
	}

	replayed, err := store.Create(ctx, createParams)
	if err != nil {
		t.Fatalf("replay completed create: %v", err)
	}
	if replayed.ID != created.ID || replayed.Status != StatusPublished ||
		replayed.PublicationURI != publicationURI || replayed.PublicationCID != publicationCID ||
		!replayed.PublishedAt.Equal(publishedAt) {
		t.Fatalf("replayed completed create=%#v", replayed)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE owner_did=$1`, 0, owner)

	conflicting := createParams
	conflicting.RequestHash[0] ^= 0xff
	if _, err := store.Create(ctx, conflicting); !errors.Is(err, ErrOperationConflict) {
		t.Fatalf("conflicting completed operation error=%v, want %v", err, ErrOperationConflict)
	}
}
