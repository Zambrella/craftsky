// appview/internal/api/moderation.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

const devModerationTokenHeader = "X-Craftsky-Dev-Moderation-Token"

type ModerationOutputInserter interface {
	InsertOutput(rctx context.Context, input ModerationOutputInput) (*ModerationOutputRow, error)
}

type syntheticModerationResponse struct {
	OutputID string `json:"outputId"`
	Status   string `json:"status"`
}

// DevModerationOzoneEventsHandler serves the dev-only synthetic moderation
// endpoint. It enforces the dedicated dev token gate, validates exactly one
// trusted synthetic output, persists it, and returns a minimal indexed response.
func DevModerationOzoneEventsHandler(expectedToken string, cfg ModerationRequestConfig, store ModerationOutputInserter, logger *slog.Logger) http.Handler {
	logger = normalizeLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		if expectedToken == "" || r.Header.Get(devModerationTokenHeader) != expectedToken {
			envelope.WriteError(w, http.StatusForbidden, "invalid_dev_moderation_token", "invalid dev moderation token", runID, nil)
			return
		}
		if store == nil {
			logger.Error("dev moderation: store missing", slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "moderation store unavailable", runID, nil)
			return
		}

		input, err := DecodeSyntheticModerationRequest(r.Body, cfg)
		if err != nil {
			if fe := new(FieldError); errors.As(err, &fe) {
				switch fe.Code {
				case "malformed_body":
					envelope.WriteError(w, http.StatusBadRequest, fe.Code, "could not parse body", runID, fe.Fields)
				case "untrusted_moderation_source":
					envelope.WriteError(w, http.StatusForbidden, fe.Code, "untrusted moderation source", runID, fe.Fields)
				default:
					envelope.WriteError(w, http.StatusUnprocessableEntity, fe.Code, "validation failed", runID, fe.Fields)
				}
				return
			}
			envelope.WriteError(w, http.StatusBadRequest, "malformed_body", "could not parse body", runID, nil)
			return
		}

		row, err := store.InsertOutput(r.Context(), input)
		if err != nil {
			logger.Error("dev moderation: insert output failed",
				apiLogErrorAttrs(runID, "moderation.dev_output.create", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "moderation output persistence failed", runID, nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(syntheticModerationResponse{OutputID: row.ID, Status: "indexed"})
	})
}
