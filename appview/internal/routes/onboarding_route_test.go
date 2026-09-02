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

	"social.craftsky/appview/internal/testdb"
)

func TestOnboardingRoutesUseCurrentMemberBodylessPolicies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		method string
		path   string
		rate   RateClass
	}{
		{method: "GET", path: "/v1/onboarding/status", rate: RateClassRead},
		{method: "POST", path: "/v1/onboarding/completion", rate: RateClassWrite},
	}
	for _, test := range tests {
		policy := mustPolicy(test.method, test.path)
		if policy.RateClass != test.rate || policy.BodyKind != BodyNoBody ||
			policy.AccessClass != AccessCurrentMember {
			t.Fatalf("policy for %s %s = %+v", test.method, test.path, policy)
		}
	}
}

func TestOnboardingRoutesEnforceAuthenticatedCurrentMemberContract(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000065_account_onboarding_completion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles(did) VALUES
			('did:plc:test'), ('did:plc:bob'), ('did:plc:departed');
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY,
			state TEXT NOT NULL,
			generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL,
			transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL,
			terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES
			('did:plc:test','active',1,1,'test',now(),now(),now()),
			('did:plc:bob','active',1,1,'test',now(),now(),now()),
			('did:plc:departed','departed',2,2,'test',now(),now(),now());
	`+string(migration))
	deps := testDeps()
	deps.DB = pool
	deps.OwnerLifecycles = newRouteOwnerLifecycleStore(t, pool)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	request := func(method, path, body, did string, auth, device bool) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if auth {
			req.Header.Set("Authorization", "Bearer onboarding-test")
		}
		if device {
			req.Header.Set("X-Craftsky-Device-Id", "onboarding-test-device")
		}
		if did != "" {
			req.Header.Set("X-Dev-DID", did)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	assertEnvelope := func(response *httptest.ResponseRecorder, status int, code string) {
		t.Helper()
		if response.Code != status {
			t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode envelope: %v", err)
		}
		if body["error"] != code || body["message"] == nil || body["requestId"] == nil {
			t.Fatalf("envelope = %#v", body)
		}
	}

	assertEnvelope(request(http.MethodGet, "/v1/onboarding/status", "", "", false, true), http.StatusUnauthorized, "unauthorized")
	assertEnvelope(request(http.MethodGet, "/v1/onboarding/status", "", "", true, false), http.StatusBadRequest, "missing_device_id")
	assertEnvelope(request(http.MethodPost, "/v1/onboarding/completion", "", "did:plc:departed", true, true), http.StatusNotFound, "profile_not_found")
	assertEnvelope(request(http.MethodGet, "/v1/onboarding/status?accountDid=did:plc:bob", "", "", true, true), http.StatusBadRequest, "invalid_request")
	assertEnvelope(request(http.MethodPost, "/v1/onboarding/completion", `{"accountDid":"did:plc:bob"}`, "", true, true), http.StatusBadRequest, "request_body_not_allowed")

	initial := request(http.MethodGet, "/v1/onboarding/status", "", "", true, true)
	if initial.Code != http.StatusOK || !strings.Contains(initial.Body.String(), `"completed":false`) {
		t.Fatalf("initial status = %d, body = %s", initial.Code, initial.Body.String())
	}
	first := request(http.MethodPost, "/v1/onboarding/completion", "", "", true, true)
	second := request(http.MethodPost, "/v1/onboarding/completion", "", "", true, true)
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("completion statuses = %d/%d; bodies = %s / %s", first.Code, second.Code, first.Body.String(), second.Body.String())
	}
	var firstBody, secondBody struct {
		Completed   bool      `json:"completed"`
		CompletedAt time.Time `json:"completedAt"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondBody); err != nil {
		t.Fatal(err)
	}
	if !firstBody.Completed || firstBody.CompletedAt.IsZero() || !firstBody.CompletedAt.Equal(secondBody.CompletedAt) {
		t.Fatalf("idempotent responses = %+v / %+v", firstBody, secondBody)
	}
	bob := request(http.MethodGet, "/v1/onboarding/status", "", "did:plc:bob", true, true)
	if bob.Code != http.StatusOK || !strings.Contains(bob.Body.String(), `"completed":false`) {
		t.Fatalf("Bob status = %d, body = %s", bob.Code, bob.Body.String())
	}
}
