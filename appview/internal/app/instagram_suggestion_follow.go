package app

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/pdscommands"
)

const instagramFollowCollection syntax.NSID = "app.bsky.graph.follow"

type instagramSuggestionFollowAdapter struct {
	commands api.SetCommandExecutor
}

func (adapter instagramSuggestionFollowAdapter) FollowSuggestion(
	ctx context.Context,
	request instagram.SuggestionFollowRequest,
) (instagram.SuggestionCommandResult, error) {
	if adapter.commands == nil {
		return instagram.SuggestionCommandResult{}, pdscommands.ErrDispatchUnavailable
	}
	if err := validateSuggestionFollowRequest(request); err != nil {
		return instagram.SuggestionCommandResult{}, err
	}
	intent, err := json.Marshal(struct {
		SuggestionID     string     `json:"suggestionId"`
		TargetDID        syntax.DID `json:"targetDid"`
		TargetGeneration int64      `json:"targetGeneration"`
	}{SuggestionID: request.SuggestionID.String(), TargetDID: request.Target, TargetGeneration: request.TargetGeneration})
	if err != nil {
		return instagram.SuggestionCommandResult{}, err
	}
	selectedURI := syntax.ATURI("at://" + request.Owner.String() + "/" + instagramFollowCollection.String() + "/" + request.Rkey.String())
	accepted := func(state instagram.SuggestionState) pdscommands.TerminalResult {
		body, _ := json.Marshal(struct {
			SuggestionID string                    `json:"suggestionId"`
			State        instagram.SuggestionState `json:"state"`
		}{SuggestionID: request.SuggestionID.String(), State: state})
		return pdscommands.TerminalResult{
			State: pdscommands.CommandAccepted, HTTPStatus: http.StatusOK, ResponseBody: body,
			ResponseHeaders: json.RawMessage(`{"Content-Type":"application/json"}`),
		}
	}
	result, err := adapter.commands.Execute(ctx, pdscommands.SetCommandRequest{
		Owner: request.Owner, OwnerGeneration: request.OwnerGeneration,
		Target: request.Target, TargetGeneration: request.TargetGeneration,
		SessionID: request.SessionID, OperationKind: "instagram.suggestion.accept",
		OperationKey: request.OperationKey, Collection: instagramFollowCollection,
		DesiredActive: true, Intent: intent, SelectedRkey: request.Rkey,
		Matches: func(record pdscommands.AuthoritativeRecord) bool {
			follow, valid := decodeInstagramAuthoritativeFollow(record)
			return valid && follow.Subject == request.Target.String()
		},
		CreateRecord: func(time.Time) (json.RawMessage, error) {
			return json.Marshal(bsky.GraphFollow{
				LexiconTypeID: instagramFollowCollection.String(),
				Subject:       request.Target.String(),
				CreatedAt:     request.CreatedAt.UTC().Format(time.RFC3339Nano),
			})
		},
		AcceptedPresent: func(record pdscommands.AuthoritativeRecord, _ bool) (pdscommands.TerminalResult, error) {
			follow, valid := decodeInstagramAuthoritativeFollow(record)
			if !valid || follow.Subject != request.Target.String() {
				return pdscommands.TerminalResult{}, pdscommands.ErrMalformedCommand
			}
			if record.URI == selectedURI {
				return accepted(instagram.SuggestionFollowed), nil
			}
			return accepted(instagram.SuggestionAlreadyFollowing), nil
		},
		AcceptedAbsent: func() pdscommands.TerminalResult {
			return accepted(instagram.SuggestionAlreadyFollowing)
		},
		Rejected: func(err error) pdscommands.TerminalResult {
			return api.RejectedCommandResult(request.RequestID, err)
		},
	})
	return instagram.SuggestionCommandResult{
		State:             instagram.SuggestionCommandState(result.State),
		HTTPStatus:        result.HTTPStatus,
		ResponseBody:      result.ResponseBody,
		ResponseHeaders:   result.ResponseHeaders,
		RetryAfterSeconds: result.RetryAfterSeconds,
	}, err
}

func validateSuggestionFollowRequest(request instagram.SuggestionFollowRequest) error {
	if request.OperationKey == uuid.Nil || request.RequestID == "" || request.SessionID == "" || request.SuggestionID == uuid.Nil ||
		request.Owner == "" || request.Target == "" || request.Owner == request.Target ||
		request.OwnerGeneration <= 0 || request.TargetGeneration <= 0 || request.Rkey == "" || request.CreatedAt.IsZero() {
		return pdscommands.ErrMalformedCommand
	}
	return nil
}

func decodeInstagramAuthoritativeFollow(record pdscommands.AuthoritativeRecord) (bsky.GraphFollow, bool) {
	var follow bsky.GraphFollow
	if record.URI == "" || record.CID == "" || json.Unmarshal(record.Record, &follow) != nil ||
		follow.LexiconTypeID != instagramFollowCollection.String() {
		return bsky.GraphFollow{}, false
	}
	if _, err := syntax.ParseDID(follow.Subject); err != nil {
		return bsky.GraphFollow{}, false
	}
	if _, err := time.Parse(time.RFC3339Nano, follow.CreatedAt); err != nil {
		return bsky.GraphFollow{}, false
	}
	return follow, true
}

var _ instagram.SuggestionFollowExecutor = instagramSuggestionFollowAdapter{}
