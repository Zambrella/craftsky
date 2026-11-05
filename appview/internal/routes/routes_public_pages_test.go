package routes

import (
	"golang.org/x/time/rate"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/api"
	"testing"
)

func TestPublicPageAdmissionBoundsAnonymousWork(t *testing.T) {
	for _, saturated := range []bool{false, true} {
		t.Run(map[bool]string{false: "rate", true: "concurrency"}[saturated], func(t *testing.T) {
			slots := make(chan struct{}, 1)
			limiter := rate.NewLimiter(0, 1)
			if saturated {
				slots <- struct{}{}
			} else {
				limiter.Allow()
			}
			called := false
			handler := publicPageAdmission(limiter, slots, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
			request := httptest.NewRequest("GET", "/v1/public/profiles/did:plc:alice", nil)
			request.Header.Set("X-Craftsky-Device-Id", "invented")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			want := 429
			if saturated {
				want = 503
			}
			if response.Code != want || called || response.Header().Get("Retry-After") == "" || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("status=%d called=%v", response.Code, called)
			}
		})
	}
}

func TestPublicPageResponsesBypassMemberDecorators(t *testing.T) {
	mw := v1Middleware{accountTypeHydrator: api.NewIdentityAccountTypeHydrator(routeAccountTypeReader{}), folderRedactor: api.SavedFolderRedactor(nil)}
	const payload = `{"did":"did:plc:alice","project":{"viewerSavedFolderId":"public extension field"}}`
	handler := mw.wrap(mustPolicy("GET", "/v1/public/profiles/{handleOrDid}"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/v1/public/profiles/did:plc:alice", nil))
	if response.Code != 200 || response.Body.String() != payload {
		t.Fatalf("public response expanded or required viewer: %d %s", response.Code, response.Body.String())
	}
}
