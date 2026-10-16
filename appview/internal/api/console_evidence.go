package api

import (
	"net/http"
	"social.craftsky/appview/internal/safetyincident"
)

func ConsoleEvidenceCapabilitiesHandler(service *safetyincident.EvidenceService, bodyLimit int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := bodyLimit
		if limit <= 0 || limit > 1<<20 {
			limit = 1 << 20
		}
		maxBytes := max(int64(0), (limit-4096)*3/4)
		maxBytes = min(maxBytes, int64(safetyincident.MaxEvidenceBytes))
		writeSafetyCommandJSON(w, 200, map[string]any{"preservationAvailable": service.Available() && maxBytes > 0, "accessAvailable": service.Available(), "maxPreservationBytes": maxBytes, "maxAccessBytes": safetyincident.MaxEvidenceBytes})
	})
}
