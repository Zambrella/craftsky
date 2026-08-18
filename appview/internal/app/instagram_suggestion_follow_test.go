package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/followwrite"
	"social.craftsky/appview/internal/instagram"
)

type stubDeterministicFollowCoordinator struct {
	request followwrite.DeterministicRequest
	result  followwrite.DeterministicResult
	err     error
}

func (stub *stubDeterministicFollowCoordinator) ExecuteDeterministic(
	_ context.Context,
	request followwrite.DeterministicRequest,
) (followwrite.DeterministicResult, error) {
	stub.request = request
	return stub.result, stub.err
}

func TestInstagramSuggestionFollowAdapterUsesOnlyDeterministicCoordinator(t *testing.T) {
	owner := syntax.DID("did:plc:importer")
	target := syntax.DID("did:plc:target")
	createdAt := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	coordinator := &stubDeterministicFollowCoordinator{result: followwrite.DeterministicResult{
		RecordURI: syntax.ATURI("at://did:plc:importer/app.bsky.graph.follow/3lsuggestion"),
		RecordCID: "bafycid",
	}}
	adapter := instagramSuggestionFollowAdapter{coordinator: coordinator}

	result, err := adapter.FollowSuggestion(context.Background(), instagram.SuggestionFollowRequest{
		OperationID: "instagram-suggestion:one", Owner: owner, Target: target,
		OwnerGeneration: 7, TargetGeneration: 11, SessionID: "oauth-session",
		Rkey: syntax.RecordKey("3lsuggestion"), CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("FollowSuggestion: %v", err)
	}
	if result.Outcome != instagram.SuggestionFollowed || result.RecordURI != coordinator.result.RecordURI || result.RecordCID != "bafycid" {
		t.Fatalf("result = %+v", result)
	}
	if coordinator.request.Owner != owner || coordinator.request.Target != target ||
		coordinator.request.OwnerGeneration != 7 || coordinator.request.SessionID != "oauth-session" ||
		coordinator.request.OperationID != "instagram-suggestion:one" || coordinator.request.Rkey != "3lsuggestion" ||
		!coordinator.request.CreatedAt.Equal(createdAt) {
		t.Fatalf("coordinator request = %+v", coordinator.request)
	}
}

func TestInstagramSuggestionFollowAdapterPropagatesUncertainOutcome(t *testing.T) {
	coordinator := &stubDeterministicFollowCoordinator{err: followwrite.ErrOutcomeUncertain}
	adapter := instagramSuggestionFollowAdapter{coordinator: coordinator}
	_, err := adapter.FollowSuggestion(context.Background(), instagram.SuggestionFollowRequest{
		OperationID: "instagram-suggestion:two",
		Owner:       syntax.DID("did:plc:importer"), Target: syntax.DID("did:plc:target"),
		OwnerGeneration: 1, TargetGeneration: 1, SessionID: "oauth-session",
		Rkey: syntax.RecordKey("3lsecond"), CreatedAt: time.Now().UTC(),
	})
	if !errors.Is(err, followwrite.ErrOutcomeUncertain) {
		t.Fatalf("error = %v, want uncertain outcome", err)
	}
}
