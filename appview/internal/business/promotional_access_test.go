package business

import (
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestPromotionalBusinessAccessAcrossPublicConsumers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		revoke string
	}{
		{"expired", `UPDATE billing_promotional_grants SET expires_at=now()-interval '1 minute'`},
		{"revoked", `UPDATE billing_promotional_grants SET gives_access=false`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := testdb.WithMigratedSchema(t)
			seedPromotionalBusiness(t, pool)
			store := NewStore(pool)
			var paidLicenses int
			if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM billing_licenses`).Scan(&paidLicenses); err != nil || paidLicenses != 0 {
				t.Fatalf("promo-only paid licenses=%d, err=%v", paidLicenses, err)
			}

			assertPromotionalBusinessVisibility(t, store, true)
			if _, err := pool.Exec(t.Context(), tc.revoke); err != nil {
				t.Fatal(err)
			}
			assertPromotionalBusinessVisibility(t, store, false)

			// Restoring the grant alongside a real paid plan must not duplicate
			// public rows; losing the grant must preserve the paid Business access.
			for _, query := range []string{
				`UPDATE billing_promotional_grants SET gives_access=true,expires_at=now()+interval '6 months'`,
				`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000091','10000000-0000-4000-8000-000000000091','project','paid','business','app','app_store','production','active',true,'business',1)`,
				`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000091','business','did:plc:promobusiness',now())`,
			} {
				if _, err := pool.Exec(t.Context(), query); err != nil {
					t.Fatal(err)
				}
			}
			assertPromotionalBusinessVisibility(t, store, true)
			if _, err := pool.Exec(t.Context(), tc.revoke); err != nil {
				t.Fatal(err)
			}
			assertPromotionalBusinessVisibility(t, store, true)

			var profiles, events int
			if err := pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM craftsky_business_profiles),(SELECT count(*) FROM craftsky_business_events)`).Scan(&profiles, &events); err != nil || profiles != 1 || events != 1 {
				t.Fatalf("retained profiles=%d events=%d, err=%v", profiles, events, err)
			}
		})
	}
}

func seedPromotionalBusiness(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, query := range []string{
		`INSERT INTO craftsky_profiles(did,record_cid) VALUES ('did:plc:promobusiness','cid')`,
		`INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES ('did:plc:promobusiness','active',1,1,'test',now(),now(),now())`,
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000091','did:plc:promobusiness','20000000-0000-4000-8000-000000000091')`,
		`INSERT INTO billing_promotional_grants(billing_account_id,project_id,revenuecat_subscription_id,environment,tier,gives_access,expires_at,accepted_generation) VALUES ('10000000-0000-4000-8000-000000000091','project','gift','production','business',true,now()+interval '6 months',1)`,
		`INSERT INTO craftsky_business_profiles(owner_did,uri,cid,raw_record,source_revision) VALUES ('did:plc:promobusiness','at://did:plc:promobusiness/social.craftsky.business.profile/self','cid','{}','rev')`,
		`INSERT INTO craftsky_business_events(uri,owner_did,rkey,cid,raw_record,source_revision,starts_at,ends_at,created_at) VALUES ('at://did:plc:promobusiness/social.craftsky.business.event/event','did:plc:promobusiness','event','cid','{"$type":"social.craftsky.business.event","name":"Fair","startsAt":"2026-10-01T10:00:00Z","endsAt":"2026-10-01T12:00:00Z","roles":["vendor"],"createdAt":"2026-09-01T09:00:00Z"}','rev',now()+interval '1 day',now()+interval '2 days',now())`,
		`INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'businessProfile',cid,'clear' FROM craftsky_business_profiles`,
		`INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'businessEvent',cid,'clear' FROM craftsky_business_events`,
	} {
		if _, err := pool.Exec(t.Context(), query); err != nil {
			t.Fatal(err)
		}
	}
}

func assertPromotionalBusinessVisibility(t *testing.T, store *Store, visible bool) {
	t.Helper()
	const owner syntax.DID = "did:plc:promobusiness"
	const visitor syntax.DID = "did:plc:visitor"
	wantType, wantCount := AccountTypeRegular, 0
	if visible {
		wantType, wantCount = AccountTypeBusiness, 1
	}
	if got, err := store.ReadAccountType(t.Context(), owner); err != nil || got != wantType {
		t.Fatalf("account type=%q, err=%v, want=%q", got, err, wantType)
	}
	batch, err := store.ReadAccountTypes(t.Context(), []syntax.DID{owner, visitor})
	if err != nil || (batch[owner] == AccountTypeBusiness) != visible || batch[visitor] == AccountTypeBusiness {
		t.Fatalf("batch account types=%v, err=%v, want owner visible=%v", batch, err, visible)
	}
	profile, err := store.ReadEligibleProfile(t.Context(), owner)
	if err != nil || (profile != nil) != visible {
		t.Fatalf("profile visible=%v, err=%v, want=%v", profile != nil, err, visible)
	}
	asOf := time.Now()
	if got, err := store.HasUpcomingEvents(t.Context(), owner, asOf); err != nil || got != visible {
		t.Fatalf("has upcoming=%v, err=%v, want=%v", got, err, visible)
	}
	_, err = store.ReadEvent(t.Context(), EventReadInput{CallerDID: visitor, OwnerDID: owner, Rkey: "event", AsOf: asOf})
	if visible && err != nil || !visible && !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("direct event err=%v, want visible=%v", err, visible)
	}
	upcoming, err := store.ListUpcomingEvents(t.Context(), UpcomingEventListInput{CallerDID: visitor, OwnerDID: owner, AsOf: asOf})
	if err != nil || len(upcoming) != wantCount {
		t.Fatalf("upcoming count=%d, err=%v, want=%d", len(upcoming), err, wantCount)
	}
	managed, err := store.ListOwnerEvents(t.Context(), OwnerEventListInput{OwnerDID: owner, AsOf: asOf, Filter: OwnerEventUpcoming})
	if err != nil || len(managed) != wantCount {
		t.Fatalf("managed count=%d, err=%v, want=%d", len(managed), err, wantCount)
	}
}
