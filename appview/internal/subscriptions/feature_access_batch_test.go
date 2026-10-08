package subscriptions

import (
	"context"
	"os"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/testdb"
)

func TestFeatureAccessBatchUsesSameEnvironmentAndProviderCriteriaAsSelfAccess(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	ctx := context.Background()
	for _, statement := range []string{
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000091','did:plc:payer','20000000-0000-4000-8000-000000000091')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000091','10000000-0000-4000-8000-000000000091','project','business-1','business','app','app_store','sandbox','active',true,'business',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000091','business','did:plc:beneficiary',now())`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	dids := []syntax.DID{"did:plc:payer", "did:plc:beneficiary"}
	for _, tc := range []struct {
		name  string
		store *Store
		want  Tier
	}{
		{"production", NewStore(pool), TierFree},
		{"sandbox", NewStoreForEnvironment(pool, "sandbox"), TierBusiness},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.store.EffectiveTiers(ctx, dids)
			if err != nil || got[dids[0]] != TierFree || got[dids[1]] != tc.want {
				t.Fatalf("batch = %v, err %v, want %s beneficiary", got, err, tc.want)
			}
		})
	}
}
