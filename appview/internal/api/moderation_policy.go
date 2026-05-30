// appview/internal/api/moderation_policy.go
package api

import "time"

type ModerationPolicy struct {
	Hidden  bool
	Warning bool
	Value   ModerationValue
}

// ComputeModerationPolicy applies MVP moderation semantics over a set of
// candidate outputs for one subject/view: expired outputs are inactive,
// same-source later negates cancel prior matching applies, and hide/takedown
// dominate warn.
func ComputeModerationPolicy(outputs []ModerationOutputRow, now time.Time) ModerationPolicy {
	activeApplies := make([]ModerationOutputRow, 0, len(outputs))
	for _, output := range outputs {
		if output.Action != ModerationActionApply || outputExpired(output, now) {
			continue
		}
		if hasLaterMatchingNegate(output, outputs, now) {
			continue
		}
		activeApplies = append(activeApplies, output)
	}

	for _, output := range activeApplies {
		if output.Value == ModerationValueHide || output.Value == ModerationValueTakedown {
			return ModerationPolicy{Hidden: true, Value: output.Value}
		}
	}
	for _, output := range activeApplies {
		if output.Value == ModerationValueWarn {
			return ModerationPolicy{Warning: true, Value: ModerationValueWarn}
		}
	}
	return ModerationPolicy{}
}

func hasLaterMatchingNegate(apply ModerationOutputRow, outputs []ModerationOutputRow, now time.Time) bool {
	for _, candidate := range outputs {
		if candidate.Action != ModerationActionNegate || outputExpired(candidate, now) {
			continue
		}
		if !candidate.IndexedAt.After(apply.IndexedAt) && !candidate.CreatedAt.After(apply.CreatedAt) {
			continue
		}
		if sameModerationTargetAndValue(apply, candidate) {
			return true
		}
	}
	return false
}

func sameModerationTargetAndValue(left, right ModerationOutputRow) bool {
	if left.SourceDID != right.SourceDID || left.SubjectType != right.SubjectType || left.SubjectDID != right.SubjectDID || left.Value != right.Value {
		return false
	}
	if left.SubjectType == ModerationSubjectPost {
		return stringPtrValue(left.SubjectURI) == stringPtrValue(right.SubjectURI) && stringPtrValue(left.SubjectRkey) == stringPtrValue(right.SubjectRkey)
	}
	return true
}

func outputExpired(output ModerationOutputRow, now time.Time) bool {
	return output.ExpiresAt != nil && !output.ExpiresAt.After(now)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
