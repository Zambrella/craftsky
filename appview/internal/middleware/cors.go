package middleware

import (
	"net/http"
)

// CORS returns middleware that handles CORS for the given allow-list.
//
// The allow-list is an explicit list of exact origins. The special value
// "*" matches any origin (used in dev); when wildcarded, the request's
// Origin header is echoed back rather than sending a literal "*" so that
// credentialed requests still work.
//
// Preflight (OPTIONS) requests short-circuit with 200 after the headers
// are set. Non-preflight requests pass through to next with the
// Access-Control-Allow-Origin header set iff the origin is allowed.
//
// Day one: only exact-string match and the "*" wildcard. No subdomain
// patterns, no regex — add them to the spec and this function together
// when a concrete case appears.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if isOriginAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			// Vary: Origin tells CDNs/proxies that the response depends on the
			// request's Origin header, so they don't serve a cached ACAO for
			// origin A to a request from origin B.
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Dev-DID")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isOriginAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if a == "*" {
			return true
		}
		if a == origin {
			return true
		}
	}
	return false
}
