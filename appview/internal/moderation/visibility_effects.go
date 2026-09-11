package moderation

import "errors"

var ErrInvalidVisibilityEffects = errors.New("invalid moderation visibility effects")

type EffectAction string

const (
	EffectApply  EffectAction = "apply"
	EffectNegate EffectAction = "negate"
)

type VisibilityOutput struct {
	Value  string
	Action string
}

type VisibilityPlan struct {
	FormalWarningAction EffectAction
	Output              *VisibilityOutput
}

func MapVisibilityEffects(effects []EffectType, action EffectAction) (VisibilityPlan, error) {
	if action != EffectApply && action != EffectNegate {
		return VisibilityPlan{}, ErrInvalidVisibilityEffects
	}
	plan := VisibilityPlan{}
	for _, effect := range effects {
		switch effect {
		case EffectFormalWarning:
			plan.FormalWarningAction = action
		case EffectVisibilityWarn, EffectVisibilityHide, EffectVisibilityTakedown:
			if plan.Output != nil {
				return VisibilityPlan{}, ErrInvalidVisibilityEffects
			}
			plan.Output = &VisibilityOutput{Value: visibilityValue(effect), Action: string(action)}
		case EffectStrike, EffectSevereSuspension:
		default:
			return VisibilityPlan{}, ErrInvalidVisibilityEffects
		}
	}
	return plan, nil
}

func visibilityValue(effect EffectType) string {
	switch effect {
	case EffectVisibilityWarn:
		return "warn"
	case EffectVisibilityHide:
		return "hide"
	case EffectVisibilityTakedown:
		return "takedown"
	default:
		return ""
	}
}
