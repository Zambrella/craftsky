package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/moderation"
)

type moderationCommanderStub struct {
	decision    moderation.TrustedCommand
	restoration moderation.SevereRestorationCommand
	decisionErr error
}

func (s *moderationCommanderStub) ResolveCase(_ context.Context, command moderation.TrustedCommand) (moderation.CommandResult, error) {
	s.decision = command
	return moderation.CommandResult{CaseID: command.CaseID, Revision: command.ExpectedRevision + 1}, s.decisionErr
}
func (*moderationCommanderStub) ConfirmAppeal(context.Context, moderation.AppealCommand) (moderation.CommandResult, error) {
	return moderation.CommandResult{}, nil
}
func (*moderationCommanderStub) ResolveAppeal(context.Context, moderation.AppealResolutionCommand) (moderation.CommandResult, error) {
	return moderation.CommandResult{}, nil
}
func (*moderationCommanderStub) ChangeEffects(context.Context, moderation.EffectChangeCommand) (moderation.CommandResult, error) {
	return moderation.CommandResult{}, nil
}
func (s *moderationCommanderStub) RestoreSevereSuspension(_ context.Context, command moderation.SevereRestorationCommand) (moderation.CommandResult, error) {
	s.restoration = command
	return moderation.CommandResult{CaseID: command.CaseID, Revision: command.ExpectedRevision + 1}, nil
}

func TestModerationAdminCommandUsesOnlyAuthenticatedActorAndSource(t *testing.T) {
	caseID := uuid.New()
	reference, err := moderation.FormatCaseReference(caseID)
	if err != nil {
		t.Fatal(err)
	}
	commander := &moderationCommanderStub{}
	handler := ModerationAdminCommandHandler(commander, syntax.DID("did:plc:moderation"), "decision")
	mux := http.NewServeMux()
	mux.Handle("POST /v1/admin/moderation/cases/{caseReference}/decisions", handler)

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/moderation/cases/"+reference+"/decisions", strings.NewReader(`{"expectedRevision":0,"disposition":"violation","reason":"spam","evidenceNotes":"reviewed evidence","consequences":["strike"]}`))
	request.Header.Set(moderationIdempotencyKeyHeader, "replay-identifier-1")
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "trusted-actor", SourceSystem: "admin-api"}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
	}
	if commander.decision.CaseID != caseID || commander.decision.ExpectedRevision != 0 ||
		commander.decision.ActorID != "trusted-actor" || commander.decision.SourceSystem != "admin-api" ||
		commander.decision.SourceDID != "did:plc:moderation" || commander.decision.ReplayID != "replay-identifier-1" ||
		commander.decision.Decision.Disposition != moderation.DispositionViolation || commander.decision.Decision.Reason != moderation.ReasonSpam ||
		commander.decision.Decision.Evidence != "reviewed evidence" || len(commander.decision.Decision.Consequences) != 1 ||
		commander.decision.Decision.Consequences[0] != moderation.EffectStrike {
		t.Fatalf("trusted command = %+v", commander.decision)
	}

	request = httptest.NewRequest(http.MethodPost, "/v1/admin/moderation/cases/"+reference+"/decisions", strings.NewReader(`{"expectedRevision":0,"actorId":"untrusted","disposition":"violation","reason":"spam","evidenceNotes":"reviewed evidence","consequences":["strike"]}`))
	request.Header.Set(moderationIdempotencyKeyHeader, "replay-identifier-2")
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "trusted-actor", SourceSystem: "admin-api"}))
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("untrusted attribution status = %d", response.Code)
	}
}

func TestModerationAdminDecisionRequiresTrustedAdapterSource(t *testing.T) {
	caseID := uuid.New()
	reference, err := moderation.FormatCaseReference(caseID)
	if err != nil {
		t.Fatal(err)
	}
	commander := &moderationCommanderStub{}
	handler := ModerationAdminCommandHandler(commander, "", "decision")
	mux := http.NewServeMux()
	mux.Handle("POST /v1/admin/moderation/cases/{caseReference}/decisions", handler)
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/moderation/cases/"+reference+"/decisions", strings.NewReader(`{"expectedRevision":0,"disposition":"noAction","evidenceNotes":"not substantiated"}`))
	request.Header.Set(moderationIdempotencyKeyHeader, "replay-identifier-3")
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "trusted-actor", SourceSystem: "admin-api"}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || commander.decision.CaseID != uuid.Nil {
		t.Fatalf("status/body/command = %d/%s/%+v", response.Code, response.Body.String(), commander.decision)
	}
}

func TestModerationAdminRestorationTargetsExactSevereEffect(t *testing.T) {
	caseID, effectID := uuid.New(), uuid.New()
	reference, err := moderation.FormatCaseReference(caseID)
	if err != nil {
		t.Fatal(err)
	}
	commander := &moderationCommanderStub{}
	handler := ModerationAdminCommandHandler(commander, syntax.DID("did:plc:moderation"), "restoration")
	mux := http.NewServeMux()
	mux.Handle("POST /v1/admin/moderation/cases/{caseReference}/restorations", handler)
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/moderation/cases/"+reference+"/restorations", strings.NewReader(`{"expectedRevision":1,"effectId":"`+effectID.String()+`","rationale":"review completed"}`))
	request.Header.Set(moderationIdempotencyKeyHeader, "restoration-replay-1")
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "trusted-actor", SourceSystem: "admin-api"}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
	}
	got := commander.restoration
	if got.CaseID != caseID || got.EffectID != effectID || got.ExpectedRevision != 1 || got.ActorID != "trusted-actor" || got.SourceSystem != "admin-api" || got.ReplayID != "restoration-replay-1" || got.Rationale != "review completed" {
		t.Fatalf("restoration command = %+v", got)
	}
	if commander.decision.CaseID != uuid.Nil {
		t.Fatalf("restoration invoked decision command = %+v", commander.decision)
	}
}

func TestModerationAdminCommandRequiresExpectedRevision(t *testing.T) {
	caseID := uuid.New()
	reference, err := moderation.FormatCaseReference(caseID)
	if err != nil {
		t.Fatal(err)
	}
	commander := &moderationCommanderStub{}
	handler := ModerationAdminCommandHandler(commander, syntax.DID("did:plc:moderation"), "decision")
	mux := http.NewServeMux()
	mux.Handle("POST /v1/admin/moderation/cases/{caseReference}/decisions", handler)

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/moderation/cases/"+reference+"/decisions", strings.NewReader(`{"disposition":"noAction","evidenceNotes":"not substantiated"}`))
	request.Header.Set(moderationIdempotencyKeyHeader, "replay-identifier-4")
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "trusted-actor", SourceSystem: "admin-api"}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_request"`) {
		t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
	}
	if commander.decision.CaseID != uuid.Nil {
		t.Fatalf("command invoked = %+v", commander.decision)
	}
}

func TestEveryModerationAdminCommandRequiresExpectedRevision(t *testing.T) {
	caseID := uuid.New()
	reference, err := moderation.FormatCaseReference(caseID)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		kind string
		path string
		body string
	}{
		{kind: "decision", path: "decisions", body: `{"disposition":"noAction","evidenceNotes":"reviewed"}`},
		{kind: "appealConfirmation", path: "appeal-confirmations", body: `{}`},
		{kind: "appealResolution", path: "appeal-resolutions", body: `{"outcome":"upheld","rationale":"reviewed"}`},
		{kind: "effectChange", path: "effect-changes", body: `{"negateEffects":["strike"],"rationale":"reviewed"}`},
		{kind: "restoration", path: "restorations", body: `{"effectId":"` + uuid.NewString() + `","rationale":"reviewed"}`},
	} {
		t.Run(test.path, func(t *testing.T) {
			commander := &moderationCommanderStub{}
			handler := ModerationAdminCommandHandler(commander, syntax.DID("did:plc:moderation"), test.kind)
			mux := http.NewServeMux()
			pattern := "/v1/admin/moderation/cases/{caseReference}/" + test.path
			mux.Handle("POST "+pattern, handler)
			request := httptest.NewRequest(http.MethodPost, "/v1/admin/moderation/cases/"+reference+"/"+test.path, strings.NewReader(test.body))
			request.Header.Set(moderationIdempotencyKeyHeader, "required-revision-"+test.path)
			request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "trusted-actor", SourceSystem: "admin-api"}))
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_request"`) {
				t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestModerationAdminCommandAcceptsExactlyOneJSONObject(t *testing.T) {
	caseID := uuid.New()
	reference, err := moderation.FormatCaseReference(caseID)
	if err != nil {
		t.Fatal(err)
	}
	commander := &moderationCommanderStub{}
	handler := ModerationAdminCommandHandler(commander, syntax.DID("did:plc:moderation"), "decision")
	mux := http.NewServeMux()
	mux.Handle("POST /v1/admin/moderation/cases/{caseReference}/decisions", handler)

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/moderation/cases/"+reference+"/decisions", strings.NewReader(`{"expectedRevision":0,"disposition":"noAction","evidenceNotes":"not substantiated"} {}`))
	request.Header.Set(moderationIdempotencyKeyHeader, "replay-identifier-5")
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "trusted-actor", SourceSystem: "admin-api"}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_request"`) {
		t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
	}
	if commander.decision.CaseID != uuid.Nil {
		t.Fatalf("command invoked = %+v", commander.decision)
	}
}

func TestModerationAdminCommandRejectsFieldsFromAnotherEndpoint(t *testing.T) {
	caseID := uuid.New()
	reference, err := moderation.FormatCaseReference(caseID)
	if err != nil {
		t.Fatal(err)
	}
	commander := &moderationCommanderStub{}
	handler := ModerationAdminCommandHandler(commander, syntax.DID("did:plc:moderation"), "decision")
	mux := http.NewServeMux()
	mux.Handle("POST /v1/admin/moderation/cases/{caseReference}/decisions", handler)

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/moderation/cases/"+reference+"/decisions", strings.NewReader(`{"expectedRevision":0,"disposition":"noAction","evidenceNotes":"not substantiated","rationale":"belongs to effect changes"}`))
	request.Header.Set(moderationIdempotencyKeyHeader, "replay-identifier-6")
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "trusted-actor", SourceSystem: "admin-api"}))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_request"`) {
		t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
	}
	if commander.decision.CaseID != uuid.Nil {
		t.Fatalf("command invoked = %+v", commander.decision)
	}
}

func TestModerationAdminQueueDistinguishesInvalidQueryFromCursor(t *testing.T) {
	handler := ModerationAdminQueueHandler(&moderation.Store{})
	for _, test := range []struct {
		name      string
		query     string
		wantError string
	}{
		{name: "state", query: "state=closed", wantError: "invalid_request"},
		{name: "subject type", query: "subjectType=comment", wantError: "invalid_request"},
		{name: "appeal status", query: "appealStatus=received", wantError: "invalid_request"},
		{name: "limit syntax", query: "limit=many", wantError: "invalid_request"},
		{name: "limit range", query: "limit=101", wantError: "invalid_request"},
		{name: "unknown filter", query: "ownerDid=did%3Aplc%3Aprivate", wantError: "invalid_request"},
		{name: "cursor", query: "cursor=not-a-cursor", wantError: "invalid_cursor"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/admin/moderation/cases?"+test.query, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"`+test.wantError+`"`) {
				t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestModerationAdminCommandMapsCuratedDomainErrors(t *testing.T) {
	caseID := uuid.New()
	reference, err := moderation.FormatCaseReference(caseID)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "missing case", err: moderation.ErrCaseNotFound, wantStatus: http.StatusNotFound, wantCode: "moderation_case_not_found"},
		{name: "closed case", err: moderation.ErrCaseNotOpen, wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "stale revision", err: moderation.ErrRevisionConflict, wantStatus: http.StatusConflict, wantCode: "case_revision_conflict"},
		{name: "replay conflict", err: moderation.ErrReplayConflict, wantStatus: http.StatusConflict, wantCode: "idempotency_conflict"},
	} {
		t.Run(test.name, func(t *testing.T) {
			commander := &moderationCommanderStub{decisionErr: test.err}
			handler := ModerationAdminCommandHandler(commander, syntax.DID("did:plc:moderation"), "decision")
			mux := http.NewServeMux()
			mux.Handle("POST /v1/admin/moderation/cases/{caseReference}/decisions", handler)
			request := httptest.NewRequest(http.MethodPost, "/v1/admin/moderation/cases/"+reference+"/decisions", strings.NewReader(`{"expectedRevision":0,"disposition":"noAction","evidenceNotes":"not substantiated"}`))
			request.Header.Set(moderationIdempotencyKeyHeader, "curated-error-replay")
			request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "trusted-actor", SourceSystem: "admin-api"}))
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"error":"`+test.wantCode+`"`) {
				t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
			}
		})
	}
}
