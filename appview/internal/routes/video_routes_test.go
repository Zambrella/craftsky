package routes

import (
	"context"
	"net/http"
	"testing"
)

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
