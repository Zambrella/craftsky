package accountdeletion

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestReceiptObserverPersistsOwnerScopedPostIndexerDeletes(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	jobID := uuid.MustParse("00000000-0000-4000-8000-000000000920")
	owner := syntax.DID("did:plc:alice")
	uri := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/post1")
	now := time.Date(2026, 8, 10, 14, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		WITH operation AS (
			INSERT INTO account_deletion_operations(
			id,owner_did,state,phase,accepted_at
			) VALUES($1,$2,'active','waitingForIndexerConvergence',$3)
			RETURNING id
		)
		INSERT INTO account_deletion_expected_records(
			job_id,uri,collection,registered_at,delete_requested_at
		) SELECT id,$4,'social.craftsky.feed.post',$3,$3 FROM operation
	`, jobID, owner, now, uri); err != nil {
		t.Fatalf("seed expected deletion receipt: %v", err)
	}

	observer := NewReceiptObserver(pool, func() time.Time { return now })
	for _, event := range []tap.Event{
		{URI: uri, DID: owner, Collection: "social.craftsky.feed.post", Action: "create", ID: 39, Rev: "rev-create"},
		{URI: "at://did:plc:alice/social.craftsky.feed.post/other", DID: owner, Collection: "social.craftsky.feed.post", Action: "delete", ID: 40, Rev: "rev-other"},
		{URI: uri, DID: owner, Collection: "social.craftsky.feed.post", Action: "delete", ID: 41, Rev: "rev-delete"},
		{URI: uri, DID: owner, Collection: "social.craftsky.feed.post", Action: "delete", ID: 41, Rev: "rev-delete"},
	} {
		if err := observer.ObserveHandled(ctx, event); err != nil {
			t.Fatalf("observe event %+v: %v", event, err)
		}
	}

	var (
		count   int
		eventID int64
		rev     string
	)
	if err := pool.QueryRow(ctx, `
		SELECT count(*),
		       (SELECT tap_event_id FROM account_deletion_index_receipts WHERE job_id=$1 AND uri=$2),
		       (SELECT repo_revision FROM account_deletion_index_receipts WHERE job_id=$1 AND uri=$2)
		FROM account_deletion_index_receipts WHERE job_id=$1
	`, jobID, uri).Scan(&count, &eventID, &rev); err != nil {
		t.Fatalf("read deletion receipt: %v", err)
	}
	if count != 2 || eventID != 41 || rev != "rev-delete" {
		t.Fatalf("receipt count=%d event=%d rev=%q", count, eventID, rev)
	}
}

func TestReceiptObserverRetainsDeleteBeforeExpectedRegistration(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	jobID := uuid.MustParse("00000000-0000-4000-8000-000000000921")
	owner := syntax.DID("did:plc:alice")
	uri := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/raced")
	now := time.Date(2026, 8, 10, 14, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,state,phase,accepted_at
		) VALUES($1,$2,'active','removingCraftskyRecords',$3)
	`, jobID, owner, now); err != nil {
		t.Fatal(err)
	}

	observer := NewReceiptObserver(pool, func() time.Time { return now })
	if err := observer.ObserveHandled(ctx, tap.Event{
		URI: uri, DID: owner, Collection: "social.craftsky.feed.post",
		Action: "delete", ID: 51, Rev: "rev-before-expected",
	}); err != nil {
		t.Fatal(err)
	}
	var receiptsBeforeRegistration int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM account_deletion_index_receipts
		WHERE job_id=$1 AND uri=$2
	`, jobID, uri).Scan(&receiptsBeforeRegistration); err != nil {
		t.Fatal(err)
	}
	if receiptsBeforeRegistration != 1 {
		t.Fatalf("receipts before expected registration=%d want=1", receiptsBeforeRegistration)
	}

	store := NewStore(pool, func() time.Time { return now })
	if err := store.RegisterExpected(ctx, jobID.String(), owner, uri, uri.Collection()); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkDeleteRequested(ctx, jobID.String(), owner, uri); err != nil {
		t.Fatal(err)
	}
	converged, err := NewConvergenceVerifier(pool).IsConverged(ctx, jobID, owner)
	if err != nil || !converged {
		t.Fatalf("converged after late expected registration=%t err=%v", converged, err)
	}
}
