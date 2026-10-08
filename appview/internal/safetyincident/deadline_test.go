package safetyincident

import (
	"errors"
	"testing"
	"time"
)

func TestDeadlinePolicyUsesOnlyExplicitApprovedTarget(t *testing.T) {
	received := time.Date(2030, 2, 1, 10, 0, 0, 0, time.FixedZone("test", 3600))

	empty, err := NewDeadlinePolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := empty.Calculate(WorkflowIntimateImage, received); !errors.Is(err, ErrDeadlineTargetNotConfigured) {
		t.Fatalf("unconfigured deadline error = %v", err)
	}
	if _, err := NewDeadlinePolicy(map[WorkflowClass]DeadlineTarget{
		WorkflowIntimateImage: {ResponseWithin: 0},
	}); !errors.Is(err, ErrDeadlineTargetNotConfigured) {
		t.Fatalf("zero target error = %v", err)
	}

	policy, err := NewDeadlinePolicy(map[WorkflowClass]DeadlineTarget{
		WorkflowIntimateImage: {
			ResponseWithin: 36 * time.Hour,
			AlertBefore:    []time.Duration{2 * time.Hour, 12 * time.Hour},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	deadline, err := policy.Calculate(WorkflowIntimateImage, received)
	if err != nil {
		t.Fatal(err)
	}
	wantDue := received.UTC().Add(36 * time.Hour)
	if !deadline.DueAt.Equal(wantDue) {
		t.Fatalf("due = %s, want %s", deadline.DueAt, wantDue)
	}
	if len(deadline.AlertAt) != 2 || !deadline.AlertAt[0].Equal(wantDue.Add(-12*time.Hour)) || !deadline.AlertAt[1].Equal(wantDue.Add(-2*time.Hour)) {
		t.Fatalf("alerts = %v", deadline.AlertAt)
	}
	if _, err := policy.Calculate(WorkflowAuthorityRequest, received); !errors.Is(err, ErrDeadlineTargetNotConfigured) {
		t.Fatalf("invented authority target: %v", err)
	}
}
