package app

import (
	"context"

	"social.craftsky/appview/internal/followwrite"
	"social.craftsky/appview/internal/instagram"
)

type deterministicFollowCoordinator interface {
	ExecuteDeterministic(context.Context, followwrite.DeterministicRequest) (followwrite.DeterministicResult, error)
}

// instagramSuggestionFollowAdapter is the only capability the Instagram
// suggestion service receives for crossing the PDS boundary. Matching and
// reconciliation never receive this adapter, an OAuth session selector, or a
// raw PDS client.
type instagramSuggestionFollowAdapter struct {
	coordinator deterministicFollowCoordinator
}

func (adapter instagramSuggestionFollowAdapter) FollowSuggestion(
	ctx context.Context,
	request instagram.SuggestionFollowRequest,
) (instagram.SuggestionFollowResult, error) {
	result, err := adapter.coordinator.ExecuteDeterministic(ctx, followwrite.DeterministicRequest{
		OperationID:     request.OperationID,
		Owner:           request.Owner,
		Target:          request.Target,
		OwnerGeneration: request.OwnerGeneration,
		SessionID:       request.SessionID,
		Rkey:            request.Rkey,
		CreatedAt:       request.CreatedAt,
	})
	if err != nil {
		return instagram.SuggestionFollowResult{}, err
	}
	return instagram.SuggestionFollowResult{
		Outcome:   instagram.SuggestionFollowed,
		RecordURI: result.RecordURI,
		RecordCID: result.RecordCID,
	}, nil
}

var _ instagram.SuggestionFollowExecutor = instagramSuggestionFollowAdapter{}
