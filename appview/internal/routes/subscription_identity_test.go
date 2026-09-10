package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestOwnerRefreshUsesOnlyServerStoredBillingIdentity(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles(did) VALUES('did:plc:identity-owner'),('did:plc:identity-other');
	`+string(migration))
	ctx := context.Background()
	store := subscriptions.NewStore(pool)
	account, _, err := store.EnsureAccount(ctx, "did:plc:identity-owner")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	deps := testDeps()
	deps.DB = pool
	deps.Subscriptions = store
	deps.Now = func() time.Time { return now }
	mux := http.NewServeMux()
	AddRoutes(ctx, mux, deps)
	request := func(did, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/reconciliation", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-Craftsky-Device-Id", "identity-device")
		req.Header.Set("X-Dev-DID", did)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	if response := request("did:plc:identity-owner", ""); response.Code != http.StatusAccepted || !strings.Contains(response.Body.String(), `"status":"pending"`) {
		t.Fatalf("owner refresh = %d; body=%s", response.Code, response.Body.String())
	}
	var generation int64
	if err := pool.QueryRow(ctx, `SELECT requested_generation FROM billing_accounts WHERE id=$1 AND revenuecat_app_user_id=$2`, account.ID, account.RevenueCatAppUserID).Scan(&generation); err != nil || generation != 1 {
		t.Fatalf("owner generation = %d, error %v", generation, err)
	}
	if response := request("did:plc:identity-other", ""); response.Code != http.StatusNotFound {
		t.Fatalf("non-owner refresh = %d; body=%s", response.Code, response.Body.String())
	}
	if response := request("did:plc:identity-owner", `{"revenueCatAppUserId":"20000000-0000-4000-8000-000000000099"}`); response.Code != http.StatusBadRequest {
		t.Fatalf("client identity override = %d; body=%s", response.Code, response.Body.String())
	}
	if err := pool.QueryRow(ctx, `SELECT requested_generation FROM billing_accounts WHERE id=$1`, account.ID).Scan(&generation); err != nil || generation != 1 {
		t.Fatalf("rejected override generation = %d, error %v", generation, err)
	}
}
