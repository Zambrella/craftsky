package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/testdb"
)

func TestLanguagePreferenceRoutesUseExactAuthenticatedContract(t *testing.T) {
	wantPolicies := map[string]struct {
		rateClass RateClass
		bodyKind  BodyKind
	}{
		"GET /v1/languages/preferences":             {RateClassRead, BodyNoBody},
		"PUT /v1/languages/preferences":             {RateClassWrite, BodyDefaultJSON},
		"POST /v1/languages/preferences/initialize": {RateClassWrite, BodyDefaultJSON},
	}
	for _, policy := range V1RoutePolicies(app.EnvDev, app.Config{Env: app.EnvDev}) {
		key := policy.Method + " " + policy.PathPattern
		want, exists := wantPolicies[key]
		if !exists {
			continue
		}
		if policy.AccessClass != AccessCurrentMember ||
			policy.RateClass != want.rateClass ||
			policy.BodyKind != want.bodyKind {
			t.Fatalf("%s policy = %+v", key, policy)
		}
		delete(wantPolicies, key)
	}
	if len(wantPolicies) != 0 {
		t.Fatalf("missing language preference policies: %v", wantPolicies)
	}

	up, err := os.ReadFile("../../migrations/000033_post_languages.up.sql")
	if err != nil {
		t.Fatalf("read language migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (
			did TEXT PRIMARY KEY
		);
		INSERT INTO craftsky_profiles(did) VALUES ('did:plc:alice');
		CREATE TABLE craftsky_posts (
			uri TEXT PRIMARY KEY,
			did TEXT NOT NULL,
			rkey TEXT NOT NULL,
			cid TEXT NOT NULL,
			record JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		);
	`)
	if _, err := pool.Exec(context.Background(), string(up)); err != nil {
		t.Fatalf("apply language migration: %v", err)
	}

	deps := testDeps()
	deps.DB = pool
	deps.LanguagePreferences = languages.NewStore(pool)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	for _, target := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/v1/languages/preferences", ""},
		{http.MethodPut, "/v1/languages/preferences", `{"primaryLanguage":"en","contentLanguages":["en"]}`},
		{http.MethodPost, "/v1/languages/preferences/initialize", `{"primaryLanguage":"en","contentLanguages":["en"]}`},
	} {
		request := httptest.NewRequest(target.method, target.path, strings.NewReader(target.body))
		request.Header.Set("X-Craftsky-Device-Id", "test-device")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s unauthenticated status = %d, body = %s", target.method, response.Code, response.Body.String())
		}
	}

	response := serveLanguagePreferencesRoute(
		mux,
		http.MethodGet,
		"/v1/languages/preferences",
		"",
	)
	if response.Code != http.StatusNotFound ||
		!strings.Contains(response.Body.String(), `"error":"language_preferences_not_found"`) {
		t.Fatalf("absent GET status = %d, body = %s", response.Code, response.Body.String())
	}

	response = serveLanguagePreferencesRoute(
		mux,
		http.MethodPut,
		"/v1/languages/preferences",
		`{"primaryLanguage":"fr","contentLanguages":["fr"]}`,
	)
	if response.Code != http.StatusNotFound {
		t.Fatalf("absent PUT status = %d, body = %s", response.Code, response.Body.String())
	}

	response = serveLanguagePreferencesRoute(
		mux,
		http.MethodPost,
		"/v1/languages/preferences/initialize",
		`{"primaryLanguage":"en","contentLanguages":["en","cy"]}`,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("initialize status = %d, body = %s", response.Code, response.Body.String())
	}

	response = serveLanguagePreferencesRoute(
		mux,
		http.MethodGet,
		"/v1/languages/preferences",
		"",
	)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"primaryLanguage":"en"`) {
		t.Fatalf("initialized GET status = %d, body = %s", response.Code, response.Body.String())
	}

	response = serveLanguagePreferencesRoute(
		mux,
		http.MethodPut,
		"/v1/languages/preferences?accountDid=did:plc:bob",
		`{"primaryLanguage":"fr","contentLanguages":["fr"]}`,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("selector query status = %d, body = %s", response.Code, response.Body.String())
	}

	response = serveLanguagePreferencesRoute(
		mux,
		http.MethodPost,
		"/v1/languages/preferences/initialize",
		`{"primaryLanguage":"fr","contentLanguages":["fr"],"did":"did:plc:bob"}`,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("selector body status = %d, body = %s", response.Code, response.Body.String())
	}
}

func serveLanguagePreferencesRoute(
	mux *http.ServeMux,
	method string,
	target string,
	body string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer test")
	request.Header.Set("X-Dev-DID", "did:plc:alice")
	request.Header.Set("X-Craftsky-Device-Id", "test-device")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}
