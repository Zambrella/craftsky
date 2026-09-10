package subscriptions

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/testdb"
)

func TestSelfAccessReadsLatestProviderTruthLocally(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	ctx := context.Background()
	store := NewStore(pool)
	beneficiary := syntax.DID("did:plc:access-beneficiary")
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	periodEnd := now.Add(-24 * time.Hour)

	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id,requested_generation,reconciled_generation,reconciliation_requested_at,reconciled_at)
		VALUES('10000000-0000-4000-8000-000000000001','did:plc:access-owner','20000000-0000-4000-8000-000000000001',2,1,$1,$2)
	`, now.Add(-2*time.Hour), periodEnd); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,current_period_ends_at,mapped_tier,accepted_generation)
		VALUES('30000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001','project','subscription','plus','app','app_store','production','active',true,$1,'plus',1)
	`, periodEnd); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at)
		VALUES('30000000-0000-4000-8000-000000000001','plus',$1,$2)
	`, beneficiary, periodEnd); err != nil {
		t.Fatal(err)
	}

	access, err := store.SelfAccess(ctx, beneficiary, now)
	if err != nil {
		t.Fatal(err)
	}
	if access.DID != beneficiary || access.EffectiveTier != TierPlus || !access.GivesAccess {
		t.Fatalf("access = %+v, want locally accepted Plus access", access)
	}
	if access.AssignedTier == nil || *access.AssignedTier != TierPlus {
		t.Fatalf("assigned tier = %v, want Plus", access.AssignedTier)
	}
	if access.AccessEndsAt == nil || !access.AccessEndsAt.Equal(periodEnd) {
		t.Fatalf("access end = %v, want informational %v", access.AccessEndsAt, periodEnd)
	}

	freeDID := syntax.DID("did:plc:access-free")
	free, err := store.SelfAccess(ctx, freeDID, now)
	if err != nil {
		t.Fatal(err)
	}
	if free.DID != freeDID || free.EffectiveTier != TierFree || free.GivesAccess || free.AssignedTier != nil {
		t.Fatalf("free access = %+v", free)
	}
}
