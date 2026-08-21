package accountdeletion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestAcceptanceBindsFreshOAuthBeforeRevokingOrdinaryAccessAndFinalizesWithoutResidue(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	store := NewStore(pool, func() time.Time { return now })
	seedAccountDeletionOwner(t, pool, owner, "active", 1, 1, now)

	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'ordinary-oauth','{}'),($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id)
		VALUES(decode(repeat('01',32),'hex'),$1,'ordinary-oauth')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateIntent(ctx, IntentRecord{
		JobID: jobID, Owner: owner,
		ConfirmationHandleHash: HashSecret("@alice.test"),
		ExpiresAt:              now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteReauthentication(ctx, jobID, owner, "deletion-oauth", HashSecret("proof")); err != nil {
		t.Fatal(err)
	}
	accepted, err := store.Accept(ctx, AcceptanceRequest{
		JobID: jobID, Owner: owner, ReauthProof: "proof", ConfirmationHandle: "@alice.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != StatusActive || accepted.DeletionOAuthSessionID != "deletion-oauth" {
		t.Fatalf("accepted operation = %+v", accepted)
	}
	if got, err := store.BoundOAuthSession(ctx, jobID, owner); err != nil || got != "deletion-oauth" {
		t.Fatalf("worker OAuth authority = (%q, %v)", got, err)
	}
	for _, scope := range []struct {
		jobID uuid.UUID
		owner syntax.DID
	}{
		{jobID: uuid.MustParse("10000000-0000-4000-8000-000000000009"), owner: owner},
		{jobID: jobID, owner: syntax.DID("did:plc:bob")},
	} {
		if _, err := store.BoundOAuthSession(ctx, scope.jobID, scope.owner); !errors.Is(err, ErrBoundOAuthUnauthorized) {
			t.Fatalf("cross-scope worker OAuth error = %v", err)
		}
	}
	// A concurrent duplicate that authenticated before revocation is harmless.
	if _, err := store.Accept(ctx, AcceptanceRequest{JobID: jobID, Owner: owner}); err != nil {
		t.Fatalf("duplicate acceptance: %v", err)
	}
	assertCount(t, pool, `SELECT count(*) FROM craftsky_sessions WHERE account_did=$1 AND lifecycle_state<>'revoked'`, owner, 0)
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner, 2)

	claimed, found, err := store.ClaimDue(ctx, "worker", time.Minute)
	if err != nil || !found {
		t.Fatalf("claim = (%+v, %t, %v)", claimed, found, err)
	}
	if err := store.CompleteAttempt(ctx, claimed); err == nil {
		t.Fatal("legacy completion bypassed owner lifecycle coordination")
	}
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_operations WHERE owner_did=$1`, owner, 1)
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner, 2)
}

func TestAcceptanceAtomicallyAdoptsKnownUncertainPDSAttempt(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 17, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000049")
	store := NewStore(pool, func() time.Time { return now })
	seedAccountDeletionOwner(t, pool, owner, "active", 1, 1, now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateIntent(ctx, IntentRecord{
		JobID: jobID, Owner: owner,
		ConfirmationHandleHash: HashSecret("@alice.test"),
		ExpiresAt:              now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteReauthentication(
		ctx,
		jobID,
		owner,
		"deletion-oauth",
		HashSecret("proof"),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
			request_fingerprint,record_fingerprint,remote_outcome,projection_disposition,
			repeat_forbidden,remote_deadline,dispatched_at,created_at,updated_at
		) VALUES(
			'acceptance-write',$1,1,'pds_record','put_record','acceptance-write',
			'at://did:plc:alice/social.craftsky.feed.post/accepted',
			decode(repeat('04',32),'hex'),decode(repeat('04',32),'hex'),'dispatched','pending',true,$2,$3,$3,$3
		)
	`, owner, now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Accept(ctx, AcceptanceRequest{
		JobID: jobID, Owner: owner,
		ReauthProof: "proof", ConfirmationHandle: "@alice.test",
	}); err != nil {
		t.Fatal(err)
	}
	assertCount(t, pool, `
		SELECT count(*) FROM account_deletion_safety_tombstones
		WHERE operation_id=$1 AND owner_generation=1 AND kind='pds_record'
		  AND exact_key='at://did:plc:alice/social.craftsky.feed.post/accepted'
		  AND source_attempt_id='acceptance-write' AND state='pending'
	`, jobID, 1)
}

func TestPendingLoginRefreshesWorkerOAuthWithoutMintingMembership(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	store := NewStore(pool, func() time.Time { return now })
	seedAccountDeletionOwner(t, pool, owner, "deleting", 2, 2, now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'old-deletion-oauth','{}'),($1,'fresh-login-oauth','{}')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation,next_attempt_at
		) VALUES('10000000-0000-4000-8000-000000000002',$1,2,'retrying',$2,'old-deletion-oauth',1,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	refreshed, err := store.RefreshBoundOAuthFromLogin(ctx, owner, "fresh-login-oauth")
	if err != nil || !refreshed {
		t.Fatalf("refresh = (%t, %v)", refreshed, err)
	}
	if got, err := store.BoundOAuthSession(ctx, uuid.MustParse("10000000-0000-4000-8000-000000000002"), owner); err != nil || got != "fresh-login-oauth" {
		t.Fatalf("bound OAuth = (%q, %v)", got, err)
	}
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner, 2)
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1 AND lifecycle_state='revocation_pending'`, owner, 1)
	assertCount(t, pool, `SELECT count(*) FROM craftsky_sessions WHERE account_did=$1`, owner, 0)
}

func TestAdoptUncertainPDSAttemptsIsExactOwnerRegisteredCollectionAndGenerationScoped(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000025")
	store := NewStore(pool, func() time.Time { return now })

	seedDeletionSafetyOwner(t, pool, owner, 7, now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,next_attempt_at
		) VALUES($1,$2,7,'active',$3,$3)
	`, jobID, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
			request_fingerprint,record_fingerprint,remote_outcome,projection_disposition,
			repeat_forbidden,remote_deadline,dispatched_at,created_at,updated_at
		) VALUES(
			'ordinary-post',$1,7,'pds_record','put_record','ordinary-post',
			'at://did:plc:alice/social.craftsky.feed.post/post-1',
			decode(repeat('01',32),'hex'),decode(repeat('01',32),'hex'),'outcome_unknown_pre_transition',
			'hidden_non_active',true,$2,$3,$3,$3
		)
	`, owner, now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}

	operation := ClaimedOperation{JobID: jobID, Owner: owner, OwnerGeneration: 7}
	if err := store.AdoptUncertainPDSAttempts(ctx, operation); err != nil {
		t.Fatal(err)
	}
	var (
		gotOwner      syntax.DID
		gotGeneration int64
		gotKey        string
		gotSource     string
	)
	if err := pool.QueryRow(ctx, `
		SELECT owner_did,owner_generation,exact_key,source_attempt_id
		FROM account_deletion_safety_tombstones
		WHERE operation_id=$1
	`, jobID).Scan(&gotOwner, &gotGeneration, &gotKey, &gotSource); err != nil {
		t.Fatal(err)
	}
	if gotOwner != owner || gotGeneration != 7 ||
		gotKey != "at://did:plc:alice/social.craftsky.feed.post/post-1" ||
		gotSource != "ordinary-post" {
		t.Fatalf("adopted safety scope = (%s, %d, %q, %q)", gotOwner, gotGeneration, gotKey, gotSource)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM owner_effect_attempts WHERE operation_id='ordinary-post'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
			request_fingerprint,record_fingerprint,remote_outcome,projection_disposition,
			repeat_forbidden,remote_deadline,dispatched_at,created_at,updated_at
		) VALUES(
			'ordinary-follow',$1,7,'pds_record','put_record','ordinary-follow',
			'at://did:plc:alice/app.bsky.graph.follow/follow-1',
			decode(repeat('05',32),'hex'),decode(repeat('05',32),'hex'),'outcome_unknown_pre_transition',
			'hidden_non_active',true,$2,$3,$3,$3
		)
	`, owner, now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	if err := store.AdoptUncertainPDSAttempts(ctx, operation); err != nil {
		t.Fatalf("valid non-CraftSky attempt must remain tracked but not block deletion adoption: %v", err)
	}
	assertCount(t, pool, `
		SELECT count(*) FROM account_deletion_safety_tombstones
		WHERE operation_id=$1 AND source_attempt_id='ordinary-follow'
	`, jobID, 0)

	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
			request_fingerprint,record_fingerprint,remote_outcome,projection_disposition,
			repeat_forbidden,remote_deadline,dispatched_at,created_at,updated_at
		) VALUES(
			'ordinary-delete',$1,7,'pds_record','delete_record','ordinary-delete',
			'at://did:plc:alice/social.craftsky.feed.post/post-delete',
			decode(repeat('06',32),'hex'),NULL,'outcome_unknown_pre_transition',
			'hidden_non_active',true,$2,$3,$3,$3
		)
	`, owner, now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	if err := store.AdoptUncertainPDSAttempts(ctx, operation); err != nil {
		t.Fatalf("uncertain delete must remain tracked without becoming delete authority: %v", err)
	}
	assertCount(t, pool, `
		SELECT count(*) FROM account_deletion_safety_tombstones
		WHERE operation_id=$1 AND source_attempt_id='ordinary-delete'
	`, jobID, 0)

	for _, invalidKey := range []string{
		"at://did:plc:bob/social.craftsky.feed.post/post-1",
		"at://did:plc:alice/social.craftsky.feed.post",
		"scheduled-media/v2/7/not-an-at-uri",
	} {
		if _, err := pool.Exec(ctx, `DELETE FROM owner_effect_attempts WHERE operation_id='ordinary-post'`); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO owner_effect_attempts(
				operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
				request_fingerprint,record_fingerprint,remote_outcome,projection_disposition,
				repeat_forbidden,remote_deadline,dispatched_at,created_at,updated_at
			) VALUES(
				'ordinary-post',$1,7,'pds_record','put_record','ordinary-post',$2,
				decode(repeat('01',32),'hex'),decode(repeat('01',32),'hex'),'outcome_unknown_pre_transition',
				'hidden_non_active',true,$3,$4,$4,$4
			)
		`, owner, invalidKey, now.Add(time.Minute), now); err != nil {
			t.Fatal(err)
		}
		if err := store.AdoptUncertainPDSAttempts(ctx, operation); err == nil {
			t.Fatalf("out-of-scope safety key %q was accepted", invalidKey)
		}
	}
}

func TestCompleteAttemptRequiresSettledSafetyAndRemovesAllTemporaryAuthorityAtomically(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 14, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000038")
	leaseToken := uuid.MustParse("20000000-0000-4000-8000-000000000038")
	store := NewStore(pool, func() time.Time { return now })
	seedDeletionSafetyOwner(t, pool, owner, 7, now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation,next_attempt_at,lease_owner,
			lease_token,lease_expires_at
		) VALUES($1,$2,7,'active',$3,'deletion-oauth',1,$3,'worker',$4,$5)
	`, jobID, owner, now, leaseToken, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_safety_tombstones(
			id,operation_id,owner_did,owner_generation,kind,exact_key,
			source_attempt_id,state,remote_deadline,next_attempt_at,
			created_at,updated_at
		) VALUES(
			'30000000-0000-4000-8000-000000000038',$1,$2,7,'pds_record',
			'at://did:plc:alice/social.craftsky.feed.post/pending',
			'ordinary-write','pending',$3,$4,$4,$4
		)
	`, jobID, owner, now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	// These outcome-uncertain attempts are deliberately outside the narrow
	// explicit-deletion authority: an app.bsky record put must never be
	// adopted, and an ordinary delete cannot create a late record. Neither may
	// keep an otherwise converged CraftSky deletion operation alive forever.
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,
			mutation_key,deterministic_key,request_fingerprint,record_fingerprint,remote_outcome,
			projection_disposition,repeat_forbidden,remote_deadline,dispatched_at,
			created_at,updated_at
		) VALUES
			('external-put',$1,7,'pds_record','put_record','external-put',
			 'at://did:plc:alice/app.bsky.graph.follow/external',
			 decode(repeat('02',32),'hex'),decode(repeat('02',32),'hex'),'outcome_unknown_pre_transition',
			 'hidden_non_active',true,$2,$3,$3,$3),
			('craftsky-delete',$1,7,'pds_record','delete_record','craftsky-delete',
			 'at://did:plc:alice/social.craftsky.feed.post/deleted',
			 decode(repeat('03',32),'hex'),NULL,'outcome_unknown_pre_transition',
			 'hidden_non_active',true,$2,$3,$3,$3)
	`, owner, now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	operation := ClaimedOperation{
		JobID: jobID, Owner: owner, OwnerGeneration: 7,
		LeaseToken: leaseToken, LeaseExpiresAt: now.Add(time.Minute),
	}

	participant := store.CompleteParticipant(operation)
	before := ownerlifecycle.Lifecycle{Owner: owner, State: ownerlifecycle.StateDeleting, Generation: 7}
	after := ownerlifecycle.Lifecycle{Owner: owner, State: ownerlifecycle.StateDeparted, Generation: 8}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := participant(ctx, tx, before, after); !errors.Is(err, ErrSafetyPending) {
		_ = tx.Rollback(ctx)
		t.Fatalf("completion with pending safety error = %v", err)
	}
	_ = tx.Rollback(ctx)
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_operations WHERE id=$1`, jobID, 1)
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner, 1)
	if _, err := pool.Exec(ctx, `
		UPDATE account_deletion_safety_tombstones
		SET state='settled',next_attempt_at=NULL,settled_at=$2,updated_at=$2
		WHERE operation_id=$1
	`, jobID, now); err != nil {
		t.Fatal(err)
	}
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := participant(ctx, tx, before, after); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_operations WHERE id=$1`, jobID, 0)
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_safety_tombstones WHERE operation_id=$1`, jobID, 0)
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner, 1)
}

func seedDeletionSafetyOwner(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	generation int64,
	now time.Time,
) {
	seedAccountDeletionOwner(t, pool, owner, "deleting", generation, 2, now)
}

func seedAccountDeletionOwner(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	state string,
	generation int64,
	authEpoch int64,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,$2,$3,$4,'accountDeletionTest',$5,$5,$5)
	`, owner, state, generation, authEpoch, now); err != nil {
		t.Fatal(err)
	}
}

func assertCount(t *testing.T, pool *pgxpool.Pool, query string, value any, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), query, value).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count = %d, want %d for %s", got, want, query)
	}
}
