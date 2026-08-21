package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
)

type recordingOnboardingProfileWriter struct {
	requests []auth.OnboardingProfileWrite
	err      error
}

func (writer *recordingOnboardingProfileWriter) PutOnboardingProfile(
	_ context.Context,
	_ auth.PDSClient,
	request auth.OnboardingProfileWrite,
) error {
	writer.requests = append(writer.requests, request)
	return writer.err
}

func TestInitializeProfileUsesDurableOnboardingWriterWithStableGenerationIdentity(t *testing.T) {
	owner := syntax.DID("did:plc:onboarding-effect")
	pds := &mockPDS{
		getRecord: func(string, string, any) (string, error) {
			return "", auth.ErrRecordNotFound
		},
		putRecord: func(string, string, any) error {
			t.Fatal("raw PutRecord must not be reachable during onboarding")
			return nil
		},
	}
	writer := &recordingOnboardingProfileWriter{}
	for _, attemptID := range []uuid.UUID{uuid.New(), uuid.New()} {
		attempt := auth.CallbackAttempt{
			State: "oauth-parent", AttemptID: attemptID, Owner: owner,
			OwnerGeneration: 7, AuthEpoch: 3, Purpose: auth.LoginOAuthPurpose,
		}
		if err := auth.InitializeProfile(context.Background(), pds, attempt, writer); err != nil {
			t.Fatalf("InitializeProfile: %v", err)
		}
	}
	if len(writer.requests) != 2 {
		t.Fatalf("durable writes = %d, want 2", len(writer.requests))
	}
	first, second := writer.requests[0], writer.requests[1]
	if first.Owner != owner || first.OwnerGeneration != 7 || first.Record == nil {
		t.Fatalf("first durable request = %+v", first)
	}
	if first.OperationID == "" || first.MutationKey == "" ||
		first.OperationID != second.OperationID || first.MutationKey != second.MutationKey {
		t.Fatalf("generation-stable identities = (%q,%q), (%q,%q)",
			first.OperationID, first.MutationKey, second.OperationID, second.MutationKey)
	}
	if len(pds.putCalls) != 0 {
		t.Fatalf("raw PDS puts = %d, want zero", len(pds.putCalls))
	}
}

func TestInitializeProfileFailsClosedWithoutDurableWriter(t *testing.T) {
	pds := &mockPDS{
		getRecord: func(string, string, any) (string, error) {
			return "", auth.ErrRecordNotFound
		},
		putRecord: func(string, string, any) error {
			t.Fatal("raw PutRecord must not be used as a fallback")
			return nil
		},
	}
	attempt := auth.CallbackAttempt{
		State: "oauth-parent", AttemptID: uuid.New(), Owner: syntax.DID("did:plc:onboarding-no-writer"),
		OwnerGeneration: 1, AuthEpoch: 1, Purpose: auth.LoginOAuthPurpose,
	}
	err := auth.InitializeProfile(context.Background(), pds, attempt, nil)
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Fatalf("error = %v, want ErrProfileInitFailed", err)
	}
}
