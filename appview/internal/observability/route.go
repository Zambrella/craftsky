package observability

import (
	"net/http"
	"strings"
)

const unmatchedRoutePattern = "unmatched"

// RoutePattern returns the low-cardinality ServeMux route pattern for telemetry.
func RoutePattern(r *http.Request) string {
	pattern := r.Pattern
	if pattern == "" {
		return unmatchedRoutePattern
	}
	return strings.TrimPrefix(pattern, r.Method+" ")
}
