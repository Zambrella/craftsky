package api

import (
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/moderation"
)

func ConsoleDetectionConfirmationHandler(pool *pgxpool.Pool) http.Handler {
	return ConsoleCommandHandler(pool, "safety_incidents", "incidentReference", "id", func(tx pgx.Tx) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := safetyActor(r)
			if !ok {
				adminError(w, r, 401, "moderator_authentication_failed", "moderator authentication failed")
				return
			}
			var body struct {
				SubjectURI string `json:"subjectUri"`
			}
			if decodeSafetyCommand(r, &body) != nil {
				adminError(w, r, 400, "invalid_request", "invalid command body")
				return
			}
			uri, err := syntax.ParseATURI(body.SubjectURI)
			if err != nil {
				adminError(w, r, 400, "invalid_request", "invalid subject reference")
				return
			}
			id, _ := uuid.Parse(r.PathValue("incidentReference"))
			result, err := moderation.NewStore(tx).ConfirmSystemDetection(r.Context(), moderation.ConfirmSystemDetectionCommand{IncidentID: id, SubjectURI: uri, Actor: actor, SourceSystem: "operator-console", ReplayID: actor.ID + ":" + r.Header.Get("Idempotency-Key"), ConfirmedAt: time.Now().UTC()})
			if err != nil {
				writeModerationCommandError(w, r, err)
				return
			}
			reference, _ := moderation.FormatCaseReference(result.ID)
			writeSafetyCommandJSON(w, 201, map[string]any{"caseReference": reference, "caseRevision": result.Revision})
		})
	})
}
