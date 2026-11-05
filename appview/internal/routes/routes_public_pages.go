package routes

import (
	"net/http"

	"golang.org/x/time/rate"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

func registerPublicPageRoutes(mux Registrar, middleware v1Middleware, store api.PublicPageReader, resolver api.HandleResolver) {
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
}

func publicPageAdmission(limiter *rate.Limiter, slots chan struct{}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		reject := func(status int, code string) {
			w.Header().Set("Retry-After", "1")
			envelope.WriteError(w, status, code, "page temporarily unavailable", middleware.GetRunID(r.Context()), nil)
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
