package api

import (
	"context"
	"encoding/json"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// HandleResolverFunc is the minimal interface WhoAmIHandler needs.
// HandleResolver from handle_resolver.go satisfies it; tests stub it.
type HandleResolverFunc interface {
	ResolveHandle(ctx context.Context, did string) (string, error)
}

// WhoAmIHandler returns the caller's DID and current handle.
//
// The DID is read from the request context (injected by the
// Authenticated middleware). The handle is resolved on every call via
// the identity directory.
//
// Errors collapse:
//   - DID missing from context → 500 internal_error (routing bug).
//   - Directory lookup failure (unknown DID, empty handle, network) →
//     502 identity_unavailable.
func WhoAmIHandler(resolver HandleResolverFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context",
				middleware.GetRunID(r.Context()), nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle for did",
				middleware.GetRunID(r.Context()), nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(WhoAmIResponse{DID: did, Handle: handle})
	})
}
