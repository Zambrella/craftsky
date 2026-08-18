package ownerlifecycle

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

type recordingTerminalPurgeObserver struct {
	mu    sync.Mutex
	calls []TerminalPurgeObservation
}

func (observer *recordingTerminalPurgeObserver) ObserveTerminalPurge(observation TerminalPurgeObservation) {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	observer.calls = append(observer.calls, observation)
}

func (observer *recordingTerminalPurgeObserver) has(operation, result string) bool {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	for _, call := range observer.calls {
		if call.Operation == operation && call.Result == result {
			return true
		}
	}
	return false
}

func TestTerminalPurgeProcessorObservesClaimsComponentsAndBacklog(t *testing.T) {
	_, store, _, _, _ := newTerminalPurgeProcessorTest(t, 1)
	observer := &recordingTerminalPurgeObserver{}
	processor, err := NewTerminalPurgeProcessor(TerminalPurgeProcessorConfig{
		Store: store, WorkerID: "terminal-observer", ComponentLimit: 1,
		RowBatchSize: 1, LeaseDuration: time.Minute, RetryDelay: time.Second,
		Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := processor.ProcessBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("processed components=%d, want 1", processed)
	}
	for _, want := range [][2]string{{"claim", "success"}, {"component", "success"}, {"backlog", "success"}} {
		if !observer.has(want[0], want[1]) {
			t.Fatalf("missing terminal purge observation %s/%s: %+v", want[0], want[1], observer.calls)
		}
	}
}

func TestTerminalPurgeProcessorRetainsFixedTombstones(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 1)
	claim := claimSpecificTerminalComponent(
		t, store, owner, generation, "owner_lifecycles", "owner",
	)
	result, err := processor.ProcessClaim(context.Background(), claim)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Complete || result.RowsAffected != 0 {
		t.Fatalf("retained tombstone result = %+v", result)
	}
	var state State
	if err := pool.QueryRow(context.Background(), `
		SELECT state FROM owner_lifecycles WHERE owner_did=$1
	`, owner).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != StateTerminal {
		t.Fatalf("retained lifecycle state = %q", state)
	}
}

func TestTerminalPurgeProcessorAnonymizesAuditRowsInBoundedBatches(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 1)
	other := syntax.DID("did:plc:audit-other")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO instagram_audit_events(owner_did,action,subject_kind,subject_id,outcome)
		VALUES($1,'disconnect','account','terminal-subject','ok'),
		      ($1,'disconnect','account','terminal-subject-2','ok'),
		      ($2,'disconnect','account','other-subject','ok')
	`, owner, other); err != nil {
		t.Fatal(err)
	}

	claim := claimSpecificTerminalComponent(
		t, store, owner, generation, "instagram_audit_events", "owner",
	)
	result, err := processor.ProcessClaim(context.Background(), claim)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected != 1 || result.Complete {
		t.Fatalf("first audit batch result = %+v", result)
	}
	var anonymized, retained int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FILTER (WHERE owner_did IS NULL AND subject_id IS NULL),
		       count(*) FILTER (WHERE owner_did=$1 AND subject_id='other-subject')
		FROM instagram_audit_events
	`, other).Scan(&anonymized, &retained); err != nil {
		t.Fatal(err)
	}
	if anonymized != 1 || retained != 1 {
		t.Fatalf("anonymized=%d unrelated-retained=%d, want 1/1", anonymized, retained)
	}
}

func TestTerminalPurgeProcessorPersistsScheduledObjectIntentBeforeRowDeletion(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 10)
	ctx := context.Background()
	mediaID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("terminal-media"))
	attemptID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("terminal-attempt"))
	objectKey := fmt.Sprintf("scheduled-media/v2/%d/%s", generation, mediaID)
	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_post_object_attempts(
			upload_attempt_id,media_id,owner_did,owner_generation,upload_generation,
			object_key,request_fingerprint,remote_outcome,remote_started_at,
			remote_deadline,dispatched_at,completed_at,created_at,updated_at
		) VALUES($1,$2,$3,$4,$4,$5,$6,'accepted',now(),now()+interval '1 minute',
		         now(),now(),now(),now())
	`, attemptID, mediaID, owner, generation, objectKey, bytes.Repeat([]byte{7}, 32)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_post_media(
			id,owner_did,object_key,state,mime_type,size_bytes,sha256,
			unclaimed_expires_at,owner_generation,upload_generation,upload_attempt_id
		) VALUES($1,$2,$3,'uploading','image/jpeg',4,$4,now()+interval '1 hour',$5,$5,$6)
	`, mediaID, owner, objectKey, bytes.Repeat([]byte{9}, 32), generation, attemptID); err != nil {
		t.Fatal(err)
	}

	claim := claimSpecificTerminalComponent(
		t, store, owner, generation, "scheduled_post_media", "owner",
	)
	result, err := processor.ProcessClaim(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected != 1 || !result.Complete {
		t.Fatalf("scheduled media result = %+v", result)
	}
	var mediaRows, cleanupRows int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM scheduled_post_media WHERE owner_did=$1),
		       (SELECT count(*) FROM scheduled_post_cleanup_jobs
		        WHERE owner_did=$1 AND object_key=$2 AND source_attempt_id=$3)
	`, owner, objectKey, attemptID).Scan(&mediaRows, &cleanupRows); err != nil {
		t.Fatal(err)
	}
	if mediaRows != 0 || cleanupRows != 1 {
		t.Fatalf("media rows=%d cleanup intents=%d, want 0/1", mediaRows, cleanupRows)
	}

	attemptClaim := claimSpecificTerminalComponent(
		t, store, owner, generation, "scheduled_post_object_attempts", "owner",
	)
	attemptResult, err := processor.ProcessClaim(ctx, attemptClaim)
	if err != nil {
		t.Fatal(err)
	}
	if attemptResult.RowsAffected != 0 || attemptResult.Complete {
		t.Fatalf("object attempt must remain while cleanup intent exists: %+v", attemptResult)
	}
}

func TestTerminalPurgeProcessorDeletesOnlyOrphanedPushInstallations(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 10)
	ctx := context.Background()
	other := syntax.DID("did:plc:push-other")
	sharedInstallation := uuid.New()
	ownerOnlyInstallation := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_installations(id,device_id,platform,fcm_token)
		VALUES($1,'shared-device','ios','shared-token'),
		      ($2,'owner-device','android','owner-token')
	`, sharedInstallation, ownerOnlyInstallation); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id)
		VALUES($1,$2,$3,$4),($5,$2,$6,$7),($8,$9,$3,$10)
	`, uuid.New(), sharedInstallation, owner, uuid.New(), uuid.New(), other,
		uuid.New(), uuid.New(), ownerOnlyInstallation, uuid.New()); err != nil {
		t.Fatal(err)
	}

	claim := claimSpecificTerminalComponent(
		t, store, owner, generation, "push_account_subscriptions", "recipient",
	)
	result, err := processor.ProcessClaim(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected != 2 || !result.Complete {
		t.Fatalf("push purge result = %+v", result)
	}
	var ownerSubscriptions, otherSubscriptions, sharedRows, ownerOnlyRows int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM push_account_subscriptions WHERE account_did=$1),
		       (SELECT count(*) FROM push_account_subscriptions WHERE account_did=$2),
		       (SELECT count(*) FROM push_installations WHERE id=$3),
		       (SELECT count(*) FROM push_installations WHERE id=$4)
	`, owner, other, sharedInstallation, ownerOnlyInstallation).Scan(
		&ownerSubscriptions, &otherSubscriptions, &sharedRows, &ownerOnlyRows,
	); err != nil {
		t.Fatal(err)
	}
	if ownerSubscriptions != 0 || otherSubscriptions != 1 || sharedRows != 1 || ownerOnlyRows != 0 {
		t.Fatalf("owner=%d other=%d shared=%d owner-only=%d, want 0/1/1/0",
			ownerSubscriptions, otherSubscriptions, sharedRows, ownerOnlyRows)
	}
}

func TestTerminalPurgeProcessorDrainsPostDependentsWithinRowBudget(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 2)
	ctx := context.Background()
	postURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/fanout")
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($1,$2,'fanout','post-cid','terminal row','{}',now())
	`, postURI, owner); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 5; index++ {
		actor := fmt.Sprintf("did:plc:fanout-actor-%d", index)
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_likes(
				uri,did,rkey,cid,subject_uri,subject_cid,record,created_at
			) VALUES($1,$2,$3,$4,$5,'post-cid','{}',now())
		`, fmt.Sprintf("at://%s/social.craftsky.feed.like/%d", actor, index),
			actor, fmt.Sprint(index), fmt.Sprintf("like-cid-%d", index), postURI); err != nil {
			t.Fatal(err)
		}
	}

	claim := claimSpecificTerminalComponent(
		t, store, owner, generation, "craftsky_posts", "owner",
	)
	result, err := processor.ProcessClaim(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected != 2 || result.Complete {
		t.Fatalf("post fan-out first batch=%+v, want two dependents and pending parent", result)
	}
	var posts, likes int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*)::int FROM craftsky_posts WHERE uri=$1),
		       (SELECT count(*)::int FROM craftsky_likes WHERE subject_uri=$1)
	`, postURI).Scan(&posts, &likes); err != nil {
		t.Fatal(err)
	}
	if posts != 1 || likes != 3 {
		t.Fatalf("post/dependents after bounded batch=%d/%d, want 1/3", posts, likes)
	}
}

func TestTerminalPurgeProcessorDrainsPushDeliveriesBeforeSubscription(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 2)
	ctx := context.Background()
	installationID := uuid.New()
	subscriptionID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_installations(id,device_id,platform,fcm_token)
		VALUES($1,'fanout-device','android','fanout-token')
	`, installationID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id)
		VALUES($1,$2,$3,$4)
	`, subscriptionID, installationID, owner, uuid.New()); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 5; index++ {
		notificationID := uuid.New()
		if _, err := pool.Exec(ctx, `
			INSERT INTO notification_events(
				id,recipient_did,actor_did,category,subject_key,source_uri,
				source_cid,source_rkey,eligibility_scope,recipient_followed_actor,
				push_enabled_snapshot,state,first_activity_at,activity_at,
				initial_push_evaluated_at
			) VALUES(
				$1,'did:plc:fanout-recipient','did:plc:fanout-actor','like',$2,$3,
				$4,$2,'everyone',false,true,'active',now(),now(),now()
			)
		`, notificationID, fmt.Sprint(index),
			fmt.Sprintf("at://did:plc:fanout-actor/social.craftsky.feed.like/%d", index),
			fmt.Sprintf("fanout-cid-%d", index)); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO push_deliveries(
				id,notification_id,account_subscription_id,status,next_attempt_at,deadline_at
			) VALUES($1,$2,$3,'pending',now(),now()+interval '1 hour')
		`, uuid.New(), notificationID, subscriptionID); err != nil {
			t.Fatal(err)
		}
	}

	claim := claimSpecificTerminalComponent(
		t, store, owner, generation, "push_account_subscriptions", "recipient",
	)
	result, err := processor.ProcessClaim(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected != 2 || result.Complete {
		t.Fatalf("push fan-out first batch=%+v, want two deliveries and pending subscription", result)
	}
	var subscriptions, deliveries int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*)::int FROM push_account_subscriptions WHERE id=$1),
		       (SELECT count(*)::int FROM push_deliveries WHERE account_subscription_id=$1)
	`, subscriptionID).Scan(&subscriptions, &deliveries); err != nil {
		t.Fatal(err)
	}
	if subscriptions != 1 || deliveries != 3 {
		t.Fatalf("subscription/deliveries after bounded batch=%d/%d, want 1/3", subscriptions, deliveries)
	}
}

func TestTerminalPurgeProcessorDrainsTapJobsBeforeSourceRecord(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 2)
	ctx := context.Background()
	uri := "at://" + owner.String() + "/social.craftsky.feed.post/tap-fanout"
	if _, err := pool.Exec(ctx, `
		INSERT INTO tap_source_records(
			uri,did,collection,rkey,source_event_id,source_fingerprint,revision,
			action,record_bytes,live,ordering_status,projection_disposition
		) VALUES($1,$2,'social.craftsky.feed.post','tap-fanout',1,$3,'1',
		         'delete',0,false,'authoritative','denied_terminal')
	`, uri, owner, bytes.Repeat([]byte{1}, 32)); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 5; index++ {
		if _, err := pool.Exec(ctx, `
			INSERT INTO tap_projection_jobs(
				source_uri,projection_kind,source_event_id,state
			) VALUES($1,$2,1,'pending')
		`, uri, fmt.Sprintf("projection-%d", index)); err != nil {
			t.Fatal(err)
		}
	}

	claim := claimSpecificTerminalComponent(
		t, store, owner, generation, "tap_source_records", "owner",
	)
	result, err := processor.ProcessClaim(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected != 2 || result.Complete {
		t.Fatalf("tap fan-out first batch=%+v, want two jobs and pending source", result)
	}
	var sources, jobs int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*)::int FROM tap_source_records WHERE uri=$1),
		       (SELECT count(*)::int FROM tap_projection_jobs WHERE source_uri=$1)
	`, uri).Scan(&sources, &jobs); err != nil {
		t.Fatal(err)
	}
	if sources != 1 || jobs != 3 {
		t.Fatalf("source/jobs after bounded batch=%d/%d, want 1/3", sources, jobs)
	}
}

func TestTerminalCascadeParentLockRejectsLateChildInsteadOfCascading(t *testing.T) {
	pool, _, _, owner, _ := newTerminalPurgeProcessorTest(t, 1)
	ctx := context.Background()
	postURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/locked")
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($1,$2,'locked','post-cid','terminal row','{}',now())
	`, postURI, owner); err != nil {
		t.Fatal(err)
	}
	entry := terminalInventoryEntry(t, "craftsky_posts", "owner")
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	targets, err := lockTerminalCascadeParentsTx(ctx, tx, entry, owner, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("locked parents=%d, want 1", len(targets))
	}

	insertDone := make(chan error, 1)
	go func() {
		_, insertErr := pool.Exec(context.Background(), `
			INSERT INTO craftsky_likes(
				uri,did,rkey,cid,subject_uri,subject_cid,record,created_at
			) VALUES(
				'at://did:plc:late/social.craftsky.feed.like/1',
				'did:plc:late','1','like-cid',$1,'post-cid','{}',now()
			)
		`, postURI)
		insertDone <- insertErr
	}()
	select {
	case insertErr := <-insertDone:
		t.Fatalf("late child insert did not wait for locked parent: %v", insertErr)
	case <-time.After(100 * time.Millisecond):
	}
	if affected, err := deleteLockedTerminalRoleBatchTx(ctx, tx, entry, targets); err != nil {
		t.Fatal(err)
	} else if affected != 1 {
		t.Fatalf("deleted parents=%d, want 1", affected)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if insertErr := <-insertDone; insertErr == nil {
		t.Fatal("late child insert succeeded after its locked parent was deleted")
	}
}

func TestTerminalCascadeLockedChildFailsClosedWithoutParentCascade(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 1)
	ctx := context.Background()
	postURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/locked-child")
	likeURI := "at://did:plc:locked/social.craftsky.feed.like/1"
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($1,$2,'locked-child','post-cid','terminal row','{}',now());
		INSERT INTO craftsky_likes(
			uri,did,rkey,cid,subject_uri,subject_cid,record,created_at
		) VALUES($3,'did:plc:locked','1','like-cid',$1,'post-cid','{}',now())
	`, postURI, owner, likeURI); err != nil {
		t.Fatal(err)
	}
	childTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = childTx.Rollback(context.Background()) }()
	if _, err := childTx.Exec(ctx, `
		SELECT 1 FROM craftsky_likes WHERE uri=$1 FOR UPDATE
	`, likeURI); err != nil {
		t.Fatal(err)
	}

	claim := claimSpecificTerminalComponent(
		t, store, owner, generation, "craftsky_posts", "owner",
	)
	if _, err := processor.ProcessClaim(ctx, claim); err == nil {
		t.Fatal("terminal purge cascaded through a locked child instead of failing closed")
	}
	var posts, likes int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*)::int FROM craftsky_posts WHERE uri=$1),
		       (SELECT count(*)::int FROM craftsky_likes WHERE uri=$2)
	`, postURI, likeURI).Scan(&posts, &likes); err != nil {
		t.Fatal(err)
	}
	if posts != 1 || likes != 1 {
		t.Fatalf("locked-child failure retained post/like=%d/%d, want 1/1", posts, likes)
	}
}

func TestTerminalCascadeDrainQueriesCoverEveryDeclaredRole(t *testing.T) {
	pool, _, _, owner, _ := newTerminalPurgeProcessorTest(t, 2)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	for _, entry := range TerminalDIDInventory() {
		if terminalCascadePolicies[entry.Table] != "drain" {
			continue
		}
		targets, err := lockTerminalCascadeParentsTx(ctx, tx, entry, owner, 2)
		if err != nil {
			t.Fatalf("lock cascade parent %s/%s: %v", entry.Component, entry.Role, err)
		}
		if _, err := drainTerminalCascadeBatchTx(ctx, tx, entry, owner, targets, 2); err != nil {
			t.Fatalf("cascade drain query %s/%s: %v", entry.Component, entry.Role, err)
		}
	}
}

func terminalInventoryEntry(t *testing.T, component, role string) TerminalDIDEntry {
	t.Helper()
	for _, entry := range TerminalDIDInventory() {
		if entry.Component == component && entry.Role == role {
			return entry
		}
	}
	t.Fatalf("missing terminal inventory entry %s/%s", component, role)
	return TerminalDIDEntry{}
}

func TestTerminalPurgeProcessorArchivesRestorationBeforeDeletingModerationParent(t *testing.T) {
	for _, test := range []struct {
		name        string
		role        string
		wantOutcome string
	}{
		{name: "terminal source", role: "source", wantOutcome: "no_work"},
		{name: "terminal subject", role: "subject", wantOutcome: "cancelled_target_terminal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 10)
			ctx := context.Background()
			other := syntax.DID("did:plc:moderation-other")
			source, subject := owner, other
			if test.role == "subject" {
				source, subject = other, owner
			}
			outputID := "terminal-moderation-" + test.role
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
				) VALUES($1,$2,'pending',now())
			`, outputID, subject); err != nil {
				t.Fatal(err)
			}

			claim := claimSpecificTerminalComponent(
				t, store, owner, generation, "moderation_outputs", test.role,
			)
			result, err := processor.ProcessClaim(ctx, claim)
			if err != nil {
				t.Fatal(err)
			}
			if result.RowsAffected != 1 || !result.Complete {
				t.Fatalf("moderation parent purge=%+v", result)
			}
			var parents, outbox int
			var outcome string
			if err := pool.QueryRow(ctx, `
				SELECT (SELECT count(*)::int FROM moderation_outputs WHERE id=$1),
				       (SELECT count(*)::int FROM moderation_restoration_outbox WHERE moderation_output_id=$1),
				       (SELECT outcome FROM moderation_restoration_history WHERE moderation_output_id=$1)
			`, outputID).Scan(&parents, &outbox, &outcome); err != nil {
				t.Fatal(err)
			}
			if parents != 0 || outbox != 0 || outcome != test.wantOutcome {
				t.Fatalf("parents=%d outbox=%d outcome=%q, want 0/0/%q", parents, outbox, outcome, test.wantOutcome)
			}
		})
	}
}

func TestTerminalPurgeProcessorCancelsQueuedRestorationBeforeDeletingModerationSource(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 10)
	ctx := context.Background()
	other := syntax.DID("did:plc:moderation-queued-target")
	outputID := "terminal-moderation-queued"
	jobID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,value,action)
		VALUES($1,$2,'account',$3,'hide','negate')
	`, outputID, owner, other); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs(
			id,owner_did,reason,status,next_attempt_at
		) VALUES($1,$2,$3,'queued',now())
	`, jobID, other, "moderationCleared:"+outputID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_restoration_outbox(
			moderation_output_id,target_did,status,reconciliation_job_id,
			created_at,processed_at
		) VALUES($1,$2,'queued',$3,now(),now())
	`, outputID, other, jobID); err != nil {
		t.Fatal(err)
	}

	claim := claimSpecificTerminalComponent(t, store, owner, generation, "moderation_outputs", "source")
	result, err := processor.ProcessClaim(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected != 1 || !result.Complete {
		t.Fatalf("queued moderation purge=%+v", result)
	}
	var jobStatus, outcome string
	if err := pool.QueryRow(ctx, `
		SELECT job.status,history.outcome
		FROM instagram_reconciliation_jobs job
		JOIN moderation_restoration_history history ON history.moderation_output_id=$2
		WHERE job.id=$1
	`, jobID, outputID).Scan(&jobStatus, &outcome); err != nil {
		t.Fatal(err)
	}
	if jobStatus != "ignored" || outcome != "queued" {
		t.Fatalf("job=%q history=%q, want ignored/queued", jobStatus, outcome)
	}
}

func TestTerminalPurgeProcessorWaitsForProcessingModerationRestoration(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 10)
	ctx := context.Background()
	other := syntax.DID("did:plc:moderation-processing-target")
	outputID := "terminal-moderation-processing"
	jobID := uuid.New()
	leaseToken := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,value,action)
		VALUES($1,$2,'account',$3,'hide','negate')
	`, outputID, owner, other); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs(
			id,owner_did,reason,status,attempts,next_attempt_at,
			lease_token,lease_expires_at
		) VALUES($1,$2,$3,'processing',1,now(),$4,now()+interval '1 minute')
	`, jobID, other, "moderationCleared:"+outputID, leaseToken); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_restoration_outbox(
			moderation_output_id,target_did,status,reconciliation_job_id,
			created_at,processed_at
		) VALUES($1,$2,'queued',$3,now(),now())
	`, outputID, other, jobID); err != nil {
		t.Fatal(err)
	}

	claim := claimSpecificTerminalComponent(t, store, owner, generation, "moderation_outputs", "source")
	result, err := processor.ProcessClaim(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected != 0 || result.Complete {
		t.Fatalf("processing restoration must block parent purge: %+v", result)
	}
	var parentRows, outboxRows int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*)::int FROM moderation_outputs WHERE id=$1),
		       (SELECT count(*)::int FROM moderation_restoration_outbox WHERE moderation_output_id=$1)
	`, outputID).Scan(&parentRows, &outboxRows); err != nil {
		t.Fatal(err)
	}
	if parentRows != 1 || outboxRows != 1 {
		t.Fatalf("processing barrier retained parent/outbox=%d/%d, want 1/1", parentRows, outboxRows)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs
		SET status='completed',terminal_at=now(),lease_token=NULL,lease_expires_at=NULL
		WHERE id=$1
	`, jobID); err != nil {
		t.Fatal(err)
	}
	claim = claimSpecificTerminalComponent(t, store, owner, generation, "moderation_outputs", "source")
	result, err = processor.ProcessClaim(ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected != 1 || !result.Complete {
		t.Fatalf("settled restoration parent purge=%+v", result)
	}
}

func newTerminalPurgeProcessorTest(
	t *testing.T,
	rowBatchSize int,
) (*pgxpool.Pool, *Store, *TerminalPurgeProcessor, syntax.DID, int64) {
	t.Helper()
	pool := testdb.WithSchema(t, "")
	applyAllTerminalInventoryMigrations(t, pool)
	fencer, err := NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewTerminalPurgeProcessor(TerminalPurgeProcessorConfig{
		Store: store, WorkerID: "terminal-purge-special-test", ComponentLimit: 100,
		RowBatchSize: rowBatchSize, LeaseDuration: time.Minute, RetryDelay: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:terminal-special")
	terminal, err := store.Terminalize(context.Background(), TerminalizeRequest{
		Owner: owner, Reason: "identityDeleted",
	})
	if err != nil {
		t.Fatal(err)
	}
	return pool, store, processor, owner, terminal.Generation
}

func TestTerminalPurgeProcessorDeletesKeyOrderedBatchesAndPreservesOtherOwners(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllTerminalInventoryMigrations(t, pool)
	fencer, err := NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	store, err := NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewTerminalPurgeProcessor(TerminalPurgeProcessorConfig{
		Store: store, WorkerID: "terminal-purge-test", ComponentLimit: 100,
		RowBatchSize: 2, LeaseDuration: time.Minute, RetryDelay: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	owner := syntax.DID("did:plc:purge-owner")
	other := syntax.DID("did:plc:purge-other")
	otherLifecycle, err := store.EnsureOnboardingOwner(ctx, other)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(ctx, TransitionRequest{
		Owner: other, ExpectedGeneration: otherLifecycle.Generation,
		To: StateActive, Reason: "profileCreated",
	}); err != nil {
		t.Fatal(err)
	}
	terminal, err := store.Terminalize(ctx, TerminalizeRequest{
		Owner: owner, Reason: "identityDeleted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'owner-cid'),($2,'other-cid')
	`, owner, other); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 5; index++ {
		uri := fmt.Sprintf("at://%s/social.craftsky.feed.post/%02d", owner, index)
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
			VALUES($1,$2,$3,$4,'terminal row','{}',now())
		`, uri, owner, fmt.Sprintf("%02d", index), fmt.Sprintf("cid-%02d", index)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($1,$2,'other','other-cid','other row','{}',now())
	`, "at://"+other.String()+"/social.craftsky.feed.post/other", other); err != nil {
		t.Fatal(err)
	}

	wantRemainingByBatch := []int{3, 1, 0}
	wantRowsByBatch := []int64{2, 2, 1}
	for iteration, wantRemaining := range wantRemainingByBatch {
		if iteration > 0 {
			now = now.Add(2 * time.Second)
		}
		claim := claimSpecificTerminalComponent(
			t, store, owner, terminal.Generation, "craftsky_posts", "owner",
		)
		result, err := processor.ProcessClaim(ctx, claim)
		if err != nil {
			t.Fatalf("process post purge batch %d: %v", iteration+1, err)
		}
		if result.RowsAffected != wantRowsByBatch[iteration] {
			t.Fatalf("post purge batch %d rows = %d", iteration+1, result.RowsAffected)
		}
		var remaining, unrelated int
		if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM craftsky_posts WHERE did=$1`, owner).Scan(&remaining); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM craftsky_posts WHERE did=$1`, other).Scan(&unrelated); err != nil {
			t.Fatal(err)
		}
		if remaining != wantRemaining || unrelated != 1 {
			t.Fatalf("after batch %d remaining=%d unrelated=%d, want %d/1", iteration+1, remaining, unrelated, wantRemaining)
		}
		if result.Complete != (wantRemaining == 0) {
			t.Fatalf("post purge batch %d complete=%t", iteration+1, result.Complete)
		}
	}
}

func claimSpecificTerminalComponent(
	t *testing.T,
	store *Store,
	owner syntax.DID,
	generation int64,
	component string,
	role string,
) PurgeClaim {
	t.Helper()
	token := uuid.New()
	now := store.now().UTC().Truncate(time.Microsecond)
	leaseExpiresAt := now.Add(time.Minute)
	result, err := store.pool.Exec(context.Background(), `
		UPDATE owner_purge_components
		SET state='running',attempts=attempts+1,lease_owner='terminal-purge-test',
		    lease_token=$5,lease_expires_at=$6,updated_at=$7
		WHERE owner_did=$1 AND owner_generation=$2 AND component=$3 AND did_role=$4
		  AND state='pending'
	`, owner, generation, component, role, token, leaseExpiresAt, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.RowsAffected() != 1 {
		t.Fatalf("terminal component %s/%s was not claimable", component, role)
	}
	return PurgeClaim{
		Owner: owner, OwnerGeneration: generation, Component: component, DIDRole: role,
		State: PurgeRunning, Attempts: 1, LeaseOwner: "terminal-purge-test",
		LeaseToken: token, LeaseExpiresAt: leaseExpiresAt,
	}
}
