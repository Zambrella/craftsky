package accountdeletion

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestPrivateCleanupReplaysEveryComponentAfterFailure(t *testing.T) {
	t.Parallel()

	owner := syntax.DID("did:plc:alice")
	var calls []string
	firstFailure := true
	cleaner, err := NewPrivateCleaner([]PrivateCleanupComponent{
		fakePrivateCleanupComponent{name: "databasePrivate", run: func(gotOwner syntax.DID) error {
			if gotOwner != owner {
				t.Fatalf("database component scope = %s", gotOwner)
			}
			calls = append(calls, "databasePrivate")
			return nil
		}},
		fakePrivateCleanupComponent{name: "scheduledPosts", run: func(syntax.DID) error {
			calls = append(calls, "scheduledPosts")
			if firstFailure {
				firstFailure = false
				return errors.New("object cleanup queue unavailable")
			}
			return nil
		}},
		fakePrivateCleanupComponent{name: "instagram", run: func(syntax.DID) error {
			calls = append(calls, "instagram")
			return nil
		}},
	})
	if err != nil {
		t.Fatalf("construct cleaner: %v", err)
	}

	if err := cleaner.Run(context.Background(), owner); err == nil {
		t.Fatal("first cleanup unexpectedly succeeded")
	}
	if got, want := calls, []string{"databasePrivate", "scheduledPosts"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first calls = %v, want %v", got, want)
	}
	if err := cleaner.Run(context.Background(), owner); err != nil {
		t.Fatalf("retry cleanup: %v", err)
	}
	if got, want := calls, []string{"databasePrivate", "scheduledPosts", "databasePrivate", "scheduledPosts", "instagram"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("retry calls = %v, want %v", got, want)
	}
}

func TestDatabasePrivateCleanupDeletesOnlyOwnerPrivateState(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	jobID := uuid.MustParse("00000000-0000-4000-8000-000000000902")
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)

	seedSQL := `
		INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'alice-profile-cid'),($2,'bob-profile-cid');
		INSERT INTO craftsky_account_types(owner_did,account_type) VALUES($1,'business'),($2,'business');
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES
			($1,'deleting',1,2,'accountDeletionTest',$3,$3,$3),
			($2,'active',1,1,'accountDeletionTest',$3,$3,$3);
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES
			('at://did:plc:alice/social.craftsky.feed.post/a',$1,'a','alice-post-cid','alice','{}',$3),
			('at://did:plc:bob/social.craftsky.feed.post/b',$2,'b','bob-post-cid','bob','{}',$3);
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'alice-deletion-oauth','{}'),($2,'bob-oauth','{}');
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation
		) VALUES($4,$1,1,'active',$3,'alice-deletion-oauth',1);

		INSERT INTO craftsky_recent_searches(id,viewer_did,search_type,display_label,normalized_payload,normalized_payload_hash)
		VALUES('alice-search',$1,'profile','Alice private search','{}','alice-search-hash'),
		      ('bob-search',$2,'profile','Bob private search','{}','bob-search-hash');
		INSERT INTO actor_mutes(owner_did,subject_did) VALUES($1,$2),($2,$1);
		INSERT INTO account_language_preferences(account_did,primary_language) VALUES($1,'en'),($2,'en');
		INSERT INTO profile_customisations(owner_did,colour,profile_border,profile_background)
		VALUES($1,'blue','none','plain'),($2,'red','none','plain');
		INSERT INTO profile_pins(owner_did,slot,post_uri,state_token,created_at,updated_at)
		VALUES($1,'standard','at://did:plc:alice/social.craftsky.feed.post/a',gen_random_uuid(),$3,$3),
		      ($2,'standard','at://did:plc:bob/social.craftsky.feed.post/b',gen_random_uuid(),$3,$3);
		INSERT INTO saved_post_folders(id,owner_did,name,created_at,updated_at)
		VALUES('10000000-0000-4000-8000-000000000001',$1,'Alice folder',$3,$3),
		      ('10000000-0000-4000-8000-000000000002',$2,'Bob folder',$3,$3);
		INSERT INTO saved_posts(owner_did,post_uri,folder_id,saved_at)
		VALUES($1,'at://did:plc:bob/social.craftsky.feed.post/b','10000000-0000-4000-8000-000000000001',$3),
		      ($2,'at://did:plc:alice/social.craftsky.feed.post/a','10000000-0000-4000-8000-000000000002',$3);
		INSERT INTO notification_preferences(account_did,category,scope,push_enabled)
		VALUES($1,'like','everyone',true),($2,'like','everyone',true);
		INSERT INTO notification_seen_state(account_did,last_seen_revision) VALUES($1,1),($2,1);
		INSERT INTO notification_events(
			id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,
			eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,
			first_activity_at,activity_at,initial_push_evaluated_at
		) VALUES
			('20000000-0000-4000-8000-000000000001',$1,$2,'like','alice-recipient','at://did:plc:bob/social.craftsky.feed.like/l','cid','l','everyone',false,true,'active',$3,$3,$3),
			('20000000-0000-4000-8000-000000000002',$2,$1,'like','alice-actor','at://did:plc:alice/social.craftsky.feed.like/l','cid','l','everyone',false,true,'active',$3,$3,$3);
		INSERT INTO push_installations(id,device_id,platform,fcm_token)
		VALUES('30000000-0000-4000-8000-000000000001','shared-device','ios','shared-token');
		INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id)
		VALUES
			('40000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000001',$1,'50000000-0000-4000-8000-000000000001'),
			('40000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000001',$2,'50000000-0000-4000-8000-000000000002');
		INSERT INTO moderation_reports(
			id,reporter_did,subject_type,subject_did,submitted_handle_snapshot,reason_type,
			forwarding_status,forwarding_prepared_at
		) VALUES('alice-report',$1,'account',$2,'alice.test','spam','prepared_not_submitted',$3),
		        ('bob-report',$2,'account',$1,'bob.test','spam','prepared_not_submitted',$3);
		INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,value,action)
		VALUES('alice-output',$1,'account',$2,'warn','apply'),
		      ('bob-output',$2,'account',$1,'warn','apply');
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
		VALUES($1,'alice.test','alice.test',$3),($2,'bob.test','bob.test',$3);
		INSERT INTO bluesky_profiles(did,record_cid) VALUES($1,'alice-bsky-cid'),($2,'bob-bsky-cid');
	`
	seedSQL = strings.NewReplacer(
		"$4", "'"+jobID.String()+"'",
		"$3", "'"+now.Format(time.RFC3339Nano)+"'",
		"$2", "'"+bob.String()+"'",
		"$1", "'"+alice.String()+"'",
	).Replace(seedSQL)
	_, err := pool.Exec(ctx, seedSQL)
	if err != nil {
		t.Fatalf("seed owner-private cleanup fixtures: %v", err)
	}

	component := NewDatabasePrivateCleanup(pool)
	if err := component.Purge(ctx, alice); err != nil {
		t.Fatalf("purge Alice private database state: %v", err)
	}
	if err := component.Purge(ctx, alice); err != nil {
		t.Fatalf("repeat Alice private database purge: %v", err)
	}

	for _, table := range []string{
		"craftsky_recent_searches", "actor_mutes", "account_language_preferences",
		"profile_customisations", "profile_pins", "saved_post_folders", "saved_posts",
		"notification_preferences", "notification_seen_state", "push_account_subscriptions",
	} {
		assertPrivateCleanupCount(t, pool, table, ownerColumnForPrivateCleanupTest(table), alice, 0)
		assertPrivateCleanupCount(t, pool, table, ownerColumnForPrivateCleanupTest(table), bob, 1)
	}
	assertPrivateCleanupCount(t, pool, "notification_events", "recipient_did", alice, 0)
	assertPrivateCleanupCount(t, pool, "notification_events", "recipient_did", bob, 1)
	assertPrivateCleanupCount(t, pool, "moderation_reports", "reporter_did", alice, 0)
	assertPrivateCleanupCount(t, pool, "moderation_reports", "reporter_did", bob, 0)
	assertPrivateCleanupCount(t, pool, "moderation_outputs", "source_did", alice, 0)
	assertPrivateCleanupCount(t, pool, "moderation_outputs", "source_did", bob, 0)
	assertPrivateCleanupCount(t, pool, "push_installations", "id", "30000000-0000-4000-8000-000000000001", 1)

	// Public/indexer-owned projections, shared caches, and the bound deletion
	// authority survive this phase and are handled by their dedicated gates.
	assertPrivateCleanupCount(t, pool, "craftsky_profiles", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "craftsky_posts", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "craftsky_account_types", "owner_did", alice, 1)
	assertPrivateCleanupCount(t, pool, "craftsky_account_types", "owner_did", bob, 1)
	assertPrivateCleanupCount(t, pool, "atproto_identity_cache", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "bluesky_profiles", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "oauth_sessions", "account_did", alice, 1)
}

func TestDatabasePrivateCleanupClassifiesModerationRestorationBeforeParentDeletion(t *testing.T) {
	for _, test := range []struct {
		name        string
		ownerSource bool
		wantOutcome string
		wantJobs    int
	}{
		{name: "deleting moderator preserves target restoration", ownerSource: true, wantOutcome: "queued", wantJobs: 1},
		{name: "deleting target cancels restoration", ownerSource: false, wantOutcome: "cancelled_target_terminal", wantJobs: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, "")
			applyAllAccountDeletionTestMigrations(t, pool)
			ctx := context.Background()
			owner := syntax.DID("did:plc:deleting-moderation-owner")
			other := syntax.DID("did:plc:moderation-other-owner")
			source, subject := owner, other
			if !test.ownerSource {
				source, subject = other, owner
			}
			jobID := uuid.New()
			outputID := "accepted-deletion-moderation-output"
			now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
			if _, err := pool.Exec(ctx, `
				INSERT INTO owner_lifecycles(
					owner_did,state,generation,auth_epoch,transition_reason,
					transitioned_at,created_at,updated_at
				) VALUES
					($1,'deleting',2,2,'accountDeletionAccepted',$3,$3,$3),
					($2,'active',1,1,'test',$3,$3,$3)
			`, owner, other, now); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO oauth_sessions(account_did,session_id,data)
				VALUES($1,'deletion-oauth','{}')
			`, owner); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO account_deletion_operations(
					id,owner_did,owner_generation,state,accepted_at,
					deletion_oauth_session_id,deletion_credential_generation
				) VALUES($1,$2,1,'active',$3,'deletion-oauth',1)
			`, jobID, owner, now); err != nil {
				t.Fatal(err)
			}
			if test.ownerSource {
				if _, err := pool.Exec(ctx, `
					INSERT INTO instagram_account_links(
						id,owner_did,state,igsid,igsid_digest_version,igsid_digest,
						username,username_normalized,discoverable,conflict_pending,
						verified_at,updated_at
					) VALUES($1,$2,'active','accepted-deletion-target',1,$3,
					         'accepted.deletion.target','accepted.deletion.target',true,false,
					         $4,$4)
				`, uuid.New(), subject, bytes.Repeat([]byte{0x55}, 32), now); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO moderation_outputs(
					id,source_did,subject_type,subject_did,value,action
				) VALUES($1,$2,'account',$3,'hide','negate')
			`, outputID, source, subject); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO moderation_restoration_outbox(
					moderation_output_id,target_did,status,created_at
				) VALUES($1,$2,'pending',$3)
			`, outputID, subject, now); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO moderation_idempotency_receipts(
					request_key_hash,request_fingerprint,output_id,output_status,
					created_at,expires_at
				) VALUES($1,$2,$3,'indexed',$4,$4::timestamptz+interval '24 hours')
			`, bytes.Repeat([]byte{0x66}, 32), bytes.Repeat([]byte{0x77}, 32), outputID, now); err != nil {
				t.Fatal(err)
			}

			if err := NewDatabasePrivateCleanup(pool).Purge(ctx, owner); err != nil {
				t.Fatalf("purge accepted deletion moderation rows: %v", err)
			}

			var parents, outbox, receipts, jobs int
			var outcome string
			if err := pool.QueryRow(ctx, `
				SELECT
					(SELECT count(*)::int FROM moderation_outputs WHERE id=$1),
					(SELECT count(*)::int FROM moderation_restoration_outbox WHERE moderation_output_id=$1),
					(SELECT count(*)::int FROM moderation_idempotency_receipts WHERE output_id=$1),
					(SELECT count(*)::int FROM instagram_reconciliation_jobs
					 WHERE reason='moderationCleared:' || $1 AND status='queued'),
					(SELECT outcome FROM moderation_restoration_history WHERE moderation_output_id=$1)
			`, outputID).Scan(&parents, &outbox, &receipts, &jobs, &outcome); err != nil {
				t.Fatal(err)
			}
			if parents != 0 || outbox != 0 || receipts != 0 || jobs != test.wantJobs || outcome != test.wantOutcome {
				t.Fatalf(
					"parents=%d outbox=%d receipts=%d jobs=%d outcome=%q, want 0/0/0/%d/%q",
					parents, outbox, receipts, jobs, outcome, test.wantJobs, test.wantOutcome,
				)
			}
		})
	}
}

func TestDatabasePrivateCleanupRetriesProcessingModerationRestoration(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	owner := syntax.DID("did:plc:deleting-processing-source")
	target := syntax.DID("did:plc:processing-target")
	operationID := uuid.New()
	reconciliationJobID := uuid.New()
	leaseToken := uuid.New()
	outputID := "accepted-deletion-processing-moderation-output"
	now := time.Date(2026, 8, 19, 12, 30, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES
			($1,'deleting',2,2,'accountDeletionAccepted',$3,$3,$3),
			($2,'active',1,1,'test',$3,$3,$3)
	`, owner, target, now); err != nil {
		t.Fatalf("seed processing moderation lifecycles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatalf("seed processing moderation OAuth session: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation
		) VALUES($1,$2,1,'active',$3,'deletion-oauth',1)
	`, operationID, owner, now); err != nil {
		t.Fatalf("seed processing moderation deletion operation: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(
			id,source_did,subject_type,subject_did,value,action
		) VALUES($1,$2,'account',$3,'hide','negate')
	`, outputID, owner, target); err != nil {
		t.Fatalf("seed processing moderation output: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs(
			id,owner_did,reason,status,next_attempt_at,lease_token,
			lease_expires_at,created_at,updated_at
		) VALUES($1,$2,'moderationCleared:' || $3,'processing',$4,$5,
		         $4::timestamptz+interval '1 minute',$4,$4)
	`, reconciliationJobID, target, outputID, now, leaseToken); err != nil {
		t.Fatalf("seed processing moderation reconciliation job: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_restoration_outbox(
			moderation_output_id,target_did,status,reconciliation_job_id,
			created_at,processed_at
		) VALUES($1,$2,'queued',$3,$4,$4)
	`, outputID, target, reconciliationJobID, now); err != nil {
		t.Fatalf("seed processing moderation outbox: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_idempotency_receipts(
			request_key_hash,request_fingerprint,output_id,output_status,
			created_at,expires_at
		) VALUES(
			decode(repeat('88',32),'hex'),decode(repeat('99',32),'hex'),
			$1,'indexed',$2,$2::timestamptz+interval '24 hours'
		)
	`, outputID, now); err != nil {
		t.Fatalf("seed processing moderation receipt: %v", err)
	}

	cleanup := NewDatabasePrivateCleanup(pool)
	if err := cleanup.Purge(ctx, owner); err == nil || !strings.Contains(err.Error(), "reconciliation work is processing") {
		t.Fatalf("processing cleanup error = %v", err)
	}
	var parents, outbox, receipts, jobs, history int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM moderation_outputs WHERE id=$1),
			(SELECT count(*) FROM moderation_restoration_outbox WHERE moderation_output_id=$1),
			(SELECT count(*) FROM moderation_idempotency_receipts WHERE output_id=$1),
			(SELECT count(*) FROM instagram_reconciliation_jobs WHERE id=$2 AND status='processing'),
			(SELECT count(*) FROM moderation_restoration_history WHERE moderation_output_id=$1)
	`, outputID, reconciliationJobID).Scan(&parents, &outbox, &receipts, &jobs, &history); err != nil {
		t.Fatal(err)
	}
	if parents != 1 || outbox != 1 || receipts != 1 || jobs != 1 || history != 0 {
		t.Fatalf(
			"processing parent=%d outbox=%d receipt=%d job=%d history=%d, want 1/1/1/1/0",
			parents, outbox, receipts, jobs, history,
		)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs
		SET status='completed',lease_token=NULL,lease_expires_at=NULL,
		    terminal_at=$2,updated_at=$2
		WHERE id=$1
	`, reconciliationJobID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := cleanup.Purge(ctx, owner); err != nil {
		t.Fatalf("retry cleanup after reconciliation completion: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM moderation_outputs WHERE id=$1),
			(SELECT count(*) FROM moderation_restoration_outbox WHERE moderation_output_id=$1),
			(SELECT count(*) FROM moderation_idempotency_receipts WHERE output_id=$1),
			(SELECT count(*) FROM instagram_reconciliation_jobs WHERE id=$2 AND status='completed'),
			(SELECT count(*) FROM moderation_restoration_history
			 WHERE moderation_output_id=$1 AND outcome='queued')
	`, outputID, reconciliationJobID).Scan(&parents, &outbox, &receipts, &jobs, &history); err != nil {
		t.Fatal(err)
	}
	if parents != 0 || outbox != 0 || receipts != 0 || jobs != 1 || history != 1 {
		t.Fatalf(
			"completed parent=%d outbox=%d receipt=%d job=%d history=%d, want 0/0/0/1/1",
			parents, outbox, receipts, jobs, history,
		)
	}
}

func TestDatabasePrivateCleanupHoldsTargetFenceThroughModerationPromotion(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	owner := syntax.DID("did:plc:deleting-moderation-source")
	target := syntax.DID("did:plc:deletion-fenced-target")
	jobID := uuid.New()
	outputID := "accepted-deletion-fenced-moderation-output"
	now := time.Date(2026, 8, 20, 13, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES
			($1,'deleting',2,2,'accountDeletionAccepted',$3,$3,$3),
			($2,'active',1,1,'test',$3,$3,$3)
	`, owner, target, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation
		) VALUES($1,$2,1,'active',$3,'deletion-oauth',1)
	`, jobID, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_account_links(
			id,owner_did,state,igsid,igsid_digest_version,igsid_digest,
			username,username_normalized,discoverable,conflict_pending,
			verified_at,updated_at
		) VALUES($1,$2,'active','deletion-fenced-target',1,$3,
		         'deletion.fenced.target','deletion.fenced.target',true,false,
		         $4,$4)
	`, uuid.New(), target, bytes.Repeat([]byte{0x31}, 32), now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(
			id,source_did,subject_type,subject_did,value,action
		) VALUES($1,$2,'account',$3,'hide','negate')
	`, outputID, owner, target); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_restoration_outbox(
			moderation_output_id,target_did,status,created_at
		) VALUES($1,$2,'pending',$3)
	`, outputID, target, now); err != nil {
		t.Fatal(err)
	}

	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	if _, err := blocker.Exec(ctx, `
		SELECT moderation_output_id
		FROM moderation_restoration_outbox
		WHERE moderation_output_id=$1
		FOR UPDATE
	`, outputID); err != nil {
		t.Fatal(err)
	}

	cleaned := make(chan error, 1)
	go func() {
		cleaned <- NewDatabasePrivateCleanup(pool).Purge(ctx, owner)
	}()
	waitForModerationParentRowLock(t, pool, outputID)

	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	terminalized := make(chan error, 1)
	go func() {
		_, err := lifecycles.Terminalize(ctx, ownerlifecycle.TerminalizeRequest{
			Owner: target, Reason: "identityDeleted",
		})
		terminalized <- err
	}()
	select {
	case err := <-terminalized:
		t.Fatalf("target terminal transition crossed accepted-deletion promotion: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	if err := blocker.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-cleaned:
		if err != nil {
			t.Fatalf("private cleanup after barrier: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("private cleanup did not resume")
	}
	select {
	case err := <-terminalized:
		if err != nil {
			t.Fatalf("target terminal transition after promotion: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("target terminal transition did not resume")
	}
}

func waitForModerationParentRowLock(t *testing.T, pool *pgxpool.Pool, outputID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		_, lockErr := tx.Exec(context.Background(), `
			SELECT id FROM moderation_outputs WHERE id=$1 FOR UPDATE NOWAIT
		`, outputID)
		_ = tx.Rollback(context.Background())
		var postgresError *pgconn.PgError
		if errors.As(lockErr, &postgresError) && postgresError.Code == "55P03" {
			return
		}
		if lockErr != nil {
			t.Fatalf("probe moderation parent row lock: %v", lockErr)
		}
		if time.Now().After(deadline) {
			t.Fatal("private cleanup did not lock moderation parent")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func ownerColumnForPrivateCleanupTest(table string) string {
	switch table {
	case "craftsky_recent_searches":
		return "viewer_did"
	case "account_language_preferences", "notification_preferences", "notification_seen_state", "push_account_subscriptions":
		return "account_did"
	default:
		return "owner_did"
	}
}

func assertPrivateCleanupCount(t *testing.T, pool *pgxpool.Pool, table, column string, value any, want int) {
	t.Helper()
	var got int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s=$1", table, column)
	if err := pool.QueryRow(context.Background(), query, value).Scan(&got); err != nil {
		t.Fatalf("count %s.%s: %v", table, column, err)
	}
	if got != want {
		t.Fatalf("%s.%s=%v count=%d want=%d", table, column, value, got, want)
	}
}

type fakePrivateCleanupComponent struct {
	name string
	run  func(syntax.DID) error
}

func (component fakePrivateCleanupComponent) Name() string { return component.name }

func (component fakePrivateCleanupComponent) Purge(_ context.Context, owner syntax.DID) error {
	return component.run(owner)
}
