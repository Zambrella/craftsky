package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

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
func WhoAmIHandler(resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context",
				runID, nil)
			return
		}
		logger.Debug("whoami: resolving handle",
			slog.String("did", did.String()),
			slog.String("run_id", runID))
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("whoami: handle resolution failed",
				slog.String("did", did.String()),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle",
				runID, nil)
			return
		}
		logger.Debug("whoami: resolved handle",
			slog.String("did", did.String()),
			slog.String("handle", handle.String()),
			slog.String("run_id", runID))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(WhoAmIResponse{DID: did, Handle: handle})
	})
}
