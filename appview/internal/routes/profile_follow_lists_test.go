package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProfileFollowListsRequireMembership(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())
	for _, kind := range []string{"followers", "following"} {
		policy := mustPolicy("GET", "/v1/profiles/{handleOrDid}/"+kind)
		if policy.AccessClass != AccessCurrentMember {
			t.Fatal("list is not member-only")
		}
		req := httptest.NewRequest("GET", "/v1/profiles/@did:plc:target/"+kind, nil)
		req.Header.Set("X-Craftsky-Device-Id", "device")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != 401 {
			t.Fatalf("anonymous %s: %d", kind, response.Code)
		}
	}
}
