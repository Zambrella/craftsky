package moderation

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

type enforcementBodyProbe struct {
	reader *strings.Reader
	reads  int
}

func newEnforcementBodyProbe(body string) *enforcementBodyProbe {
	return &enforcementBodyProbe{reader: strings.NewReader(body)}
}

func (body *enforcementBodyProbe) Read(buffer []byte) (int, error) {
	body.reads++
	return body.reader.Read(buffer)
}

func (*enforcementBodyProbe) Close() error { return nil }

type recordingEnforcementPDS struct {
	records        map[syntax.DID]bool
	mutations      int
	ownerDeletions int
}

func (pds *recordingEnforcementPDS) mutate(owner syntax.DID) {
	pds.mutations++
	pds.records[owner] = true
}

func (pds *recordingEnforcementPDS) deleteForOwner(owner syntax.DID) {
	pds.ownerDeletions++
	delete(pds.records, owner)
}

func TestStoredStandingTransitionsDriveEnforcementWithoutModerationPDSMutation(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	now := time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now })
	pds := &recordingEnforcementPDS{records: map[syntax.DID]bool{}}

	thresholdOwner := syntax.DID("did:plc:enforcement-threshold")
	pds.records[thresholdOwner] = true
	var thresholdResults []CommandResult
	for index := 1; index <= 3; index++ {
		caseRow := seedAdjudicationCase(t, pool, store, "enforcement-threshold-report-"+string(rune('0'+index)), thresholdOwner)
		thresholdResults = append(thresholdResults, resolveStrike(t, service, caseRow, "enforcement-threshold-replay-"+string(rune('0'+index))))
	}
	assertEnforcementStanding(t, store, thresholdOwner, true)
	assertEnforcementPDSCounts(t, pds, 0, 0)

	publicHandlerCalls := 0
	publicWrite := func(owner syntax.DID, body *enforcementBodyProbe) *httptest.ResponseRecorder {
		handler := middleware.ModerationEnforcement(store, false, nil)(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			publicHandlerCalls++
			if _, err := io.ReadAll(request.Body); err != nil {
				t.Errorf("read allowed public write body: %v", err)
			}
			pds.mutate(owner)
			w.WriteHeader(http.StatusNoContent)
		}))
		request := httptest.NewRequest(http.MethodPost, "/v1/posts", body)
		request.ContentLength = int64(body.reader.Len())
		request = request.WithContext(ctxkeys.WithDID(request.Context(), owner))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	deniedBody := newEnforcementBodyProbe(`{"text":"must not be read"}`)
	response := publicWrite(thresholdOwner, deniedBody)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"error":"account_suspended"`) {
		t.Fatalf("threshold denial status/body = %d/%s", response.Code, response.Body.String())
	}
	if publicHandlerCalls != 0 || deniedBody.reads != 0 {
		t.Fatalf("denied threshold handler/body work = %d/%d, want 0/0", publicHandlerCalls, deniedBody.reads)
	}
	assertEnforcementPDSCounts(t, pds, 0, 0)
	if !pds.records[thresholdOwner] {
		t.Fatal("threshold enforcement removed the owner's PDS record")
	}

	readCalls := 0
	invokeAllowedEnforcement(t, store, thresholdOwner, http.MethodGet, "/v1/account/moderation", func(w http.ResponseWriter, _ *http.Request) {
		readCalls++
		w.WriteHeader(http.StatusNoContent)
	})
	if readCalls != 1 {
		t.Fatalf("suspended read calls = %d, want 1", readCalls)
	}

	invokeAllowedEnforcement(t, store, thresholdOwner, http.MethodDelete, "/v1/posts/self/record", func(w http.ResponseWriter, _ *http.Request) {
		pds.deleteForOwner(thresholdOwner)
		w.WriteHeader(http.StatusNoContent)
	})
	assertEnforcementPDSCounts(t, pds, 0, 1)
	if pds.records[thresholdOwner] {
		t.Fatal("explicit owner removal did not delete the owner's PDS record")
	}

	adminCalls := 0
	thirdCaseID := thresholdResults[2].CaseID
	invokeAllowedEnforcement(t, store, thresholdOwner, http.MethodPost, "/v1/moderation/cases/effects", func(w http.ResponseWriter, _ *http.Request) {
		adminCalls++
		_, err := service.ChangeEffects(ctx, EffectChangeCommand{
			CaseID: thirdCaseID, ExpectedRevision: thresholdResults[2].Revision,
			SourceSystem: "integration-admin", ReplayID: "enforcement-threshold-reversal",
			ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
			Negate: []EffectType{EffectStrike}, Rationale: "review reversed the strike",
		})
		if err != nil {
			t.Errorf("reverse threshold strike: %v", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if adminCalls != 1 {
		t.Fatalf("suspended admin calls = %d, want 1", adminCalls)
	}
	assertEnforcementStanding(t, store, thresholdOwner, false)
	assertEnforcementPDSCounts(t, pds, 0, 1)

	allowedBody := newEnforcementBodyProbe(`{"text":"allowed after reversal"}`)
	response = publicWrite(thresholdOwner, allowedBody)
	if response.Code != http.StatusNoContent || publicHandlerCalls != 1 || allowedBody.reads == 0 {
		t.Fatalf("post-reversal status/handler/body work = %d/%d/%d", response.Code, publicHandlerCalls, allowedBody.reads)
	}
	assertEnforcementPDSCounts(t, pds, 1, 1)

	severeOwner := syntax.DID("did:plc:enforcement-severe")
	pds.records[severeOwner] = true
	severeCase := seedAdjudicationCase(t, pool, store, "enforcement-severe-report", severeOwner)
	severeResult, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: severeCase.ID, ExpectedRevision: 0, SourceSystem: "integration-admin",
		ReplayID: "enforcement-severe-decision", ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence",
			SeverityRationale: "immediate risk", Consequences: []EffectType{EffectStrike, EffectSevereSuspension}},
	})
	if err != nil {
		t.Fatalf("resolve severe case: %v", err)
	}
	for index := 1; index <= 2; index++ {
		now = now.Add(time.Hour)
		caseRow := seedAdjudicationCase(t, pool, store, "enforcement-severe-threshold-report-"+string(rune('0'+index)), severeOwner)
		resolveStrike(t, service, caseRow, "enforcement-severe-threshold-replay-"+string(rune('0'+index)))
	}
	assertEnforcementStanding(t, store, severeOwner, true)
	assertEnforcementPDSCounts(t, pds, 1, 1)

	now = StrikeDeadline(time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC))
	processed, err := NewExpiryProcessor(store, nil, func() time.Time { return now }).ProcessDue(ctx, 1)
	if err != nil || processed != 1 {
		t.Fatalf("expire threshold strike = %d, %v", processed, err)
	}
	standing, err := store.Standing(ctx, severeOwner)
	if err != nil {
		t.Fatal(err)
	}
	if standing.ActiveStrikeCount != 2 || standing.ThresholdSuspended || !standing.SevereSuspended || !standing.Suspended {
		t.Fatalf("standing after expiry = %+v, want independent severe suspension", standing)
	}
	assertEnforcementPDSCounts(t, pds, 1, 1)

	deniedAfterExpiry := newEnforcementBodyProbe(`{"text":"still denied"}`)
	response = publicWrite(severeOwner, deniedAfterExpiry)
	if response.Code != http.StatusForbidden || publicHandlerCalls != 1 || deniedAfterExpiry.reads != 0 {
		t.Fatalf("post-expiry denial status/handler/body work = %d/%d/%d", response.Code, publicHandlerCalls, deniedAfterExpiry.reads)
	}
	assertEnforcementPDSCounts(t, pds, 1, 1)
	if !pds.records[severeOwner] {
		t.Fatal("expiry enforcement removed the owner's PDS record")
	}

	var severeEffectID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT logical_effect_id FROM moderation_active_case_effects WHERE case_id=$1 AND effect_type='severeSuspension'`, severeCase.ID).Scan(&severeEffectID); err != nil {
		t.Fatalf("read severe effect: %v", err)
	}
	if _, err := service.RestoreSevereSuspension(ctx, SevereRestorationCommand{
		CaseID: severeCase.ID, EffectID: severeEffectID, ExpectedRevision: severeResult.Revision + 1,
		SourceSystem: "integration-admin", ReplayID: "enforcement-severe-restoration",
		ActorID: "moderator", Rationale: "restoration approved",
	}); err != nil {
		t.Fatalf("restore severe suspension: %v", err)
	}
	assertEnforcementStanding(t, store, severeOwner, false)
	assertEnforcementPDSCounts(t, pds, 1, 1)

	restoredBody := newEnforcementBodyProbe(`{"text":"allowed after restoration"}`)
	response = publicWrite(severeOwner, restoredBody)
	if response.Code != http.StatusNoContent || publicHandlerCalls != 2 || restoredBody.reads == 0 {
		t.Fatalf("post-restoration status/handler/body work = %d/%d/%d", response.Code, publicHandlerCalls, restoredBody.reads)
	}
	assertEnforcementPDSCounts(t, pds, 2, 1)
}

func invokeAllowedEnforcement(t *testing.T, store *Store, owner syntax.DID, method, path string, next http.HandlerFunc) {
	t.Helper()
	handler := middleware.ModerationEnforcement(store, true, nil)(next)
	request := httptest.NewRequest(method, path, nil)
	request = request.WithContext(ctxkeys.WithDID(request.Context(), owner))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("allowed %s %s status = %d, want %d", method, path, response.Code, http.StatusNoContent)
	}
}

func assertEnforcementStanding(t *testing.T, store *Store, owner syntax.DID, suspended bool) {
	t.Helper()
	got, err := store.IsSuspended(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if got != suspended {
		t.Fatalf("IsSuspended(%s) = %t, want %t", owner, got, suspended)
	}
}

func assertEnforcementPDSCounts(t *testing.T, pds *recordingEnforcementPDS, mutations, ownerDeletions int) {
	t.Helper()
	if pds.mutations != mutations || pds.ownerDeletions != ownerDeletions {
		t.Fatalf("PDS mutation/owner-deletion calls = %d/%d, want %d/%d", pds.mutations, pds.ownerDeletions, mutations, ownerDeletions)
	}
}
