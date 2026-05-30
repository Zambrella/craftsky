// appview/internal/api/moderation_policy_test.go
package api_test

import (
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func TestComputeModerationPolicy_NegationExpiryAndPrecedence(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	postURI := "at://did:plc:bob/social.craftsky.feed.post/3lf2abc"

	policy := api.ComputeModerationPolicy([]api.ModerationOutputRow{
		moderationOutput("same-source hide apply", "did:plc:labeler", api.ModerationSubjectPost, "did:plc:bob", &postURI, api.ModerationValueHide, api.ModerationActionApply, nil, now.Add(-3*time.Minute)),
		moderationOutput("same-source hide negate", "did:plc:labeler", api.ModerationSubjectPost, "did:plc:bob", &postURI, api.ModerationValueHide, api.ModerationActionNegate, nil, now.Add(-2*time.Minute)),
		moderationOutput("cross-source hide remains", "did:plc:ozone", api.ModerationSubjectPost, "did:plc:bob", &postURI, api.ModerationValueHide, api.ModerationActionApply, nil, now.Add(-time.Minute)),
		moderationOutput("active warn dominated", "did:plc:labeler", api.ModerationSubjectPost, "did:plc:bob", &postURI, api.ModerationValueWarn, api.ModerationActionApply, &future, now.Add(-time.Minute)),
		moderationOutput("expired takedown ignored", "did:plc:labeler", api.ModerationSubjectPost, "did:plc:bob", &postURI, api.ModerationValueTakedown, api.ModerationActionApply, &past, now.Add(-time.Minute)),
	}, now)

	if !policy.Hidden {
		t.Fatalf("policy.Hidden = false, want true from cross-source hide")
	}
	if policy.Warning {
		t.Fatalf("policy.Warning = true, want false because hide dominates warn")
	}
	if policy.Value != api.ModerationValueHide {
		t.Fatalf("policy.Value = %q, want hide", policy.Value)
	}
}

func TestComputeModerationPolicy_WarnOnlyAndHideTakedownEnforcement(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	warn := api.ComputeModerationPolicy([]api.ModerationOutputRow{
		moderationOutput("warn", "did:plc:labeler", api.ModerationSubjectAccount, "did:plc:bob", nil, api.ModerationValueWarn, api.ModerationActionApply, nil, now),
	}, now)
	if warn.Hidden || !warn.Warning || warn.Value != api.ModerationValueWarn {
		t.Fatalf("warn policy = %+v, want visible warning", warn)
	}

	for _, value := range []api.ModerationValue{api.ModerationValueHide, api.ModerationValueTakedown} {
		value := value
		t.Run(string(value), func(t *testing.T) {
			t.Parallel()
			policy := api.ComputeModerationPolicy([]api.ModerationOutputRow{
				moderationOutput(string(value), "did:plc:labeler", api.ModerationSubjectAccount, "did:plc:bob", nil, value, api.ModerationActionApply, nil, now),
			}, now)
			if !policy.Hidden || policy.Warning {
				t.Fatalf("policy = %+v, want hidden with no warning", policy)
			}
		})
	}
}

func moderationOutput(id, source string, subjectType api.ModerationSubjectType, subjectDID string, subjectURI *string, value api.ModerationValue, action api.ModerationAction, expiresAt *time.Time, indexedAt time.Time) api.ModerationOutputRow {
	return api.ModerationOutputRow{
		ID:          id,
		SourceDID:   source,
		SubjectType: subjectType,
		SubjectDID:  subjectDID,
		SubjectURI:  subjectURI,
		Value:       value,
		Action:      action,
		ExpiresAt:   expiresAt,
		CreatedAt:   indexedAt,
		IndexedAt:   indexedAt,
	}
}
