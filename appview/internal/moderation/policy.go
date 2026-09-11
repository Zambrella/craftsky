package moderation

import (
	"errors"
	"strings"
)

var ErrInvalidDecision = errors.New("invalid moderation decision")

type Disposition string

const (
	DispositionViolation Disposition = "violation"
	DispositionNoAction  Disposition = "noAction"
)

type Reason string

const (
	ReasonHarassment           Reason = "harassment"
	ReasonHate                 Reason = "hate"
	ReasonSpam                 Reason = "spam"
	ReasonMisleading           Reason = "misleading"
	ReasonSuspectedAIGenerated Reason = "suspected_ai_generated"
	ReasonAdultOrGraphic       Reason = "adult_or_graphic"
	ReasonImpersonation        Reason = "impersonation"
	ReasonOffTopic             Reason = "off_topic"
	ReasonIntellectualProperty Reason = "intellectual_property"
	ReasonOther                Reason = "other"
)

var approvedReasons = []Reason{
	ReasonHarassment,
	ReasonHate,
	ReasonSpam,
	ReasonMisleading,
	ReasonSuspectedAIGenerated,
	ReasonAdultOrGraphic,
	ReasonImpersonation,
	ReasonOffTopic,
	ReasonIntellectualProperty,
	ReasonOther,
}

type EffectType string

const (
	EffectFormalWarning      EffectType = "formalWarning"
	EffectVisibilityWarn     EffectType = "visibilityWarn"
	EffectVisibilityHide     EffectType = "visibilityHide"
	EffectVisibilityTakedown EffectType = "visibilityTakedown"
	EffectStrike             EffectType = "strike"
	EffectSevereSuspension   EffectType = "severeSuspension"
)

type Decision struct {
	Disposition       Disposition
	Reason            Reason
	Evidence          string
	UserSafeDetail    string
	SeverityRationale string
	Consequences      []EffectType
}

func ApprovedReasons() []Reason {
	return append([]Reason(nil), approvedReasons...)
}

func ValidateDecision(decision Decision) error {
	if decision.Disposition == DispositionNoAction {
		if decision.Reason != "" || decision.UserSafeDetail != "" ||
			decision.SeverityRationale != "" || len(decision.Consequences) != 0 {
			return ErrInvalidDecision
		}
		return nil
	}
	if decision.Disposition != DispositionViolation ||
		!validReason(decision.Reason) || strings.TrimSpace(decision.Evidence) == "" ||
		len(decision.Consequences) == 0 {
		return ErrInvalidDecision
	}

	seen := make(map[EffectType]struct{}, len(decision.Consequences))
	visibilityEffects := 0
	hasSevere := false
	for _, effect := range decision.Consequences {
		if _, duplicate := seen[effect]; duplicate || !validEffect(effect) {
			return ErrInvalidDecision
		}
		seen[effect] = struct{}{}
		if isVisibilityEffect(effect) {
			visibilityEffects++
		}
		if effect == EffectSevereSuspension {
			hasSevere = true
		}
		if decision.Reason == ReasonOther && (effect == EffectStrike || effect == EffectSevereSuspension) {
			return ErrInvalidDecision
		}
	}
	if visibilityEffects > 1 {
		return ErrInvalidDecision
	}
	if hasSevere {
		if !severeReason(decision.Reason) || strings.TrimSpace(decision.SeverityRationale) == "" {
			return ErrInvalidDecision
		}
	} else if decision.SeverityRationale != "" {
		return ErrInvalidDecision
	}
	return nil
}

func validReason(reason Reason) bool {
	for _, approved := range approvedReasons {
		if reason == approved {
			return true
		}
	}
	return false
}

func validEffect(effect EffectType) bool {
	switch effect {
	case EffectFormalWarning, EffectVisibilityWarn, EffectVisibilityHide,
		EffectVisibilityTakedown, EffectStrike, EffectSevereSuspension:
		return true
	default:
		return false
	}
}

func isVisibilityEffect(effect EffectType) bool {
	return effect == EffectVisibilityWarn || effect == EffectVisibilityHide || effect == EffectVisibilityTakedown
}

func severeReason(reason Reason) bool {
	return reason == ReasonHarassment || reason == ReasonHate ||
		reason == ReasonAdultOrGraphic || reason == ReasonImpersonation
}
