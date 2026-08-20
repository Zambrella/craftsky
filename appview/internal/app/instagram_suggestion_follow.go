package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

const instagramFollowCollection syntax.NSID = "app.bsky.graph.follow"

// instagramSuggestionEffectCoordinator is the only capability the explicit
// suggestion-acceptance service receives for crossing the PDS boundary. It
// enters the combined participant/session fence once and lends a scoped
// follow-only adapter to the callback. Matching and background reconciliation
// never receive this coordinator.
type instagramSuggestionEffectCoordinator struct {
	factory pdseffects.GuardedExecutorFactory
}

func (coordinator instagramSuggestionEffectCoordinator) WithSuggestionEffects(
	ctx context.Context,
	owner syntax.DID,
	sessionID string,
	expected []ownerlifecycle.ExpectedOwner,
	operation instagram.SuggestionEffectOperation,
) error {
	if coordinator.factory == nil || owner == "" || strings.TrimSpace(sessionID) == "" ||
		len(expected) == 0 || operation == nil {
		return errors.New("instagram suggestion effect coordinator is unavailable")
	}
	guarded, err := coordinator.factory(ctx, owner, sessionID)
	if err != nil {
		return err
	}
	if guarded == nil {
		return errors.New("instagram suggestion guarded executor is unavailable")
	}
	return guarded.WithGuardedEffects(
		ctx,
		expected,
		func(effectCtx context.Context, executor pdseffects.EffectExecutor) error {
			return operation(effectCtx, instagramSuggestionFollowAdapter{
				executor: executor,
				expected: append([]ownerlifecycle.ExpectedOwner(nil), expected...),
			})
		},
	)
}

type instagramSuggestionFollowAdapter struct {
	executor pdseffects.EffectExecutor
	expected []ownerlifecycle.ExpectedOwner
}

func (adapter instagramSuggestionFollowAdapter) FollowSuggestion(
	ctx context.Context,
	request instagram.SuggestionFollowRequest,
) (instagram.SuggestionFollowResult, error) {
	if adapter.executor == nil {
		return instagram.SuggestionFollowResult{}, errors.New("instagram suggestion follow executor is unavailable")
	}
	if err := validateSuggestionFollowScope(request, adapter.expected); err != nil {
		return instagram.SuggestionFollowResult{}, err
	}
	result, err := adapter.executor.PutRecord(ctx, pdseffects.PutRecordRequest{
		OperationID: request.OperationID,
		MutationKey: request.MutationKey,
		Owner:       request.Owner, OwnerGeneration: request.OwnerGeneration,
		ExpectedOwners: adapter.expected,
		Collection:     instagramFollowCollection,
		Rkey:           request.Rkey,
		Record: &bsky.GraphFollow{
			LexiconTypeID: instagramFollowCollection.String(),
			Subject:       request.Target.String(),
			CreatedAt:     request.CreatedAt.UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return instagram.SuggestionFollowResult{}, err
	}
	return instagram.SuggestionFollowResult{
		Outcome:   instagram.SuggestionFollowed,
		RecordURI: result.URI,
		RecordCID: result.CID.String(),
	}, nil
}

func validateSuggestionFollowScope(
	request instagram.SuggestionFollowRequest,
	expected []ownerlifecycle.ExpectedOwner,
) error {
	if request.OperationID == "" || request.MutationKey != request.OperationID ||
		request.Owner == "" || request.Target == "" || request.Owner == request.Target ||
		request.OwnerGeneration <= 0 || request.TargetGeneration <= 0 ||
		request.Rkey == "" || request.CreatedAt.IsZero() {
		return errors.New("invalid Instagram suggestion follow request")
	}
	ownerMatched := false
	targetMatched := false
	for _, item := range expected {
		if item.AllowMissing {
			continue
		}
		if item.Owner == request.Owner && item.Generation == request.OwnerGeneration {
			ownerMatched = true
		}
		if item.Owner == request.Target && item.Generation == request.TargetGeneration {
			targetMatched = true
		}
	}
	if !ownerMatched || !targetMatched {
		return ownerlifecycle.ErrGenerationChanged
	}
	return nil
}

var _ instagram.SuggestionEffectCoordinator = instagramSuggestionEffectCoordinator{}
var _ instagram.SuggestionFollowExecutor = instagramSuggestionFollowAdapter{}
