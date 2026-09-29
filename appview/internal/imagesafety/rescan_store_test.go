package imagesafety_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/testdb"
)

const rescanTapPreStateDDL = `
CREATE TABLE tap_source_records (uri TEXT PRIMARY KEY);
CREATE TABLE bluesky_profiles (
    did TEXT PRIMARY KEY,
    avatar_cid TEXT,
    avatar_mime TEXT,
    banner_cid TEXT,
    banner_mime TEXT,
    record_cid TEXT NOT NULL
);
CREATE TABLE tap_projection_jobs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_uri TEXT NOT NULL REFERENCES tap_source_records(uri) ON DELETE CASCADE,
    projection_kind TEXT NOT NULL,
    source_event_id BIGINT NOT NULL CHECK (source_event_id > 0),
    state TEXT NOT NULL CHECK (state IN ('pending', 'blocked', 'processing', 'complete', 'permanent_denied')),
    dependency_kind TEXT,
    dependency_key TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    lease_owner TEXT,
    lease_token UUID,
    lease_expires_at TIMESTAMPTZ,
    last_reason_code TEXT,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tap_projection_jobs_dependency_check CHECK (
        (state = 'blocked'
            AND dependency_kind IN ('member_did', 'subject_uri', 'repository_did')
            AND dependency_key IS NOT NULL
            AND btrim(dependency_key) <> '' AND char_length(dependency_key) <= 2048)
        OR
        (state <> 'blocked' AND dependency_kind IS NULL AND dependency_key IS NULL)
    )
);
`

func TestTargetedRescanStartsOneNewGenerationAndBlocksCurrentSubjects(t *testing.T) {
	pool := testdb.WithSchema(t, rescanTapPreStateDDL)
	migration, err := os.ReadFile("../../migrations/000076_image_safety.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply image safety migration: %v", err)
	}

	ctx := context.Background()
	resultID := uuid.MustParse("40000000-0000-4000-8000-000000000001")
	const subjectURI = "at://did:plc:rescan/social.craftsky.feed.post/one"
	const profileURI = "at://did:plc:rescan/app.bsky.actor.profile/self"
	if _, err := pool.Exec(ctx, `INSERT INTO tap_source_records(uri) VALUES($1)`, subjectURI); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tap_projection_jobs(source_uri,projection_kind,source_event_id,state,completed_at)
		VALUES($1,'record',1,'complete',now())
	`, subjectURI); err != nil {
		t.Fatalf("seed projection: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_scan_results(
			id,blob_cid,scanner_id,policy_version,corpus_version,state,completed_at
		) VALUES($1,'bafytarget','fixture','policy-1','corpus-1','clear',now())
	`, resultID); err != nil {
		t.Fatalf("seed scan result: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
		VALUES($1,'post','record-cid','clear')
	`, subjectURI); err != nil {
		t.Fatalf("seed subject: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_subject_requirements(
			subject_uri,source_cid,subject_kind,image_slot,blob_cid,scan_result_id
		) VALUES($1,'record-cid','post','images.0','bafytarget',$2)
	`, subjectURI, resultID); err != nil {
		t.Fatalf("seed requirement: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO tap_source_records(uri) VALUES($1)`, profileURI); err != nil {
		t.Fatalf("seed profile source: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO bluesky_profiles(did,avatar_cid,avatar_mime,record_cid)
		VALUES('did:plc:rescan','bafytarget','image/jpeg','profile-cid')
	`); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_image_candidates(
			profile_did,slot,source_cid,candidate_blob_cid,declared_mime,scan_result_id
		) VALUES('did:plc:rescan','avatar','profile-cid','bafytarget','image/jpeg',$1)
	`, resultID); err != nil {
		t.Fatalf("seed profile candidate: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tap_projection_jobs(source_uri,projection_kind,source_event_id,state,completed_at)
		VALUES($1,'record',2,'complete',now())
	`, profileURI); err != nil {
		t.Fatalf("seed profile projection: %v", err)
	}

	store := imagesafety.NewRescanStore(pool, func() time.Time {
		return time.Date(2030, 9, 22, 13, 0, 0, 0, time.UTC)
	})
	started, err := store.Target(ctx, resultID, 1)
	if err != nil {
		t.Fatalf("target rescan: %v", err)
	}
	if !started {
		t.Fatal("target rescan did not start")
	}

	var resultState, subjectState, scanJobState, projectionState, dependencyKind, dependencyKey string
	var version, jobVersion int64
	var scanJobs int
	if err := pool.QueryRow(ctx, `
		SELECT result.state,result.current_version,
		       subject.visibility_state,
		       job.state,job.scan_version,
		       projection.state,projection.dependency_kind,projection.dependency_key,
		       (SELECT count(*) FROM image_scan_jobs WHERE scan_result_id=$1)
		FROM image_scan_results result
		JOIN image_subject_requirements requirement ON requirement.scan_result_id=result.id
		JOIN image_subject_states subject ON subject.subject_uri=requirement.subject_uri
		JOIN image_scan_jobs job ON job.scan_result_id=result.id
		JOIN tap_projection_jobs projection ON projection.source_uri=requirement.subject_uri
		WHERE result.id=$1
	`, resultID).Scan(
		&resultState, &version, &subjectState, &scanJobState, &jobVersion,
		&projectionState, &dependencyKind, &dependencyKey, &scanJobs,
	); err != nil {
		t.Fatal(err)
	}
	if resultState != "pending" || version != 2 || subjectState != "blocked" {
		t.Fatalf("result=%s/v%d subject=%s", resultState, version, subjectState)
	}
	if scanJobState != "queued" || jobVersion != 2 || scanJobs != 1 {
		t.Fatalf("scan job=%s/v%d count=%d", scanJobState, jobVersion, scanJobs)
	}
	if projectionState != "blocked" || dependencyKind != "image_subject_uri" || dependencyKey != subjectURI {
		t.Fatalf("projection=%s dependency=%s/%s", projectionState, dependencyKind, dependencyKey)
	}
	var avatarCID *string
	var profileProjectionState, profileDependency string
	if err := pool.QueryRow(ctx, `
		SELECT profile.avatar_cid,projection.state,projection.dependency_key
		FROM bluesky_profiles profile
		JOIN tap_projection_jobs projection
		  ON projection.source_uri='at://' || profile.did || '/app.bsky.actor.profile/self'
		WHERE profile.did='did:plc:rescan'
	`).Scan(&avatarCID, &profileProjectionState, &profileDependency); err != nil {
		t.Fatal(err)
	}
	if avatarCID != nil || profileProjectionState != "blocked" || profileDependency != profileURI {
		t.Fatalf("avatar=%v profile projection=%s/%s", avatarCID, profileProjectionState, profileDependency)
	}

	started, err = store.Target(ctx, resultID, 1)
	if err != nil {
		t.Fatalf("repeat target rescan: %v", err)
	}
	if started {
		t.Fatal("repeated expected generation started duplicate work")
	}
}
