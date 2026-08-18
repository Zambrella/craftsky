package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
)

func TestV1CatalogueRejectsDuplicatePolicy(t *testing.T) {
	t.Parallel()

	policy := RoutePolicy{Method: http.MethodGet, PathPattern: "/v1/things/{id}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember}
	if _, err := NewV1Catalogue([]RoutePolicy{policy, policy}); err == nil {
		t.Fatal("NewV1Catalogue() error = nil, want duplicate policy rejection")
	}
}

func TestV1CatalogueRejectsMissingOrInvalidAccessClass(t *testing.T) {
	t.Parallel()

	for _, accessClass := range []AccessClass{0, AccessClass(255)} {
		policy := RoutePolicy{
			Method:      http.MethodGet,
			PathPattern: "/v1/things/{id}",
			RateClass:   RateClassRead,
			BodyKind:    BodyNoBody,
			AccessClass: accessClass,
		}
		if _, err := NewV1Catalogue([]RoutePolicy{policy}); err == nil || !strings.Contains(err.Error(), "access class") {
			t.Fatalf("NewV1Catalogue(access class %d) error = %v, want access-class rejection", accessClass, err)
		}
	}
}

func TestV1CatalogueDrivesPreflightForEveryPolicy(t *testing.T) {
	policies := V1RoutePolicies(app.EnvDev, app.Config{
		Env:                 app.EnvDev,
		EnableDevModeration: true,
		DevModerationToken:  "configured",
	})
	catalogue, err := NewV1Catalogue(policies)
	if err != nil {
		t.Fatal(err)
	}
	handler := catalogue.RoutingHandler(middleware.CORS(
		[]string{"https://app.craftsky.social"},
		catalogue,
	)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("preflight reached application handler")
	})))

	for _, policy := range policies {
		policy := policy
		t.Run(policy.Method+" "+policy.PathPattern, func(t *testing.T) {
			path := concretePolicyPath(policy.PathPattern)
			req := httptest.NewRequest(http.MethodOptions, path, nil)
			req.Header.Set("Origin", "https://app.craftsky.social")
			req.Header.Set("Access-Control-Request-Method", policy.Method)
			req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, X-Craftsky-Device-Id")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("allowed preflight status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
			}
			methods := recorder.Header().Get("Access-Control-Allow-Methods")
			if !headerListContains(methods, policy.Method) || !headerListContains(methods, http.MethodOptions) {
				t.Fatalf("Access-Control-Allow-Methods = %q, missing %s/OPTIONS", methods, policy.Method)
			}

			disallowed := httptest.NewRequest(http.MethodOptions, path, nil)
			disallowed.Header.Set("Origin", "https://evil.example")
			disallowed.Header.Set("Access-Control-Request-Method", policy.Method)
			disallowedRecorder := httptest.NewRecorder()
			handler.ServeHTTP(disallowedRecorder, disallowed)
			if disallowedRecorder.Code != http.StatusForbidden {
				t.Fatalf("disallowed preflight status = %d, want 403", disallowedRecorder.Code)
			}
		})
	}
}

func concretePolicyPath(pattern string) string {
	parts := strings.Split(pattern, "/")
	for index, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			parts[index] = "sample"
		}
	}
	return strings.Join(parts, "/")
}

func headerListContains(value, want string) bool {
	for _, candidate := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(candidate), want) {
			return true
		}
	}
	return false
}

func TestV1RoutingOwnsUnknownAndWrongMethodErrors(t *testing.T) {
	t.Parallel()

	catalogue, err := NewV1Catalogue([]RoutePolicy{
		{Method: http.MethodGet, PathPattern: "/v1/things/{id}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: http.MethodPatch, PathPattern: "/v1/things/{id}", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
	})
	if err != nil {
		t.Fatal(err)
	}
	nextCalled := false
	handler := catalogue.RoutingHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name      string
		method    string
		path      string
		wantCode  int
		wantError string
		wantAllow string
		wantNext  bool
	}{
		{name: "allowed", method: http.MethodPatch, path: "/v1/things/one", wantCode: http.StatusNoContent, wantNext: true},
		{name: "unknown", method: http.MethodGet, path: "/v1/not-registered", wantCode: http.StatusNotFound, wantError: "not_found"},
		{name: "wrong method", method: http.MethodPost, path: "/v1/things/one", wantCode: http.StatusMethodNotAllowed, wantError: "method_not_allowed", wantAllow: "GET, PATCH"},
		{name: "HEAD explicitly rejected", method: http.MethodHead, path: "/v1/things/one", wantCode: http.StatusMethodNotAllowed, wantError: "method_not_allowed", wantAllow: "GET, PATCH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled = false
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req = req.WithContext(ctxkeys.WithRunID(req.Context(), "routing-test"))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.wantCode, recorder.Body.String())
			}
			if nextCalled != tt.wantNext {
				t.Fatalf("next called = %t, want %t", nextCalled, tt.wantNext)
			}
			if got := recorder.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("Allow = %q, want %q", got, tt.wantAllow)
			}
			if tt.wantError == "" {
				return
			}
			if got := recorder.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var body envelope.Error
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Error != tt.wantError || body.RequestID != "routing-test" {
				t.Fatalf("envelope = %#v", body)
			}
		})
	}
}

func TestV1RoutingRejectsNonCanonicalPathsWithoutRedirect(t *testing.T) {
	t.Parallel()

	catalogue, err := NewV1Catalogue([]RoutePolicy{
		{Method: http.MethodGet, PathPattern: "/v1/things/{id}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := catalogue.RoutingHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("non-canonical path reached handler")
	}))

	paths := []string{
		"//v1/things/one",
		"/../v1/things/one",
		"/%2e%2e/v1/things/one",
		"/v1//things/one",
		"/v1/./things/one",
		"/v1/things/../one",
		"/v1/things/%2e",
		"/v1/things%2fone",
		"/v1/things%5cone",
		"/v1/things/%00",
		"/v1/things/one/",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req = req.WithContext(ctxkeys.WithRunID(req.Context(), "canonical-test"))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusNotFound && recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400/404; body=%s", recorder.Code, recorder.Body.String())
			}
			if recorder.Header().Get("Location") != "" {
				t.Fatalf("Location = %q, want no redirect", recorder.Header().Get("Location"))
			}
			if !strings.Contains(recorder.Body.String(), `"requestId":"canonical-test"`) {
				t.Fatalf("body = %q, want canonical envelope", recorder.Body.String())
			}
		})
	}
}

func TestPolicyMuxValidatesHandlerPolicyBijection(t *testing.T) {
	t.Parallel()

	catalogue, err := NewV1Catalogue([]RoutePolicy{
		{Method: http.MethodGet, PathPattern: "/v1/one", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: http.MethodPost, PathPattern: "/v1/two", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := NewPolicyMux(http.NewServeMux(), catalogue)
	mux.Handle("GET /v1/one", http.NotFoundHandler())
	if err := mux.Validate(); err == nil || !strings.Contains(err.Error(), "POST /v1/two") {
		t.Fatalf("Validate() error = %v, want missing policy handler", err)
	}

	mux.Handle("POST /v1/not-in-policy", http.NotFoundHandler())
	if err := mux.Validate(); err == nil || !strings.Contains(err.Error(), "not-in-policy") {
		t.Fatalf("Validate() error = %v, want handler without policy", err)
	}
}

func TestPolicyMuxRejectsMethodNeutralV1HandlerOutsideCatalogue(t *testing.T) {
	t.Parallel()

	catalogue, err := NewV1Catalogue([]RoutePolicy{
		{Method: http.MethodGet, PathPattern: "/v1/one", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := NewPolicyMux(http.NewServeMux(), catalogue)
	mux.Handle("GET /v1/one", http.NotFoundHandler())
	mux.Handle("/v1/one", http.NotFoundHandler())

	if err := mux.Validate(); err == nil || !strings.Contains(err.Error(), "* /v1/one") {
		t.Fatalf("Validate() error = %v, want method-neutral handler rejection", err)
	}
}
