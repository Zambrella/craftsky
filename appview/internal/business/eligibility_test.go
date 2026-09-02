package business

import (
	"reflect"
	"testing"
	"time"
)

func TestEventEligibility(t *testing.T) {
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	base := EventPolicyInput{
		OwnerCurrent: true,
		AccountType:  AccountTypeBusiness,
		StartsAt:     now.Add(time.Hour),
		EndsAt:       now.Add(2 * time.Hour),
		Status:       "scheduled",
		AsOf:         now,
	}

	visitor := EvaluateEvent(base)
	if !visitor.VisitorDirect || !visitor.Upcoming || visitor.OwnerManagement {
		t.Fatalf("eligible visitor result = %+v", visitor)
	}
	if visitor.PublicSuppressionReasons == nil || visitor.UpcomingExclusionReasons == nil {
		t.Fatalf("diagnostic arrays must be non-nil: %+v", visitor)
	}

	ownerInput := base
	ownerInput.CallerIsOwner = true
	ownerInput.AccountType = AccountTypeRegular
	owner := EvaluateEvent(ownerInput)
	if !owner.OwnerManagement || !owner.DirectVisible || owner.VisitorDirect || owner.Upcoming {
		t.Fatalf("regular owner result = %+v", owner)
	}
	if want := []string{"owner-not-business"}; !reflect.DeepEqual(owner.PublicSuppressionReasons, want) {
		t.Fatalf("owner public reasons = %v, want %v", owner.PublicSuppressionReasons, want)
	}

	departedOwner := ownerInput
	departedOwner.OwnerCurrent = false
	if got := EvaluateEvent(departedOwner); got.OwnerManagement || got.DirectVisible {
		t.Fatalf("departed owner result = %+v", got)
	}

	blocked := base
	blocked.Blocked = true
	if got := EvaluateEvent(blocked); got.VisitorDirect || got.Upcoming || got.DirectVisible {
		t.Fatalf("blocked visitor result = %+v", got)
	}

	for name, mutate := range map[string]func(*EventPolicyInput){
		"ended": func(input *EventPolicyInput) {
			input.StartsAt = now.Add(-2 * time.Hour)
			input.EndsAt = now.Add(-time.Hour)
		},
		"cancelled": func(input *EventPolicyInput) { input.Status = "cancelled" },
		"postponed": func(input *EventPolicyInput) { input.Status = "postponed" },
	} {
		t.Run(name+" remains directly readable", func(t *testing.T) {
			input := base
			mutate(&input)
			got := EvaluateEvent(input)
			if !got.VisitorDirect || got.Upcoming || !got.DirectVisible {
				t.Fatalf("result = %+v", got)
			}
		})
	}

	suppressed := base
	suppressed.CallerIsOwner = true
	suppressed.OwnerCurrent = false
	suppressed.AccountType = AccountTypeRegular
	suppressed.EndsAt = suppressed.StartsAt
	suppressed.Moderated = true
	got := EvaluateEvent(suppressed)
	wantPublic := []string{"owner-not-business", "invalid-time-range", "record-moderated"}
	wantUpcoming := []string{"owner-not-business", "invalid-time-range", "record-moderated"}
	if !reflect.DeepEqual(got.PublicSuppressionReasons, wantPublic) || !reflect.DeepEqual(got.UpcomingExclusionReasons, wantUpcoming) {
		t.Fatalf("canonical reasons = (%v, %v), want (%v, %v)", got.PublicSuppressionReasons, got.UpcomingExclusionReasons, wantPublic, wantUpcoming)
	}

	overDuration := base
	overDuration.EndsAt = overDuration.StartsAt.Add(31*24*time.Hour + time.Second)
	if got := EvaluateEvent(overDuration); !reflect.DeepEqual(got.PublicSuppressionReasons, []string{"duration-exceeds-limit"}) {
		t.Fatalf("over-duration reasons = %v", got.PublicSuppressionReasons)
	}

	if !IsBusinessClassified(true, AccountTypeBusiness) || IsBusinessClassified(true, AccountTypeRegular) || IsBusinessClassified(false, AccountTypeBusiness) {
		t.Fatal("business classification did not require current membership plus business type")
	}

	if _, ok := reflect.TypeOf(EventPolicyInput{}).FieldByName("Declaration"); ok {
		t.Fatal("event eligibility depends on declaration presence")
	}
}
