package routes

import (
	"golang.org/x/time/rate"
	"social.craftsky/appview/internal/api"
)

func registerPublicStatsRoute(mux Registrar, middleware v1Middleware, store api.PublicStatsReader) {
	// Each request aggregates the database without caching. Keep its admission
	// budget separate from profile/post reads, independent of untrusted headers.
	handler := publicPageAdmission(rate.NewLimiter(2, 4), make(chan struct{}, 2), api.PublicStatsHandler(store))
	const path = "/v1/public/stats"
	mux.Handle("GET "+path, middleware.wrap(mustPolicy("GET", path), handler))
}
