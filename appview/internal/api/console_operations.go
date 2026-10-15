package api

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/safetyincident"
	"time"
)

func ConsoleScanRetryHandler(pool *pgxpool.Pool) http.Handler {
	return ConsoleCommandHandler(pool, "image_scan_jobs", "scanId", "scan_result_id", func(tx pgx.Tx) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				ExpectedVersion int64 `json:"expectedVersion"`
			}
			if decodeSafetyCommand(r, &body) != nil || body.ExpectedVersion < 1 {
				adminError(w, r, 400, "invalid_request", "invalid retry body")
				return
			}
			id, _ := uuid.Parse(r.PathValue("scanId"))
			changed, err := imagesafety.NewWorkerStore(tx, time.Now).ManualRetry(r.Context(), id, body.ExpectedVersion)
			if err != nil {
				consoleCommandFailure(w, r, err)
				return
			}
			if !changed {
				adminError(w, r, 409, "invalid_workflow_state", "scan is not retryable at this version")
				return
			}
			writeSafetyCommandJSON(w, 200, map[string]any{"id": id, "state": "queued"})
		})
	})
}

func ConsoleHoldReleaseHandler(pool *pgxpool.Pool) http.Handler {
	return ConsoleCommandHandler(pool, "safety_legal_holds", "holdId", "id", func(tx pgx.Tx) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := safetyActor(r)
			if !ok {
				adminError(w, r, 401, "moderator_authentication_failed", "moderator authentication failed")
				return
			}
			var body struct {
				Reason string `json:"reason"`
			}
			if decodeSafetyCommand(r, &body) != nil {
				adminError(w, r, 400, "invalid_request", "invalid release body")
				return
			}
			id, _ := uuid.Parse(r.PathValue("holdId"))
			if err := safetyincident.NewHoldService(tx).Release(r.Context(), actor, id, body.Reason, time.Now().UTC()); err != nil {
				writeSafetyCommandError(w, r, err)
				return
			}
			writeSafetyCommandJSON(w, 200, map[string]any{"id": id, "state": "released"})
		})
	})
}
