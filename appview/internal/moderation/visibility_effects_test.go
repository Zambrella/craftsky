package moderation

import (
	"errors"
	"testing"
)

func TestMapVisibilityEffects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		effects          []EffectType
		action           EffectAction
		wantFormalAction EffectAction
		wantValue        string
		wantAction       string
		wantOutput       bool
		wantErr          error
	}{
		{name: "formal only", effects: []EffectType{EffectFormalWarning}, action: EffectApply, wantFormalAction: EffectApply},
		{name: "formal reversal", effects: []EffectType{EffectFormalWarning}, action: EffectNegate, wantFormalAction: EffectNegate},
		{name: "viewer warn only", effects: []EffectType{EffectVisibilityWarn}, action: EffectApply, wantValue: "warn", wantAction: "apply", wantOutput: true},
		{name: "formal and viewer warn", effects: []EffectType{EffectFormalWarning, EffectVisibilityWarn}, action: EffectApply, wantFormalAction: EffectApply, wantValue: "warn", wantAction: "apply", wantOutput: true},
		{name: "hide", effects: []EffectType{EffectVisibilityHide}, action: EffectApply, wantValue: "hide", wantAction: "apply", wantOutput: true},
		{name: "takedown reversal", effects: []EffectType{EffectVisibilityTakedown}, action: EffectNegate, wantValue: "takedown", wantAction: "negate", wantOutput: true},
		{name: "account effects have no visibility output", effects: []EffectType{EffectStrike, EffectSevereSuspension}, action: EffectApply},
		{name: "multiple viewer effects rejected", effects: []EffectType{EffectVisibilityWarn, EffectVisibilityHide}, action: EffectApply, wantErr: ErrInvalidVisibilityEffects},
		{name: "unknown effect rejected", effects: []EffectType{"shadowBan"}, action: EffectApply, wantErr: ErrInvalidVisibilityEffects},
		{name: "unknown action rejected", effects: []EffectType{EffectVisibilityWarn}, action: "replace", wantErr: ErrInvalidVisibilityEffects},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			plan, err := MapVisibilityEffects(test.effects, test.action)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("MapVisibilityEffects() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr != nil {
				return
			}
			if plan.FormalWarningAction != test.wantFormalAction {
				t.Errorf("FormalWarningAction = %q, want %q", plan.FormalWarningAction, test.wantFormalAction)
			}
			if (plan.Output != nil) != test.wantOutput {
				t.Fatalf("Output present = %t, want %t", plan.Output != nil, test.wantOutput)
			}
			if plan.Output != nil && (plan.Output.Value != test.wantValue || plan.Output.Action != test.wantAction) {
				t.Errorf("Output = %+v, want value/action %s/%s", plan.Output, test.wantValue, test.wantAction)
			}
		})
	}
}
