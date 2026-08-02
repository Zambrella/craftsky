package scheduledposts

import (
	"errors"
	"testing"
)

func TestStatusCountsTowardCapacity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status Status
		want   bool
	}{
		{status: StatusScheduled, want: true},
		{status: StatusPublishing, want: true},
		{status: StatusRetrying, want: true},
		{status: StatusNeedsAttention, want: true},
		{status: StatusPublished},
		{status: StatusDeleted},
		{status: Status("unknown")},
	}

	for _, test := range tests {
		t.Run(string(test.status), func(t *testing.T) {
			t.Parallel()
			if got := test.status.CountsTowardCapacity(); got != test.want {
				t.Fatalf("Status(%q).CountsTowardCapacity() = %t, want %t", test.status, got, test.want)
			}
		})
	}
}

func TestLifecycleStateRules(t *testing.T) {
	t.Parallel()

	statuses := []Status{
		StatusScheduled,
		StatusPublishing,
		StatusRetrying,
		StatusNeedsAttention,
		StatusPublished,
		StatusDeleted,
	}
	allowed := map[[2]Status]bool{
		{StatusScheduled, StatusPublishing}:      true,
		{StatusScheduled, StatusDeleted}:         true,
		{StatusPublishing, StatusRetrying}:       true,
		{StatusPublishing, StatusNeedsAttention}: true,
		{StatusPublishing, StatusPublished}:      true,
		{StatusRetrying, StatusPublishing}:       true,
		{StatusRetrying, StatusScheduled}:        true,
		{StatusRetrying, StatusNeedsAttention}:   true,
		{StatusRetrying, StatusDeleted}:          true,
		{StatusNeedsAttention, StatusScheduled}:  true,
		{StatusNeedsAttention, StatusPublishing}: true,
		{StatusNeedsAttention, StatusDeleted}:    true,
	}

	for _, from := range statuses {
		for _, to := range statuses {
			err := ValidateStatusTransition(from, to)
			if allowed[[2]Status{from, to}] {
				if err != nil {
					t.Errorf("ValidateStatusTransition(%q, %q) error = %v, want nil", from, to, err)
				}
				continue
			}
			if !errors.Is(err, ErrInvalidStatusTransition) {
				t.Errorf("ValidateStatusTransition(%q, %q) error = %v, want ErrInvalidStatusTransition", from, to, err)
			}
		}
	}

	for _, status := range []Status{StatusScheduled, StatusRetrying, StatusNeedsAttention} {
		if !status.AllowsMemberMutation() {
			t.Errorf("status %q must allow member mutation", status)
		}
	}
	for _, status := range []Status{StatusPublishing, StatusPublished, StatusDeleted, Status("unknown")} {
		if status.AllowsMemberMutation() {
			t.Errorf("status %q must lock member mutation", status)
		}
	}

	for _, status := range []Status{StatusScheduled, StatusRetrying} {
		if !status.AutomaticClaimable() {
			t.Errorf("status %q must be automatically claimable", status)
		}
	}
	for _, status := range []Status{StatusPublishing, StatusNeedsAttention, StatusPublished, StatusDeleted} {
		if status.AutomaticClaimable() {
			t.Errorf("status %q must not be automatically claimable", status)
		}
	}

	if err := ValidateWorkerVersion(4, 4); err != nil {
		t.Fatalf("ValidateWorkerVersion(4, 4) error = %v, want nil", err)
	}
	if err := ValidateWorkerVersion(5, 4); !errors.Is(err, ErrStaleWorkerVersion) {
		t.Fatalf("ValidateWorkerVersion(5, 4) error = %v, want ErrStaleWorkerVersion", err)
	}
}
