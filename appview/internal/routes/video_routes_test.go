package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	appmiddleware "social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

func TestVideoAuthorizationRouteUsesExactCurrentMemberPolicy(t *testing.T) {
	t.Parallel()
	policy := mustPolicy(http.MethodPost, "/v1/blobs/videos/authorization")
	if policy.RateClass != RateClassUpload || policy.BodyKind != BodyNoBody || policy.AccessClass != AccessCurrentMember {
		t.Fatalf("policy = %+v", policy)
	}
}

func TestAddRoutesRegistersVideoAuthorization(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)
	request, err := http.NewRequest(http.MethodPost, "/v1/blobs/videos/authorization", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, pattern := mux.Handler(request); pattern != "POST /v1/blobs/videos/authorization" {
		t.Fatalf("registered pattern = %q", pattern)
	}
}

func TestVideoLimitsRouteUsesExactCurrentMemberPolicy(t *testing.T) {
	t.Parallel()
	policy := mustPolicy(http.MethodGet, "/v1/blobs/videos/limits")
	if policy.RateClass != RateClassRead || policy.BodyKind != BodyNoBody || policy.AccessClass != AccessCurrentMember {
		t.Fatalf("policy = %+v", policy)
	}
}

func TestAddRoutesRegistersVideoLimits(t *testing.T) {
	t.Parallel()
	deps := testDeps()
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)
	request, err := http.NewRequest(http.MethodGet, "/v1/blobs/videos/limits", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, pattern := mux.Handler(request); pattern != "GET /v1/blobs/videos/limits" {
		t.Fatalf("registered pattern = %q", pattern)
	}
}

func TestVideoCaptionRouteUsesExactCurrentMemberPolicy(t *testing.T) {
	t.Parallel()
	policy := mustPolicy(http.MethodGet, "/v1/posts/{did}/{rkey}/video-captions/{captionCid}")
	if policy.RateClass != RateClassRead || policy.BodyKind != BodyNoBody || policy.AccessClass != AccessCurrentMember {
		t.Fatalf("policy = %+v", policy)
	}
}

func TestVideoRouteCatalogueExcludesProhibitedProxies(t *testing.T) {
	t.Parallel()
	forbidden := map[string]bool{
		"POST /v1/blobs/videos":             true,
		"GET /v1/blobs/videos/jobs/{jobId}": true,
		"GET /v1/blobs/{cid}":               true,
	}
	for _, policy := range baseV1RoutePolicies() {
		key := policy.Method + " " + policy.PathPattern
		if forbidden[key] {
			t.Fatalf("prohibited route policy exists: %s", key)
		}
	}
}

func TestDisabledVideoRoutesNeverCallLaunchServices(t *testing.T) {
	identity := func(next http.Handler) http.Handler { return next }
	authenticated := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(appmiddleware.WithDID(r.Context(), syntax.DID("did:plc:test"))))
		})
	}
	mux := http.NewServeMux()
	registerVideoRoutes(videoRouteBundle{
		mux: mux,
		middleware: v1Middleware{
			authCurrentMember: authenticated,
			deviceID:          identity,
			member:            identity,
			rateLimit:         map[RateClass]func(http.Handler) http.Handler{},
			observer:          observability.New(observability.Config{Env: "test"}),
			suspension:        unsuspendedReader{},
			eligibility:       unrestrictedEligibilityReader{},
		},
		enabled: false,
	})

	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/v1/blobs/videos/authorization", nil),
		httptest.NewRequest(http.MethodGet, "/v1/blobs/videos/limits", nil),
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s status/cache = %d/%q, body=%s", request.Method, request.URL.Path, response.Code, response.Header().Get("Cache-Control"), response.Body.String())
		}
	}
}
