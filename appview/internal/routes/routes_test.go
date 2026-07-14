package routes

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

const routeModerationDDL = `
CREATE TABLE moderation_outputs (
    id                 TEXT PRIMARY KEY,
    source_did         TEXT NOT NULL,
    subject_type       TEXT NOT NULL,
    subject_did        TEXT NOT NULL,
    subject_collection TEXT,
    subject_rkey       TEXT,
    subject_uri        TEXT,
    value              TEXT NOT NULL,
    action             TEXT NOT NULL,
    internal_reason    TEXT,
    expires_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL,
    indexed_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// stubResolver is a minimal api.HandleResolver used by the routing
// tests so they don't depend on the real PLC directory.
type stubResolver struct{ handle syntax.Handle }

func (s stubResolver) ResolveHandle(_ context.Context, _ syntax.DID) (syntax.Handle, error) {
	return s.handle, nil
}
func (s stubResolver) ResolveDID(_ context.Context, _ syntax.Handle) (syntax.DID, error) {
	return "", nil
}

var _ api.HandleResolver = stubResolver{}

func testDeps() *app.Deps {
	return &app.Deps{
		Config:         app.Config{Env: app.EnvDev, AllowedOrigins: []string{"*"}, DevDID: "did:plc:test"},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService:    &auth.MockAuthService{DefaultDID: "did:plc:test"},
		HandleResolver: stubResolver{handle: syntax.Handle("stub-handle.example")},
	}
}

func TestV1RoutePoliciesCoverRegisteredRoutes(t *testing.T) {
	policies := V1RoutePolicies(app.EnvDev, app.Config{Env: app.EnvDev, EnableDevModeration: true, DevModerationToken: "secret"})
	if len(policies) == 0 {
		t.Fatal("V1RoutePolicies returned no policies")
	}
	seen := map[string]RoutePolicy{}
	for _, policy := range policies {
		if policy.Method == "" || policy.PathPattern == "" {
			t.Fatalf("policy has empty method/path: %+v", policy)
		}
		if policy.RateClass == "" {
			t.Fatalf("%s %s has empty rate class", policy.Method, policy.PathPattern)
		}
		if policy.BodyKind == "" {
			t.Fatalf("%s %s has empty body kind", policy.Method, policy.PathPattern)
		}
		if !policy.RateClass.Valid() {
			t.Fatalf("%s %s has invalid rate class %q", policy.Method, policy.PathPattern, policy.RateClass)
		}
		if !policy.BodyKind.Valid() {
			t.Fatalf("%s %s has invalid body kind %q", policy.Method, policy.PathPattern, policy.BodyKind)
		}
		seen[policy.Method+" "+policy.PathPattern] = policy
	}

	for _, want := range []struct {
		key       string
		rateClass RateClass
		bodyKind  BodyKind
	}{
		{"POST /v1/auth/login", RateClassAuth, BodyDefaultJSON},
		{"GET /v1/whoami", RateClassRead, BodyNoBody},
		{"GET /v1/search/posts", RateClassSearch, BodyNoBody},
		{"POST /v1/posts", RateClassWrite, BodyDefaultJSON},
		{"POST /v1/blobs/images", RateClassUpload, BodyUpload},
		{"GET /v1/notifications/new-count", RateClassRead, BodyNoBody},
		{"POST /v1/notifications/seen", RateClassWrite, BodyNoBody},
		{"GET /v1/dev/media/{name}", RateClassDevOnly, BodyNoBody},
		{"GET /v1/dev/panic", RateClassDevOnly, BodyNoBody},
		{"POST /v1/dev/moderation/ozone-events", RateClassDevOnly, BodyDefaultJSON},
	} {
		got, ok := seen[want.key]
		if !ok {
			t.Fatalf("missing policy for %s", want.key)
		}
		if got.RateClass != want.rateClass || got.BodyKind != want.bodyKind {
			t.Fatalf("%s policy = (%s, %s), want (%s, %s)", want.key, got.RateClass, got.BodyKind, want.rateClass, want.bodyKind)
		}
	}

	prodPolicies := V1RoutePolicies(app.EnvProd, app.Config{Env: app.EnvProd, EnableDevModeration: true, DevModerationToken: "secret"})
	for _, policy := range prodPolicies {
		if policy.DevOnly {
			t.Fatalf("prod policy includes dev-only route: %+v", policy)
		}
	}
}

func TestAddRoutes_V1WhoAmIAuthenticatedReturnsDIDAndHandle(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Dev-DID", "did:plc:from-header")
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		DID    string `json:"did"`
		Handle string `json:"handle"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body.DID != "did:plc:from-header" {
		t.Errorf("body.did = %q, want did:plc:from-header", body.DID)
	}
	if body.Handle != "stub-handle.example" {
		t.Errorf("body.handle = %q, want stub-handle.example", body.Handle)
	}
	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &topLevel); err != nil {
		t.Fatalf("body not valid JSON object: %v", err)
	}
	if _, ok := topLevel["data"]; ok {
		t.Fatalf("success response has synthetic data wrapper: %s", rec.Body.String())
	}
}

func TestAddRoutes_V1WhoAmIWithoutAuthReturns401(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_V1WhoAmIWithoutDeviceIDReturns400(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_MetricsEndpointIsRemoved(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 after /metrics removal; body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, "craftsky_appview") || strings.Contains(body, "# HELP") {
		t.Fatalf("/metrics returned metrics output after removal: %s", body)
	}
}

func TestAddRoutes_NoMetricsAuthBypassAndV1RoutesStillEnforceDevice(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	mux.ServeHTTP(metricsRec, metricsReq)
	if metricsRec.Code != http.StatusNotFound {
		t.Fatalf("/metrics status = %d, want 404 after removal; body=%s", metricsRec.Code, metricsRec.Body.String())
	}

	v1Req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	v1Req.Header.Set("Authorization", "Bearer anything")
	v1Rec := httptest.NewRecorder()
	mux.ServeHTTP(v1Rec, v1Req)
	if v1Rec.Code == http.StatusOK {
		t.Fatalf("/v1/whoami without auth/device status = 200, want an auth/device error")
	}
	if !strings.Contains(v1Rec.Body.String(), "missing_device_id") {
		t.Fatalf("/v1/whoami body = %q, want missing_device_id", v1Rec.Body.String())
	}
}

func TestAddRoutes_BodyPolicyRunsThroughMux(t *testing.T) {
	t.Run("default JSON route rejects oversized body before auth", func(t *testing.T) {
		deps := testDeps()
		deps.Config.JSONBodyLimitBytes = 8
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)

		req := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader("123456789"))
		req.Header.Set("Authorization", "Bearer anything")
		req.Header.Set("X-Craftsky-Device-Id", "dev-test")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413; body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "request_body_too_large") {
			t.Fatalf("body = %q, want request_body_too_large", rec.Body.String())
		}
	})

	t.Run("no-body route rejects unexpected body before auth", func(t *testing.T) {
		deps := testDeps()
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)

		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer anything")
		req.Header.Set("X-Craftsky-Device-Id", "dev-test")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "request_body_not_allowed") {
			t.Fatalf("body = %q, want request_body_not_allowed", rec.Body.String())
		}
	})
}

func TestAddRoutes_RateLimitRejectsBeforeHandlerWork(t *testing.T) {
	deps := testDeps()
	deps.RateLimiter = middleware.NewLocalRateLimiter(middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
		middleware.RateClassRead: {Window: time.Minute, PerDevice: 1},
	}}, func() time.Time { return time.Unix(100, 0) })
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		req.Header.Set("Authorization", "Bearer anything")
		req.Header.Set("X-Dev-DID", "did:plc:from-header")
		req.Header.Set("X-Craftsky-Device-Id", "dev-test")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if i == 0 && rec.Code != http.StatusOK {
			t.Fatalf("first status = %d, want 200; body=%s", rec.Code, rec.Body.String())
		}
		if i == 1 {
			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("second status = %d, want 429; body=%s", rec.Code, rec.Body.String())
			}
			if rec.Header().Get("Retry-After") == "" {
				t.Fatal("Retry-After header is empty")
			}
			if rec.Header().Get("X-RateLimit-Limit") != "" || rec.Header().Get("X-RateLimit-Remaining") != "" {
				t.Fatalf("unexpected public rate-limit headers: %v", rec.Header())
			}
			if !strings.Contains(rec.Body.String(), "rate_limited") {
				t.Fatalf("body = %q, want rate_limited", rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "stub-handle.example") {
				t.Fatalf("rate-limited response appears to include handler output: %q", rec.Body.String())
			}
		}
	}
}

func TestAddRoutes_AllV1PoliciesEnforcedThroughMux(t *testing.T) {
	for _, policy := range V1RoutePolicies(app.EnvDev, app.Config{Env: app.EnvDev, EnableDevModeration: true, DevModerationToken: "secret-token"}) {
		if policy.RateClass == RateClassDevOnly {
			continue
		}
		if policy.BodyKind == BodyNoBody {
			t.Run(policy.Method+" "+policy.PathPattern+" rejects unexpected body", func(t *testing.T) {
				deps := testDeps()
				mux := http.NewServeMux()
				AddRoutes(context.Background(), mux, deps)

				req := httptest.NewRequest(policy.Method, samplePath(policy.PathPattern), strings.NewReader("unexpected"))
				req.Header.Set("Authorization", "Bearer anything")
				req.Header.Set("X-Dev-DID", "did:plc:from-header")
				req.Header.Set("X-Craftsky-Device-Id", "dev-test")
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, req)

				if rec.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
				}
				if !strings.Contains(rec.Body.String(), "request_body_not_allowed") {
					t.Fatalf("body = %q, want request_body_not_allowed", rec.Body.String())
				}
			})
		}

		if policy.RateClass != RateClassExempt {
			t.Run(policy.Method+" "+policy.PathPattern+" applies rate class", func(t *testing.T) {
				mux := muxWithPolicyProbe(policy, routeTestRateLimitConfig(policy.RateClass))

				for i := 0; i < 2; i++ {
					req := httptest.NewRequest(policy.Method, samplePath(policy.PathPattern), nil)
					req.Header.Set("Authorization", "Bearer anything")
					req.Header.Set("X-Dev-DID", "did:plc:from-header")
					req.Header.Set("X-Craftsky-Device-Id", "dev-test")
					rec := httptest.NewRecorder()
					mux.ServeHTTP(rec, req)
					if i == 1 {
						if rec.Code != http.StatusTooManyRequests {
							t.Fatalf("second status = %d, want 429; body=%s", rec.Code, rec.Body.String())
						}
						if !strings.Contains(rec.Body.String(), "rate_limited") {
							t.Fatalf("body = %q, want rate_limited", rec.Body.String())
						}
					}
				}
			})
		}
	}
}

func muxWithPolicyProbe(policy RoutePolicy, cfg middleware.RateLimitConfig) *http.ServeMux {
	mux := http.NewServeMux()
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	limiter := middleware.NewLocalRateLimiter(cfg, func() time.Time { return time.Unix(200, 0) })
	if policy.RateClass != RateClassExempt && policy.RateClass != RateClassDevOnly {
		handler = middleware.RateLimit(limiter, middleware.RateClass(policy.RateClass), nil)(handler)
	}
	handler = middleware.DeviceID(nil, nil)(handler)
	handler = middleware.Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:test"}, nil)(handler)
	handler = middleware.BodyLimit(middleware.BodyLimitConfig{DefaultJSONBytes: 1, UploadBytes: 1}, middleware.BodyKind(policy.BodyKind), nil)(handler)
	mux.Handle(policy.Method+" "+policy.PathPattern, handler)
	return mux
}

func routeTestRateLimitConfig(class RateClass) middleware.RateLimitConfig {
	mwClass := middleware.RateClass(class)
	return middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
		mwClass: {Window: time.Minute, PerDevice: 1},
	}}
}

func samplePath(pattern string) string {
	replacer := strings.NewReplacer(
		"{handleOrDid}", "@alice.example",
		"{did}", "did:plc:alice",
		"{rkey}", "post1",
		"{tag}", "sock",
		"{id}", "recent_123",
	)
	path := replacer.Replace(pattern)
	switch pattern {
	case "/v1/facets/mentions", "/v1/facets/hashtags", "/v1/search/suggestions", "/v1/search/hashtags", "/v1/search/profiles", "/v1/search/posts":
		return path + "?q=sock"
	case "/v1/facets/mentions/resolve":
		return path + "?handle=alice.example"
	}
	return path
}

func TestAddRoutes_PostRepliesRequiresAuthenticatedDevice(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/posts/did:plc:alice/root/replies", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_TimelineRequiresAuthenticatedDevice(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/feed/timeline", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_TimelineRequiresDeviceID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/feed/timeline", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_NotificationsRequiresAuthenticatedDevice(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/notifications", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_NotificationsRequiresDeviceID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/notifications", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_PostRepliesRequiresDeviceID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/posts/did:plc:alice/root/replies", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_PostCommentsRequiresAuthenticatedDevice(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/posts/did:plc:alice/root/comments", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_PostCommentsRequiresDeviceID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/posts/did:plc:alice/root/comments", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_ProfileCommentsRequiresAuthenticatedDevice(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/profiles/@alice.example/comments", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_ProfileCommentsRequiresDeviceID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/profiles/@alice.example/comments", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_ProfileProjectsRequiresAuthenticatedDevice(t *testing.T) {
	for _, tc := range []struct {
		name       string
		headers    map[string]string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "requires auth",
			headers:    map[string]string{"X-Craftsky-Device-Id": "dev-test"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "requires device",
			headers:    map[string]string{"Authorization": "Bearer anything"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "missing_device_id",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())

			req := httptest.NewRequest("GET", "/v1/profiles/@alice.example/projects", nil)
			for name, value := range tc.headers {
				req.Header.Set(name, value)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantBody != "" && !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body = %q, want containing %q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestAddRoutes_V1LoginWithoutDeviceIDReturns400(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("POST", "/v1/auth/login", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_LegacyUnprefixedWhoAmIReturns404(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (legacy path should be gone)", rec.Code)
	}
}

func TestAddRoutes_HealthStaysUnprefixed(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/health", nil)
	_, pattern := mux.Handler(req)
	if pattern == "/" || pattern == "" {
		t.Errorf("pattern = %q; /health must be registered at a top-level path", pattern)
	}
}

func TestAddRoutes_OAuthClientMetadataStaysUnprefixed(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/oauth/client-metadata.json", nil)
	_, pattern := mux.Handler(req)
	if pattern == "/" || pattern == "" {
		t.Errorf("pattern = %q; /oauth/client-metadata.json must be registered", pattern)
	}
}

func TestAddRoutes_UnknownPathReturns404(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestRoutes_GetProfileByHandleOrDIDRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@alice.example", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("GET /v1/profiles/@{handleOrDid} without auth: status = %d, want 401", rr.Code)
	}
}

func TestRoutes_GetProfileMeRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/me", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("GET /v1/profiles/me without auth: status = %d, want 401", rr.Code)
	}
}

func TestRoutes_ReportEndpointsRequireAuthenticatedDevice(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "post report", path: "/v1/posts/did:plc:bob/3lf2abc/reports"},
		{name: "profile report", path: "/v1/profiles/bob.craftsky.social/reports"},
	} {
		t.Run(tc.name+" requires auth", func(t *testing.T) {
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())

			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{"reasonType":"spam"}`))
			req.Header.Set("X-Craftsky-Device-Id", "dev-test")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401; body = %s", rr.Code, rr.Body.String())
			}
		})

		t.Run(tc.name+" requires device", func(t *testing.T) {
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())

			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{"reasonType":"spam"}`))
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "missing_device_id") {
				t.Fatalf("body = %q, want missing_device_id", rr.Body.String())
			}
		})
	}
}

func TestRoutes_DevModerationRouteUnavailableUnlessEnabled(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  app.Config
	}{
		{name: "prod with flag", cfg: app.Config{Env: app.EnvProd, EnableDevModeration: true, DevModerationToken: "secret"}},
		{name: "dev flag off", cfg: app.Config{Env: app.EnvDev, EnableDevModeration: false, DevModerationToken: "secret"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps := testDeps()
			deps.Config = tc.cfg
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, deps)

			req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{"subject":{"type":"post","did":"did:plc:bob","rkey":"rk"},"value":"hide","action":"apply"}`))
			req.Header.Set("X-Craftsky-Dev-Moderation-Token", "secret")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRoutes_DevPanicRouteIsDevOnly(t *testing.T) {
	t.Run("dev route is registered", func(t *testing.T) {
		deps := testDeps()
		deps.Config = app.Config{Env: app.EnvDev}
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)

		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("GET /v1/dev/panic did not panic")
			}
			if recovered != "synthetic appview dev panic" {
				t.Fatalf("panic = %#v, want synthetic appview dev panic", recovered)
			}
		}()
		mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/dev/panic", nil))
	})

	t.Run("prod route is not registered", func(t *testing.T) {
		deps := testDeps()
		deps.Config = app.Config{Env: app.EnvProd}
		mux := http.NewServeMux()
		AddRoutes(context.Background(), mux, deps)

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/dev/panic", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestRoutes_DevModerationRouteRequiresToken(t *testing.T) {
	deps := testDeps()
	deps.Config = app.Config{Env: app.EnvDev, EnableDevModeration: true, DevModerationToken: "secret-token"}
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "missing"},
		{name: "invalid", token: "wrong"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{"subject":{"type":"post","did":"did:plc:bob","rkey":"rk"},"value":"hide","action":"apply"}`))
			if tc.token != "" {
				req.Header.Set("X-Craftsky-Dev-Moderation-Token", tc.token)
			}
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403; body = %s", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "invalid_dev_moderation_token") {
				t.Fatalf("body = %q, want invalid_dev_moderation_token", rr.Body.String())
			}
		})
	}
}

func TestRoutes_DevModerationRoutePersistsValidOutput(t *testing.T) {
	pool := testdb.WithSchema(t, routeModerationDDL)
	deps := testDeps()
	deps.DB = pool
	deps.ModerationStore = api.NewModerationStore(pool)
	deps.Config = app.Config{
		Env:                         app.EnvDev,
		EnableDevModeration:         true,
		DevModerationToken:          "secret-token",
		DevLabelerDID:               "did:plc:labeler",
		TrustedModerationSourceDIDs: []string{"did:plc:labeler"},
	}
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{
		"subject":{"type":"post","did":"did:plc:bob","rkey":"3lf2abc"},
		"value":"hide",
		"action":"apply",
		"internalReason":"private fixture"
	}`))
	req.Header.Set("X-Craftsky-Dev-Moderation-Token", "secret-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var body struct {
		OutputID string `json:"outputId"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body.OutputID == "" || body.Status != "indexed" {
		t.Fatalf("body = %+v, want outputId and indexed status", body)
	}

	var count int
	var subjectURI, internalReason string
	if err := pool.QueryRow(context.Background(), `SELECT count(*)::int, max(subject_uri), max(internal_reason) FROM moderation_outputs`).Scan(&count, &subjectURI, &internalReason); err != nil {
		t.Fatalf("query persisted output: %v", err)
	}
	if count != 1 || subjectURI != "at://did:plc:bob/social.craftsky.feed.post/3lf2abc" || internalReason != "private fixture" {
		t.Fatalf("persisted count=%d subjectURI=%q internalReason=%q", count, subjectURI, internalReason)
	}
}

func TestRoutes_DevModerationRouteRejectsInvalidWithoutMutation(t *testing.T) {
	pool := testdb.WithSchema(t, routeModerationDDL)
	deps := testDeps()
	deps.DB = pool
	deps.ModerationStore = api.NewModerationStore(pool)
	deps.Config = app.Config{
		Env:                         app.EnvDev,
		EnableDevModeration:         true,
		DevModerationToken:          "secret-token",
		DevLabelerDID:               "did:plc:labeler",
		TrustedModerationSourceDIDs: []string{"did:plc:labeler"},
	}
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{
		"sourceDid":"did:plc:untrusted",
		"subject":{"type":"account","did":"did:plc:bob"},
		"value":"warn",
		"action":"apply"
	}`))
	req.Header.Set("X-Craftsky-Dev-Moderation-Token", "secret-token")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "untrusted_moderation_source") {
		t.Fatalf("body = %q, want untrusted_moderation_source", rr.Body.String())
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*)::int FROM moderation_outputs`).Scan(&count); err != nil {
		t.Fatalf("count moderation outputs: %v", err)
	}
	if count != 0 {
		t.Fatalf("stored outputs = %d, want 0", count)
	}
}

func TestRoutes_ProfileSocialGraphEndpointsRequireAuthenticatedDevice(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"/v1/profiles/@alice.example/mutual-followers",
		"/v1/profiles/me/followers",
		"/v1/profiles/me/following",
	} {
		t.Run(path+" requires auth", func(t *testing.T) {
			t.Parallel()
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("X-Craftsky-Device-Id", "dev-test")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rr.Code)
			}
		})
		t.Run(path+" requires device", func(t *testing.T) {
			t.Parallel()
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer anything")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rr.Code)
			}
			if !strings.Contains(rr.Body.String(), "missing_device_id") {
				t.Errorf("body = %q, want containing 'missing_device_id'", rr.Body.String())
			}
		})
	}
}

func TestRoutes_PutProfileMeRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PUT /v1/profiles/me without auth: status = %d, want 401", rr.Code)
	}
}

func TestRoutes_PostPostLikesRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("POST /v1/posts/{did}/{rkey}/likes without auth: status = %d, want 401", rr.Code)
	}
}

func TestRoutes_DeletePostLikesRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("DELETE /v1/posts/{did}/{rkey}/likes without auth: status = %d, want 401", rr.Code)
	}
}

func TestAddRoutes_ImageBlobUploadRouteRegistered(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest(http.MethodPost, "/v1/blobs/images", nil)
	_, pattern := mux.Handler(req)
	if pattern == "/" || pattern == "" {
		t.Fatalf("pattern = %q; /v1/blobs/images must be registered", pattern)
	}
}

func TestAddRoutes_FacetRoutesRegisteredAndRequireAuthenticatedDevice(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "mention suggestions", path: "/v1/facets/mentions?q=ali"},
		{name: "mention resolve", path: "/v1/facets/mentions/resolve?handle=alice.craftsky.social"},
		{name: "hashtag suggestions", path: "/v1/facets/hashtags?q=sock"},
	} {
		t.Run(tc.name+" registered", func(t *testing.T) {
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			_, pattern := mux.Handler(req)
			if pattern == "/" || pattern == "" {
				t.Fatalf("pattern = %q; %s must be registered", pattern, tc.path)
			}
		})

		t.Run(tc.name+" requires auth", func(t *testing.T) {
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("X-Craftsky-Device-Id", "dev-test")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rr.Code)
			}
		})

		t.Run(tc.name+" requires device", func(t *testing.T) {
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer anything")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rr.Code)
			}
			if !strings.Contains(rr.Body.String(), "missing_device_id") {
				t.Errorf("body = %q, want containing 'missing_device_id'", rr.Body.String())
			}
		})
	}
}

func TestAddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "project list", method: http.MethodGet, path: "/v1/projects?craftType=knitting"},
		{name: "hashtag posts", method: http.MethodGet, path: "/v1/search/hashtags/sock/posts"},
		{name: "search suggestions", method: http.MethodGet, path: "/v1/search/suggestions?q=sock"},
		{name: "hashtag search", method: http.MethodGet, path: "/v1/search/hashtags?q=sock"},
		{name: "profile search", method: http.MethodGet, path: "/v1/search/profiles?q=ali"},
		{name: "post search", method: http.MethodGet, path: "/v1/search/posts?q=sock"},
		{name: "project search", method: http.MethodGet, path: "/v1/search/projects"},
		{name: "top hashtags", method: http.MethodGet, path: "/v1/search/hashtags/top?craftTypes=knitting"},
		{name: "list recents", method: http.MethodGet, path: "/v1/search/recent"},
		{name: "save recent", method: http.MethodPost, path: "/v1/search/recent"},
		{name: "delete recent", method: http.MethodDelete, path: "/v1/search/recent/recent_123"},
	} {
		t.Run(tc.name+" registered", func(t *testing.T) {
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())
			req := httptest.NewRequest(tc.method, tc.path, nil)
			_, pattern := mux.Handler(req)
			if pattern == "/" || pattern == "" {
				t.Fatalf("pattern = %q; %s %s must be registered", pattern, tc.method, tc.path)
			}
		})

		t.Run(tc.name+" requires auth", func(t *testing.T) {
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("X-Craftsky-Device-Id", "dev-test")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401; body = %s", rr.Code, rr.Body.String())
			}
			assertErrorEnvelope(t, rr.Body.Bytes())
		})

		t.Run(tc.name+" requires device", func(t *testing.T) {
			mux := http.NewServeMux()
			AddRoutes(context.Background(), mux, testDeps())
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Authorization", "Bearer anything")
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
			}
			assertErrorEnvelope(t, rr.Body.Bytes())
			if !strings.Contains(rr.Body.String(), "missing_device_id") {
				t.Errorf("body = %q, want containing 'missing_device_id'", rr.Body.String())
			}
		})
	}
}

func TestSearchProjectsRouteRejectsBrowseFilters(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())
	req := httptest.NewRequest(http.MethodGet, "/v1/search/projects?q=sock&craftType=knitting&material=alpaca", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s, want 400", rr.Code, rr.Body.String())
	}
	assertErrorEnvelope(t, rr.Body.Bytes())
	if !strings.Contains(rr.Body.String(), "validation_error") {
		t.Fatalf("body = %s, want validation_error", rr.Body.String())
	}
}

func assertErrorEnvelope(t *testing.T, body []byte) {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("body not valid JSON: %v; body=%s", err, string(body))
	}
	for _, key := range []string{"error", "message", "requestId"} {
		if _, ok := env[key]; !ok {
			t.Fatalf("error envelope missing %s: %v", key, env)
		}
	}
}

func TestAddRoutes_ImageBlobUploadRequiresAuth(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest(http.MethodPost, "/v1/blobs/images", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestAddRoutes_ImageBlobUploadRequiresDeviceID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest(http.MethodPost, "/v1/blobs/images", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rr.Body.String())
	}
}

func TestRoutes_PostPostRepostsRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("POST /v1/posts/{did}/{rkey}/reposts without auth: status = %d, want 401", rr.Code)
	}
}

func TestRoutes_DeletePostRepostsRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("DELETE /v1/posts/{did}/{rkey}/reposts without auth: status = %d, want 401", rr.Code)
	}
}

func TestRoutes_PostPostRepostsRequiresDeviceID(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/posts/{did}/{rkey}/reposts without device: status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rr.Body.String())
	}
}

func TestRoutes_DeletePostRepostsRequiresDeviceID(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("DELETE /v1/posts/{did}/{rkey}/reposts without device: status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rr.Body.String())
	}
}

func TestRoutes_PostPostsRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"hi"}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("POST /v1/posts without auth: status = %d, want 401", rr.Code)
	}
}

func TestRoutes_PostProfileFollowRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles/@bob.example/follows", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("POST /v1/profiles/@{handleOrDid}/follows without auth: status = %d, want 401", rr.Code)
	}
}

func TestRoutes_PostProfileFollowRequiresDeviceID(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles/@bob.example/follows", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/profiles/@{handleOrDid}/follows without device: status = %d, want 400", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body["error"] != "missing_device_id" {
		t.Errorf("error = %v, want missing_device_id", body["error"])
	}
	if _, ok := body["message"]; !ok {
		t.Error("missing message field")
	}
	if _, ok := body["requestId"]; !ok {
		t.Error("missing requestId field")
	}
}

func TestRoutes_DeleteProfileFollowRequiresAuth(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodDelete, "/v1/profiles/@bob.example/follows", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("DELETE /v1/profiles/@{handleOrDid}/follows without auth: status = %d, want 401", rr.Code)
	}
}

func TestRoutes_DeleteProfileFollowRequiresDeviceID(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	req := httptest.NewRequest(http.MethodDelete, "/v1/profiles/@bob.example/follows", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("DELETE /v1/profiles/@{handleOrDid}/follows without device: status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rr.Body.String())
	}
}
