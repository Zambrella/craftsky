package middleware

import (
	"net/http"
	"sort"
	"strings"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

type CORSMethodCatalogue interface {
	AllowedMethods(path string) ([]string, bool)
}

var corsAllowedRequestHeaders = []string{
	"Authorization",
	"Content-Type",
	"X-Craftsky-Device-Id",
	"X-Dev-DID",
	"X-Requested-With",
}

// CORS returns middleware that handles CORS for the given allow-list.
//
// The allow-list is an explicit list of exact origins. The special value
// "*" matches any origin (used in dev); when wildcarded, the request's
// Origin header is echoed back rather than sending a literal "*". V1 uses
// bearer Authorization headers and does not enable cookie credential CORS.
//
// Preflight (OPTIONS) requests short-circuit with 204 after the headers
// are set. Non-preflight requests pass through to next with the
// Access-Control-Allow-Origin header set iff the origin is allowed.
//
// Day one: only exact-string match and the "*" wildcard. No subdomain
// patterns, no regex — add them to the spec and this function together
// when a concrete case appears.
func CORS(allowedOrigins []string, catalogue CORSMethodCatalogue) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			originAllowed := isOriginAllowed(origin, allowedOrigins)
			if originAllowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Add("Vary", "Origin")
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")

			if r.Method != http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			// Preflight is terminal at this boundary. Do not let net/http drain an
			// attacker-controlled body merely to preserve connection reuse.
			RejectBodyWithoutDrain(w, r)

			if catalogue == nil {
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			methods, known := catalogue.AllowedMethods(r.URL.Path)
			if !known {
				envelope.WriteError(w, http.StatusNotFound, "not_found", "route not found", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			sort.Strings(methods)
			allow := append(append([]string(nil), methods...), http.MethodOptions)
			w.Header().Set("Allow", strings.Join(methods, ", "))
			if !originAllowed {
				w.Header().Del("Access-Control-Allow-Origin")
				envelope.WriteError(w, http.StatusForbidden, "cors_origin_denied", "origin is not allowed", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			requestedMethod := r.Header.Get("Access-Control-Request-Method")
			if !containsExact(methods, requestedMethod) {
				envelope.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed for this route", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			if !corsHeadersAllowed(r.Header.Get("Access-Control-Request-Headers")) {
				envelope.WriteError(w, http.StatusForbidden, "cors_headers_denied", "requested headers are not allowed", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(allow, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(corsAllowedRequestHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
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

func corsHeadersAllowed(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	for _, requested := range strings.Split(raw, ",") {
		requested = strings.TrimSpace(requested)
		if requested == "" || !containsFold(corsAllowedRequestHeaders, requested) {
			return false
		}
	}
	return true
}

func containsExact(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}
