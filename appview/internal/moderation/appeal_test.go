package moderation

import (
	"errors"
	"testing"
)

func TestAppealLifecyclePolicy(t *testing.T) {
	if err := ValidateAppealConfirmation(false, AppealStatusNone); !errors.Is(err, ErrInvalidAppealTransition) {
		t.Fatalf("non-visible confirmation error = %v", err)
	}
	if err := ValidateAppealConfirmation(true, AppealStatusNone); err != nil {
		t.Fatalf("first visible confirmation: %v", err)
	}
	for _, status := range []AppealStatus{AppealStatusPending, AppealStatusUpheld, AppealStatusChanged} {
		if err := ValidateAppealConfirmation(true, status); !errors.Is(err, ErrInvalidAppealTransition) {
			t.Fatalf("second lifecycle from %q error = %v", status, err)
		}
	}
	for _, outcome := range []AppealStatus{AppealStatusUpheld, AppealStatusChanged} {
		if err := ValidateAppealResolution(AppealStatusPending, outcome); err != nil {
			t.Fatalf("pending to %q: %v", outcome, err)
		}
	}
	if err := ValidateAppealResolution(AppealStatusNone, AppealStatusUpheld); !errors.Is(err, ErrInvalidAppealTransition) {
		t.Fatalf("unconfirmed resolution error = %v", err)
	}
	if err := ValidateAppealResolution(AppealStatusPending, AppealStatusPending); !errors.Is(err, ErrInvalidAppealTransition) {
		t.Fatalf("pending outcome error = %v", err)
	}
}
