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

// Preview returns only indexed text from the captured version. No remote media,
// raw records or restricted evidence is fetched or embedded.
func ConsoleCasePreviewHandler(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ref, err := moderation.ParseCaseReference(r.PathValue("caseReference"))
		if err != nil {
			adminError(w, r, 400, "invalid_request", "invalid case reference")
			return
		}
		if pool == nil {
			adminError(w, r, 503, "admin_read_unavailable", "admin read unavailable")
			return
		}
		var kind, did, uri, cid string
		var restricted bool
		err = pool.QueryRow(r.Context(), `SELECT subject_type,subject_did,COALESCE(subject_uri,''),COALESCE(subject_cid_snapshot,''),origin='systemDetected' OR EXISTS(SELECT 1 FROM safety_incident_subjects s WHERE s.subject_uri=c.subject_uri OR (c.subject_type='account' AND s.owner_did=c.subject_did)) FROM moderation_cases c WHERE id=$1`, ref.UUID()).Scan(&kind, &did, &uri, &cid, &restricted)
		if err != nil {
			consoleReadError(w, r, err)
			return
		}
		result := map[string]any{"state": "unavailable", "subjectType": kind, "subjectDid": did, "mediaAvailable": false}
		if restricted {
			result["state"] = "restricted"
			writeSafetyCommandJSON(w, 200, result)
			return
		}
		var text, title string
		switch kind {
		case "post":
			err = pool.QueryRow(r.Context(), `SELECT text,'' FROM craftsky_posts p WHERE uri=$1 AND cid=$2 AND EXISTS(SELECT 1 FROM image_subject_states s WHERE s.subject_uri=p.uri AND s.source_cid=p.cid AND s.visibility_state='clear')`, uri, cid).Scan(&text, &title)
		case "event":
			err = pool.QueryRow(r.Context(), `SELECT COALESCE(raw_record->>'description',''),COALESCE(raw_record->>'name','') FROM craftsky_business_events e WHERE uri=$1 AND cid=$2 AND EXISTS(SELECT 1 FROM image_subject_states s WHERE s.subject_uri=e.uri AND s.source_cid=e.cid AND s.visibility_state='clear')`, uri, cid).Scan(&text, &title)
		case "account":
			err = pool.QueryRow(r.Context(), `SELECT COALESCE(description,''),COALESCE(display_name,'') FROM bluesky_profiles WHERE did=$1 AND ($2='' OR record_cid=$2)`, did, cid).Scan(&text, &title)
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			consoleReadError(w, r, err)
			return
		}
		if err == nil {
			result["state"] = "available"
			result["text"] = text
			result["title"] = title
		}
		writeSafetyCommandJSON(w, 200, result)
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
