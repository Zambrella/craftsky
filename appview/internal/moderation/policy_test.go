package moderation

import (
	"errors"
	"testing"
)

func TestValidateDecision(t *testing.T) {
	t.Parallel()

	validViolation := Decision{
		Disposition: DispositionViolation,
		Reason:      ReasonSpam,
		Evidence:    "confirmed repeated unsolicited promotion",
		Consequences: []EffectType{
			EffectFormalWarning,
		},
	}
	tests := []struct {
		name     string
		decision Decision
		wantErr  error
	}{
		{name: "formal warning", decision: validViolation},
		{name: "viewer warning only", decision: replaceEffects(validViolation, EffectVisibilityWarn)},
		{name: "formal and viewer warnings", decision: replaceEffects(validViolation, EffectFormalWarning, EffectVisibilityWarn)},
		{name: "no action", decision: Decision{Disposition: DispositionNoAction, Evidence: "allegation not substantiated"}},
		{name: "violation needs consequence", decision: replaceEffects(validViolation), wantErr: ErrInvalidDecision},
		{name: "violation needs evidence", decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Consequences: []EffectType{EffectStrike}}, wantErr: ErrInvalidDecision},
		{name: "no action prohibits consequence", decision: Decision{Disposition: DispositionNoAction, Consequences: []EffectType{EffectFormalWarning}}, wantErr: ErrInvalidDecision},
		{name: "no action prohibits reason", decision: Decision{Disposition: DispositionNoAction, Reason: ReasonSpam}, wantErr: ErrInvalidDecision},
		{name: "unknown disposition", decision: Decision{Disposition: "dismissed"}, wantErr: ErrInvalidDecision},
		{name: "unknown reason", decision: Decision{Disposition: DispositionViolation, Reason: "unknown", Evidence: "evidence", Consequences: []EffectType{EffectFormalWarning}}, wantErr: ErrInvalidDecision},
		{name: "duplicate effect", decision: replaceEffects(validViolation, EffectStrike, EffectStrike), wantErr: ErrInvalidDecision},
		{name: "multiple visibility effects", decision: replaceEffects(validViolation, EffectVisibilityWarn, EffectVisibilityHide), wantErr: ErrInvalidDecision},
		{name: "other formal warning", decision: Decision{Disposition: DispositionViolation, Reason: ReasonOther, Evidence: "evidence", Consequences: []EffectType{EffectFormalWarning}}},
		{name: "other visibility", decision: Decision{Disposition: DispositionViolation, Reason: ReasonOther, Evidence: "evidence", Consequences: []EffectType{EffectVisibilityTakedown}}},
		{name: "other cannot strike", decision: Decision{Disposition: DispositionViolation, Reason: ReasonOther, Evidence: "evidence", Consequences: []EffectType{EffectStrike}}, wantErr: ErrInvalidDecision},
		{name: "other cannot suspend", decision: Decision{Disposition: DispositionViolation, Reason: ReasonOther, Evidence: "evidence", SeverityRationale: "severe", Consequences: []EffectType{EffectSevereSuspension}}, wantErr: ErrInvalidDecision},
		{name: "severe reason needs rationale", decision: Decision{Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence", Consequences: []EffectType{EffectSevereSuspension}}, wantErr: ErrInvalidDecision},
		{name: "severe only", decision: Decision{Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence", SeverityRationale: "immediate safety risk", Consequences: []EffectType{EffectSevereSuspension}}},
		{name: "severe and independent strike", decision: Decision{Disposition: DispositionViolation, Reason: ReasonHarassment, Evidence: "evidence", SeverityRationale: "immediate safety risk", Consequences: []EffectType{EffectSevereSuspension, EffectStrike}}},
		{name: "ineligible severe reason", decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", SeverityRationale: "severe", Consequences: []EffectType{EffectSevereSuspension}}, wantErr: ErrInvalidDecision},
		{name: "rationale without severe effect", decision: Decision{Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence", SeverityRationale: "severe", Consequences: []EffectType{EffectStrike}}, wantErr: ErrInvalidDecision},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateDecision(test.decision)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ValidateDecision() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestEveryNamedReasonExceptOtherMayExplicitlyIssueStrike(t *testing.T) {
	t.Parallel()

	for _, reason := range ApprovedReasons() {
		decision := Decision{
			Disposition: DispositionViolation,
			Reason:      reason,
			Evidence:    "evidence",
			Consequences: []EffectType{
				EffectStrike,
			},
		}
		err := ValidateDecision(decision)
		if reason == ReasonOther {
			if !errors.Is(err, ErrInvalidDecision) {
				t.Fatalf("reason %q strike error = %v, want ErrInvalidDecision", reason, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("reason %q explicit strike error = %v, want nil", reason, err)
		}
	}
}

func TestOnlyApprovedSevereReasonsMaySuspend(t *testing.T) {
	t.Parallel()

	eligible := map[Reason]bool{
		ReasonHarassment:     true,
		ReasonHate:           true,
		ReasonAdultOrGraphic: true,
		ReasonImpersonation:  true,
	}
	for _, reason := range ApprovedReasons() {
		decision := Decision{
			Disposition:       DispositionViolation,
			Reason:            reason,
			Evidence:          "evidence",
			SeverityRationale: "immediate safety risk",
			Consequences:      []EffectType{EffectSevereSuspension},
		}
		err := ValidateDecision(decision)
		if eligible[reason] && err != nil {
			t.Fatalf("reason %q severe suspension error = %v, want nil", reason, err)
		}
		if !eligible[reason] && !errors.Is(err, ErrInvalidDecision) {
			t.Fatalf("reason %q severe suspension error = %v, want ErrInvalidDecision", reason, err)
		}
	}
}

func replaceEffects(decision Decision, effects ...EffectType) Decision {
	decision.Consequences = effects
	return decision
}
