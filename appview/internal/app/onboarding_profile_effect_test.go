package app

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

type recordingOnboardingProfilePutter struct {
	owner      syntax.DID
	generation int64
	record     any
	result     syntax.CID
	err        error
}

func (putter *recordingOnboardingProfilePutter) PutProfile(
	_ context.Context,
	_ auth.PDSClient,
	owner syntax.DID, generation int64, record any,
) (syntax.CID, error) {
	putter.owner, putter.generation, putter.record = owner, generation, record
	return putter.result, putter.err
}

func TestOnboardingProfileEffectAdapterReturnsAuthoritativeCID(t *testing.T) {
	putter := &recordingOnboardingProfilePutter{result: "bafyonboardingprofile"}
	adapter := onboardingProfileEffectAdapter{executor: putter}

	cid, err := adapter.PutOnboardingProfile(
		context.Background(), nil, auth.OnboardingProfileWrite{
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
		Owner: syntax.DID("did:plc:onboarding"), OwnerGeneration: 7, Record: record,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want mapped durable error", err)
	}
	gotRecord, ok := putter.record.(map[string]any)
	if !ok {
		t.Fatalf("mapped record type = %T, want map[string]any", putter.record)
	}
	if putter.owner != "did:plc:onboarding" || putter.generation != 7 ||
		gotRecord["$type"] != "social.craftsky.actor.profile" {
		t.Fatalf("mapped request = owner %s, generation %d, record %v", putter.owner, putter.generation, gotRecord)
	}
}
