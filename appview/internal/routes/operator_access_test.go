package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
)

type operatorFixtureAuthenticator struct {
	moderator ctxkeys.Moderator
	token     string
}

func (a operatorFixtureAuthenticator) Authenticate(_ context.Context, token string) (ctxkeys.Moderator, error) {
	if token != a.token || token == "" {
		return ctxkeys.Moderator{}, middleware.ErrModeratorAuthentication
	}
	return a.moderator, nil
}

func TestAuthenticatedOperatorWithoutGrantsCannotReachAdminOperations(t *testing.T) {
	deps := testDeps()
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = middleware.ModeratorDatabaseAuthentication(operatorFixtureAuthenticator{token: "fixture-token", moderator: ctxkeys.Moderator{ActorID: "unprivileged", Role: "safetyAdministrator", SourceSystem: "test", Permissions: map[string]bool{}}}, nil)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	for _, endpoint := range []string{
		"GET /v1/admin/safety/status",
		"GET /v1/admin/safety/incidents/4638932e-28cd-48a6-8a1a-80b00ab22fa5",
		"GET /v1/admin/moderation/cases",
		"GET /v1/admin/moderation/cases/invalid-reference",
		"POST /v1/admin/moderation/cases/MOD-4638932e-28cd-48a6-8a1a-80b00ab22fa5/decisions",
		"POST /v1/admin/safety/external-intakes",
		"POST /v1/admin/eligibility/reviews",
		"POST /v1/admin/safety/evidence/4638932e-28cd-48a6-8a1a-80b00ab22fa5/access",
		"POST /v1/admin/safety/incidents/4638932e-28cd-48a6-8a1a-80b00ab22fa5/holds",
		"POST /v1/admin/safety/workflows/authority-requests",
	} {
		t.Run(endpoint, func(t *testing.T) {
			method, path, _ := strings.Cut(endpoint, " ")
			r := httptest.NewRequest(method, path, nil)
			if method == http.MethodPost {
				r = httptest.NewRequest(method, path, strings.NewReader(`{}`))
			}
			r.Header.Set("Authorization", "Bearer fixture-token")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Fatalf("status = %d, body=%s", w.Code, w.Body)
			}
		})
	}
}

func TestOperatorIdentityExposesEffectiveGrantsWithoutCredentials(t *testing.T) {
	deps := testDeps()
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = middleware.ModeratorDatabaseAuthentication(operatorFixtureAuthenticator{token: "identity-credential-canary", moderator: ctxkeys.Moderator{ActorID: "named-moderator", Role: "moderator", SourceSystem: "admin-api", Permissions: map[string]bool{"moderation.read": true, "evidence.access": true}}}, nil)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	r := httptest.NewRequest("GET", "/v1/admin/identity", nil)
	r.Header.Set("Authorization", "Bearer identity-credential-canary")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"actorId":"named-moderator"`) || !strings.Contains(w.Body.String(), `"moderation.read":true`) || !strings.Contains(w.Body.String(), `"evidence.access":false`) {
		t.Fatalf("identity: %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "credential-canary") {
		t.Fatal("credential leaked")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("identity must not be cached")
	}
}

func TestMissingOperatorAuthenticatorFailsClosed(t *testing.T) {
	deps := testDeps()
	deps.Config.ModerationAdminEnabled = true
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	r := httptest.NewRequest("GET", "/v1/admin/identity", nil)
	r.Header.Set("Authorization", "Bearer stale-shared-token")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("missing authenticator: %d %s", w.Code, w.Body)
	}
}
