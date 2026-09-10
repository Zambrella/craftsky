package subscriptions

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestAssignLicenseRevalidatesOwnerTargetDeviceAndUniquenessAtomically(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY,
			state TEXT NOT NULL,
			generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL,
			transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL,
			terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		CREATE TABLE craftsky_sessions (
			token_hash BYTEA PRIMARY KEY,
			account_did TEXT NOT NULL,
			last_device_id TEXT,
			last_seen_at TIMESTAMPTZ NOT NULL,
			idle_expires_at TIMESTAMPTZ NOT NULL,
			lifecycle_state TEXT NOT NULL,
			revoked_at TIMESTAMPTZ
		);
		CREATE FUNCTION appview_owner_is_active(candidate_did TEXT)
		RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
			SELECT COALESCE((SELECT state='active' FROM owner_lifecycles WHERE owner_did=candidate_did),false)
		$$;
	`+string(migration))
	ctx := context.Background()
	observer := &recordingBillingObserver{}
	store := NewStore(pool, observer)
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:assignment-owner")
	targetA := syntax.DID("did:plc:assignment-a")
	targetB := syntax.DID("did:plc:assignment-b")
	licenseA := uuid.MustParse("30000000-0000-4000-8000-000000000021")
	licenseB := uuid.MustParse("30000000-0000-4000-8000-000000000022")

	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did) VALUES($1),($2),($3)`, owner, targetA, targetB); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
		VALUES($1,'active',1,1,'test',$4,$4,$4),($2,'active',1,1,'test',$4,$4,$4),($3,'active',1,1,'test',$4,$4,$4)
	`, owner, targetA, targetB, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_sessions(token_hash,account_did,last_device_id,last_seen_at,idle_expires_at,lifecycle_state)
		VALUES(decode('01','hex'),$1,'device-a',$3::timestamptz,$3::timestamptz+interval '30 days','active'),
		      (decode('02','hex'),$2,'wrong-device',$3::timestamptz,$3::timestamptz+interval '30 days','active'),
		      (decode('03','hex'),$4,'device-a',$3::timestamptz,$3::timestamptz+interval '30 days','active')
	`, targetA, targetB, now, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id)
		VALUES('10000000-0000-4000-8000-000000000021',$1,'20000000-0000-4000-8000-000000000021'),
		      ('10000000-0000-4000-8000-000000000022','did:plc:other-owner','20000000-0000-4000-8000-000000000022')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation)
		VALUES('40000000-0000-4000-8000-000000000021','10000000-0000-4000-8000-000000000021','project','sub-a','plus','app','app_store','production','active',true,'plus',1),
		      ('40000000-0000-4000-8000-000000000022','10000000-0000-4000-8000-000000000021','project','sub-b','business','app','app_store','production','expired',false,'business',1)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_licenses(id,provider_subscription_id,tier,assignable)
		VALUES($1,'40000000-0000-4000-8000-000000000021','plus',true),
		      ($2,'40000000-0000-4000-8000-000000000022','business',true)
	`, licenseA, licenseB); err != nil {
		t.Fatal(err)
	}

	assigned, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: licenseA, TargetDID: targetA, DeviceID: "device-a", Now: now})
	if err != nil || assigned.TargetDID != targetA {
		t.Fatalf("initial assignment = %+v, error %v", assigned, err)
	}
	if _, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: licenseB, TargetDID: targetA, DeviceID: "device-a", Now: now}); !errors.Is(err, ErrDIDAlreadyAssigned) {
		t.Fatalf("dormant uniqueness error = %v, want ErrDIDAlreadyAssigned", err)
	}
	if _, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: licenseA, TargetDID: targetB, DeviceID: "device-a", Now: now}); !errors.Is(err, ErrAssignmentTargetIneligible) {
		t.Fatalf("wrong-device error = %v, want ErrAssignmentTargetIneligible", err)
	}
	if _, err := store.Assign(ctx, AssignParams{OwnerDID: syntax.DID("did:plc:other-owner"), LicenseID: licenseA, TargetDID: targetA, DeviceID: "device-a", Now: now}); !errors.Is(err, ErrLicenseNotFound) {
		t.Fatalf("wrong-owner error = %v, want ErrLicenseNotFound", err)
	}

	var preserved syntax.DID
	if err := pool.QueryRow(ctx, `SELECT assigned_did FROM billing_licenses WHERE id=$1`, licenseA).Scan(&preserved); err != nil {
		t.Fatal(err)
	}
	if preserved != targetA {
		t.Fatalf("failed changes replaced prior assignment with %q", preserved)
	}

	if _, err := pool.Exec(ctx, `UPDATE craftsky_sessions SET last_device_id='device-a' WHERE account_did=$1`, targetB); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"deletion_pending", "departed"} {
		if _, err := pool.Exec(ctx, `UPDATE owner_lifecycles SET state=$2 WHERE owner_did=$1`, targetB, state); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: licenseA, TargetDID: targetB, DeviceID: "device-a", Now: now.Add(8 * 24 * time.Hour)}); !errors.Is(err, ErrAssignmentTargetIneligible) {
			t.Fatalf("%s target error = %v, want ErrAssignmentTargetIneligible", state, err)
		}
	}

	if _, err := pool.Exec(ctx, `UPDATE owner_lifecycles SET state='active' WHERE owner_did=$1`, targetB); err != nil {
		t.Fatal(err)
	}
	transition, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fenceKey, err := ownerlifecycle.FenceKey(targetB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transition.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, fenceKey); err != nil {
		t.Fatal(err)
	}
	if _, err := transition.Exec(ctx, `UPDATE owner_lifecycles SET state='deletion_pending' WHERE owner_did=$1`, targetB); err != nil {
		t.Fatal(err)
	}
	assignmentDone := make(chan error, 1)
	go func() {
		_, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: licenseA, TargetDID: targetB, DeviceID: "device-a", Now: now.Add(8 * 24 * time.Hour)})
		assignmentDone <- err
	}()
	select {
	case err := <-assignmentDone:
		_ = transition.Rollback(ctx)
		t.Fatalf("assignment completed before lifecycle transition committed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := transition.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-assignmentDone; !errors.Is(err, ErrAssignmentTargetIneligible) {
		t.Fatalf("concurrent lifecycle transition assignment error = %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT assigned_did FROM billing_licenses WHERE id=$1`, licenseA).Scan(&preserved); err != nil || preserved != targetA {
		t.Fatalf("concurrent lifecycle failure replaced prior assignment with %q: %v", preserved, err)
	}

	ownerTransition, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ownerFenceKey, err := ownerlifecycle.FenceKey(owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ownerTransition.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, ownerFenceKey); err != nil {
		t.Fatal(err)
	}
	if _, err := ownerTransition.Exec(ctx, `UPDATE owner_lifecycles SET state='deletion_pending' WHERE owner_did=$1`, owner); err != nil {
		t.Fatal(err)
	}
	assignmentDone = make(chan error, 1)
	go func() {
		_, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: licenseA, TargetDID: owner, DeviceID: "device-a", Now: now.Add(8 * 24 * time.Hour)})
		assignmentDone <- err
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM pg_stat_activity
				WHERE pid <> pg_backend_pid()
				  AND datname=current_database()
				  AND wait_event_type='Lock'
				  AND wait_event='advisory'
				  AND query LIKE '%pg_advisory_xact_lock_shared%'
			)
		`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("assignment did not wait on the owner lifecycle fence")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := ownerTransition.Exec(ctx, `SELECT id FROM billing_accounts WHERE owner_did=$1 FOR UPDATE`, owner); err != nil {
		_ = ownerTransition.Rollback(ctx)
		t.Fatalf("lifecycle transition could not preserve fence-before-billing lock order: %v", err)
	}
	if err := ownerTransition.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-assignmentDone; !errors.Is(err, ErrAssignmentTargetIneligible) {
		t.Fatalf("concurrent owner transition assignment error = %v", err)
	}

	if len(observer.assignments) != 8 {
		t.Fatalf("assignment observations = %v", observer.assignments)
	}
	for _, observation := range observer.assignments {
		if observation[0] != "assign" {
			t.Fatalf("unexpected assignment operation %q", observation[0])
		}
		switch observation[1] {
		case "success", "already_assigned", "target_ineligible", "not_found":
		default:
			t.Fatalf("unbounded assignment outcome %q", observation[1])
		}
	}
}
