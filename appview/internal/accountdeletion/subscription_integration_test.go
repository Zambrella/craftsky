package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestSubscriptionDeletionParticipantBlocksClosesAndUnassigns(t *testing.T) {
	accountsMigration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	eventsMigration, err := os.ReadFile("../../migrations/000070_revenuecat_events.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(accountsMigration)+string(eventsMigration))
	ctx := context.Background()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:deleting-billing-owner")
	target := syntax.DID("did:plc:deleting-target")
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id)
		VALUES('10000000-0000-4000-8000-000000000041',$1,'20000000-0000-4000-8000-000000000041'),
		      ('10000000-0000-4000-8000-000000000042','did:plc:other-billing-owner','20000000-0000-4000-8000-000000000042')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,pending_payment,auto_renewal_status,mapped_tier,accepted_generation)
		VALUES('30000000-0000-4000-8000-000000000041','10000000-0000-4000-8000-000000000041','project','owner-sub','plus','app','app_store','production','active',true,false,'will_renew','plus',0),
		      ('30000000-0000-4000-8000-000000000042','10000000-0000-4000-8000-000000000042','project','target-sub','business','app','app_store','production','active',true,false,'will_renew','business',0)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_licenses(id,provider_subscription_id,tier,assigned_did,assigned_at)
		VALUES('40000000-0000-4000-8000-000000000041','30000000-0000-4000-8000-000000000041','plus',$1,now()),
		      ('40000000-0000-4000-8000-000000000042','30000000-0000-4000-8000-000000000042','business',$2,now())
	`, owner, target); err != nil {
		t.Fatal(err)
	}
	participant := subscriptions.NewDeletionParticipant()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if billingOwner, err := participant.BeginDeletion(ctx, tx, owner, now); err != nil || !billingOwner {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	for _, did := range []syntax.DID{owner, target} {
		var assignments int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM billing_licenses WHERE assigned_did=$1`, did).Scan(&assignments); err != nil || assignments != 1 {
			t.Fatalf("reversible intent assignment for %s = %d, error %v", did, assignments, err)
		}
	}
	var requested, deletionRequested int64
	if err := pool.QueryRow(ctx, `SELECT requested_generation,deletion_requested_generation FROM billing_accounts WHERE owner_did=$1`, owner).Scan(&requested, &deletionRequested); err != nil || requested != 1 || deletionRequested != 1 {
		t.Fatalf("deletion generation = %d/%d, error %v", requested, deletionRequested, err)
	}

	tx, _ = pool.Begin(ctx)
	if _, err := participant.ConfirmDeletion(ctx, tx, owner, now); !errors.Is(err, subscriptions.ErrProviderBillingMustBeResolved) {
		_ = tx.Rollback(ctx)
		t.Fatalf("active billing confirmation error = %v", err)
	}
	_ = tx.Rollback(ctx)
	if _, err := pool.Exec(ctx, `
		UPDATE provider_subscriptions SET status='active',gives_access=true,pending_payment=false,auto_renewal_status='will_not_renew',accepted_generation=1 WHERE billing_account_id='10000000-0000-4000-8000-000000000041'
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE billing_accounts SET reconciled_generation=1,reconciled_at=$1 WHERE id='10000000-0000-4000-8000-000000000041'`, now); err != nil {
		t.Fatal(err)
	}
	tx, _ = pool.Begin(ctx)
	if closed, err := participant.ConfirmDeletion(ctx, tx, owner, now); err != nil || !closed {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	var closedOwner *string
	var closedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT state,owner_did,closed_at FROM billing_accounts WHERE id='10000000-0000-4000-8000-000000000041'`).Scan(&state, &closedOwner, &closedAt); err != nil || state != "closed" || closedOwner != nil || closedAt == nil {
		t.Fatalf("closed marker = %q/%v/%v, error %v", state, closedOwner, closedAt, err)
	}
	var liveRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM provider_subscriptions WHERE billing_account_id='10000000-0000-4000-8000-000000000041'`).Scan(&liveRows); err != nil || liveRows != 0 {
		t.Fatalf("closed live subscriptions = %d, error %v", liveRows, err)
	}

	tx, _ = pool.Begin(ctx)
	if billingOwner, err := participant.BeginDeletion(ctx, tx, target, now); err != nil || billingOwner {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var assigned *string
	if err := pool.QueryRow(ctx, `SELECT assigned_did FROM billing_licenses WHERE id='40000000-0000-4000-8000-000000000042'`).Scan(&assigned); err != nil || assigned == nil || *assigned != target.String() {
		t.Fatalf("pending target assignment = %v, error %v", assigned, err)
	}
	tx, _ = pool.Begin(ctx)
	if closed, err := participant.ConfirmDeletion(ctx, tx, target, now); err != nil || closed {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT assigned_did FROM billing_licenses WHERE id='40000000-0000-4000-8000-000000000042'`).Scan(&assigned); err != nil || assigned != nil {
		t.Fatalf("accepted target assignment = %v, error %v", assigned, err)
	}

	store := subscriptions.NewStore(pool)
	accepted, err := store.AcceptRevenueCatEvent(ctx, subscriptions.RevenueCatEvent{
		ID: "delayed-event", Type: "RENEWAL", RevenueCatAppUserID: uuid.MustParse("20000000-0000-4000-8000-000000000041"),
	}, now.Add(time.Minute))
	if err != nil || !accepted {
		t.Fatalf("delayed event = accepted %t, error %v", accepted, err)
	}
	if err := pool.QueryRow(ctx, `SELECT outcome FROM revenuecat_events WHERE event_id='delayed-event'`).Scan(&state); err != nil || state != "ignored_closed" {
		t.Fatalf("delayed event outcome = %q, error %v", state, err)
	}
}

func TestSubscriptionDeletionParticipantBlocksMixedProviderBilling(t *testing.T) {
	accountsMigration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(accountsMigration))
	ctx := context.Background()
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	participant := subscriptions.NewDeletionParticipant()

	tests := []struct {
		name              string
		autoRenewalStatus any
		pendingPayment    bool
	}{
		{name: "will renew", autoRenewalStatus: "will_renew"},
		{name: "will change product", autoRenewalStatus: "will_change_product"},
		{name: "will pause", autoRenewalStatus: "will_pause"},
		{name: "requires price increase consent", autoRenewalStatus: "requires_price_increase_consent"},
		{name: "has already renewed", autoRenewalStatus: "has_already_renewed"},
		{name: "missing", autoRenewalStatus: nil},
		{name: "empty", autoRenewalStatus: ""},
		{name: "unknown", autoRenewalStatus: "future_provider_state"},
		{name: "pending payment", autoRenewalStatus: "will_not_renew", pendingPayment: true},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			accountID := uuid.New()
			owner := syntax.DID(fmt.Sprintf("did:plc:mixed-deletion-%d", i))
			safeSubscriptionID := uuid.New()
			blockingSubscriptionID := uuid.New()
			licenseID := uuid.New()
			if _, err := pool.Exec(ctx, `
				INSERT INTO billing_accounts(
					id,owner_did,revenuecat_app_user_id,requested_generation,
					reconciled_generation,deletion_requested_generation,reconciled_at
				) VALUES($1,$2,$3,1,1,1,$4)
			`, accountID, owner, uuid.New(), now); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO provider_subscriptions(
					id,billing_account_id,project_id,revenuecat_subscription_id,product_id,
					app_id,store,environment,status,gives_access,pending_payment,
					auto_renewal_status,mapped_tier,accepted_generation
				) VALUES
					($1,$2,'project',$3,'plus','app','app_store','production','active',true,false,'will_not_renew','plus',1),
					($4,$2,'project',$5,'business','app','app_store','production','expired',false,$6,$7,'business',1)
			`, safeSubscriptionID, accountID, "safe-"+test.name, blockingSubscriptionID, "blocking-"+test.name, test.pendingPayment, test.autoRenewalStatus); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO billing_licenses(id,provider_subscription_id,tier,assigned_did,assigned_at)
				VALUES($1,$2,'plus',$3,$4)
			`, licenseID, safeSubscriptionID, owner, now); err != nil {
				t.Fatal(err)
			}

			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := participant.ConfirmDeletion(ctx, tx, owner, now); !errors.Is(err, subscriptions.ErrProviderBillingMustBeResolved) {
				_ = tx.Rollback(ctx)
				t.Fatalf("confirmation error = %v, want provider billing blocker", err)
			}
			if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				t.Fatal(err)
			}

			var state string
			var ownerDID *string
			if err := pool.QueryRow(ctx, `SELECT state,owner_did FROM billing_accounts WHERE id=$1`, accountID).Scan(&state, &ownerDID); err != nil || state != "active" || ownerDID == nil || *ownerDID != owner.String() {
				t.Fatalf("billing account after blocked deletion = %q/%v, error %v", state, ownerDID, err)
			}
			var subscriptionsCount, assignmentsCount int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM provider_subscriptions WHERE billing_account_id=$1`, accountID).Scan(&subscriptionsCount); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM billing_licenses WHERE assigned_did=$1`, owner).Scan(&assignmentsCount); err != nil {
				t.Fatal(err)
			}
			if subscriptionsCount != 2 || assignmentsCount != 1 {
				t.Fatalf("state mutated after blocked deletion: subscriptions=%d assignments=%d", subscriptionsCount, assignmentsCount)
			}
		})
	}
}

func TestAcceptedSelfAssignedDeletionSerializesWithSnapshotApply(t *testing.T) {
	accountsMigration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(accountsMigration))
	ctx := context.Background()
	now := time.Date(2026, 9, 9, 13, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:deletion-snapshot-race")
	accountID := uuid.MustParse("10000000-0000-4000-8000-000000000051")
	leaseToken := uuid.MustParse("50000000-0000-4000-8000-000000000051")
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts(
			id,owner_did,revenuecat_app_user_id,requested_generation,reconciled_generation,
			claimed_generation,lease_token,lease_expires_at,deletion_requested_generation
		) VALUES($1,$2,'20000000-0000-4000-8000-000000000051',2,1,2,$3,$4,1)
	`, accountID, owner, leaseToken, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_subscriptions(
			id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,
			store,environment,status,gives_access,pending_payment,auto_renewal_status,mapped_tier,accepted_generation
		) VALUES(
			'30000000-0000-4000-8000-000000000051',$1,'project','race-sub','prod-plus','app',
			'app_store','production','expired',false,false,'will_not_renew','plus',1
		)
	`, accountID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_licenses(id,provider_subscription_id,tier,assigned_did,assigned_at)
		VALUES('40000000-0000-4000-8000-000000000051','30000000-0000-4000-8000-000000000051','plus',$1,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}

	const pauseLockID int64 = 4200910
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION pause_accepted_billing_cleanup()
		RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(4200910);
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER pause_accepted_billing_cleanup
		BEFORE UPDATE ON billing_licenses
		FOR EACH ROW EXECUTE FUNCTION pause_accepted_billing_cleanup();
	`); err != nil {
		t.Fatal(err)
	}
	blocker, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Release()
	if _, err := blocker.Exec(ctx, `SELECT pg_advisory_lock($1)`, pauseLockID); err != nil {
		t.Fatal(err)
	}
	locked := true
	defer func() {
		if !locked {
			return
		}
		var unlocked bool
		if err := blocker.QueryRow(context.Background(), `SELECT pg_advisory_unlock($1)`, pauseLockID).Scan(&unlocked); err != nil || !unlocked {
			t.Errorf("release deletion pause lock: unlocked=%t err=%v", unlocked, err)
		}
	}()

	participant := subscriptions.NewDeletionParticipant()
	deletionDone := make(chan error, 1)
	go func() {
		tx, err := pool.Begin(ctx)
		if err == nil {
			var closed bool
			closed, err = participant.ConfirmDeletion(ctx, tx, owner, now)
			if err == nil && !closed {
				err = errors.New("accepted deletion did not close billing account")
			}
			if err == nil {
				err = tx.Commit(ctx)
			} else {
				_ = tx.Rollback(ctx)
			}
		}
		deletionDone <- err
	}()
	waitForAdvisoryWaiter(t, pool, pauseLockID)

	catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{
		ProjectID: "project", AppIDs: []string{"app"},
		Products: map[string]subscriptions.ProductMapping{
			"prod-plus": {AppID: "app", Tier: subscriptions.TierPlus},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshotDone := make(chan error, 1)
	go func() {
		snapshotDone <- subscriptions.NewStore(pool).ApplySnapshot(ctx, subscriptions.SnapshotClaim{
			BillingAccountID: accountID, Generation: 2, LeaseToken: leaseToken,
		}, subscriptions.CompleteSnapshot{Complete: true, Subscriptions: []subscriptions.ProviderSubscriptionSnapshot{
			{ID: "race-sub", ProductID: "prod-plus", Store: "app_store", Environment: "production", Status: "expired", AutoRenewalStatus: "will_not_renew"},
		}}, catalog)
	}()
	waitForSnapshotLockWaiter(t, pool)

	var unlocked bool
	if err := blocker.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, pauseLockID).Scan(&unlocked); err != nil || !unlocked {
		t.Fatalf("release deletion pause lock: unlocked=%t err=%v", unlocked, err)
	}
	locked = false
	if err := <-deletionDone; err != nil {
		t.Fatalf("accepted deletion race: %v", err)
	}
	if err := <-snapshotDone; !errors.Is(err, subscriptions.ErrStaleSnapshot) {
		t.Fatalf("snapshot after accepted deletion = %v, want ErrStaleSnapshot", err)
	}

	var state string
	var ownerDID *string
	if err := pool.QueryRow(ctx, `SELECT state,owner_did FROM billing_accounts WHERE id=$1`, accountID).Scan(&state, &ownerDID); err != nil || state != "closed" || ownerDID != nil {
		t.Fatalf("billing marker after race = %q/%v, error %v", state, ownerDID, err)
	}
	var liveRows, assignedRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM provider_subscriptions WHERE billing_account_id=$1`, accountID).Scan(&liveRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM billing_licenses WHERE assigned_did=$1`, owner).Scan(&assignedRows); err != nil {
		t.Fatal(err)
	}
	if liveRows != 0 || assignedRows != 0 {
		t.Fatalf("live billing state after race = subscriptions %d assignments %d", liveRows, assignedRows)
	}
}

func waitForAdvisoryWaiter(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, key int64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS(
				SELECT 1 FROM pg_locks
				WHERE locktype='advisory'
				  AND classid=(($1::bigint >> 32) & 4294967295)::oid
				  AND objid=($1::bigint & 4294967295)::oid
				  AND NOT granted
			)
		`, key).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for accepted deletion assignment cleanup")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func waitForSnapshotLockWaiter(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS(
				SELECT 1 FROM pg_stat_activity
				WHERE pid <> pg_backend_pid()
				  AND datname=current_database()
				  AND wait_event_type='Lock'
				  AND (
					query LIKE '%SELECT reconciled_generation, state%'
					OR query LIKE '%INSERT INTO billing_licenses(provider_subscription_id,tier,assignable,anomaly)%'
				  )
			)
		`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for snapshot row lock")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
