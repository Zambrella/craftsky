package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/testdb"
)

func TestLanguagePreferenceRoutesPreserveOwnerScopedBehavior(t *testing.T) {
	up, err := testdb.ReadMigration("000033_post_languages.up.sql")
	if err != nil {
		t.Fatalf("read language migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (
			did TEXT PRIMARY KEY
		);
		INSERT INTO craftsky_profiles(did) VALUES ('did:plc:alice');
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
		) VALUES('did:plc:alice','active',1,1,'test',now(),now(),now());
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
	deps.OwnerLifecycles = newRouteOwnerLifecycleStore(t, pool)
	deps.LanguagePreferences = languages.NewStore(pool)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

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
