package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

func TestLanguagePreferencesHandlersReadAndReplaceAuthenticatedAccount(t *testing.T) {
	store := newLanguagePreferencesAPIStore(t)

	get := api.GetLanguagePreferencesHandler(store)
	getResponse := httptest.NewRecorder()
	get.ServeHTTP(getResponse, languagePreferencesRequest(
		http.MethodGet,
		"/v1/languages/preferences",
		"",
		"did:plc:alice",
	))
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body = %s", getResponse.Code, getResponse.Body.String())
	}
	assertLanguagePreferencesJSON(t, getResponse.Body.Bytes(), "en", []string{"en"})

	put := api.PutLanguagePreferencesHandler(store)
	putResponse := httptest.NewRecorder()
	put.ServeHTTP(putResponse, languagePreferencesRequest(
		http.MethodPut,
		"/v1/languages/preferences",
		`{"primaryLanguage":"es","contentLanguages":["es","en"]}`,
		"did:plc:alice",
	))
	if putResponse.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", putResponse.Code, putResponse.Body.String())
	}
	assertLanguagePreferencesJSON(t, putResponse.Body.Bytes(), "es", []string{"es", "en"})

	bob, err := store.Get(context.Background(), syntax.DID("did:plc:bob"))
	if err != nil {
		t.Fatalf("read Bob after Alice replacement: %v", err)
	}
	if bob.PrimaryLanguage != "fr" ||
		len(bob.ContentLanguages) != 2 ||
		bob.ContentLanguages[0] != "fr" ||
		bob.ContentLanguages[1] != "cy" {
		t.Fatalf("Bob changed after Alice replacement: %+v", bob)
	}
}

func TestPutLanguagePreferencesRejectsInvalidOrAccountSelectingRequestsAtomically(t *testing.T) {
	tests := []struct {
		name   string
		target string
		body   string
	}{
		{
			name:   "invalid primary",
			target: "/v1/languages/preferences",
			body:   `{"primaryLanguage":"not_a_tag","contentLanguages":["en"]}`,
		},
		{
			name:   "unknown account selector",
			target: "/v1/languages/preferences",
			body:   `{"primaryLanguage":"fr","contentLanguages":["fr"],"accountDid":"did:plc:bob"}`,
		},
		{
			name:   "unexpected query selector",
			target: "/v1/languages/preferences?did=did:plc:bob",
			body:   `{"primaryLanguage":"fr","contentLanguages":["fr"]}`,
		},
		{
			name:   "trailing JSON",
			target: "/v1/languages/preferences",
			body:   `{"primaryLanguage":"fr","contentLanguages":["fr"]} {}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newLanguagePreferencesAPIStore(t)
			response := httptest.NewRecorder()
			api.PutLanguagePreferencesHandler(store).ServeHTTP(
				response,
				languagePreferencesRequest(
					http.MethodPut,
					test.target,
					test.body,
					"did:plc:alice",
				),
			)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"error":"invalid_request"`) {
				t.Fatalf("error envelope = %s, want invalid_request", response.Body.String())
			}

			alice, err := store.Get(context.Background(), syntax.DID("did:plc:alice"))
			if err != nil {
				t.Fatalf("read Alice after rejection: %v", err)
			}
			if alice.PrimaryLanguage != "en" ||
				len(alice.ContentLanguages) != 1 ||
				alice.ContentLanguages[0] != "en" {
				t.Fatalf("Alice changed after rejected request: %+v", alice)
			}
			bob, err := store.Get(context.Background(), syntax.DID("did:plc:bob"))
			if err != nil {
				t.Fatalf("read Bob after rejection: %v", err)
			}
			if bob.PrimaryLanguage != "fr" {
				t.Fatalf("Bob changed after rejected request: %+v", bob)
			}
		})
	}
}

func TestInitializeLanguagePreferencesReturnsExistingAuthoritativeRow(t *testing.T) {
	store := newLanguagePreferencesAPIStore(t)
	response := httptest.NewRecorder()
	api.InitializeLanguagePreferencesHandler(store).ServeHTTP(
		response,
		languagePreferencesRequest(
			http.MethodPost,
			"/v1/languages/preferences/initialize",
			`{"primaryLanguage":"es","contentLanguages":["es"]}`,
			"did:plc:alice",
		),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	assertLanguagePreferencesJSON(t, response.Body.Bytes(), "en", []string{"en"})

	stored, err := store.Get(context.Background(), syntax.DID("did:plc:alice"))
	if err != nil {
		t.Fatalf("read existing preferences: %v", err)
	}
	if stored.PrimaryLanguage != "en" ||
		len(stored.ContentLanguages) != 1 ||
		stored.ContentLanguages[0] != "en" {
		t.Fatalf("stored preferences were overwritten: %+v", stored)
	}
}

func newLanguagePreferencesAPIStore(t *testing.T) *languages.Store {
	t.Helper()
	up, err := os.ReadFile("../../migrations/000033_post_languages.up.sql")
	if err != nil {
		t.Fatalf("read language migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_posts (
			uri TEXT PRIMARY KEY,
			did TEXT NOT NULL,
			rkey TEXT NOT NULL,
			cid TEXT NOT NULL,
			record JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY,
			state TEXT NOT NULL,
			generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL DEFAULT 1,
			transition_reason TEXT NOT NULL DEFAULT 'test',
			transitioned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		INSERT INTO owner_lifecycles(owner_did,state,generation)
		VALUES('did:plc:alice','active',1),('did:plc:bob','active',1);
	`)
	if _, err := pool.Exec(context.Background(), string(up)); err != nil {
		t.Fatalf("apply language migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO account_language_preferences (
			account_did,
			primary_language,
			content_languages
		) VALUES
			('did:plc:alice', 'en', ARRAY['en']::text[]),
			('did:plc:bob', 'fr', ARRAY['fr', 'cy']::text[])
	`); err != nil {
		t.Fatalf("seed preferences: %v", err)
	}
	return languages.NewStore(pool)
}

func languagePreferencesRequest(
	method string,
	target string,
	body string,
	did syntax.DID,
) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	ctx := middleware.WithDID(request.Context(), did)
	return request.WithContext(middleware.WithOwnerGeneration(ctx, 1))
}

func assertLanguagePreferencesJSON(
	t *testing.T,
	body []byte,
	wantPrimary string,
	wantContent []string,
) {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["primaryLanguage"] != wantPrimary {
		t.Fatalf("primaryLanguage = %v, want %q", response["primaryLanguage"], wantPrimary)
	}
	content, ok := response["contentLanguages"].([]any)
	if !ok || len(content) != len(wantContent) {
		t.Fatalf("contentLanguages = %v, want %v", response["contentLanguages"], wantContent)
	}
	for index, want := range wantContent {
		if content[index] != want {
			t.Fatalf("contentLanguages = %v, want %v", content, wantContent)
		}
	}
	if _, exists := response["appLanguage"]; exists {
		t.Fatal("response unexpectedly contains appLanguage")
	}
}
