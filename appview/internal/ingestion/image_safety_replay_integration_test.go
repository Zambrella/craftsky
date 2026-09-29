package ingestion_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const imageReplayFixtureDDL = `
CREATE TABLE image_replay_serving_posts (
    uri TEXT PRIMARY KEY,
    cid TEXT NOT NULL
);
`

type imageReplayServingProjector struct{}

func (imageReplayServingProjector) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	if event.Action == "delete" {
		_, err := tx.Exec(ctx, `DELETE FROM image_replay_serving_posts WHERE uri=$1`, event.URI)
		return tap.Applied(), err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO image_replay_serving_posts(uri,cid) VALUES($1,$2)
		ON CONFLICT(uri) DO UPDATE SET cid=EXCLUDED.cid
	`, event.URI, event.CID)
	return tap.Applied(), err
}

func TestImageSafetyReplayUsesCurrentSourceAndDeleteCannotResurrect(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL+imageReplayFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	migration, err := os.ReadFile("../../migrations/000073_image_safety.up.sql")
	if err != nil {
		t.Fatalf("read image safety migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply image safety migration: %v", err)
	}

	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:image-replay','profile')`); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	const (
		firstBlob  = "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"
		secondBlob = "bafkreidjq52a7nre4puzipwf3gwfkgnxftvbwnp3jppfogo7her2g3ai64"
	)
	key := imagesafety.ScanKey{ScannerID: "fixture", PolicyVersion: "policy-1", CorpusVersion: "corpus-1"}
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_scan_results(
			id,blob_cid,scanner_id,policy_version,corpus_version,state,completed_at
		) VALUES('30000000-0000-4000-8000-000000000001',$1,$2,$3,$4,'clear',now())
	`, firstBlob, key.ScannerID, key.PolicyVersion, key.CorpusVersion); err != nil {
		t.Fatalf("seed clear scan: %v", err)
	}

	projector := index.NewImageSafetyCraftskyPost(imageReplayServingProjector{}, key)
	uri := syntax.ATURI("at://did:plc:image-replay/social.craftsky.feed.post/one")
	event := imageReplayEvent(1, uri, "3aaaaaaaaaaa2", "record-a", firstBlob, "create")
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatalf("ingest initial source: %v", err)
	}
	projectImageReplayClaim(t, store, projector, "initial")
	assertImageReplayServing(t, pool, uri, "record-a", true)

	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatalf("redeliver identical source: %v", err)
	}
	assertNoImageReplayClaim(t, store)
	assertImageReplayCounts(t, pool, 1, 0)

	event = imageReplayEvent(2, uri, "3aaaaaaaaaaa3", "record-b", firstBlob, "update")
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatalf("ingest same-blob edit: %v", err)
	}
	projectImageReplayClaim(t, store, projector, "same-blob-edit")
	assertImageReplayServing(t, pool, uri, "record-b", true)
	assertImageReplayCounts(t, pool, 1, 0)

	event = imageReplayEvent(3, uri, "3aaaaaaaaaaa4", "record-c", secondBlob, "update")
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatalf("ingest changed-blob edit: %v", err)
	}
	projectImageReplayClaim(t, store, projector, "changed-blob-edit")
	assertImageReplayServing(t, pool, uri, "record-b", false)
	assertImageReplayCounts(t, pool, 2, 1)

	restarted, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatalf("restart ingestion store: %v", err)
	}
	if err := restarted.WakeDependency(ctx, tap.Dependency{Kind: "image_subject_uri", Key: uri.String()}); err != nil {
		t.Fatalf("wake blocked source after restart: %v", err)
	}
	projectImageReplayClaim(t, restarted, projector, "restart-replay")
	assertImageReplayServing(t, pool, uri, "record-b", false)
	assertImageReplayCounts(t, pool, 2, 1)

	deleted := imageReplayEvent(4, uri, "3aaaaaaaaaaa5", "", "", "delete")
	if _, err := restarted.IngestRecord(ctx, deleted); err != nil {
		t.Fatalf("ingest delete: %v", err)
	}
	projectImageReplayClaim(t, restarted, projector, "delete")
	assertImageReplayServing(t, pool, uri, "", false)

	if _, err := pool.Exec(ctx, `
		UPDATE image_scan_results SET state='clear',completed_at=now(),updated_at=now()
		WHERE blob_cid=$1
	`, secondBlob); err != nil {
		t.Fatalf("complete orphaned scan: %v", err)
	}
	if err := restarted.WakeDependency(ctx, tap.Dependency{Kind: "image_subject_uri", Key: uri.String()}); err != nil {
		t.Fatalf("wake deleted subject: %v", err)
	}
	assertNoImageReplayClaim(t, restarted)
	assertImageReplayServing(t, pool, uri, "", false)
}

func imageReplayEvent(id uint64, uri syntax.ATURI, revision string, recordCID syntax.CID, blobCID string, action string) tap.Event {
	event := tap.Event{
		ID: id, URI: uri, DID: "did:plc:image-replay", Collection: "social.craftsky.feed.post",
		Rkey: "one", Rev: syntax.TID(revision), CID: recordCID, Action: action,
	}
	if action != "delete" {
		event.Record = json.RawMessage(`{
			"$type":"social.craftsky.feed.post","text":"image","createdAt":"2026-09-22T12:00:00Z",
			"images":[{"image":{"$type":"blob","ref":{"$link":"` + blobCID + `"},"mimeType":"image/jpeg","size":42}}]
		}`)
	}
	return event
}

func projectImageReplayClaim(t *testing.T, store *ingestion.Store, projector index.TransactionalIndexer, worker string) {
	t.Helper()
	claims, err := store.ClaimProjectionJobs(context.Background(), ingestion.ProjectionClaimRequest{
		Worker: worker, LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim %s: claims=%+v err=%v", worker, claims, err)
	}
	err = store.Project(context.Background(), claims[0], func(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
		return projector.Project(ctx, tx, tap.Event{
			ID: source.SourceEventID, URI: source.URI, CID: source.CID, DID: source.DID,
			Collection: source.Collection, Rkey: source.Rkey, Rev: source.Revision,
			Action: source.Action, Record: source.Record, Live: source.Live,
		})
	})
	if err != nil {
		t.Fatalf("project %s: %v", worker, err)
	}
}

func assertNoImageReplayClaim(t *testing.T, store *ingestion.Store) {
	t.Helper()
	claims, err := store.ClaimProjectionJobs(context.Background(), ingestion.ProjectionClaimRequest{
		Worker: "none", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(claims) != 0 {
		t.Fatalf("claims=%+v, want none", claims)
	}
}

func assertImageReplayCounts(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, wantResults, wantJobs int) {
	t.Helper()
	var results, jobs int
	if err := pool.QueryRow(context.Background(), `
		SELECT (SELECT count(*) FROM image_scan_results),
		       (SELECT count(*) FROM image_scan_jobs)
	`).Scan(&results, &jobs); err != nil {
		t.Fatal(err)
	}
	if results != wantResults || jobs != wantJobs {
		t.Fatalf("results/jobs=%d/%d, want %d/%d", results, jobs, wantResults, wantJobs)
	}
}

func assertImageReplayServing(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, uri syntax.ATURI, wantCID string, wantEligible bool) {
	t.Helper()
	var cid *string
	var eligible bool
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT cid FROM image_replay_serving_posts WHERE uri=$1),
			appview_image_subject_is_clear($1, COALESCE((SELECT cid FROM image_replay_serving_posts WHERE uri=$1),''))
	`, uri).Scan(&cid, &eligible); err != nil {
		t.Fatal(err)
	}
	if cid == nil {
		if wantCID != "" {
			t.Fatalf("serving CID is absent, want %q", wantCID)
		}
	} else if *cid != wantCID {
		t.Fatalf("serving CID=%q, want %q", *cid, wantCID)
	}
	if eligible != wantEligible {
		t.Fatalf("eligible=%t, want %t", eligible, wantEligible)
	}
}
