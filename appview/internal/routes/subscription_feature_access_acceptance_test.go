package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestSubscriptionFeatureAccessAssignedDIDNotBillingOwner(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `CREATE TABLE craftsky_profiles(did TEXT PRIMARY KEY); INSERT INTO craftsky_profiles(did) VALUES ('did:plc:payer'),('did:plc:beneficiary'),('did:plc:business');`+string(migration))
	ctx := context.Background()
	for _, statement := range []string{
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000001','did:plc:payer','20000000-0000-4000-8000-000000000001')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001','project','subscription','plus','app','app_store','production','active',true,'plus',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000001','plus','did:plc:beneficiary',now())`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001','project','business-subscription','business','app','app_store','production','active',true,'business',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000002','business','did:plc:business',now())`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	deps := testDeps()
	deps.DB = pool
	deps.Subscriptions = subscriptions.NewStore(pool)
	called := 0
	deps.routeHandlerDecorator = func(_ RoutePolicy, _ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called++
			w.WriteHeader(http.StatusCreated)
		})
	}
	mux := http.NewServeMux()
	AddRoutes(ctx, mux, deps)
	for _, tc := range []struct {
		did    string
		method string
		path   string
		want   int
	}{
		{did: "did:plc:payer", method: http.MethodPost, path: "/v1/saved-post-folders", want: http.StatusForbidden},
		{did: "did:plc:beneficiary", method: http.MethodPost, path: "/v1/saved-post-folders", want: http.StatusCreated},
		{did: "did:plc:payer", method: http.MethodPatch, path: "/v1/saved-post-folders/00000000-0000-4000-8000-000000000001", want: http.StatusForbidden},
		{did: "did:plc:payer", method: http.MethodDelete, path: "/v1/saved-post-folders/00000000-0000-4000-8000-000000000001", want: http.StatusForbidden},
		{did: "did:plc:payer", method: http.MethodPost, path: "/v1/scheduled-posts", want: http.StatusForbidden},
		{did: "did:plc:beneficiary", method: http.MethodPost, path: "/v1/scheduled-posts", want: http.StatusCreated},
		{did: "did:plc:payer", method: http.MethodPut, path: "/v1/scheduled-posts/00000000-0000-4000-8000-000000000001", want: http.StatusForbidden},
		{did: "did:plc:payer", method: http.MethodPost, path: "/v1/scheduled-posts/00000000-0000-4000-8000-000000000001/publication", want: http.StatusForbidden},
		{did: "did:plc:payer", method: http.MethodPut, path: "/v1/scheduled-post-media/00000000-0000-4000-8000-000000000001", want: http.StatusForbidden},
		{did: "did:plc:payer", method: http.MethodGet, path: "/v1/scheduled-posts", want: http.StatusCreated},
		{did: "did:plc:payer", method: http.MethodGet, path: "/v1/saved-posts", want: http.StatusCreated},
		{did: "did:plc:payer", method: http.MethodGet, path: "/v1/saved-post-folders", want: http.StatusForbidden},
		{did: "did:plc:payer", method: http.MethodPut, path: "/v1/posts/did:plc:payer/3mzzzzzzzzzzz/pin", want: http.StatusForbidden},
		{did: "did:plc:payer", method: http.MethodDelete, path: "/v1/posts/did:plc:payer/3mzzzzzzzzzzz/pin", want: http.StatusForbidden},
		{did: "did:plc:beneficiary", method: http.MethodPut, path: "/v1/posts/did:plc:beneficiary/3mzzzzzzzzzzz/pin", want: http.StatusCreated},
		{did: "did:plc:payer", method: http.MethodGet, path: "/v1/profiles/me/follower-growth", want: http.StatusForbidden},
		{did: "did:plc:payer", method: http.MethodPut, path: "/v1/profiles/me/customisation", want: http.StatusForbidden},
		{did: "did:plc:beneficiary", method: http.MethodGet, path: "/v1/profiles/me/follower-growth", want: http.StatusCreated},
		{did: "did:plc:beneficiary", method: http.MethodPut, path: "/v1/profiles/me/business", want: http.StatusForbidden},
		{did: "did:plc:payer", method: http.MethodGet, path: "/v1/events", want: http.StatusForbidden},
		{did: "did:plc:beneficiary", method: http.MethodDelete, path: "/v1/events/did:plc:beneficiary/3mzzzzzzzzzzz", want: http.StatusForbidden},
		{did: "did:plc:business", method: http.MethodPut, path: "/v1/profiles/me/business", want: http.StatusCreated},
		{did: "did:plc:payer", method: http.MethodGet, path: "/v1/events/did:plc:payer/3mzzzzzzzzzzz", want: http.StatusForbidden},
		{did: "did:plc:beneficiary", method: http.MethodGet, path: "/v1/events/did:plc:business/3mzzzzzzzzzzz", want: http.StatusCreated},
	} {
		var body strings.Reader
		if tc.method != http.MethodGet && tc.method != http.MethodDelete && !strings.HasSuffix(tc.path, "/pin") {
			body = *strings.NewReader(`{"name":"Work"}`)
		}
		r := httptest.NewRequest(tc.method, tc.path, &body)
		r.Header.Set("Authorization", "Bearer test-token")
		r.Header.Set("X-Craftsky-Device-Id", "test-device")
		r.Header.Set("X-Dev-DID", tc.did)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s %s: status = %d, want %d; body=%s", tc.did, tc.path, w.Code, tc.want, w.Body.String())
		}
		if tc.want == http.StatusForbidden && !strings.Contains(w.Body.String(), `"error":"subscription_required"`) {
			t.Fatalf("%s: missing paid-access error: %s", tc.did, w.Body.String())
		}
	}
	if called != 8 {
		t.Fatalf("permitted handler called %d times, want 8", called)
	}
}

// seedPlusRouteAccess sets up paid route fixtures in schemas containing migration 000076.
func seedPlusRouteAccess(t *testing.T, pool *pgxpool.Pool, dids ...string) {
	seedPaidRouteAccess(t, pool, subscriptions.TierPlus, dids...)
}

func seedPaidRouteAccess(t *testing.T, pool *pgxpool.Pool, tier subscriptions.Tier, dids ...string) {
	t.Helper()
	ctx := context.Background()
	for _, did := range dids {
		accountID, subscriptionID := uuid.New(), uuid.New()
		if _, err := pool.Exec(ctx, `INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ($1,$2,$3)`, accountID, did, uuid.New()); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ($1,$2,'test-project',$3,$4,'test-app','app_store','production','active',true,$4,1)`, subscriptionID, accountID, uuid.NewString(), tier); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ($1,$2,$3,now())`, subscriptionID, tier, did); err != nil {
			t.Fatal(err)
		}
	}
}
