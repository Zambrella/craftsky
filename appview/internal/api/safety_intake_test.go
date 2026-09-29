package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/safetyintake"
)

type safetyIntakeStoreFixture struct {
	accepted       safetyintake.EmailMetadata
	correspondence safetyintake.CorrespondenceMetadata
	appeal         safetyintake.AppealLink
	err            error
}

func (fixture *safetyIntakeStoreFixture) AcceptEmail(_ context.Context, metadata safetyintake.EmailMetadata) (safetyintake.Intake, error) {
	fixture.accepted = metadata
	return safetyintake.Intake{ID: uuid.MustParse("10000000-0000-4000-8000-000000000001"), Reference: "CS-SAF-TEST"}, fixture.err
}

func (fixture *safetyIntakeStoreFixture) AppendCorrespondence(_ context.Context, metadata safetyintake.CorrespondenceMetadata) (uuid.UUID, bool, error) {
	fixture.correspondence = metadata
	return uuid.MustParse("20000000-0000-4000-8000-000000000001"), false, fixture.err
}

func (fixture *safetyIntakeStoreFixture) BindAppeal(_ context.Context, link safetyintake.AppealLink) (uuid.UUID, error) {
	fixture.appeal = link
	return uuid.MustParse("30000000-0000-4000-8000-000000000001"), fixture.err
}

func TestSafetyExternalIntakeUsesAuthenticatedActorAndStrictMetadataBody(t *testing.T) {
	store := &safetyIntakeStoreFixture{}
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/external-intakes", strings.NewReader(`{
		"providerMessageReference":"message-1",
		"receivedAt":"2030-09-22T12:00:00Z",
		"canonicalSubject":"at://did:plc:test/social.craftsky.feed.post/one",
		"kind":"allegation",
		"urgency":"priority",
		"senderContact":"reporter@example.invalid",
		"hasMediaAttachment":false,
		"attachmentStatus":"none"
	}`))
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{
		ActorID: "authenticated-operator", SourceSystem: "admin-api", Role: "safetyAdministrator",
	}))
	recorder := httptest.NewRecorder()
	SafetyExternalIntakeHandler(store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.accepted.OwnerActorID != "authenticated-operator" || store.accepted.SenderContact != "reporter@example.invalid" {
		t.Fatalf("accepted metadata=%+v", store.accepted)
	}
	if strings.Contains(recorder.Body.String(), "reporter@example.invalid") || strings.Contains(recorder.Body.String(), "sha256") {
		t.Fatalf("contact data leaked in response: %s", recorder.Body.String())
	}
	var response map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["reference"] != "CS-SAF-TEST" || response["replayed"] != false {
		t.Fatalf("response=%v", response)
	}

	strictRequest := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/external-intakes", strings.NewReader(`{"rawMime":"forbidden"}`))
	strictRequest = strictRequest.WithContext(ctxkeys.WithModerator(strictRequest.Context(), ctxkeys.Moderator{ActorID: "operator"}))
	strictRecorder := httptest.NewRecorder()
	SafetyExternalIntakeHandler(store).ServeHTTP(strictRecorder, strictRequest)
	if strictRecorder.Code != http.StatusBadRequest || !strings.Contains(strictRecorder.Body.String(), `"error":"malformed_body"`) {
		t.Fatalf("strict status=%d body=%s", strictRecorder.Code, strictRecorder.Body.String())
	}
}

func TestSafetyExternalIntakeCorrespondenceAndAppealUsePathAndAuthenticatedActor(t *testing.T) {
	store := &safetyIntakeStoreFixture{}
	intakeID := uuid.MustParse("40000000-0000-4000-8000-000000000001")
	moderator := ctxkeys.Moderator{ActorID: "appeal-operator", SourceSystem: "admin-api", Role: "safetyAdministrator"}

	correspondenceRequest := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/external-intakes/"+intakeID.String()+"/correspondence", strings.NewReader(`{
		"providerMessageReference":"message-2",
		"receivedAt":"2030-09-22T12:01:00Z",
		"senderContact":"representative@example.invalid",
		"hasMediaAttachment":true,
		"attachmentStatus":"rejected"
	}`))
	correspondenceRequest.SetPathValue("intakeId", intakeID.String())
	correspondenceRequest = correspondenceRequest.WithContext(ctxkeys.WithModerator(correspondenceRequest.Context(), moderator))
	correspondenceRecorder := httptest.NewRecorder()
	SafetyExternalCorrespondenceHandler(store).ServeHTTP(correspondenceRecorder, correspondenceRequest)
	if correspondenceRecorder.Code != http.StatusCreated || store.correspondence.IntakeID != intakeID || !store.correspondence.HasMediaAttachment {
		t.Fatalf("status=%d correspondence=%+v body=%s", correspondenceRecorder.Code, store.correspondence, correspondenceRecorder.Body.String())
	}

	caseID := uuid.MustParse("50000000-0000-4000-8000-000000000001")
	appealRequest := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/external-intakes/"+intakeID.String()+"/appeal-link", strings.NewReader(`{
		"caseId":"`+caseID.String()+`",
		"verifiedOwnerDid":"did:plc:appeal-owner",
		"replayId":"message-2",
		"receivedAt":"2030-09-22T12:01:00Z"
	}`))
	appealRequest.SetPathValue("intakeId", intakeID.String())
	appealRequest = appealRequest.WithContext(ctxkeys.WithModerator(appealRequest.Context(), moderator))
	appealRecorder := httptest.NewRecorder()
	SafetyExternalAppealLinkHandler(store).ServeHTTP(appealRecorder, appealRequest)
	if appealRecorder.Code != http.StatusCreated || store.appeal.IntakeID != intakeID || store.appeal.CaseID != caseID ||
		store.appeal.ActorID != "appeal-operator" || store.appeal.SourceSystem != "externalEmail" {
		t.Fatalf("status=%d appeal=%+v body=%s", appealRecorder.Code, store.appeal, appealRecorder.Body.String())
	}
	if !store.appeal.ReceivedAt.Equal(time.Date(2030, 9, 22, 12, 1, 0, 0, time.UTC)) {
		t.Fatalf("receivedAt=%s", store.appeal.ReceivedAt)
	}
}

func TestSafetyExternalIntakeRequiresModeratorAndMapsUnsafeAttachment(t *testing.T) {
	unauthorized := httptest.NewRecorder()
	SafetyExternalIntakeHandler(&safetyIntakeStoreFixture{}).ServeHTTP(
		unauthorized,
		httptest.NewRequest(http.MethodPost, "/v1/admin/safety/external-intakes", strings.NewReader(`{}`)),
	)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d", unauthorized.Code)
	}

	store := &safetyIntakeStoreFixture{err: safetyintake.ErrUnsafeAttachment}
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/external-intakes", strings.NewReader(`{
		"providerMessageReference":"message-3",
		"receivedAt":"2030-09-22T12:00:00Z",
		"canonicalSubject":"https://craftsky.social/post/test",
		"kind":"incident",
		"urgency":"urgent",
		"hasMediaAttachment":true,
		"attachmentStatus":"none"
	}`))
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "operator"}))
	recorder := httptest.NewRecorder()
	SafetyExternalIntakeHandler(store).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), `"error":"unsafe_attachment_status"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
