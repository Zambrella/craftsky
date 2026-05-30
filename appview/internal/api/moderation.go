// appview/internal/api/moderation.go
package api

import (
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

const devModerationTokenHeader = "X-Craftsky-Dev-Moderation-Token"

// DevModerationOzoneEventsHandler serves the dev-only synthetic moderation
// endpoint. The initial slice enforces the dedicated dev token gate before any
// moderation mutation can happen; request validation/persistence follows in the
// moderation store loops.
func DevModerationOzoneEventsHandler(expectedToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		if expectedToken == "" || r.Header.Get(devModerationTokenHeader) != expectedToken {
			envelope.WriteError(w, http.StatusForbidden, "invalid_dev_moderation_token", "invalid dev moderation token", runID, nil)
			return
		}
		envelope.WriteError(w, http.StatusNotImplemented, "not_implemented", "dev moderation ingestion not implemented", runID, nil)
	})
}
