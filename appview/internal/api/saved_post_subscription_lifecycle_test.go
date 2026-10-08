package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestSavedFolderAssociationsSurviveEffectivePlusLossAndReturnOnRestoration(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	owner, author := syntax.DID("did:plc:folders-owner"), syntax.DID("did:plc:folders-author")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES ($1,'cid'),($2,'cid')`, owner, author); err != nil {
		t.Fatal(err)
	}
	for _, rkey := range []string{"root", "filed"} {
		if _, err := pool.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES ('at://did:plc:folders-author/social.craftsky.feed.post/'||$1,$2,$1,'cid','saved','{}',now())`, rkey, author); err != nil {
			t.Fatal(err)
		}
	}
	folder := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO saved_post_folders(id,owner_did,name,created_at,updated_at) VALUES($1,$2,'Ideas',now(),now())`, folder, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO saved_posts(owner_did,post_uri,folder_id,saved_at) VALUES ($1,'at://did:plc:folders-author/social.craftsky.feed.post/root',NULL,now()),($1,'at://did:plc:folders-author/social.craftsky.feed.post/filed',$2,now())`, owner, folder); err != nil {
		t.Fatal(err)
	}
	accountID, subID := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES($1,'did:plc:folders-payer',$2)`, accountID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES($1,$2,'project','sub','plus','app','app_store','production','active',true,'plus',1)`, subID, accountID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES($1,'plus',$2,now())`, subID, owner); err != nil {
		t.Fatal(err)
	}
	access := subscriptions.NewStore(pool)
	store := api.NewSavedPostStore(pool)
	filter := api.SavedPostListFilter{Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortNewest, Limit: 50}
	if _, err := pool.Exec(ctx, `UPDATE provider_subscriptions SET gives_access=false WHERE id=$1`, subID); err != nil {
		t.Fatal(err)
	}
	if current, err := access.SelfAccess(ctx, owner, time.Now()); err != nil || current.AllowsPlus() {
		t.Fatalf("lapsed access=%+v err=%v", current, err)
	}
	flat, _, err := store.ListSavedRefs(ctx, owner, filter)
	if err != nil || len(flat) != 2 {
		t.Fatalf("flat refs=%+v err=%v", flat, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE provider_subscriptions SET gives_access=true WHERE id=$1`, subID); err != nil {
		t.Fatal(err)
	}
	if current, err := access.SelfAccess(ctx, owner, time.Now()); err != nil || !current.AllowsPlus() {
		t.Fatalf("restored access=%+v err=%v", current, err)
	}
	filter.Scope, filter.FolderID = api.SavedPostScopeFolder, folder.String()
	filed, _, err := store.ListSavedRefs(ctx, owner, filter)
	if err != nil || len(filed) != 1 || filed[0].FolderID == nil || *filed[0].FolderID != folder.String() {
		t.Fatalf("restored folder refs=%+v err=%v", filed, err)
	}
}

func TestFreeOrdinaryResaveHidesRetainedFolderID(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	owner, author := syntax.DID("did:plc:resave-owner"), syntax.DID("did:plc:resave-author")
	uri := syntax.ATURI("at://did:plc:resave-author/social.craftsky.feed.post/one")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES ($1,'cid'),($2,'cid')`, owner, author); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES ($1,'active',1,1,'test',now(),now(),now())`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES ($1,$2,'one','cid','saved','{}',now())`, uri, author); err != nil {
		t.Fatal(err)
	}
	folder := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO saved_post_folders(id,owner_did,name,created_at,updated_at) VALUES ($1,$2,'Ideas',now(),now())`, folder, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO saved_posts(owner_did,post_uri,folder_id,saved_at) VALUES ($1,$2,$3,now())`, owner, uri, folder); err != nil {
		t.Fatal(err)
	}
	accountID, subID := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ($1,'did:plc:resave-payer',$2)`, accountID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ($1,$2,'project','sub','plus','app','app_store','production','active',false,'plus',1)`, subID, accountID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ($1,'plus',$2,now())`, subID, owner); err != nil {
		t.Fatal(err)
	}
	store := api.NewSavedPostStore(pool)
	access := subscriptions.NewStore(pool)
	request := httptest.NewRequest(http.MethodPost, "/v1/posts/"+author.String()+"/one/saves", nil)
	request.SetPathValue("did", author.String())
	request.SetPathValue("rkey", "one")
	request = request.WithContext(ownerlifecycle.WithExpectedGeneration(middleware.WithDID(request.Context(), owner), 1))
	response := httptest.NewRecorder()
	api.SavePostHandler(&fakeSavedPostTargetResolver{uri: uri}, store, access).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("ordinary save status = %d: %s", response.Code, response.Body.String())
	}
	var state api.SavedPostState
	if err := json.Unmarshal(response.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.FolderID != nil {
		t.Fatalf("Free response disclosed retained folder: %s", response.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE provider_subscriptions SET gives_access=true WHERE id=$1`, subID); err != nil {
		t.Fatal(err)
	}
	restored, err := store.ReadState(ctx, owner, uri)
	if err != nil || restored.FolderID == nil || *restored.FolderID != folder.String() {
		t.Fatalf("restored membership = %+v, err=%v", restored, err)
	}
}
