package scheduledposts

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestLifecycleDeadlinesRemoveOnlyEligiblePrivateContent(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	deadline := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	live, err := store.Create(ctx, capacityCreateParams(owner, 1, deadline.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	expired, err := store.Create(ctx, capacityCreateParams(owner, 2, deadline.Add(-31*24*time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `
		UPDATE scheduled_posts
		SET status='needs_attention', needs_attention_at=$2,
		    needs_attention_expires_at=$3, next_attempt_at=$2
		WHERE id=$1
	`, expired.ID, deadline.Add(-30*24*time.Hour), deadline); err != nil {
		t.Fatal(err)
	}
	liveMedia := uuid.New()
	orphanMedia := uuid.New()
	for _, fixture := range []struct {
		id         uuid.UUID
		scheduleID *uuid.UUID
		ordinal    *int
	}{
		{id: liveMedia, scheduleID: &live.ID, ordinal: intPointer(0)},
		{id: orphanMedia},
	} {
		insertReadyPrivateMediaFixture(
			t, store, owner, fixture.id, fixture.scheduleID, fixture.ordinal,
			"bafk-"+fixture.id.String(), deadline,
		)
	}
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO scheduled_post_publication_tombstones (
			schedule_id, owner_did, owner_generation, operation_id, request_hash,
			publication_uri, publication_cid, published_at, expires_at
		) VALUES ($1,$2,1,$3,decode(repeat('04',32),'hex'),$4,'bafk-published',$5,$6)
	`, uuid.New(), owner, uuid.New(), "at://did:plc:alice/social.craftsky.feed.post/3retained",
		deadline.Add(-30*24*time.Hour), deadline); err != nil {
		t.Fatal(err)
	}

	if err := store.SweepExpiredLifecycle(ctx, deadline.Add(-time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE id=$1`, 1, expired.ID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 1, orphanMedia)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_publication_tombstones`, 1)

	if err := store.SweepExpiredLifecycle(ctx, deadline); err != nil {
		t.Fatal(err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE id=$1`, 0, expired.ID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 0, orphanMedia)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_publication_tombstones`, 0)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE id=$1`, 1, live.ID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 1, liveMedia)
	orphanKey, _, _ := NewGenerationObjectKey(owner, 1, orphanMedia)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 1, orphanKey)
}
