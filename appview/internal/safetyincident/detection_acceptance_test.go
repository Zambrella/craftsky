package safetyincident_test

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/testdb"
)

const detectionPreStateDDL = `
CREATE TABLE tap_source_records (uri TEXT PRIMARY KEY);
CREATE TABLE tap_projection_jobs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_uri TEXT NOT NULL REFERENCES tap_source_records(uri) ON DELETE CASCADE,
    projection_kind TEXT NOT NULL,
    source_event_id BIGINT NOT NULL CHECK (source_event_id > 0),
    state TEXT NOT NULL CHECK (state IN ('pending', 'blocked', 'processing', 'complete', 'permanent_denied')),
    dependency_kind TEXT,
    dependency_key TEXT,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_reason_code TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tap_projection_jobs_dependency_check CHECK (
        (state = 'blocked'
            AND dependency_kind IN ('member_did', 'subject_uri', 'repository_did')
            AND dependency_key IS NOT NULL AND btrim(dependency_key) <> '')
        OR
        (state <> 'blocked' AND dependency_kind IS NULL AND dependency_key IS NULL)
    )
);
CREATE TABLE moderation_reports (id TEXT PRIMARY KEY);
CREATE TABLE moderation_cases (id UUID PRIMARY KEY);
CREATE TABLE moderation_decisions (id UUID PRIMARY KEY);
CREATE TABLE moderation_effect_events (id UUID PRIMARY KEY);
`

type matchFetcher struct{}

func (matchFetcher) Fetch(context.Context, imagesafety.BlobSource) ([]byte, error) {
	return []byte("benign synthetic fixture"), nil
}

type matchScanner struct{}

func (matchScanner) Scan(context.Context, imagesafety.ScanInput) (imagesafety.ScanResult, error) {
	return imagesafety.ScanResult{
		State:                      imagesafety.StateMatch,
		ProviderReference:          "provider-match-7",
		IntegrityMetadataReference: "integrity-7",
	}, nil
}

func TestMatchCreatesRestrictedIncidentWithoutGuiltOrRenderedBytes(t *testing.T) {
	pool := testdb.WithSchema(t, detectionPreStateDDL)
	applyMigration(t, pool, "../../migrations/000073_image_safety.up.sql")
	applyMigration(t, pool, "../../migrations/000074_safety_incidents.up.sql")

	ctx := context.Background()
	now := time.Date(2030, 9, 22, 16, 0, 0, 0, time.UTC)
	resultID := uuid.MustParse("74000000-0000-4000-8000-000000000001")
	jobID := uuid.MustParse("74000000-0000-4000-8000-000000000002")
	const firstURI = "at://did:plc:first/social.craftsky.feed.post/one"
	const secondURI = "at://did:plc:second/social.craftsky.feed.post/two"
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		statements := []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO tap_source_records(uri) VALUES($1),($2)`, []any{firstURI, secondURI}},
			{`INSERT INTO tap_projection_jobs(source_uri,projection_kind,source_event_id,state,dependency_kind,dependency_key)
			 VALUES($1,'record',1,'blocked','image_subject_uri',$1),($2,'record',2,'blocked','image_subject_uri',$2)`,
				[]any{firstURI, secondURI}},
			{`INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state,created_at,updated_at)
			 VALUES($1,'bafymatch','fixture','policy-1','corpus-1','pending',$2,$2)`, []any{resultID, now}},
			{`INSERT INTO image_scan_jobs(id,scan_result_id,state,next_attempt_at,created_at,updated_at)
			 VALUES($1,$2,'queued',$3,$3,$3)`, []any{jobID, resultID, now}},
			{`INSERT INTO image_blob_sources(blob_cid,source_did,source_uri,source_cid,declared_mime,declared_size,observed_at)
			 VALUES('bafymatch','did:plc:first',$1,'record-one','image/jpeg',24,$2)`, []any{firstURI, now}},
			{`INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state,updated_at)
			 VALUES($1,'post','record-one','blocked',$3),($2,'post','record-two','blocked',$3)`, []any{firstURI, secondURI, now}},
			{`INSERT INTO image_subject_requirements(subject_uri,source_cid,subject_kind,image_slot,blob_cid,scan_result_id,created_at)
			 VALUES($1,'record-one','post','images.0','bafymatch',$3,$4),
			       ($2,'record-two','post','images.0','bafymatch',$3,$4)`, []any{firstURI, secondURI, resultID, now}},
		}
		for _, statement := range statements {
			if _, err := tx.Exec(ctx, statement.sql, statement.args...); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("seed match: %v", err)
	}

	clock := func() time.Time { return now }
	incidentStore := safetyincident.NewStore(pool, clock)
	worker, err := imagesafety.NewWorker(imagesafety.WorkerOptions{
		Store: imagesafety.NewWorkerStore(pool, clock), Fetcher: matchFetcher{}, Scanner: matchScanner{},
		MatchRecorder: incidentStore,
		Config: imagesafety.Config{Environment: imagesafety.EnvironmentTest, Mode: imagesafety.ScannerModeStub,
			ScannerID: "fixture", PolicyVersion: "policy-1", CorpusVersion: "corpus-1"},
		RetryPolicy: imagesafety.RetryPolicy{MaxAttempts: 3, InitialBackoff: time.Second, MaxBackoff: time.Minute},
		WorkerID:    "worker", LeaseDuration: time.Minute, OperationLimit: 30 * time.Second,
		PollInterval: time.Second, Now: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.ProcessOne(ctx)
	if err != nil || !processed {
		t.Fatalf("process match: processed=%t err=%v", processed, err)
	}

	var incidents, links, events, reports, cases, decisions, effects int
	var scanState, firstVisibility, secondVisibility string
	if err := pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM safety_incidents),
		  (SELECT count(*) FROM safety_incident_subjects),
		  (SELECT count(*) FROM safety_incident_events),
		  (SELECT count(*) FROM moderation_reports),
		  (SELECT count(*) FROM moderation_cases),
		  (SELECT count(*) FROM moderation_decisions),
		  (SELECT count(*) FROM moderation_effect_events),
		  (SELECT state FROM image_scan_results WHERE id=$1),
		  (SELECT visibility_state FROM image_subject_states WHERE subject_uri=$2),
		  (SELECT visibility_state FROM image_subject_states WHERE subject_uri=$3)
	`, resultID, firstURI, secondURI).Scan(
		&incidents, &links, &events, &reports, &cases, &decisions, &effects,
		&scanState, &firstVisibility, &secondVisibility,
	); err != nil {
		t.Fatal(err)
	}
	if incidents != 1 || links != 2 || events != 1 {
		t.Fatalf("incidents=%d links=%d events=%d", incidents, links, events)
	}
	if reports != 0 || cases != 0 || decisions != 0 || effects != 0 {
		t.Fatalf("automated guilt artifacts reports=%d cases=%d decisions=%d effects=%d", reports, cases, decisions, effects)
	}
	if scanState != "match" || firstVisibility != "blocked" || secondVisibility != "blocked" {
		t.Fatalf("scan=%s visibility=%s/%s", scanState, firstVisibility, secondVisibility)
	}

	claim, found, err := imagesafety.NewWorkerStore(pool, clock).Claim(ctx, "replay", time.Minute)
	if err != nil || found || claim.JobID != uuid.Nil {
		t.Fatalf("completed match was reclaimable: found=%t claim=%+v err=%v", found, claim, err)
	}
	work, err := incidentStore.SafeWork(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(work) != 1 || work[0].Reference == "" || work[0].Kind != "imageMatch" || work[0].State != "detected" {
		t.Fatalf("safe work=%+v", work)
	}
	if work[0].Reference == "provider-match-7" || work[0].Reference == "bafymatch" || work[0].Owner != "" || work[0].Cover != "" {
		t.Fatalf("restricted metadata leaked: %+v", work[0])
	}
	incidentID := uuid.MustParse(work[0].Reference)
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return incidentStore.RecordMatchTx(ctx, tx, imagesafety.MatchDetection{
			ResultID: resultID, ProviderReference: "provider-match-7",
			IntegrityMetadataReference: "integrity-7", DetectedAt: now,
		})
	}); err != nil {
		t.Fatalf("replay detection: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM safety_incidents),
		       (SELECT count(*) FROM safety_incident_subjects),
		       (SELECT count(*) FROM safety_incident_events)
	`).Scan(&incidents, &links, &events); err != nil {
		t.Fatal(err)
	}
	if incidents != 1 || links != 2 || events != 1 {
		t.Fatalf("replayed incidents=%d links=%d events=%d", incidents, links, events)
	}

	request := httptest.NewRequest("GET", "/v1/admin/safety/incidents/"+incidentID.String(), nil)
	request.SetPathValue("incidentReference", incidentID.String())
	recorder := httptest.NewRecorder()
	api.SafetyIncidentDetailHandler(incidentStore).ServeHTTP(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("safe detail status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, forbidden := range []string{"bafymatch", "did:plc", firstURI, secondURI, "image_bytes", "raw_result", "payload"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("safe detail leaked %q: %s", forbidden, body)
		}
	}
	for _, required := range []string{"provider-match-7", "integrity-7", `"post":2`} {
		if !strings.Contains(body, required) {
			t.Fatalf("safe detail missing %q: %s", required, body)
		}
	}

	var byteColumns int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.columns
		WHERE table_schema=current_schema()
		  AND table_name LIKE 'safety_incident%'
		  AND (data_type='bytea' OR column_name IN ('image_bytes','content','payload','raw_result'))
	`).Scan(&byteColumns); err != nil {
		t.Fatal(err)
	}
	if byteColumns != 0 {
		t.Fatalf("restricted incident schema has %d rendered/raw byte columns", byteColumns)
	}

}

func applyMigration(t *testing.T, pool *pgxpool.Pool, path string) {
	t.Helper()
	migration, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply %s: %v", path, err)
	}
}
