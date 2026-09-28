package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/moderation"
)

type ModerationCommander interface {
	ResolveCase(context.Context, moderation.TrustedCommand) (moderation.CommandResult, error)
	ConfirmAppeal(context.Context, moderation.AppealCommand) (moderation.CommandResult, error)
	ResolveAppeal(context.Context, moderation.AppealResolutionCommand) (moderation.CommandResult, error)
	ChangeEffects(context.Context, moderation.EffectChangeCommand) (moderation.CommandResult, error)
	RestoreSevereSuspension(context.Context, moderation.SevereRestorationCommand) (moderation.CommandResult, error)
}

type adminDecisionRequest struct {
	ExpectedRevision  *int64                  `json:"expectedRevision"`
	Disposition       moderation.Disposition  `json:"disposition"`
	Reason            moderation.Reason       `json:"reason"`
	EvidenceNotes     string                  `json:"evidenceNotes"`
	UserSafeDetail    string                  `json:"userSafeDetail"`
	SeverityRationale string                  `json:"severityRationale"`
	Consequences      []moderation.EffectType `json:"consequences"`
}

type adminAppealConfirmationRequest struct {
	ExpectedRevision        *int64 `json:"expectedRevision"`
	CorrespondenceReference string `json:"correspondenceReference"`
}

type adminAppealResolutionRequest struct {
	ExpectedRevision *int64                  `json:"expectedRevision"`
	Outcome          moderation.AppealStatus `json:"outcome"`
	ReversedEffects  []moderation.EffectType `json:"reversedEffects"`
	Rationale        string                  `json:"rationale"`
}

type adminEffectChangeRequest struct {
	ExpectedRevision *int64                  `json:"expectedRevision"`
	NegateEffects    []moderation.EffectType `json:"negateEffects"`
	ApplyEffects     []moderation.EffectType `json:"applyEffects"`
	Rationale        string                  `json:"rationale"`
}

type adminRestorationRequest struct {
	ExpectedRevision *int64    `json:"expectedRevision"`
	EffectID         uuid.UUID `json:"effectId"`
	Rationale        string    `json:"rationale"`
}

func ModerationAdminQueueHandler(store *moderation.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed := map[string]bool{"state": true, "subjectType": true, "appealStatus": true, "cursor": true, "limit": true}
		for key, values := range r.URL.Query() {
			if !allowed[key] || len(values) != 1 {
				adminError(w, r, http.StatusBadRequest, "invalid_request", "invalid queue parameters")
				return
			}
		}
		limit := 50
		if raw := r.URL.Query().Get("limit"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil {
				adminError(w, r, 400, "invalid_request", "invalid limit")
				return
			}
			limit = value
		}
		page, err := store.Queue(r.Context(), moderation.QueueFilter{State: r.URL.Query().Get("state"), SubjectType: r.URL.Query().Get("subjectType"), AppealStatus: r.URL.Query().Get("appealStatus"), Cursor: r.URL.Query().Get("cursor"), Limit: limit})
		if errors.Is(err, moderation.ErrInvalidQueueCursor) {
			adminError(w, r, 400, "invalid_cursor", "invalid queue parameters")
			return
		}
		if errors.Is(err, moderation.ErrInvalidQueueFilter) {
			adminError(w, r, 400, "invalid_request", "invalid queue parameters")
			return
		}
		if err != nil {
			adminError(w, r, 500, "internal_error", "moderation queue unavailable")
			return
		}
		writeModerationJSON(w, page)
	})
}

func ModerationAdminDetailHandler(store *moderation.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail, err := store.AdminDetail(r.Context(), r.PathValue("caseReference"))
		if errors.Is(err, moderation.ErrInvalidCaseReference) {
			adminError(w, r, 400, "invalid_case_reference", "invalid moderation case reference")
			return
		}
		if errors.Is(err, moderation.ErrCaseNotFound) {
			adminError(w, r, 404, "moderation_case_not_found", "moderation case not found")
			return
		}
		if err != nil {
			adminError(w, r, 500, "internal_error", "moderation case unavailable")
			return
		}
		writeModerationJSON(w, detail)
	})
}

func ModerationAdminCommandHandler(commander ModerationCommander, sourceDID syntax.DID, kind string) http.Handler {
	decisionAdapter := moderation.NewTrustedInputAdapter(commander)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		moderatorIdentity, ok := ctxkeys.GetModerator(r.Context())
		if !ok {
			adminError(w, r, 401, "moderator_authentication_failed", "moderator authentication failed")
			return
		}
		reference, err := moderation.ParseCaseReference(r.PathValue("caseReference"))
		if err != nil {
			adminError(w, r, 400, "invalid_case_reference", "invalid moderation case reference")
			return
		}
		replayID := r.Header.Get(moderationIdempotencyKeyHeader)
		if err := ValidateModerationIdempotencyKey(replayID); err != nil {
			adminError(w, r, 400, "invalid_idempotency_key", "invalid Idempotency-Key")
			return
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		var result moderation.CommandResult
		switch kind {
		case "decision":
			var request adminDecisionRequest
			if !decodeAdminCommand(w, r, decoder, &request) {
				return
			}
			var command moderation.TrustedCommand
			command, err = decisionAdapter.Normalize(
				moderation.AuthenticatedSource{SourceSystem: moderatorIdentity.SourceSystem, ActorID: moderatorIdentity.ActorID, SourceDID: sourceDID},
				replayID,
				moderation.ResolveRequest{
					CaseID: reference.UUID(), ExpectedRevision: *request.ExpectedRevision,
					Decision: moderation.Decision{Disposition: request.Disposition, Reason: request.Reason, Evidence: request.EvidenceNotes, UserSafeDetail: request.UserSafeDetail, SeverityRationale: request.SeverityRationale, Consequences: request.Consequences},
				},
			)
			if err == nil {
				result, err = decisionAdapter.Invoke(r.Context(), command)
			}
		case "appealConfirmation":
			var request adminAppealConfirmationRequest
			if !decodeAdminCommand(w, r, decoder, &request) {
				return
			}
			var correspondenceID uuid.UUID
			if request.CorrespondenceReference != "" {
				correspondenceID, err = uuid.Parse(request.CorrespondenceReference)
				if err != nil {
					adminError(w, r, http.StatusUnprocessableEntity, "validation_failed", "moderation command is invalid")
					return
				}
			}
			result, err = commander.ConfirmAppeal(r.Context(), moderation.AppealCommand{CaseID: reference.UUID(), ExpectedRevision: *request.ExpectedRevision, SourceSystem: moderatorIdentity.SourceSystem, ReplayID: replayID, ActorID: moderatorIdentity.ActorID, CorrespondenceID: correspondenceID})
		case "appealResolution":
			var request adminAppealResolutionRequest
			if !decodeAdminCommand(w, r, decoder, &request) {
				return
			}
			result, err = commander.ResolveAppeal(r.Context(), moderation.AppealResolutionCommand{CaseID: reference.UUID(), ExpectedRevision: *request.ExpectedRevision, SourceSystem: moderatorIdentity.SourceSystem, ReplayID: replayID, ActorID: moderatorIdentity.ActorID, SourceDID: sourceDID, Outcome: request.Outcome, Negate: request.ReversedEffects, Rationale: request.Rationale})
		case "effectChange":
			var request adminEffectChangeRequest
			if !decodeAdminCommand(w, r, decoder, &request) {
				return
			}
			result, err = commander.ChangeEffects(r.Context(), moderation.EffectChangeCommand{CaseID: reference.UUID(), ExpectedRevision: *request.ExpectedRevision, SourceSystem: moderatorIdentity.SourceSystem, ReplayID: replayID, ActorID: moderatorIdentity.ActorID, SourceDID: sourceDID, Negate: request.NegateEffects, Apply: request.ApplyEffects, Rationale: request.Rationale})
		case "restoration":
			var request adminRestorationRequest
			if !decodeAdminCommand(w, r, decoder, &request) {
				return
			}
			result, err = commander.RestoreSevereSuspension(r.Context(), moderation.SevereRestorationCommand{
				CaseID: reference.UUID(), EffectID: request.EffectID, ExpectedRevision: *request.ExpectedRevision,
				SourceSystem: moderatorIdentity.SourceSystem, ReplayID: replayID,
				ActorID: moderatorIdentity.ActorID, Rationale: request.Rationale,
			})
		default:
			err = moderation.ErrInvalidCommand
		}
		if err != nil {
			writeModerationCommandError(w, r, err)
			return
		}
		writeModerationJSON(w, map[string]any{"caseReference": reference.String(), "revision": result.Revision, "replayed": result.Replayed})
	})
}

func decodeAdminCommand(w http.ResponseWriter, r *http.Request, decoder *json.Decoder, target any) bool {
	if decoder.Decode(target) != nil {
		adminError(w, r, http.StatusBadRequest, "invalid_request", "invalid command body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		adminError(w, r, http.StatusBadRequest, "invalid_request", "invalid command body")
		return false
	}
	value, ok := targetExpectedRevision(target)
	if !ok || value == nil {
		adminError(w, r, http.StatusBadRequest, "invalid_request", "invalid command body")
		return false
	}
	return true
}

func targetExpectedRevision(target any) (*int64, bool) {
	switch request := target.(type) {
	case *adminDecisionRequest:
		return request.ExpectedRevision, true
	case *adminAppealConfirmationRequest:
		return request.ExpectedRevision, true
	case *adminAppealResolutionRequest:
		return request.ExpectedRevision, true
	case *adminEffectChangeRequest:
		return request.ExpectedRevision, true
	case *adminRestorationRequest:
		return request.ExpectedRevision, true
	default:
		return nil, false
	}
}

func writeModerationCommandError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, moderation.ErrCaseNotFound), errors.Is(err, pgx.ErrNoRows):
		adminError(w, r, 404, "moderation_case_not_found", "moderation case not found")
	case errors.Is(err, moderation.ErrRevisionConflict):
		adminError(w, r, 409, "case_revision_conflict", "case revision conflict")
	case errors.Is(err, moderation.ErrReplayConflict):
		adminError(w, r, 409, "idempotency_conflict", "idempotency conflict")
	case errors.Is(err, moderation.ErrInvalidDecision), errors.Is(err, moderation.ErrInvalidCommand), errors.Is(err, moderation.ErrInvalidEffectChange), errors.Is(err, moderation.ErrInvalidAppealTransition), errors.Is(err, moderation.ErrCaseNotOpen):
		adminError(w, r, 422, "validation_failed", "moderation command is invalid")
	default:
		adminError(w, r, 500, "internal_error", "moderation command failed")
	}
}
func adminError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	envelope.WriteError(w, status, code, message, ctxkeys.GetRunID(r.Context()), nil)
}
