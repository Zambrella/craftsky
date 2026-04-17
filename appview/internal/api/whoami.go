package api

import (
	"encoding/json"
	"net/http"

	"social.craftsky/appview/internal/middleware"
)

// WhoAmIHandler returns the caller's authenticated DID as JSON. It
// assumes middleware.Authenticated has run — if not, it returns 500
// with a "no did in context" body, which would be a routing bug.
func WhoAmIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			http.Error(w, "no did in context", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"did": did})
	})
}
