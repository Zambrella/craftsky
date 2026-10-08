package imagesafety_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/testdb"
)

type workerFetcher struct {
	err error
}

func (fetcher *workerFetcher) Fetch(context.Context, imagesafety.BlobSource) ([]byte, error) {
	if fetcher.err != nil {
		return nil, fetcher.err
	}
	var body bytes.Buffer
	err := jpeg.Encode(&body, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil)
	return body.Bytes(), err
}

func TestWorkerSurvivesRetryRestartExhaustionAndManualRetry(t *testing.T) {
	pool := testdb.WithSchema(t, rescanTapPreStateDDL)
	migration, err := os.ReadFile("../../migrations/000081_image_safety.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply image safety migration: %v", err)
	}

	ctx := context.Background()
	now := time.Date(2030, 9, 22, 13, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	resultID := uuid.MustParse("50000000-0000-4000-8000-000000000001")
	jobID := uuid.MustParse("50000000-0000-4000-8000-000000000002")
	const subjectURI = "at://did:plc:worker/social.craftsky.feed.post/one"
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		statements := []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO tap_source_records(uri) VALUES($1)`, []any{subjectURI}},
			{`
		INSERT INTO tap_projection_jobs(
			source_uri,projection_kind,source_event_id,state,dependency_kind,dependency_key
		) VALUES($1,'record',1,'blocked','image_subject_uri',$1)`, []any{subjectURI}},
			{`
		INSERT INTO image_scan_results(
			id,blob_cid,scanner_id,policy_version,corpus_version,state,created_at,updated_at
		) VALUES($1,'bafyworker','fixture','policy-1','corpus-1','pending',$2,$2)`, []any{resultID, now}},
			{`
		INSERT INTO image_scan_jobs(
			id,scan_result_id,state,next_attempt_at,created_at,updated_at
		) VALUES($1,$2,'queued',$3,$3,$3)`, []any{jobID, resultID, now}},
			{`
		INSERT INTO image_blob_sources(
			blob_cid,source_did,source_uri,source_cid,declared_mime,declared_size,observed_at
		) VALUES('bafyworker','did:plc:worker',$1,'record-cid','image/jpeg',7,$2)`, []any{subjectURI, now}},
			{`
		INSERT INTO image_subject_states(
			subject_uri,subject_kind,source_cid,visibility_state,updated_at
		) VALUES($1,'post','record-cid','blocked',$2)`, []any{subjectURI, now}},
			{`
		INSERT INTO image_subject_requirements(
			subject_uri,source_cid,subject_kind,image_slot,blob_cid,scan_result_id,created_at
		) VALUES($1,'record-cid','post','images.0','bafyworker',$2,$3)`, []any{subjectURI, resultID, now}},
		}
		for _, statement := range statements {
			if _, err := tx.Exec(ctx, statement.sql, statement.args...); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("seed worker state: %v", err)
	}

	store := imagesafety.NewWorkerStore(pool, clock)
	fetcher := &workerFetcher{err: context.DeadlineExceeded}
	config := imagesafety.Config{
		Environment: imagesafety.EnvironmentProduction, Mode: imagesafety.ScannerModeManual,
		ScannerID: "fixture", PolicyVersion: "policy-1", CorpusVersion: "corpus-1",
	}
	newWorker := func() *imagesafety.Worker {
		worker, err := imagesafety.NewWorker(imagesafety.WorkerOptions{
			Store: store, Fetcher: fetcher,
			Scanner: imagesafety.ManualModerationScanner{},
			Config:  config,
			RetryPolicy: imagesafety.RetryPolicy{
				MaxAttempts: 3, InitialBackoff: time.Minute, MaxBackoff: 2 * time.Minute,
			},
			WorkerID: "worker", LeaseDuration: time.Minute,
			OperationLimit: 30 * time.Second, PollInterval: time.Second, Now: clock,
		})
		if err != nil {
			t.Fatal(err)
		}
		return worker
	}

	for attempt, advance := range []time.Duration{time.Minute, 2 * time.Minute, 0} {
		processed, err := newWorker().ProcessOne(ctx)
		if err != nil || !processed {
			t.Fatalf("attempt %d: processed=%t err=%v", attempt+1, processed, err)
		}
		now = now.Add(advance)
	}
	var state, category string
	var attempts int
	if err := pool.QueryRow(ctx, `
		SELECT state,safe_error_category,attempts FROM image_scan_jobs WHERE id=$1
	`, jobID).Scan(&state, &category, &attempts); err != nil {
		t.Fatal(err)
	}
	if state != "dead_letter" || category != "fetch" || attempts != 3 {
		t.Fatalf("dead letter=%s/%s attempts=%d", state, category, attempts)
	}

	retried, err := store.ManualRetry(ctx, resultID, 1)
	if err != nil || !retried {
		t.Fatalf("manual retry: retried=%t err=%v", retried, err)
	}
	fetcher.err = nil
	processed, err := newWorker().ProcessOne(ctx)
	if err != nil || !processed {
		t.Fatalf("manual retry processing: processed=%t err=%v", processed, err)
	}
	var resultState, projectionState string
	var jobs int
	if err := pool.QueryRow(ctx, `
		SELECT result.state,projection.state,
		       (SELECT count(*) FROM image_scan_jobs WHERE scan_result_id=result.id)
		FROM image_scan_results result
		JOIN tap_projection_jobs projection ON projection.source_uri=$2
		WHERE result.id=$1
	`, resultID, subjectURI).Scan(&resultState, &projectionState, &jobs); err != nil {
		t.Fatal(err)
	}
	if resultState != "clear" || projectionState != "pending" || jobs != 0 {
		t.Fatalf("result=%s projection=%s jobs=%d", resultState, projectionState, jobs)
	}

	rescan := imagesafety.NewRescanStore(pool, clock)
	started, err := rescan.Target(ctx, resultID, 1)
	if err != nil || !started {
		t.Fatalf("target rescan: started=%t err=%v", started, err)
	}
	stale, found, err := store.Claim(ctx, "stale-worker", time.Minute)
	if err != nil || !found {
		t.Fatalf("claim stale generation: found=%t err=%v", found, err)
	}
	now = now.Add(time.Minute)
	current, found, err := store.Claim(ctx, "restarted-worker", time.Minute)
	if err != nil || !found {
		t.Fatalf("reclaim generation: found=%t err=%v", found, err)
	}
	if _, err := store.Complete(ctx, stale, imagesafety.StateClear); !errors.Is(err, imagesafety.ErrLeaseLost) {
		t.Fatalf("stale completion error=%v", err)
	}
	if _, err := store.Complete(ctx, current, imagesafety.StateClear); err != nil {
		t.Fatalf("current completion: %v", err)
	}
}
