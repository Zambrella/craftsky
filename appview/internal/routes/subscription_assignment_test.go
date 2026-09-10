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

func TestAssignmentRoutesEnforceDeviceUniquenessCooldownAndUnassignment(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY, state TEXT NOT NULL, generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL, transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL, terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		CREATE TABLE craftsky_sessions (
			token_hash BYTEA PRIMARY KEY, account_did TEXT NOT NULL,
			last_device_id TEXT, last_seen_at TIMESTAMPTZ NOT NULL,
			idle_expires_at TIMESTAMPTZ NOT NULL, lifecycle_state TEXT NOT NULL,
			revoked_at TIMESTAMPTZ
		);
		CREATE FUNCTION appview_owner_is_active(candidate_did TEXT)
		RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
			SELECT COALESCE((SELECT state='active' FROM owner_lifecycles WHERE owner_did=candidate_did),false)
		$$;
	`+string(migration))
	ctx := context.Background()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did) VALUES('did:plc:route-owner'),('did:plc:route-a'),('did:plc:route-b')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
		SELECT did,'active',1,1,'test',$1,$1,$1 FROM craftsky_profiles
	`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_sessions(token_hash,account_did,last_device_id,last_seen_at,idle_expires_at,lifecycle_state)
		VALUES(decode('11','hex'),'did:plc:route-a','route-device',$1,$1::timestamptz+interval '30 days','active'),
		      (decode('12','hex'),'did:plc:route-b','wrong-device',$1,$1::timestamptz+interval '30 days','active')
	`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id)
		VALUES('10000000-0000-4000-8000-000000000031','did:plc:route-owner','20000000-0000-4000-8000-000000000031');
		INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation)
		VALUES('30000000-0000-4000-8000-000000000031','10000000-0000-4000-8000-000000000031','project','sub-plus','plus','app','app_store','production','active',true,'plus',1),
		      ('30000000-0000-4000-8000-000000000032','10000000-0000-4000-8000-000000000031','project','sub-business','business','app','app_store','production','expired',false,'business',1);
		INSERT INTO billing_licenses(id,provider_subscription_id,tier)
		VALUES('40000000-0000-4000-8000-000000000031','30000000-0000-4000-8000-000000000031','plus'),
		      ('40000000-0000-4000-8000-000000000032','30000000-0000-4000-8000-000000000032','business')
	`); err != nil {
		t.Fatal(err)
	}

	deps := testDeps()
	deps.DB = pool
	deps.Subscriptions = subscriptions.NewStore(pool)
	deps.Now = func() time.Time { return now }
	mux := http.NewServeMux()
	AddRoutes(ctx, mux, deps)
	request := func(method, licenseID, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "/v1/billing/licenses/"+licenseID+"/assignment", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-Craftsky-Device-Id", "route-device")
		req.Header.Set("X-Dev-DID", "did:plc:route-owner")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}

	licenseA := "40000000-0000-4000-8000-000000000031"
	licenseB := "40000000-0000-4000-8000-000000000032"
	if response := request(http.MethodPut, licenseA, `{"targetDid":"did:plc:route-a"}`); response.Code != http.StatusOK {
		t.Fatalf("initial assignment = %d; body=%s", response.Code, response.Body.String())
	}
	if response := request(http.MethodPut, licenseB, `{"targetDid":"did:plc:route-a"}`); response.Code != http.StatusConflict {
		t.Fatalf("dormant duplicate = %d; body=%s", response.Code, response.Body.String())
	}
	if response := request(http.MethodPut, licenseB, `{"targetDid":"did:plc:route-b"}`); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("wrong device = %d; body=%s", response.Code, response.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_sessions SET last_device_id='route-device' WHERE account_did='did:plc:route-b'`); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	if response := request(http.MethodPut, licenseA, `{"targetDid":"did:plc:route-b"}`); response.Code != http.StatusOK {
		t.Fatalf("first target change = %d; body=%s", response.Code, response.Body.String())
	}
	now = now.Add(time.Hour)
	if response := request(http.MethodPut, licenseA, `{"targetDid":"did:plc:route-a"}`); response.Code != http.StatusConflict {
		t.Fatalf("cooldown = %d; body=%s", response.Code, response.Body.String())
	}
	if response := request(http.MethodDelete, licenseA, ""); response.Code != http.StatusNoContent {
		t.Fatalf("unassign = %d; body=%s", response.Code, response.Body.String())
	}
	var assigned *string
	var lastChange *time.Time
	if err := pool.QueryRow(ctx, `SELECT assigned_did,last_target_change_at FROM billing_licenses WHERE id=$1`, licenseA).Scan(&assigned, &lastChange); err != nil {
		t.Fatal(err)
	}
	if assigned != nil || lastChange == nil {
		t.Fatalf("unassignment state = assigned %v, last change %v", assigned, lastChange)
	}
}
