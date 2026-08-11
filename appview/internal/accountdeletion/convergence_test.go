package accountdeletion

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/testdb"
)

func TestConvergenceRequiresReceiptsAndAbsentOrRetractedIndexedEffects(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	jobID := uuid.MustParse("00000000-0000-4000-8000-000000000930")
	owner := syntax.DID("did:plc:alice")
	postURI := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/post1")
	likeURI := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.like/like1")
	now := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	seedSQL := `
		INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile-cid');
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($2,$1,'post1','post-cid','post','{}',$3);
		INSERT INTO account_deletion_operations(id,owner_did,state,phase,accepted_at)
		VALUES($4,$1,'active','waitingForIndexerConvergence',$3);
		INSERT INTO account_deletion_expected_records(job_id,uri,collection,registered_at,delete_requested_at)
		VALUES($4,$2,'social.craftsky.feed.post',$3,$3);
		INSERT INTO account_deletion_index_receipts(job_id,uri,collection,tap_event_id,repo_revision,handled_at)
		VALUES($4,$2,'social.craftsky.feed.post',51,'rev-post',$3);
		INSERT INTO notification_events(
			id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,
			eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,
			first_activity_at,activity_at,initial_push_evaluated_at
		) VALUES(
			'00000000-0000-4000-8000-000000000931','did:plc:bob',$1,'reply','reply',$2,'post-cid','post1',
			'everyone',false,true,'active',$3,$3,$3
		)
	`
	seedSQL = strings.NewReplacer(
		"$4", "'"+jobID.String()+"'",
		"$3", "'"+now.Format(time.RFC3339Nano)+"'",
		"$2", "'"+postURI.String()+"'",
		"$1", "'"+owner.String()+"'",
	).Replace(seedSQL)
	if _, err := pool.Exec(ctx, seedSQL); err != nil {
		t.Fatalf("seed convergence fixture: %v", err)
	}

	verifier := NewConvergenceVerifier(pool)
	converged, err := verifier.IsConverged(ctx, jobID, owner)
	if err != nil || converged {
		t.Fatalf("live indexed effects converged=%t err=%v", converged, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_posts WHERE uri=$1`, postURI); err != nil {
		t.Fatalf("simulate successful post delete indexing: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE notification_events
		SET state='retracted',retracted_at=$2,retraction_reason='source_deleted'
		WHERE source_uri=$1
	`, postURI, now); err != nil {
		t.Fatalf("simulate successful delete indexing: %v", err)
	}
	converged, err = verifier.IsConverged(ctx, jobID, owner)
	if err != nil || !converged {
		t.Fatalf("retracted indexed effects converged=%t err=%v", converged, err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_expected_records(job_id,uri,collection,registered_at,delete_requested_at)
		VALUES($1,$2,'social.craftsky.feed.like',$3,$3)
	`, jobID, likeURI, now); err != nil {
		t.Fatalf("seed missing receipt: %v", err)
	}
	converged, err = verifier.IsConverged(ctx, jobID, owner)
	if err != nil || converged {
		t.Fatalf("missing receipt converged=%t err=%v", converged, err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_index_receipts(job_id,uri,collection,tap_event_id,repo_revision,handled_at)
		VALUES($1,$2,'social.craftsky.feed.like',52,'rev-like',$3)
	`, jobID, likeURI, now); err != nil {
		t.Fatalf("seed final receipt: %v", err)
	}
	converged, err = verifier.IsConverged(ctx, jobID, owner)
	if err != nil || !converged {
		t.Fatalf("fully receipted convergence=%t err=%v", converged, err)
	}
}
