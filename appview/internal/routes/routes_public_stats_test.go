package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
)

type routePublicStatsReader struct {
	calls int
}

func (s *routePublicStatsReader) PublicStats(context.Context) (*api.PublicStats, error) {
	s.calls++
	return &api.PublicStats{Accounts: 3}, nil
}

func TestPublicStatsRouteIsAnonymousAndAdmitted(t *testing.T) {
	store := &routePublicStatsReader{}
	mux := http.NewServeMux()
	mw := v1Middleware{
		authCurrentMember:   func(http.Handler) http.Handler { t.Fatal("stats selected member authentication"); return nil },
		deviceID:            func(http.Handler) http.Handler { t.Fatal("stats selected device authentication"); return nil },
		accountTypeHydrator: api.NewIdentityAccountTypeHydrator(routeAccountTypeReader{}),
		folderRedactor:      api.SavedFolderRedactor(nil),
	}
	registerPublicStatsRoute(mux, mw, store)
	policy := mustPolicy("GET", "/v1/public/stats")
	if policy.AccessClass != AccessAnonymous {
		t.Fatal("statistics require authentication")
	}
	for i := 0; i < 5; i++ {
		request := httptest.NewRequest("GET", "/v1/public/stats", nil)
		request.Header.Set("Authorization", "Bearer ignored")
		request.Header.Set("X-Craftsky-Device-Id", "invented")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		want := 200
		if i == 4 {
			want = 429
			if response.Header().Get("Retry-After") == "" {
				t.Fatal("rate-limit response missing retry hint")
			}
		}
		if response.Code != want || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("request %d status=%d body=%s", i, response.Code, response.Body.String())
		}
	}
	if store.calls != 4 {
		t.Fatalf("stats store received %d calls, want admission limited to 4", store.calls)
	}
}

func TestPublicStatsCORSPreflight(t *testing.T) {
	catalogue, err := NewV1Catalogue(V1RoutePolicies(EnvProd, Config{Env: EnvProd}))
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware.CORS([]string{"https://stats.craftsky.social"}, catalogue)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("preflight reached stats query")
	}))
	request := httptest.NewRequest("OPTIONS", "/v1/public/stats", nil)
	request.Header.Set("Origin", "https://stats.craftsky.social")
	request.Header.Set("Access-Control-Request-Method", "GET")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 204 || response.Header().Get("Access-Control-Allow-Origin") != "https://stats.craftsky.social" || response.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
}
