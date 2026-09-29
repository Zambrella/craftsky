package retention

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPolicyCalculatesDeletionAndScopedHoldPrecedence(t *testing.T) {
	t.Parallel()
	created := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	evidenceID := uuid.New()
	policy := Policy{Periods: map[DataClass]time.Duration{DataClassRestrictedEvidence: 30 * 24 * time.Hour}}

	decision, err := policy.Decide(DataClassRestrictedEvidence, evidenceID, created, created.Add(31*24*time.Hour), nil)
	if err != nil || !decision.Due || decision.Held || !decision.DeleteAt.Equal(created.Add(30*24*time.Hour)) {
		t.Fatalf("unheld decision=%+v err=%v", decision, err)
	}
	decision, err = policy.Decide(DataClassRestrictedEvidence, evidenceID, created, created.Add(31*24*time.Hour), []HoldCoverage{{EvidenceID: evidenceID, ExpiresAt: created.Add(60 * 24 * time.Hour)}})
	if err != nil || decision.Due || !decision.Held || !decision.DeleteAt.Equal(created.Add(60*24*time.Hour)) {
		t.Fatalf("active hold decision=%+v err=%v", decision, err)
	}
	decision, err = policy.Decide(DataClassRestrictedEvidence, evidenceID, created, created.Add(61*24*time.Hour), []HoldCoverage{{EvidenceID: evidenceID, ExpiresAt: created.Add(60 * 24 * time.Hour)}})
	if err != nil || !decision.Due || decision.Held {
		t.Fatalf("expired hold decision=%+v err=%v", decision, err)
	}
	decision, err = policy.Decide(DataClassRestrictedEvidence, evidenceID, created, created.Add(31*24*time.Hour), []HoldCoverage{{EvidenceID: uuid.New(), ExpiresAt: created.Add(60 * 24 * time.Hour)}})
	if err != nil || !decision.Due || decision.Held {
		t.Fatalf("out-of-scope hold decision=%+v err=%v", decision, err)
	}
}
