package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestBillingIdentityIsExplicitStableAndPrivate(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles(did) VALUES
			('did:plc:billing-owner'), ('did:plc:unrelated');
	`+string(migration))
	deps := testDeps()
	deps.DB = pool
	deps.Subscriptions = subscriptions.NewStore(pool)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	request := func(method, path, did string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-Craftsky-Device-Id", "billing-device")
		req.Header.Set("X-Dev-DID", did)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}

	var before int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM billing_accounts`).Scan(&before); err != nil || before != 0 {
		t.Fatalf("billing accounts before explicit ensure = %d, error %v", before, err)
	}

	first := request(http.MethodPut, "/v1/billing/account", "did:plc:billing-owner")
	if first.Code != http.StatusCreated {
		t.Fatalf("first ensure = %d, want 201; body=%s", first.Code, first.Body.String())
	}
	second := request(http.MethodPut, "/v1/billing/account", "did:plc:billing-owner")
	if second.Code != http.StatusOK {
		t.Fatalf("repeated ensure = %d, want 200; body=%s", second.Code, second.Body.String())
	}
	ownerRead := request(http.MethodGet, "/v1/billing/account", "did:plc:billing-owner")
	if ownerRead.Code != http.StatusOK {
		t.Fatalf("owner read = %d, want 200; body=%s", ownerRead.Code, ownerRead.Body.String())
	}

	decodeIdentity := func(response *httptest.ResponseRecorder) string {
		t.Helper()
		var body struct {
			RevenueCatAppUserID string `json:"revenueCatAppUserId"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.RevenueCatAppUserID == "" {
			t.Fatalf("decode billing identity: %v; body=%s", err, response.Body.String())
		}
		return body.RevenueCatAppUserID
	}
	stableID := decodeIdentity(first)
	if decodeIdentity(second) != stableID || decodeIdentity(ownerRead) != stableID {
		t.Fatal("billing identity changed across ensure/read")
	}

	unrelated := request(http.MethodGet, "/v1/billing/account", "did:plc:unrelated")
	if unrelated.Code != http.StatusNotFound {
		t.Fatalf("unrelated read = %d, want 404; body=%s", unrelated.Code, unrelated.Body.String())
	}
	var errorBody map[string]any
	if err := json.Unmarshal(unrelated.Body.Bytes(), &errorBody); err != nil {
		t.Fatal(err)
	}
	if errorBody["error"] != "billing_account_not_found" || errorBody["requestId"] == nil {
		t.Fatalf("non-canonical or leaking error = %#v", errorBody)
	}
	if _, leaked := errorBody["revenueCatAppUserId"]; leaked {
		t.Fatalf("unrelated response leaked billing identity: %#v", errorBody)
	}
}

func TestSelfAccessRouteIsDIDBoundLocalAndMinimal(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles(did) VALUES ('did:plc:access-member');
	`+string(migration))
	ctx := context.Background()
	end := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id)
		VALUES('10000000-0000-4000-8000-000000000011','did:plc:access-owner','20000000-0000-4000-8000-000000000011')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,current_period_ends_at,mapped_tier,accepted_generation)
		VALUES('30000000-0000-4000-8000-000000000011','10000000-0000-4000-8000-000000000011','provider-project','provider-subscription-canary','provider-product-canary','app','app_store','production','active',true,$1,'business',1)
	`, end); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at,anomaly)
		VALUES('30000000-0000-4000-8000-000000000011','business','did:plc:access-member',now(),'same_tier_duplicate')
	`); err != nil {
		t.Fatal(err)
	}

	deps := testDeps()
	deps.DB = pool
	deps.Subscriptions = subscriptions.NewStore(pool)
	mux := http.NewServeMux()
	AddRoutes(ctx, mux, deps)
	request := func() map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/v1/subscriptions/access", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-Craftsky-Device-Id", "access-device")
		req.Header.Set("X-Dev-DID", "did:plc:access-member")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("self access status = %d; body=%s", response.Code, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}

	paid := request()
	if paid["did"] != "did:plc:access-member" || paid["effectiveTier"] != "business" || paid["givesAccess"] != true || paid["assignedTier"] != "business" {
		t.Fatalf("paid access = %#v", paid)
	}
	for _, prohibited := range []string{"billingAccountId", "revenueCatAppUserId", "productId", "store", "status", "anomaly", "providerSubscriptionId"} {
		if _, ok := paid[prohibited]; ok {
			t.Fatalf("self access leaked %s: %#v", prohibited, paid)
		}
	}
	if len(paid) != 5 {
		t.Fatalf("self access fields = %#v, want exactly five", paid)
	}

	if _, err := pool.Exec(ctx, `UPDATE provider_subscriptions SET gives_access=false WHERE id='30000000-0000-4000-8000-000000000011'`); err != nil {
		t.Fatal(err)
	}
	dormant := request()
	if dormant["effectiveTier"] != "free" || dormant["givesAccess"] != false || dormant["assignedTier"] != "business" {
		t.Fatalf("dormant access = %#v", dormant)
	}
}

func TestOwnerBillingStateUsesPrivateCamelCaseContract(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles(did) VALUES ('did:plc:state-owner'),('did:plc:state-non-owner');
	`+string(migration))
	ctx := context.Background()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id,requested_generation,reconciled_generation,reconciliation_requested_at,reconciled_at)
		VALUES('10000000-0000-4000-8000-000000000061','did:plc:state-owner','20000000-0000-4000-8000-000000000061',2,1,$1,$2)
	`, now.Add(-2*time.Hour), now.Add(-3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,pending_payment,auto_renewal_status,current_period_ends_at,mapped_tier,accepted_generation,anomaly)
		VALUES('30000000-0000-4000-8000-000000000061','10000000-0000-4000-8000-000000000061','project','raw-provider-subscription-canary','plus-monthly','app','app_store','production','active',true,false,'will_renew',$1,'plus',1,'same_tier_duplicate')
	`, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_licenses(id,provider_subscription_id,tier,assigned_did,assigned_at,assignable,anomaly)
		VALUES('40000000-0000-4000-8000-000000000061','30000000-0000-4000-8000-000000000061','plus','did:plc:state-beneficiary',$1,false,'same_tier_duplicate')
	`, now); err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Subscriptions = subscriptions.NewStore(pool)
	deps.Now = func() time.Time { return now }
	mux := http.NewServeMux()
	AddRoutes(ctx, mux, deps)
	req := httptest.NewRequest(http.MethodGet, "/v1/billing/account", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Craftsky-Device-Id", "state-device")
	req.Header.Set("X-Dev-DID", "did:plc:state-owner")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("owner state = %d; body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["revenueCatAppUserId"] != "20000000-0000-4000-8000-000000000061" || body["reconciliationStale"] != true {
		t.Fatalf("owner readiness/staleness = %#v", body)
	}
	subscriptionsBody, ok := body["subscriptions"].([]any)
	if !ok || len(subscriptionsBody) != 1 {
		t.Fatalf("subscriptions = %#v", body["subscriptions"])
	}
	licensesBody, ok := body["licenses"].([]any)
	if !ok || len(licensesBody) != 1 {
		t.Fatalf("licenses = %#v", body["licenses"])
	}
	encoded := response.Body.String()
	if strings.Contains(encoded, "raw-provider-subscription-canary") || strings.Contains(encoded, "project\"") {
		t.Fatalf("owner response leaked provider identity: %s", encoded)
	}
	license := licensesBody[0].(map[string]any)
	if license["assignedDid"] != "did:plc:state-beneficiary" || license["anomaly"] != "same_tier_duplicate" || license["assignable"] != false {
		t.Fatalf("license state = %#v", license)
	}
	nonOwnerRequest := httptest.NewRequest(http.MethodGet, "/v1/billing/account", nil)
	nonOwnerRequest.Header.Set("Authorization", "Bearer test-token")
	nonOwnerRequest.Header.Set("X-Craftsky-Device-Id", "state-device")
	nonOwnerRequest.Header.Set("X-Dev-DID", "did:plc:state-non-owner")
	nonOwnerResponse := httptest.NewRecorder()
	mux.ServeHTTP(nonOwnerResponse, nonOwnerRequest)
	if nonOwnerResponse.Code != http.StatusNotFound || strings.Contains(nonOwnerResponse.Body.String(), "20000000-0000-4000-8000-000000000061") || strings.Contains(nonOwnerResponse.Body.String(), "raw-provider-subscription-canary") {
		t.Fatalf("non-owner response leaked billing state: %d %s", nonOwnerResponse.Code, nonOwnerResponse.Body.String())
	}
}
