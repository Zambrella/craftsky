package subscriptions

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/testdb"
)

func TestFeatureAccessSnapshotLossClearsPinsButCancellationWithAccessDoesNot(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	did := syntax.DID("did:plc:snapshot-pins")
	for _, statement := range []string{
		`INSERT INTO craftsky_profiles(did,record_cid) VALUES ('did:plc:snapshot-pins','cid')`,
		`INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES ('did:plc:snapshot-pins','active',1,1,'test',now(),now(),now())`,
		`INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES ('at://did:plc:snapshot-pins/social.craftsky.feed.post/one','did:plc:snapshot-pins','one','cid','one','{}',now())`,
		`INSERT INTO profile_pins(owner_did,slot,post_uri,state_token,created_at,updated_at) VALUES ('did:plc:snapshot-pins','standard','at://did:plc:snapshot-pins/social.craftsky.feed.post/one',gen_random_uuid(),now(),now())`,
		`INSERT INTO saved_post_folders(id,owner_did,name,created_at,updated_at) VALUES ('00000000-0000-4000-8000-000000000001','did:plc:snapshot-pins','Ideas',now(),now())`,
		`INSERT INTO saved_posts(owner_did,post_uri,folder_id,saved_at) VALUES ('did:plc:snapshot-pins','at://did:plc:snapshot-pins/social.craftsky.feed.post/one','00000000-0000-4000-8000-000000000001',now())`,
		`INSERT INTO profile_customisations(owner_did,colour,profile_background) VALUES ('did:plc:snapshot-pins','cobalt','cubedark')`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	accountID, subscriptionID := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES($1,'did:plc:snapshot-pins',$2)`, accountID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES($1,$2,'project','sub','plus','app','app_store','production','active',true,'plus',1)`, subscriptionID, accountID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES($1,'plus',$2,now())`, subscriptionID, did); err != nil {
		t.Fatal(err)
	}
	catalog, err := NewCatalog(CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]ProductMapping{"plus": {AppID: "app", Tier: TierPlus}}})
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	for generation := int64(2); generation <= 3; generation++ {
		access := generation == 2
		token := uuid.New()
		if _, err := pool.Exec(ctx, `UPDATE billing_accounts SET requested_generation=$2,claimed_generation=$2,lease_token=$3,lease_expires_at=now()+interval '1 hour' WHERE id=$1`, accountID, generation, token); err != nil {
			t.Fatal(err)
		}
		snapshot := CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{{ID: "sub", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: access, AutoRenewalStatus: "will_not_renew"}}}
		claim := SnapshotClaim{BillingAccountID: accountID, Generation: generation, LeaseToken: token}
		if !access {
			entered, release := make(chan struct{}), make(chan struct{})
			publishing := make(chan error, 1)
			go func() {
				publishing <- store.WithPlusAccess(ctx, did, func(context.Context) error {
					close(entered)
					<-release
					return nil
				})
			}()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("publisher did not reach access fence")
			}
			reconciled := make(chan error, 1)
			go func() { reconciled <- store.ApplySnapshot(ctx, claim, snapshot, catalog) }()
			select {
			case err := <-reconciled:
				t.Fatalf("loss committed before publishing effect settled: %v", err)
			case <-time.After(100 * time.Millisecond):
			}
			close(release)
			if err := <-publishing; err != nil {
				t.Fatal(err)
			}
			if err := <-reconciled; err != nil {
				t.Fatal(err)
			}
		} else if err := store.ApplySnapshot(ctx, claim, snapshot, catalog); err != nil {
			t.Fatal(err)
		}
		var pins int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM profile_pins WHERE owner_did=$1`, did).Scan(&pins); err != nil {
			t.Fatal(err)
		}
		want := 0
		if access {
			want = 1
		}
		if pins != want {
			t.Fatalf("generation %d: pins=%d, want %d", generation, pins, want)
		}
	}
	for _, next := range []struct {
		generation int64
		access     bool
	}{
		{generation: 4, access: false}, // Replay the lapsed provider state.
		{generation: 5, access: true},  // Restore access without restoring old pins.
	} {
		token := uuid.New()
		if _, err := pool.Exec(ctx, `UPDATE billing_accounts SET requested_generation=$2,claimed_generation=$2,lease_token=$3,lease_expires_at=now()+interval '1 hour' WHERE id=$1`, accountID, next.generation, token); err != nil {
			t.Fatal(err)
		}
		claim := SnapshotClaim{BillingAccountID: accountID, Generation: next.generation, LeaseToken: token}
		snapshot := CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{{ID: "sub", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: next.access, AutoRenewalStatus: "will_not_renew"}}}
		if err := store.ApplySnapshot(ctx, claim, snapshot, catalog); err != nil {
			t.Fatal(err)
		}
		current, err := store.SelfAccess(ctx, did, time.Now())
		if err != nil || current.AllowsPlus() != next.access {
			t.Fatalf("generation %d access=%+v err=%v", next.generation, current, err)
		}
		var pins, retained int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM profile_pins WHERE owner_did=$1`, did).Scan(&pins); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM saved_posts s JOIN saved_post_folders f ON f.id=s.folder_id AND f.owner_did=s.owner_did JOIN profile_customisations c ON c.owner_did=s.owner_did WHERE s.owner_did=$1 AND c.colour='cobalt' AND c.profile_background='cubedark'`, did).Scan(&retained); err != nil {
			t.Fatal(err)
		}
		if pins != 0 || retained != 1 {
			t.Fatalf("generation %d: pins=%d retained folder/customisation=%d", next.generation, pins, retained)
		}
	}
}
