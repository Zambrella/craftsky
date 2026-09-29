package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/safetyincident"
)

type safetyHealthFixture struct {
	health imagesafety.QueueHealth
}

func (fixture safetyHealthFixture) Health(context.Context) (imagesafety.QueueHealth, error) {
	return fixture.health, nil
}

type safetyWorkFixture []SafetyWorkItem

func (fixture safetyWorkFixture) SafeWork(context.Context) ([]SafetyWorkItem, error) {
	return fixture, nil
}

func TestSafetyAdminStatusOrdersSafeWorkAndAggregatesScannerHealth(t *testing.T) {
	now := time.Date(2030, 9, 22, 13, 0, 0, 0, time.UTC)
	overdue := now.Add(-time.Minute)
	upcoming := now.Add(time.Hour)
	handler := SafetyAdminStatusHandler(
		safetyHealthFixture{health: imagesafety.QueueHealth{
			Queued: 4, Leased: 2, DeadLettered: 1,
			OldestDueAge: 20 * time.Minute, MaxAttempts: 5,
		}},
		func() time.Time { return now },
		15*time.Minute,
		safetyWorkFixture{
			{Reference: "safe-report", Kind: "report", State: "untriaged", Priority: 5, CreatedAt: now.Add(-3 * time.Hour)},
			{Reference: "safe-appeal", Kind: "appeal", State: "pending", Priority: 4, CreatedAt: now.Add(-2 * time.Hour), Deadline: &upcoming},
			{Reference: "safe-detection", Kind: "detection", State: "detected", Priority: 3, CreatedAt: now.Add(-90 * time.Minute)},
			{Reference: "safe-retention", Kind: "retentionExpiry", State: "due", Priority: 2, CreatedAt: now.Add(-30 * time.Minute), Deadline: &upcoming},
			{Reference: "safe-awaiting", Kind: "intake", State: "awaitingInformation", Priority: 5, CreatedAt: now.Add(-4 * time.Hour)},
			{Reference: "safe-urgent", Kind: "incident", State: "untriaged", Priority: 1, CreatedAt: now.Add(-time.Hour), Deadline: &overdue, Owner: "founder", Cover: "trained-cover"},
		},
	)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/v1/admin/safety/status", nil))
	if recorder.Code != 200 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "blob") || strings.Contains(recorder.Body.String(), "did:plc") {
		t.Fatalf("restricted values leaked: %s", recorder.Body.String())
	}
	var body struct {
		ImageScans safetyImageScanStatus `json:"imageScans"`
		Work       []safetyWorkStatus    `json:"work"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.ImageScans.Queued != 4 || body.ImageScans.DeadLettered != 1 || !body.ImageScans.Alert {
		t.Fatalf("image scan health=%+v", body.ImageScans)
	}
	if len(body.Work) != 6 || body.Work[0].Reference != "safe-urgent" || !body.Work[0].Alert ||
		body.Work[0].Owner != "founder" || body.Work[0].Cover != "trained-cover" {
		t.Fatalf("prioritized work=%+v", body.Work)
	}
}

func TestSafetyAdminStatusExcludesRestrictedCanariesAndUsesCanonicalErrors(t *testing.T) {
	now := time.Date(2030, 9, 22, 13, 0, 0, 0, time.UTC)
	handler := SafetyAdminStatusHandler(
		safetyHealthFixture{}, func() time.Time { return now }, time.Minute,
		safetyWorkFixture{{
			Reference: "safe-reference", Kind: "urgentThreat", State: "untriaged", Priority: 1,
			CreatedAt: now.Add(-time.Hour), Owner: "assigned-helper", Cover: "named-cover",
			Sensitive: safetyincident.RestrictedFields{
				Content: "CANARY-CONTENT", Hash: "CANARY-HASH", Evidence: "CANARY-EVIDENCE",
				Contact: "CANARY-CONTACT", Credential: "CANARY-CREDENTIAL",
				InternalJudgment: "CANARY-JUDGMENT", ProviderPayload: "CANARY-PROVIDER",
			},
		}},
	)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/v1/admin/safety/status", nil))
	if recorder.Code != 200 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	for _, canary := range []string{"CANARY-CONTENT", "CANARY-HASH", "CANARY-EVIDENCE", "CANARY-CONTACT", "CANARY-CREDENTIAL", "CANARY-JUDGMENT", "CANARY-PROVIDER"} {
		if strings.Contains(recorder.Body.String(), canary) {
			t.Fatalf("restricted canary %s leaked: %s", canary, recorder.Body.String())
		}
	}

	errorRecorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/v1/admin/safety/status", nil)
	SafetyAdminStatusHandler(nil, nil, 0).ServeHTTP(errorRecorder, request)
	var failure map[string]any
	if err := json.NewDecoder(errorRecorder.Body).Decode(&failure); err != nil {
		t.Fatal(err)
	}
	if errorRecorder.Code != 503 || len(failure) != 3 || failure["error"] != "safety_status_unavailable" ||
		failure["message"] != "safety status unavailable" || failure["requestId"] != "" {
		t.Fatalf("non-canonical error: status=%d body=%v", errorRecorder.Code, failure)
	}
}
