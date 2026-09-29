package business

import (
	"context"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"testing"
	"time"

	"social.craftsky/appview/internal/testdb"
)

func TestBusinessProfileServingFollowsLicenseRatherThanLegacyFlag(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	for _, statement := range []string{
		`INSERT INTO craftsky_profiles(did,record_cid) VALUES ('did:plc:licensed','cid'),('did:plc:unlicensed','cid')`,
		`INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES ('did:plc:licensed','active',1,1,'test',now(),now(),now()),('did:plc:unlicensed','active',1,1,'test',now(),now(),now())`,
		`INSERT INTO craftsky_business_profiles(owner_did,uri,cid,raw_record,source_revision) VALUES ('did:plc:licensed','at://did:plc:licensed/social.craftsky.business.profile/self','cid','{}','rev'),('did:plc:unlicensed','at://did:plc:unlicensed/social.craftsky.business.profile/self','cid','{}','rev')`,
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000091','did:plc:licensed','20000000-0000-4000-8000-000000000091')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000091','10000000-0000-4000-8000-000000000091','project','subscription','business','app','app_store','production','active',true,'business',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000091','business','did:plc:licensed',now())`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore(pool)
	for _, tc := range []struct {
		did  syntax.DID
		want AccountType
	}{{"did:plc:licensed", AccountTypeBusiness}, {"did:plc:unlicensed", AccountTypeRegular}} {
		got, err := store.ReadAccountType(ctx, tc.did)
		if err != nil || got != tc.want {
			t.Fatalf("%s account type = %q, err %v, want %q", tc.did, got, err, tc.want)
		}
	}
	batch, err := store.ReadAccountTypes(ctx, []syntax.DID{"did:plc:licensed", "did:plc:unlicensed"})
	if err != nil || batch["did:plc:licensed"] != AccountTypeBusiness || batch["did:plc:unlicensed"] == AccountTypeBusiness {
		t.Fatalf("account type batch = %+v, err %v", batch, err)
	}
	licensed, err := store.ReadEligibleProfile(ctx, "did:plc:licensed")
	if err != nil || licensed == nil {
		t.Fatalf("licensed profile = %+v, err %v", licensed, err)
	}
	unlicensed, err := store.ReadEligibleProfile(ctx, "did:plc:unlicensed")
	if err != nil || unlicensed != nil {
		t.Fatalf("unlicensed profile = %+v, err %v", unlicensed, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE provider_subscriptions SET gives_access=false WHERE id='30000000-0000-4000-8000-000000000091'`); err != nil {
		t.Fatal(err)
	}
	lapsed, err := store.ReadEligibleProfile(ctx, "did:plc:licensed")
	if err != nil || lapsed != nil {
		t.Fatalf("lapsed profile=%+v, err=%v", lapsed, err)
	}
	if got, err := store.ReadAccountType(ctx, "did:plc:licensed"); err != nil || got != AccountTypeRegular {
		t.Fatalf("lapsed type=%s, err=%v", got, err)
	}
	var retained int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_business_profiles WHERE owner_did='did:plc:licensed'`).Scan(&retained); err != nil || retained != 1 {
		t.Fatalf("retained rows=%d, err=%v", retained, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE provider_subscriptions SET gives_access=true WHERE id='30000000-0000-4000-8000-000000000091'`); err != nil {
		t.Fatal(err)
	}
	restored, err := store.ReadEligibleProfile(ctx, "did:plc:licensed")
	if err != nil || restored == nil {
		t.Fatalf("restored profile=%+v, err=%v", restored, err)
	}
}

func TestUpcomingBusinessServingFollowsEffectiveLicense(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	for _, query := range []string{
		`INSERT INTO craftsky_profiles(did,record_cid) VALUES ('did:plc:licensed','cid'),('did:plc:unlicensed','cid')`,
		`INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES ('did:plc:licensed','active',1,1,'test',now(),now(),now()),('did:plc:unlicensed','active',1,1,'test',now(),now(),now())`,
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000091','did:plc:licensed','20000000-0000-4000-8000-000000000091')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000091','10000000-0000-4000-8000-000000000091','project','subscription','business','app','app_store','production','active',true,'business',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000091','business','did:plc:licensed',now())`,
		`INSERT INTO craftsky_business_events(uri,owner_did,rkey,cid,raw_record,source_revision,starts_at,ends_at,created_at) VALUES ('at://did:plc:licensed/social.craftsky.business.event/event','did:plc:licensed','event','cid','{"$type":"social.craftsky.business.event","name":"Fair","startsAt":"2026-10-01T10:00:00Z","endsAt":"2026-10-01T12:00:00Z","roles":["vendor"],"createdAt":"2026-09-01T09:00:00Z"}','rev',now()+interval '1 day',now()+interval '2 days',now()),('at://did:plc:unlicensed/social.craftsky.business.event/event','did:plc:unlicensed','event','cid','{"$type":"social.craftsky.business.event","name":"Fair","startsAt":"2026-10-01T10:00:00Z","endsAt":"2026-10-01T12:00:00Z","roles":["vendor"],"createdAt":"2026-09-01T09:00:00Z"}','rev',now()+interval '1 day',now()+interval '2 days',now())`,
	} {
		if _, err := pool.Exec(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore(pool)
	for _, tc := range []struct {
		did  string
		want bool
	}{{"did:plc:licensed", true}, {"did:plc:unlicensed", false}} {
		got, err := store.HasUpcomingEvents(ctx, syntax.DID(tc.did), time.Now())
		if err != nil || got != tc.want {
			t.Fatalf("%s has upcoming = %v, error %v, want %v", tc.did, got, err, tc.want)
		}
		_, err = store.ReadEvent(ctx, EventReadInput{CallerDID: "did:plc:visitor", OwnerDID: syntax.DID(tc.did), Rkey: "event", AsOf: time.Now()})
		if tc.want && err != nil || !tc.want && err != ErrEventNotFound {
			t.Fatalf("%s event read error = %v, want visible %v", tc.did, err, tc.want)
		}
		upcoming, err := store.ListUpcomingEvents(ctx, UpcomingEventListInput{CallerDID: "did:plc:visitor", OwnerDID: syntax.DID(tc.did), AsOf: time.Now()})
		if err != nil || (len(upcoming) == 1) != tc.want {
			t.Fatalf("%s upcoming count = %d, err %v, want visible %v", tc.did, len(upcoming), err, tc.want)
		}
		managed, err := store.ListOwnerEvents(ctx, OwnerEventListInput{OwnerDID: syntax.DID(tc.did), AsOf: time.Now(), Filter: OwnerEventUpcoming})
		if err != nil || (len(managed) == 1) != tc.want {
			t.Fatalf("%s managed count = %d, err %v, want %v", tc.did, len(managed), err, tc.want)
		}
	}
}
