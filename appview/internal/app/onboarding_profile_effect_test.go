package app

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/pdseffects"
)

type recordingOnboardingProfilePutter struct {
	request pdseffects.OnboardingProfileRequest
	result  pdseffects.RecordResult
	err     error
}

func (putter *recordingOnboardingProfilePutter) PutProfile(
	_ context.Context,
	_ auth.PDSClient,
	request pdseffects.OnboardingProfileRequest,
) (pdseffects.RecordResult, error) {
	putter.request = request
	return putter.result, putter.err
}

func TestOnboardingProfileEffectAdapterReturnsAuthoritativeCID(t *testing.T) {
	putter := &recordingOnboardingProfilePutter{
		result: pdseffects.RecordResult{CID: "bafyonboardingprofile"},
	}
	adapter := onboardingProfileEffectAdapter{executor: putter}

	cid, err := adapter.PutOnboardingProfile(
		context.Background(), nil, auth.OnboardingProfileWrite{
			OperationID: "onboarding:cid", MutationKey: "generation:cid",
			Owner: "did:plc:onboarding-cid", OwnerGeneration: 1,
			Record: map[string]any{"crafts": []string{}},
		},
	)
	if err != nil || cid != "bafyonboardingprofile" {
		t.Fatalf("PutOnboardingProfile CID/error = %q/%v", cid, err)
	}
}

func TestOnboardingProfileEffectAdapterMapsOnlyTheNarrowProfileRequest(t *testing.T) {
	wantErr := errors.New("durable attempt unavailable")
	putter := &recordingOnboardingProfilePutter{err: wantErr}
	adapter := onboardingProfileEffectAdapter{executor: putter}
	record := map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []string{}}
	_, err := adapter.PutOnboardingProfile(context.Background(), nil, auth.OnboardingProfileWrite{
		OperationID: "onboarding:one", MutationKey: "generation:one",
		Owner: syntax.DID("did:plc:onboarding"), OwnerGeneration: 7, Record: record,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want mapped durable error", err)
	}
	got := putter.request
	gotRecord, ok := got.Record.(map[string]any)
	if !ok {
		t.Fatalf("mapped record type = %T, want map[string]any", got.Record)
	}
	if got.OperationID != "onboarding:one" || got.MutationKey != "generation:one" ||
		got.Owner != "did:plc:onboarding" || got.OwnerGeneration != 7 ||
		gotRecord["$type"] != "social.craftsky.actor.profile" || got.ExpectedCID != "" {
		t.Fatalf("mapped request = %+v", got)
	}
}
