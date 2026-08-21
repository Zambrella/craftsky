package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

type recordingSuggestionEffects struct {
	putRequests []pdseffects.PutRecordRequest
	putResult   pdseffects.RecordResult
	putErr      error
}

func (*recordingSuggestionEffects) ResolveExpectedOwners(
	context.Context,
	int64,
	[]syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	return nil, errors.New("unexpected owner resolution")
}

func (*recordingSuggestionEffects) ReadRecord(
	context.Context,
	pdseffects.ReadRecordRequest,
	any,
) (syntax.CID, error) {
	return "", errors.New("unexpected record read")
}

func (effects *recordingSuggestionEffects) PutRecord(
	_ context.Context,
	request pdseffects.PutRecordRequest,
) (pdseffects.RecordResult, error) {
	effects.putRequests = append(effects.putRequests, request)
	return effects.putResult, effects.putErr
}

func (*recordingSuggestionEffects) DeleteRecord(
	context.Context,
	pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	return pdseffects.RecordResult{}, errors.New("unexpected record delete")
}

func (*recordingSuggestionEffects) UploadBlob(
	context.Context,
	pdseffects.UploadBlobRequest,
) (*auth.UploadedBlob, error) {
	return nil, errors.New("unexpected blob upload")
}

func TestInstagramSuggestionFollowAdapterUsesStableDurablePut(t *testing.T) {
	owner := syntax.DID("did:plc:importer")
	target := syntax.DID("did:plc:target")
	createdAt := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	expected := []ownerlifecycle.ExpectedOwner{
		{Owner: owner, Generation: 7},
		{Owner: target, Generation: 11},
	}
	effects := &recordingSuggestionEffects{putResult: pdseffects.RecordResult{
		URI: "at://did:plc:importer/app.bsky.graph.follow/3lsuggestion",
		CID: "bafycid",
	}}
	adapter := instagramSuggestionFollowAdapter{executor: effects, expected: expected}

	result, err := adapter.FollowSuggestion(context.Background(), instagram.SuggestionFollowRequest{
		OperationID: "instagram-suggestion:one", MutationKey: "instagram-suggestion:one",
		Owner: owner, Target: target, OwnerGeneration: 7, TargetGeneration: 11,
		Rkey: syntax.RecordKey("3lsuggestion"), CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("FollowSuggestion: %v", err)
	}
	if result.Outcome != instagram.SuggestionFollowed ||
		result.RecordURI != effects.putResult.URI || result.RecordCID != "bafycid" {
		t.Fatalf("result = %+v", result)
	}
	if len(effects.putRequests) != 1 {
		t.Fatalf("PutRecord calls = %d, want 1", len(effects.putRequests))
	}
	request := effects.putRequests[0]
	if request.Owner != owner || request.OwnerGeneration != 7 ||
		request.OperationID != "instagram-suggestion:one" ||
		request.MutationKey != request.OperationID || request.Rkey != "3lsuggestion" ||
		request.Collection != instagramFollowCollection || len(request.ExpectedOwners) != 2 {
		t.Fatalf("durable request = %+v", request)
	}
	record, ok := request.Record.(*bsky.GraphFollow)
	if !ok || record.Subject != target.String() || record.CreatedAt != createdAt.Format(time.RFC3339) {
		t.Fatalf("follow record = %#v", request.Record)
	}
}

func TestInstagramSuggestionFollowAdapterPropagatesAmbiguousAndConflictOutcomes(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "ambiguous", err: &pdseffects.OutcomeAmbiguousError{OperationID: "instagram-suggestion:two"}},
		{name: "same key conflict", err: &pdseffects.ConflictError{OperationID: "instagram-suggestion:two"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			effects := &recordingSuggestionEffects{putErr: test.err}
			adapter := instagramSuggestionFollowAdapter{
				executor: effects,
				expected: []ownerlifecycle.ExpectedOwner{
					{Owner: "did:plc:importer", Generation: 1},
					{Owner: "did:plc:target", Generation: 1},
				},
			}
			_, err := adapter.FollowSuggestion(context.Background(), instagram.SuggestionFollowRequest{
				OperationID: "instagram-suggestion:two", MutationKey: "instagram-suggestion:two",
				Owner: "did:plc:importer", Target: "did:plc:target",
				OwnerGeneration: 1, TargetGeneration: 1,
				Rkey: "3lsecond", CreatedAt: time.Now().UTC(),
			})
			if !errors.Is(err, test.err) {
				t.Fatalf("error = %v, want %v", err, test.err)
			}
		})
	}
}

type recordingGuardedCoordinator struct {
	calls    int
	expected []ownerlifecycle.ExpectedOwner
	executor pdseffects.EffectExecutor
}

func (guarded *recordingGuardedCoordinator) WithGuardedEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation pdseffects.GuardedEffectOperation,
) error {
	guarded.calls++
	guarded.expected = append([]ownerlifecycle.ExpectedOwner(nil), expected...)
	return operation(ctx, guarded.executor)
}

func TestInstagramSuggestionCoordinatorEntersGuardOnceWithoutFenceReentry(t *testing.T) {
	owner := syntax.DID("did:plc:importer")
	target := syntax.DID("did:plc:target")
	expected := []ownerlifecycle.ExpectedOwner{
		{Owner: owner, Generation: 3},
		{Owner: target, Generation: 5},
	}
	effects := &recordingSuggestionEffects{putResult: pdseffects.RecordResult{
		URI: "at://did:plc:importer/app.bsky.graph.follow/3lguarded",
		CID: "bafyguarded",
	}}
	guarded := &recordingGuardedCoordinator{executor: effects}
	var factoryCalls int
	coordinator := instagramSuggestionEffectCoordinator{factory: func(
		_ context.Context,
		gotOwner syntax.DID,
		gotSession string,
	) (pdseffects.GuardedEffectCoordinator, error) {
		factoryCalls++
		if gotOwner != owner || gotSession != "oauth-session" {
			t.Fatalf("factory owner/session = %s/%s", gotOwner, gotSession)
		}
		return guarded, nil
	}}

	err := coordinator.WithSuggestionEffects(
		context.Background(), owner, "oauth-session", expected,
		func(ctx context.Context, follow instagram.SuggestionFollowExecutor) error {
			_, err := follow.FollowSuggestion(ctx, instagram.SuggestionFollowRequest{
				OperationID: "instagram-suggestion:guarded",
				MutationKey: "instagram-suggestion:guarded",
				Owner:       owner, Target: target, OwnerGeneration: 3, TargetGeneration: 5,
				Rkey: "3lguarded", CreatedAt: time.Now().UTC(),
			})
			return err
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if factoryCalls != 1 || guarded.calls != 1 || len(effects.putRequests) != 1 {
		t.Fatalf("factory/guard/put calls = %d/%d/%d, want 1/1/1", factoryCalls, guarded.calls, len(effects.putRequests))
	}
}

var _ pdseffects.EffectExecutor = (*recordingSuggestionEffects)(nil)
var _ pdseffects.GuardedEffectCoordinator = (*recordingGuardedCoordinator)(nil)
