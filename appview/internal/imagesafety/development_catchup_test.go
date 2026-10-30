package imagesafety_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/testdb"
)

func TestDevelopmentCatchUpRequeuesUnresolvedImagesWithoutRewritingVerdicts(t *testing.T) {
	pool := testdb.WithSchema(t, rescanTapPreStateDDL)
	ctx := context.Background()
	migration, err := os.ReadFile("../../migrations/000081_image_safety.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	// Include exhausted failures, successful scans, matches, profiles, and work
	// blocked for another reason. Only image-dependent unresolved work may move.
	fixtures := []struct{ state, dependency, kind string }{
		{"pending", "image_subject_uri", "post"},
		{"error", "image_subject_uri", "post"},
		{"unavailable", "image_subject_uri", "post"},
		{"clear", "image_subject_uri", "post"},
		{"match", "image_subject_uri", "post"},
		{"pending", "image_subject_uri", "profile"},
		{"pending", "member_did", "post"},
	}
	uris := make([]string, len(fixtures))
	for i, fixture := range fixtures {
		uri := fmt.Sprintf("at://did:plc:catchup%d/social.craftsky.feed.post/one", i)
		if fixture.kind == "profile" {
			uri = fmt.Sprintf("at://did:plc:catchup%d/app.bsky.actor.profile/self", i)
		}
		uris[i] = uri
		resultID, blob := uuid.New(), fmt.Sprintf("blob-%d", i)
		if _, err := pool.Exec(ctx, `INSERT INTO tap_source_records(uri) VALUES($1)`, uri); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO tap_projection_jobs(source_uri,projection_kind,source_event_id,state,dependency_kind,dependency_key)
    VALUES($1,'record',1,'blocked',$2,$1)`, uri, fixture.dependency); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state)
    VALUES($1,$2,'fixture','policy','corpus',$3)`, resultID, blob, fixture.state); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO image_scan_jobs(id,scan_result_id,state) VALUES($1,$2,'dead_letter')`, uuid.New(), resultID); err != nil {
			t.Fatal(err)
		}
		if fixture.kind == "profile" {
			if _, err := pool.Exec(ctx, `INSERT INTO profile_image_candidates(profile_did,slot,source_cid,candidate_blob_cid,declared_mime,scan_result_id)
     VALUES($1,'avatar','record-cid',$2,'image/jpeg',$3)`, fmt.Sprintf("did:plc:catchup%d", i), blob, resultID); err != nil {
				t.Fatal(err)
			}
		} else {
			if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
     VALUES($1,'post','record-cid','blocked')`, uri); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO image_subject_requirements(subject_uri,source_cid,subject_kind,image_slot,blob_cid,scan_result_id)
     VALUES($1,'record-cid','post','images.0',$2,$3)`, uri, blob, resultID); err != nil {
				t.Fatal(err)
			}
		}
	}
	config := imagesafety.Config{Environment: imagesafety.EnvironmentDevelopment, Mode: imagesafety.ScannerModeDevelopmentAutoApprove,
		ScannerID: imagesafety.DevelopmentApprovalScannerID, PolicyVersion: imagesafety.DevelopmentApprovalPolicyVersion, CorpusVersion: imagesafety.DevelopmentApprovalCorpusVersion}
	for _, env := range []imagesafety.Environment{imagesafety.EnvironmentProduction, imagesafety.EnvironmentTest} {
		invalid := config
		invalid.Environment = env
		if _, err := imagesafety.CatchUpDevelopment(ctx, nil, invalid); err == nil {
			t.Fatalf("accepted %s", env)
		}
	}
	count, err := imagesafety.CatchUpDevelopment(ctx, pool, config)
	if err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("requeued=%d, want 4", count)
	}
	for i, fixture := range fixtures {
		var state, originalVerdict string
		if err := pool.QueryRow(ctx, `SELECT state FROM tap_projection_jobs WHERE source_uri=$1`, uris[i]).Scan(&state); err != nil {
			t.Fatal(err)
		}
		want := "blocked"
		if fixture.dependency == "image_subject_uri" && fixture.state != "clear" && fixture.state != "match" {
			want = "pending"
		}
		if state != want {
			t.Fatalf("fixture %d state=%s, want %s", i, state, want)
		}
		if err := pool.QueryRow(ctx, `SELECT state FROM image_scan_results WHERE blob_cid=$1`, fmt.Sprintf("blob-%d", i)).Scan(&originalVerdict); err != nil {
			t.Fatal(err)
		}
		if originalVerdict != fixture.state {
			t.Fatalf("fixture %d verdict changed to %s", i, originalVerdict)
		}
	}
	var jobs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM image_scan_jobs`).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if jobs != 2 {
		t.Fatalf("remaining jobs=%d, want clear and match retained", jobs)
	}
	count, err = imagesafety.CatchUpDevelopment(ctx, pool, config)
	if err != nil || count != 0 {
		t.Fatalf("repeat catch-up=%d err=%v", count, err)
	}
}
