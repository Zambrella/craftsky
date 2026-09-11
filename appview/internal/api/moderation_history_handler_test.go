package api_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/testdb"
)

const moderationOwnerReadTestDDL = `
CREATE TABLE moderation_cases (
 id UUID PRIMARY KEY, subject_key TEXT NOT NULL, subject_type TEXT NOT NULL,
 subject_did TEXT NOT NULL, subject_collection TEXT, subject_rkey TEXT,
 subject_uri TEXT, subject_cid_snapshot TEXT, owner_did TEXT NOT NULL,
 safe_snapshot JSONB NOT NULL, state TEXT NOT NULL, revision BIGINT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL, resolved_at TIMESTAMPTZ, updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE moderation_account_standings (
 owner_did TEXT PRIMARY KEY, active_strike_count INTEGER NOT NULL DEFAULT 0,
 threshold_suspended BOOLEAN NOT NULL DEFAULT false, severe_suspended BOOLEAN NOT NULL DEFAULT false,
 effective_suspended BOOLEAN GENERATED ALWAYS AS (threshold_suspended OR severe_suspended) STORED,
 revision BIGINT NOT NULL DEFAULT 0, updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE moderation_case_events (
 id UUID PRIMARY KEY, case_id UUID NOT NULL REFERENCES moderation_cases(id), event_type TEXT NOT NULL,
 actor_id TEXT NOT NULL, source_system TEXT NOT NULL, replay_id TEXT NOT NULL,
 request_fingerprint BYTEA NOT NULL, expected_revision BIGINT NOT NULL,
 result_revision BIGINT NOT NULL, created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE moderation_decisions (
 id UUID PRIMARY KEY, case_id UUID NOT NULL REFERENCES moderation_cases(id),
 case_event_id UUID NOT NULL UNIQUE REFERENCES moderation_case_events(id), disposition TEXT NOT NULL,
 reason TEXT, internal_evidence_notes TEXT, user_safe_detail TEXT, severity_rationale TEXT,
 created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE moderation_effect_events (
 id UUID PRIMARY KEY, case_id UUID NOT NULL REFERENCES moderation_cases(id),
 case_event_id UUID NOT NULL REFERENCES moderation_case_events(id), logical_effect_id UUID NOT NULL,
 effect_type TEXT NOT NULL, action TEXT NOT NULL, moderation_output_id TEXT, rationale TEXT,
 created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE moderation_appeals (
 case_id UUID PRIMARY KEY REFERENCES moderation_cases(id), confirmed_event_id UUID NOT NULL,
 resolved_event_id UUID, status TEXT NOT NULL, confirmed_at TIMESTAMPTZ NOT NULL, resolved_at TIMESTAMPTZ
);
`

func TestModerationHistoryHandlersAuthorizeOwnerAndExposeOnlyCuratedChronology(t *testing.T) {
	pool := testdb.WithSchema(t, moderationOwnerReadTestDDL)
	ctx := context.Background()
	owner := syntax.DID("did:plc:owner")
	other := syntax.DID("did:plc:other")
	caseID := uuid.New()
	noActionCaseID := uuid.New()
	base := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	uri := "at://did:plc:owner/social.craftsky.feed.post/3deleted"
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,subject_cid_snapshot,owner_did,safe_snapshot,state,revision,created_at,resolved_at,updated_at)
		VALUES($1,'post:deleted','post',$2,'social.craftsky.feed.post','3deleted',$3,'bafy-before-delete',$2,'{"type":"post","privateSource":"raw-snapshot-sentinel"}','resolved',3,$4,$4,$4),
		($5,'account:no-action','account',$2,NULL,NULL,NULL,NULL,$2,'{"type":"account","did":"did:plc:owner"}','resolved',1,$4,$4,$4)`, caseID, owner, uri, base, noActionCaseID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_account_standings(owner_did,active_strike_count,threshold_suspended,severe_suspended,revision,updated_at) VALUES($1,1,false,false,3,$2)`, owner, base); err != nil {
		t.Fatal(err)
	}

	eventIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for index, eventID := range eventIDs {
		occurredAt := base.Add(time.Duration(index+1) * time.Minute)
		if _, err := pool.Exec(ctx, `INSERT INTO moderation_case_events(id,case_id,event_type,actor_id,source_system,replay_id,request_fingerprint,expected_revision,result_revision,created_at) VALUES($1,$2,$3,'private-moderator-sentinel','private-source-sentinel',$4,$5,$6,$7,$8)`, eventID, caseID, []string{"decision", "effectsChanged", "effectsChanged"}[index], "private-replay-"+eventID.String(), make([]byte, 32), index, index+1, occurredAt); err != nil {
			t.Fatal(err)
		}
		action := []string{"apply", "negate", "apply"}[index]
		if _, err := pool.Exec(ctx, `INSERT INTO moderation_effect_events(id,case_id,case_event_id,logical_effect_id,effect_type,action,rationale,created_at) VALUES($1,$2,$3,$4,'formalWarning',$5,'private-rationale-sentinel',$6)`, uuid.New(), caseID, eventID, uuid.New(), action, occurredAt); err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			if _, err := pool.Exec(ctx, `INSERT INTO moderation_decisions(id,case_id,case_event_id,disposition,reason,internal_evidence_notes,user_safe_detail,severity_rationale,created_at) VALUES($1,$2,$3,'violation','spam','private-evidence-sentinel','owner-safe-detail','private-severity-sentinel',$4)`, uuid.New(), caseID, eventID, occurredAt); err != nil {
				t.Fatal(err)
			}
		}
	}
	noActionEventID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_case_events(id,case_id,event_type,actor_id,source_system,replay_id,request_fingerprint,expected_revision,result_revision,created_at) VALUES($1,$2,'decision','private-no-action-moderator','admin-api','private-no-action-replay',$3,0,1,$4)`, noActionEventID, noActionCaseID, make([]byte, 32), base.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_decisions(id,case_id,case_event_id,disposition,internal_evidence_notes,created_at) VALUES($1,$2,$3,'noAction','private-no-action-notes',$4)`, uuid.New(), noActionCaseID, noActionEventID, base.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}

	store := moderation.NewStore(pool)
	handler := api.ModerationHistoryHandler(store)
	first := serveOwnerModeration(t, handler, owner, "/v1/moderation/history?limit=2", "")
	if first.Code != http.StatusOK {
		t.Fatalf("first page = %d %s", first.Code, first.Body.String())
	}
	var firstPage moderation.OwnerHistoryPage
	if err := json.Unmarshal(first.Body.Bytes(), &firstPage); err != nil {
		t.Fatal(err)
	}
	if len(firstPage.Items) != 2 || firstPage.Cursor == "" || !firstPage.Items[0].OccurredAt.After(firstPage.Items[1].OccurredAt) {
		t.Fatalf("first page = %+v", firstPage)
	}
	second := serveOwnerModeration(t, handler, owner, "/v1/moderation/history?limit=2&cursor="+firstPage.Cursor, "")
	var secondPage moderation.OwnerHistoryPage
	if second.Code != http.StatusOK || json.Unmarshal(second.Body.Bytes(), &secondPage) != nil || len(secondPage.Items) != 1 || secondPage.Cursor != "" {
		t.Fatalf("second page = %d %s", second.Code, second.Body.String())
	}
	all := append(firstPage.Items, secondPage.Items...)
	if all[2].Reason != moderation.ReasonSpam || all[2].UserSafeDetail != "owner-safe-detail" || all[2].SafeSnapshot.URI.String() != uri || all[2].SafeSnapshot.CID != "bafy-before-delete" {
		t.Fatalf("oldest owner item = %+v", all[2])
	}
	wire := first.Body.String() + second.Body.String()
	for _, private := range []string{"raw-snapshot-sentinel", "private-moderator-sentinel", "private-source-sentinel", "private-replay-", "private-rationale-sentinel", "private-evidence-sentinel", "private-severity-sentinel", "private-no-action"} {
		if strings.Contains(wire, private) {
			t.Fatalf("owner wire leaked %q: %s", private, wire)
		}
	}
	for _, internalID := range eventIDs {
		if strings.Contains(wire, internalID.String()) {
			t.Fatalf("owner wire leaked event ID %s: %s", internalID, wire)
		}
	}
	reference, _ := moderation.FormatCaseReference(caseID)
	if strings.Count(wire, reference) != 3 {
		t.Fatalf("canonical case reference count in %s", wire)
	}

	entryHandler := api.ModerationHistoryEntryHandler(store)
	mixedCase := strings.ToUpper(reference)
	ownerEntry := serveOwnerModeration(t, entryHandler, owner, "/v1/moderation/history/"+mixedCase, mixedCase)
	if ownerEntry.Code != http.StatusOK || !strings.Contains(ownerEntry.Body.String(), reference) {
		t.Fatalf("mixed-case owner entry = %d %s", ownerEntry.Code, ownerEntry.Body.String())
	}
	otherEntry := serveOwnerModeration(t, entryHandler, other, "/v1/moderation/history/"+mixedCase, mixedCase)
	if otherEntry.Code != http.StatusNotFound || strings.Contains(otherEntry.Body.String(), reference) {
		t.Fatalf("cross-owner entry = %d %s", otherEntry.Code, otherEntry.Body.String())
	}
	otherHistory := serveOwnerModeration(t, handler, other, "/v1/moderation/history", "")
	if otherHistory.Code != http.StatusOK || otherHistory.Body.String() != "{\"items\":[]}\n" {
		t.Fatalf("cross-owner history = %d %s", otherHistory.Code, otherHistory.Body.String())
	}
	standingHandler := api.ModerationStandingHandler(store)
	ownerStanding := serveOwnerModeration(t, standingHandler, owner, "/v1/moderation/standing", "")
	otherStanding := serveOwnerModeration(t, standingHandler, other, "/v1/moderation/standing", "")
	if ownerStanding.Code != http.StatusOK || !strings.Contains(ownerStanding.Body.String(), `"activeStrikeCount":1`) {
		t.Fatalf("owner standing = %d %s", ownerStanding.Code, ownerStanding.Body.String())
	}
	if otherStanding.Code != http.StatusOK || !strings.Contains(otherStanding.Body.String(), `"activeStrikeCount":0`) {
		t.Fatalf("other standing = %d %s", otherStanding.Code, otherStanding.Body.String())
	}
}

func TestModerationHistoryHandlerRejectsNonCanonicalCursorJSON(t *testing.T) {
	pool := testdb.WithSchema(t, moderationOwnerReadTestDDL)
	cursor := base64.RawURLEncoding.EncodeToString([]byte(`{"createdAt":"2026-09-10T12:00:00Z","id":"` + uuid.NewString() + `","internalId":"private"}`))
	response := serveOwnerModeration(t, api.ModerationHistoryHandler(moderation.NewStore(pool)), syntax.DID("did:plc:owner"), "/v1/moderation/history?cursor="+cursor, "")
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_cursor"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func serveOwnerModeration(t *testing.T, handler http.Handler, owner syntax.DID, target, reference string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request = request.WithContext(ctxkeys.WithDID(request.Context(), owner))
	if reference != "" {
		request.SetPathValue("caseReference", reference)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
