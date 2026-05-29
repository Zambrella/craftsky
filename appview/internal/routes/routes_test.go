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

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
)

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
