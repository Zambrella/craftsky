package api_test

import (
	"reflect"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/relationships"
)

type policyActorFixture struct {
	accountType   business.AccountType
	businessTypes []string
	offerings     []string
}

type authorizationOutcome struct {
	operation    relationships.Operation
	state        relationships.State
	ownsResource bool
	decision     relationships.Authorization
}

type policyNeutralitySnapshot struct {
	authorization []authorizationOutcome
	moderation    []api.ModerationPolicy
}

func TestBusinessPolicyNeutrality(t *testing.T) {
	assertBusinessPolicyNeutrality(t)
}

func assertBusinessPolicyNeutrality(t *testing.T) {
	t.Helper()
	regular := policyActorFixture{accountType: business.AccountTypeRegular}
	businessActor := policyActorFixture{
		accountType:   business.AccountTypeBusiness,
		businessTypes: []string{"dyer", "teacher"},
		offerings:     []string{"yarn", "classes"},
	}
	if reflect.DeepEqual(regular, businessActor) {
		t.Fatal("regular and business fixtures must differ only in business presentation state")
	}

	regularPolicy := capturePolicyNeutralitySnapshot(regular)
	businessPolicy := capturePolicyNeutralitySnapshot(businessActor)
	if !reflect.DeepEqual(businessPolicy.authorization, regularPolicy.authorization) {
		t.Fatalf("relationship authorization changed with business state\nregular: %#v\nbusiness: %#v", regularPolicy.authorization, businessPolicy.authorization)
	}
	if !reflect.DeepEqual(businessPolicy.moderation, regularPolicy.moderation) {
		t.Fatalf("moderation priority changed with business state\nregular: %#v\nbusiness: %#v", regularPolicy.moderation, businessPolicy.moderation)
	}
}

func capturePolicyNeutralitySnapshot(_ policyActorFixture) policyNeutralitySnapshot {
	operations := []relationships.Operation{
		relationships.OperationFollowCreate,
		relationships.OperationLikeCreate,
		relationships.OperationRepostCreate,
		relationships.OperationReplyCreate,
		relationships.OperationQuoteCreate,
		relationships.OperationMentionCreate,
		relationships.OperationFollowDelete,
		relationships.OperationLikeDelete,
		relationships.OperationRepostDelete,
		relationships.OperationContentDelete,
		relationships.OperationReport,
		relationships.OperationBlockCreate,
		relationships.OperationBlockDelete,
		relationships.OperationMuteCreate,
		relationships.OperationMuteDelete,
	}
	states := []relationships.State{
		{},
		{Muted: true},
		{Blocking: true},
		{BlockedBy: true},
		{Blocking: true, BlockedBy: true},
	}
	var snapshot policyNeutralitySnapshot
	for _, operation := range operations {
		for _, state := range states {
			for _, ownsResource := range []bool{false, true} {
				snapshot.authorization = append(snapshot.authorization, authorizationOutcome{
					operation:    operation,
					state:        state,
					ownsResource: ownsResource,
					decision:     relationships.Authorize(operation, state, ownsResource),
				})
			}
		}
	}

	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	postURI := "at://did:plc:actor/social.craftsky.feed.post/neutral"
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	sets := [][]api.ModerationOutputRow{
		{},
		{moderationOutput("warn", "did:plc:labeler", api.ModerationSubjectPost, "did:plc:actor", &postURI, api.ModerationValueWarn, api.ModerationActionApply, nil, now)},
		{moderationOutput("hide", "did:plc:labeler", api.ModerationSubjectPost, "did:plc:actor", &postURI, api.ModerationValueHide, api.ModerationActionApply, &future, now)},
		{moderationOutput("expired", "did:plc:labeler", api.ModerationSubjectPost, "did:plc:actor", &postURI, api.ModerationValueTakedown, api.ModerationActionApply, &past, now)},
		{
			moderationOutput("apply", "did:plc:labeler", api.ModerationSubjectPost, "did:plc:actor", &postURI, api.ModerationValueHide, api.ModerationActionApply, nil, now.Add(-time.Minute)),
			moderationOutput("negate", "did:plc:labeler", api.ModerationSubjectPost, "did:plc:actor", &postURI, api.ModerationValueHide, api.ModerationActionNegate, nil, now),
		},
	}
	for _, outputs := range sets {
		snapshot.moderation = append(snapshot.moderation, api.ComputeModerationPolicy(outputs, now))
	}
	return snapshot
}
