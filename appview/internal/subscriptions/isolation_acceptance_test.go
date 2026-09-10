package subscriptions

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/testdb"
)

const isolationSocialDDL = `
	CREATE TABLE owner_lifecycles (
		owner_did TEXT PRIMARY KEY, state TEXT NOT NULL, generation BIGINT NOT NULL,
		auth_epoch BIGINT NOT NULL, transition_reason TEXT NOT NULL,
		transitioned_at TIMESTAMPTZ NOT NULL, terminal_at TIMESTAMPTZ,
		purge_completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	);
	CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY, record_cid TEXT NOT NULL);
	CREATE TABLE craftsky_posts (uri TEXT PRIMARY KEY, did TEXT NOT NULL, cid TEXT NOT NULL, record JSONB NOT NULL);
	CREATE TABLE atproto_follows (uri TEXT PRIMARY KEY, follower_did TEXT NOT NULL, subject_did TEXT NOT NULL);
	CREATE TABLE actor_mutes (owner_did TEXT NOT NULL, subject_did TEXT NOT NULL, PRIMARY KEY(owner_did,subject_did));
	CREATE TABLE saved_posts (owner_did TEXT NOT NULL, post_uri TEXT NOT NULL, PRIMARY KEY(owner_did,post_uri));
	CREATE TABLE moderation_reports (id TEXT PRIMARY KEY, reporter_did TEXT NOT NULL, details TEXT);
	CREATE TABLE scheduled_posts (id UUID PRIMARY KEY, owner_did TEXT NOT NULL, payload JSONB NOT NULL);
	INSERT INTO craftsky_profiles VALUES
		('did:plc:isolation-owner','profile-owner'),
		('did:plc:isolation-beneficiary','profile-beneficiary'),
		('did:plc:isolation-target','profile-target'),
		('did:plc:isolation-other-owner','profile-other-owner');
	INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
	SELECT did,'active',1,1,'test',now(),now(),now() FROM craftsky_profiles;
	CREATE FUNCTION appview_owner_is_active(candidate_did TEXT)
	RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
		SELECT COALESCE((SELECT state='active' FROM owner_lifecycles WHERE owner_did=candidate_did),false)
	$$;
	INSERT INTO craftsky_posts VALUES('at://did:plc:isolation-owner/social.craftsky.feed.post/post','did:plc:isolation-owner','post-cid','{"text":"unchanged"}');
	INSERT INTO atproto_follows VALUES('at://did:plc:isolation-owner/app.bsky.graph.follow/follow','did:plc:isolation-owner','did:plc:isolation-target');
	INSERT INTO actor_mutes VALUES('did:plc:isolation-owner','did:plc:muted');
	INSERT INTO saved_posts VALUES('did:plc:isolation-owner','at://did:plc:isolation-owner/social.craftsky.feed.post/post');
	INSERT INTO moderation_reports VALUES('report','did:plc:isolation-owner','retained-private-report');
	INSERT INTO scheduled_posts VALUES('90000000-0000-4000-8000-000000000001','did:plc:isolation-owner','{"text":"retained-private-draft"}');
	CREATE FUNCTION reject_billing_social_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
	BEGIN RAISE EXCEPTION 'billing touched social/private sentinel table %', TG_TABLE_NAME; END $$;
	CREATE TRIGGER isolate_profiles BEFORE UPDATE OR DELETE ON craftsky_profiles FOR EACH ROW EXECUTE FUNCTION reject_billing_social_mutation();
	CREATE TRIGGER isolate_posts BEFORE UPDATE OR DELETE ON craftsky_posts FOR EACH ROW EXECUTE FUNCTION reject_billing_social_mutation();
	CREATE TRIGGER isolate_follows BEFORE UPDATE OR DELETE ON atproto_follows FOR EACH ROW EXECUTE FUNCTION reject_billing_social_mutation();
	CREATE TRIGGER isolate_mutes BEFORE UPDATE OR DELETE ON actor_mutes FOR EACH ROW EXECUTE FUNCTION reject_billing_social_mutation();
	CREATE TRIGGER isolate_saved BEFORE UPDATE OR DELETE ON saved_posts FOR EACH ROW EXECUTE FUNCTION reject_billing_social_mutation();
	CREATE TRIGGER isolate_reports BEFORE UPDATE OR DELETE ON moderation_reports FOR EACH ROW EXECUTE FUNCTION reject_billing_social_mutation();
	CREATE TRIGGER isolate_scheduled BEFORE UPDATE OR DELETE ON scheduled_posts FOR EACH ROW EXECUTE FUNCTION reject_billing_social_mutation();
	CREATE TABLE craftsky_sessions (
		token_hash BYTEA PRIMARY KEY, account_did TEXT NOT NULL, last_device_id TEXT,
		last_seen_at TIMESTAMPTZ NOT NULL, idle_expires_at TIMESTAMPTZ NOT NULL,
		lifecycle_state TEXT NOT NULL, revoked_at TIMESTAMPTZ
	);
`

type isolationSnapshotProvider struct {
	t             *testing.T
	wantCustomer  uuid.UUID
	snapshots     []CompleteSnapshot
	expectedCalls int
}

func (provider *isolationSnapshotProvider) ListCustomerSubscriptions(_ context.Context, customer uuid.UUID) (CompleteSnapshot, error) {
	provider.t.Helper()
	if customer != provider.wantCustomer || len(provider.snapshots) == 0 {
		provider.t.Fatalf("unexpected external provider call for %s", customer)
	}
	provider.expectedCalls++
	snapshot := provider.snapshots[0]
	provider.snapshots = provider.snapshots[1:]
	return snapshot, nil
}

func snapshotIsolationTables(t *testing.T, ctx context.Context, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) map[string]string {
	t.Helper()
	snapshots := make(map[string]string)
	for _, table := range []string{"craftsky_profiles", "craftsky_posts", "atproto_follows", "actor_mutes", "saved_posts", "moderation_reports", "scheduled_posts"} {
		query := "SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY to_jsonb(row_data)::text), '[]'::jsonb)::text FROM " + pgx.Identifier{table}.Sanitize() + " row_data"
		var snapshot string
		if err := pool.QueryRow(ctx, query).Scan(&snapshot); err != nil {
			t.Fatal(err)
		}
		snapshots[table] = snapshot
	}
	return snapshots
}

func TestBillingLifecycleIsIsolatedFromSocialAndRetainedPrivateState(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, isolationSocialDDL+string(migration))
	ctx := context.Background()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:isolation-owner")
	beneficiary := syntax.DID("did:plc:isolation-beneficiary")
	target := syntax.DID("did:plc:isolation-target")
	store := NewStore(pool)
	socialBefore := snapshotIsolationTables(t, ctx, pool)
	account, _, err := store.EnsureAccount(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_sessions(token_hash,account_did,last_device_id,last_seen_at,idle_expires_at,lifecycle_state)
		VALUES(decode('01','hex'),$1,'isolation-device',$2,$2::timestamptz+interval '1 day','active')
	`, beneficiary, now); err != nil {
		t.Fatal(err)
	}
	catalog, err := NewCatalog(CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]ProductMapping{
		"plus": {AppID: "app", Tier: TierPlus},
	}})
	if err != nil {
		t.Fatal(err)
	}
	provider := &isolationSnapshotProvider{t: t, wantCustomer: account.RevenueCatAppUserID, snapshots: []CompleteSnapshot{
		{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{{ID: "plus-primary", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true}}},
		{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
			{ID: "plus-primary", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
			{ID: "plus-duplicate", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
		}},
		{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{{ID: "plus-primary", ProductID: "plus", Store: "app_store", Environment: "production", Status: "expired", GivesAccess: false}}},
		{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{{ID: "plus-primary", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true}}},
		{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
			{ID: "plus-primary", ProductID: "plus", Store: "app_store", Environment: "production", Status: "expired", GivesAccess: false, AutoRenewalStatus: "will_not_renew"},
			{ID: "plus-duplicate", ProductID: "plus", Store: "app_store", Environment: "production", Status: "expired", GivesAccess: false, AutoRenewalStatus: "will_not_renew"},
		}},
	}}
	reconciler := NewReconciler(store, provider, catalog, time.Minute, func() time.Time { return now })
	reconcile := func(trigger ReconciliationTrigger) {
		t.Helper()
		if err := reconciler.Trigger(ctx, account.ID, trigger); err != nil {
			t.Fatal(err)
		}
		if processed, err := reconciler.ProcessOne(ctx); err != nil || !processed {
			t.Fatalf("reconcile %s = processed %t, error %v", trigger, processed, err)
		}
	}

	reconcile(TriggerRestore)
	var primaryLicense uuid.UUID
	if err := pool.QueryRow(ctx, `
		SELECT license.id FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE subscription.revenuecat_subscription_id='plus-primary'
	`).Scan(&primaryLicense); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: primaryLicense, TargetDID: beneficiary, DeviceID: "isolation-device", Now: now}); err != nil {
		t.Fatal(err)
	}
	reconcile(TriggerWebhook)
	var duplicateAnomaly string
	if err := pool.QueryRow(ctx, `
		SELECT license.anomaly FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE subscription.revenuecat_subscription_id='plus-duplicate'
	`).Scan(&duplicateAnomaly); err != nil || duplicateAnomaly != "same_tier_duplicate" {
		t.Fatalf("duplicate anomaly = %q, error %v", duplicateAnomaly, err)
	}
	reconcile(TriggerScheduled)
	access, err := store.SelfAccess(ctx, beneficiary, now)
	if err != nil || access.EffectiveTier != TierFree || access.AssignedTier == nil {
		t.Fatalf("access loss = %#v, error %v", access, err)
	}
	reconcile(TriggerOwnerRefresh)
	access, err = store.SelfAccess(ctx, beneficiary, now)
	if err != nil || access.EffectiveTier != TierPlus || !access.GivesAccess {
		t.Fatalf("access recovery = %#v, error %v", access, err)
	}

	otherAccount, _, err := store.EnsureAccount(ctx, syntax.DID("did:plc:isolation-other-owner"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation)
		VALUES('30000000-0000-4000-8000-000000000099',$1,'project','other-sub','plus','app','app_store','production','active',true,'plus',1)
	`, otherAccount.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at)
		VALUES('30000000-0000-4000-8000-000000000099','plus',$1,$2)
	`, target, now); err != nil {
		t.Fatal(err)
	}
	participant := NewDeletionParticipant()
	withTransaction(t, pool, func(tx pgx.Tx) error {
		billingOwner, err := participant.BeginDeletion(ctx, tx, target, now)
		if billingOwner {
			t.Fatal("assigned non-owner was treated as billing owner")
		}
		return err
	})
	var assigned *syntax.DID
	if err := pool.QueryRow(ctx, `SELECT assigned_did FROM billing_licenses WHERE provider_subscription_id='30000000-0000-4000-8000-000000000099'`).Scan(&assigned); err != nil || assigned == nil || *assigned != target {
		t.Fatalf("pending non-owner deletion assignment = %v, error %v", assigned, err)
	}
	withTransaction(t, pool, func(tx pgx.Tx) error {
		_, err := participant.ConfirmDeletion(ctx, tx, target, now)
		return err
	})

	withTransaction(t, pool, func(tx pgx.Tx) error {
		billingOwner, err := participant.BeginDeletion(ctx, tx, owner, now)
		if !billingOwner {
			t.Fatal("billing owner deletion was not recognized")
		}
		return err
	})
	if processed, err := reconciler.ProcessOne(ctx); err != nil || !processed {
		t.Fatalf("post-deletion reconciliation = processed %t, error %v", processed, err)
	}
	withTransaction(t, pool, func(tx pgx.Tx) error {
		_, err := participant.ConfirmDeletion(ctx, tx, owner, now)
		return err
	})

	if provider.expectedCalls != 5 || len(provider.snapshots) != 0 {
		t.Fatalf("provider calls = %d, remaining snapshots %d", provider.expectedCalls, len(provider.snapshots))
	}
	socialAfter := snapshotIsolationTables(t, ctx, pool)
	for table, before := range socialBefore {
		if after := socialAfter[table]; after != before {
			t.Fatalf("%s changed:\nbefore: %s\nafter:  %s", table, before, after)
		}
	}
}

func withTransaction(t *testing.T, pool interface {
	Begin(context.Context) (pgx.Tx, error)
}, run func(pgx.Tx) error) {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := run(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
}
