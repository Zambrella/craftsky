package api

import (
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/moderation"
	"time"
)

func ConsoleCaseCreationHandler(pool *pgxpool.Pool) http.Handler {
	return ConsoleCommandHandler(pool, "", "", "", func(tx pgx.Tx) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, _ := safetyActor(r)
			identity, _ := ctxkeys.GetModerator(r.Context())
			var body struct {
				SubjectType moderation.SubjectType `json:"subjectType"`
				SubjectDID  string                 `json:"subjectDid"`
				SubjectURI  string                 `json:"subjectUri"`
				IntakeID    uuid.UUID              `json:"intakeId"`
				Reason      string                 `json:"reason"`
			}
			if decodeSafetyCommand(r, &body) != nil {
				adminError(w, r, 400, "invalid_request", "invalid command body")
				return
			}
			did, err := syntax.ParseDID(body.SubjectDID)
			if err != nil {
				adminError(w, r, 400, "invalid_request", "invalid account reference")
				return
			}
			var uri syntax.ATURI
			if body.SubjectURI != "" {
				uri, err = syntax.ParseATURI(body.SubjectURI)
				if err != nil {
					adminError(w, r, 400, "invalid_request", "invalid subject reference")
					return
				}
			}
			result, err := moderation.NewStore(tx).OpenOperatorCase(r.Context(), moderation.OperatorCaseCommand{SubjectType: body.SubjectType, DID: did, URI: uri, IntakeID: body.IntakeID, Actor: actor, SourceSystem: identity.SourceSystem, Reason: body.Reason, At: time.Now().UTC()})
			if err != nil {
				writeModerationCommandError(w, r, err)
				return
			}
			reference, _ := moderation.FormatCaseReference(result.ID)
			writeSafetyCommandJSON(w, 201, map[string]any{"caseReference": reference, "caseRevision": result.Revision})
		})
	})
}

func ConsoleAccountStandingHandler(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			adminError(w, r, 400, "invalid_request", "invalid account reference")
			return
		}
		if pool == nil {
			adminError(w, r, 503, "admin_read_unavailable", "admin read unavailable")
			return
		}
		standing, err := moderation.NewStore(pool).Standing(r.Context(), did)
		if err != nil {
			consoleReadError(w, r, err)
			return
		}
		writeSafetyCommandJSON(w, 200, standing)
	})
}

func ConsoleAccountEligibilityHandler(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			adminError(w, r, 400, "invalid_request", "invalid account reference")
			return
		}
		if pool == nil {
			adminError(w, r, 503, "admin_read_unavailable", "admin read unavailable")
			return
		}
		var state string
		var revision int64
		err = pool.QueryRow(r.Context(), `SELECT state,revision FROM account_age_eligibility WHERE account_did=$1`, did).Scan(&state, &revision)
		if errors.Is(err, pgx.ErrNoRows) {
			state = "eligible"
			err = nil
		}
		if err != nil {
			consoleReadError(w, r, err)
			return
		}
		writeSafetyCommandJSON(w, 200, map[string]any{"state": state, "revision": revision})
	})
}
