package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/safetyincident"
)

func SafetyIncidentDetailHandler(store *safetyincident.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		incidentID, err := uuid.Parse(r.PathValue("incidentReference"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid incident reference", runID, nil)
			return
		}
		detail, err := store.SafeDetail(r.Context(), incidentID)
		if errors.Is(err, pgx.ErrNoRows) {
			envelope.WriteError(w, http.StatusNotFound, "not_found", "incident not found", runID, nil)
			return
		}
		if err != nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "safety_incident_unavailable", "safety incident unavailable", runID, nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(detail)
	})
}
