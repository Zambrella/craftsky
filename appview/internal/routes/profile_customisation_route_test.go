package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const profileCustomisationRouteTestDDL = `
CREATE TABLE craftsky_profiles (
    did        TEXT NOT NULL PRIMARY KEY,
    record_cid TEXT NOT NULL
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:test', 'test-cid');
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
) VALUES('did:plc:test','active',1,1,'test',now(),now(),now());
`

func TestProfileCustomisationRouteUsesAuthenticatedCurrentMemberPolicy(t *testing.T) {
	policy, ok := profileCustomisationRoutePolicy(EnvDev)
	if !ok {
		t.Fatal("PUT /v1/profiles/me/customisation route policy missing")
	}
	if policy.RateClass != RateClassWrite || policy.BodyKind != BodyDefaultJSON ||
		policy.AccessClass != AccessCurrentMember {
		t.Fatalf("customisation route policy = %+v", policy)
	}

	migration, err := os.ReadFile("../../migrations/000036_profile_customisation.up.sql")
	if err != nil {
		t.Fatalf("read customisation migration: %v", err)
	}
	pool := testdb.WithSchema(t, profileCustomisationRouteTestDDL+string(migration))
	deps := testDeps()
	deps.DB = pool
	deps.OwnerLifecycles = newRouteOwnerLifecycleStore(t, pool)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	body := `{"colour":"orchid","profileBorder":"thin","profileBackground":"skewdark"}`
	request := func(authenticated, device bool, devDID string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me/customisation", strings.NewReader(body))
		if authenticated {
			req.Header.Set("Authorization", "Bearer test-token")
		}
		if device {
			req.Header.Set("X-Craftsky-Device-Id", "test-device")
		}
		if devDID != "" {
			req.Header.Set("X-Dev-DID", devDID)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}

	if got := request(false, true, ""); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", got.Code)
	}
	if got := request(true, false, ""); got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), "missing_device_id") {
		t.Fatalf("missing-device response = %d %s", got.Code, got.Body.String())
	}
	if got := request(true, true, "did:plc:departed"); got.Code != http.StatusNotFound {
		t.Fatalf("departed-member status = %d, want 404; body=%s", got.Code, got.Body.String())
	}

	response := request(true, true, "")
	if response.Code != http.StatusOK {
		t.Fatalf("valid status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	var got api.ProfileCustomisation
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode valid response: %v", err)
	}
	want := api.ProfileCustomisation{Colour: "orchid", Border: "thin", Background: "skewdark"}
	if got != want {
		t.Fatalf("response = %+v, want %+v", got, want)
	}
	stored, err := api.NewProfileCustomisationStore(pool).Read(context.Background(), "did:plc:test")
	if err != nil || stored != want {
		t.Fatalf("stored = %+v, %v; want %+v", stored, err, want)
	}
}

func profileCustomisationRoutePolicy(env Environment) (RoutePolicy, bool) {
	for _, policy := range V1RoutePolicies(env, Config{Env: env}) {
		if policy.Method == http.MethodPut && policy.PathPattern == "/v1/profiles/me/customisation" {
			return policy, true
		}
	}
	return RoutePolicy{}, false
}
