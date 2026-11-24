package scheduledposts

import (
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
	"social.craftsky/appview/internal/testlog"
)

// CheckProjectPublicationForTest lets an external test drive the real public API
// around the same scheduled worker and PDS fixture used by the recovery tests.
func CheckProjectPublicationForTest(t *testing.T, project json.RawMessage, check func(*pgxpool.Pool, syntax.DID, syntax.RecordKey, bool)) {
	t.Helper()
	ctx := t.Context()
	pool := testdb.WithMigratedSchema(t)
	fixture := newPublicationRecoveryFixtureWithPool(t, pool, 0, false, true, project)
	if status := fixture.status(t); status != StatusScheduled {
		t.Fatalf("initial status = %s, want scheduled", status)
	}
	// Discover the target key without claiming the schedule so both direct reads
	// use the exact same resource, including while the schedule is still pending.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	key, err := allocatePublicationRkey(ctx, tx, fixture.owner, fixture.id, fixture.now)
	rollbackErr := tx.Rollback(ctx)
	if err != nil || rollbackErr != nil {
		t.Fatalf("preview publication key: %v, rollback: %v", err, rollbackErr)
	}
	check(pool, fixture.owner, key, false)

	published := reconcileScheduledPublicationLostResponse(t, fixture)
	fixture.assertSingleFinalizedRecord(t, published)
	if fixture.pds.rkey != key.String() {
		t.Fatalf("published record key = %s, want %s", fixture.pds.rkey, key)
	}
	source := ingestion.SourceRecord{
		DID: fixture.owner, Rkey: key, Collection: PostCollection, Action: "create",
		URI: syntax.ATURI("at://" + fixture.owner.String() + "/" + PostCollection + "/" + key.String()),
		CID: "bafyreischeduled", Revision: "3aaaaaaaaaaa2", Record: published,
	}
	postIndex := index.NewImageSafetyCraftskyPost(
		index.NewCraftskyPost(pool, testlog.Discard(), notifications.NewService()),
		imagesafety.ScanKey{ScannerID: "fixture", PolicyVersion: "fixture", CorpusVersion: "fixture"},
	)
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	outcome, err := postIndex.Project(ctx, tx, source)
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("project PDS record: outcome = %+v, error = %v", outcome, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	check(pool, fixture.owner, key, true)
}
