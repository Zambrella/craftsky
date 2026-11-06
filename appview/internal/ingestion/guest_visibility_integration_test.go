package ingestion_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// IT-006 / RULE-004 / AC-017: an actual duplicate delivery may establish unknown
// state, without requesting historical events or re-running completed jobs.
func TestGuestVisibilityInitializesFromNormalDuplicateDelivery(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	did := syntax.DID("did:plc:duplicate-visibility")
	lifecycle, err := lifecycles.EnsureOnboardingOwner(ctx, did)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycles.Transition(ctx, ownerlifecycle.TransitionRequest{Owner: did, ExpectedGeneration: lifecycle.Generation, To: ownerlifecycle.StateActive, Reason: "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES ($1,'member')`, did); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profiles(did,record_cid) VALUES ($1,'profile')`, did); err != nil {
		t.Fatal(err)
	}
	ev := tap.Event{ID: 777, URI: syntax.ATURI("at://" + did.String() + "/app.bsky.actor.profile/self"), DID: did, Collection: "app.bsky.actor.profile", Rkey: "self", Rev: "3m234567abc22", CID: "profile", Action: "update", Record: json.RawMessage(`{"$type":"app.bsky.actor.profile"}`)}
	ingest := func() {
		t.Helper()
		if _, err := service.IngestRecord(ctx, ev); err != nil {
			t.Fatal(err)
		}
	}
	ingest()
	if _, err := pool.Exec(ctx, `UPDATE tap_projection_jobs SET state='complete',completed_at=now() WHERE source_uri=$1`, ev.URI); err != nil {
		t.Fatal(err)
	}
	ev.ID++
	ingest()
	var hidden *bool
	if err := pool.QueryRow(ctx, `SELECT guest_visibility_hidden FROM bluesky_profiles WHERE did=$1`, did).Scan(&hidden); err != nil {
		t.Fatal(err)
	}
	if hidden == nil || *hidden {
		t.Fatalf("normal duplicate did not establish public state: %v", hidden)
	}
	ev.ID++
	ingest()
	job, err := store.ProjectionJob(ctx, ev.URI)
	if err != nil || job.State != "complete" {
		t.Fatalf("duplicate scheduled completed projection: %+v %v", job, err)
	}
	var jobs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tap_repository_jobs WHERE did=$1`, did).Scan(&jobs); err != nil || jobs != 0 {
		t.Fatalf("duplicate enqueued repository repair: count=%d err=%v", jobs, err)
	}
}
