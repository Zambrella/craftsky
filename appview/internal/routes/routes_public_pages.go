package routes

import (
	"net/http"
	"time"

	"golang.org/x/time/rate"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

func registerPublicPageRoutes(mux Registrar, middleware v1Middleware, store api.PublicBrowsingReader, resolver api.HandleResolver, captions http.Handler) {
	// Anonymous callers have no trusted device/session key. Bound the aggregate
	// work across these routes independently of optional client headers.
	limiter := rate.NewLimiter(20, 40)
	slots := make(chan struct{}, 8)
	for _, route := range []struct{ path, resource string }{
		{"/v1/public/profiles/{handleOrDid}", "profile"},
		{"/v1/public/profiles/{handleOrDid}/posts", "posts"},
		{"/v1/public/profiles/{handleOrDid}/projects", "projects"},
		{"/v1/public/posts/{did}/{rkey}", "post"},
	} {
		mux.Handle("GET "+route.path, middleware.wrap(mustPolicy("GET", route.path), publicPageAdmission(limiter, slots, api.PublicPagesHandler(store, resolver, route.resource))))
	}
	for _, route := range []struct{ path, resource string }{
		{"/v1/public/projects", "projectFeed"},
		{"/v1/public/search/hashtags/{tag}/posts", "tagPosts"},
		{"/v1/public/profiles/{handleOrDid}/events", "events"},
		{"/v1/public/events/{did}/{rkey}", "event"},
		{"/v1/public/posts/{did}/{rkey}/likes", "interactions/likes"},
		{"/v1/public/posts/{did}/{rkey}/reposts", "interactions/reposts"},
		{"/v1/public/posts/{did}/{rkey}/quotes", "interactions/quotes"},
		{"/v1/public/search/profiles", "search/profiles"},
		{"/v1/public/search/posts", "search/posts"},
		{"/v1/public/search/projects", "search/projects"},
		{"/v1/public/search/hashtags", "search/hashtags"},
		{"/v1/public/profiles/{handleOrDid}/comments", "comments"},
		{"/v1/public/profiles/{handleOrDid}/reposts", "reposts"},
		{"/v1/public/posts/{did}/{rkey}/thread", "thread"},
	} {
		mux.Handle("GET "+route.path, middleware.wrap(mustPolicy("GET", route.path), publicPageAdmission(limiter, slots, api.PublicBrowsingHandler(store, resolver, route.resource))))
	}
	path := "/v1/public/posts/{did}/{rkey}/video-captions/{captionCid}"
	mux.Handle("GET "+path, middleware.wrap(mustPolicy("GET", path), publicPageAdmission(limiter, slots, captions)))
}

func registerPublicReportRoutes(mux Registrar, middleware v1Middleware, store api.PublicPageReader, resolver api.HandleResolver, reports api.ReportCreator) {
	// Shared process-wide admission cannot be bypassed by rotating caller-supplied
	// device IDs, cookies or forwarded headers. Reports have a separate budget.
	limiter := rate.NewLimiter(rate.Every(6*time.Second), 10)
	slots := make(chan struct{}, 2)
	for _, route := range []struct{ path, resource string }{
		{"/v1/public/posts/{did}/{rkey}/reports", "post"},
		{"/v1/public/profiles/{handleOrDid}/reports", "profile"},
	} {
		mux.Handle("POST "+route.path, middleware.wrap(mustPolicy("POST", route.path), publicPageAdmission(limiter, slots, api.PublicReportHandler(store, resolver, reports, route.resource))))
	}
}

func publicPageAdmission(limiter *rate.Limiter, slots chan struct{}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		reject := func(status int, code string) {
			w.Header().Set("Retry-After", "1")
			envelope.WriteError(w, status, code, "page temporarily unavailable", middleware.GetRunID(r.Context()), nil)
		}
		if r.Context().Err() != nil {
			reject(503, "unavailable")
			return
		}
		if !limiter.Allow() {
			reject(429, "rate_limited")
			return
		}
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			reject(503, "unavailable")
			return
		}
		next.ServeHTTP(w, r)
	})
}
