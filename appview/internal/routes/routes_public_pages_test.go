package routes

import (
	"context"
	"golang.org/x/time/rate"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/api"
	"strings"
	"testing"
)

func TestPublicReportRoutesValidateWithoutAccount(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())
	for _, path := range []string{
		"/v1/public/posts/did:plc:alice/post1/reports",
		"/v1/public/profiles/did:plc:alice/reports",
	} {
		r := httptest.NewRequest("POST", path, strings.NewReader(`{"reasonType":"not-a-reason"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 422 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
	}
}

func TestPublicReportAdmissionCannotBeBypassedWithDeviceHeaders(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())
	for i := 0; i < 11; i++ {
		r := httptest.NewRequest("POST", "/v1/public/profiles/did:plc:alice/reports", strings.NewReader(`{"reasonType":"invalid"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Craftsky-Device-Id", string(rune('a'+i)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		want := 422
		if i == 10 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("attempt %d: %d %s", i, w.Code, w.Body.String())
		}
	}
}

func TestPublicReportBodyLimit(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())
	r := httptest.NewRequest("POST", "/v1/public/profiles/did:plc:alice/reports", strings.NewReader(`{"reasonType":"spam","details":"`+strings.Repeat("x", 8192)+`"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 413 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

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

// IT-008 / RULE-001 / AC-006: real guest registrations are read-only and
// validation precedes database work, even with spoofed credentials.
func TestPublicBrowsingRouteAllowlist(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())
	for _, path := range []string{
		"/v1/public/projects?limit=21",
		"/v1/public/search/hashtags/needle/posts?limit=0",
		"/v1/public/profiles/did:plc:alice/events?limit=0",
		"/v1/public/events/did:plc:alice/event?limit=0",
		"/v1/public/posts/did:plc:alice/root/likes?limit=0",
		"/v1/public/posts/did:plc:alice/root/reposts?limit=0",
		"/v1/public/posts/did:plc:alice/root/quotes?limit=0",
		"/v1/public/search/profiles?q=knit&sort=popular",
		"/v1/public/search/posts?q=knit&limit=21",
		"/v1/public/search/projects?q=knit&craftType=knitting",
		"/v1/public/search/hashtags?q=knit&viewerDid=did:plc:owner",
		"/v1/public/profiles/did:plc:alice/comments?limit=0",
		"/v1/public/profiles/did:plc:alice/reposts?limit=0",
		"/v1/public/posts/did:plc:alice/root/thread?limit=0",
	} {
		request := httptest.NewRequest("GET", path, nil)
		request.Header.Set("Authorization", "Bearer canary")
		request.Header.Set("Cookie", "session=canary")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != 400 || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
	for _, path := range []string{"/v1/public/profiles/recent", "/v1/public/search/suggestions", "/v1/public/search/hashtags/top", "/v1/public/search/recent", "/v1/public/profiles/me"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 404 {
			t.Fatalf("forbidden discovery/private path %s returned %d", path, response.Code)
		}
	}
}

// IT-009: cancellation never starts fresh anonymous dependency work and always
// returns the admission slot to the shared pool.
func TestPublicPageAdmissionCancellationReleasesCapacity(t *testing.T) {
	slots := make(chan struct{}, 1)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	h := publicPageAdmission(rate.NewLimiter(rate.Inf, 1), slots, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest("GET", "/v1/public/projects", nil).WithContext(canceled))
	if called || len(slots) != 0 {
		t.Fatal("canceled request entered public dependency work")
	}
	ctx, stop := context.WithCancel(context.Background())
	entered, done := make(chan struct{}), make(chan struct{})
	h = publicPageAdmission(rate.NewLimiter(rate.Inf, 1), slots, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }))
	go func() {
		defer close(done)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/public/projects", nil).WithContext(ctx))
	}()
	<-entered
	if len(slots) != 1 {
		t.Fatal("in-flight request did not own slot")
	}
	stop()
	<-done
	if len(slots) != 0 {
		t.Fatal("canceled in-flight request retained admission slot")
	}
}
