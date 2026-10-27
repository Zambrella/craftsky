package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/postrecord"
)

// hydratePostEditResult is response-only; the durable result remains content
// free. Every replay rechecks current visibility and the matching version.
func hydratePostEditResult(ctx context.Context, store postAuthorReader, resolver HandleResolver, owner syntax.DID, key syntax.RecordKey, command pdscommands.CommandResult) pdscommands.CommandResult {
	reader, ok := store.(postEditSeedStore)
	if !ok || command.State != pdscommands.CommandAccepted {
		return command
	}
	var result PostEditResult
	if json.Unmarshal(command.ResponseBody, &result) != nil {
		return command
	}
	snapshot, err := reader.ReadEditSnapshot(ctx, owner, key)
	if err != nil {
		if !errors.Is(err, ErrPostEditPending) && !errors.Is(err, ErrPostNotFound) {
			observability.ReportRequestFailure(ctx, err, "api.CommandEditPostHandler", "accepted_refresh_query")
		}
		return command
	}
	if snapshot.Post.CID != result.AcceptedCID.String() {
		return command
	}
	handle, err := resolver.ResolveHandle(ctx, owner)
	if err != nil {
		observability.ReportRequestFailure(ctx, err, "api.CommandEditPostHandler", "accepted_refresh_identity")
		return command
	}
	summaries, err := reader.EngagementSummaries(ctx, owner.String(), nil, []string{snapshot.Post.URI})
	if err != nil {
		observability.ReportRequestFailure(ctx, err, "api.CommandEditPostHandler", "accepted_refresh_engagement")
		return command
	}
	post := buildPostResponse(snapshot.Post, handle, store)
	applyEngagementSummary(post, summaries[snapshot.Post.URI])
	if err := attachQuoteView(ctx, reader, resolver, post); err != nil {
		observability.ReportRequestFailure(ctx, err, "api.CommandEditPostHandler", "accepted_refresh_quote")
		return command
	}
	result.Post = post
	result.ProjectionPending = false
	if !result.Changed {
		result.EditedAt = post.EditedAt
		result.EditTimeSource = post.EditTimeSource
		result.RecordRevision = snapshot.Revision
	}
	if body, err := json.Marshal(result); err == nil {
		command.ResponseBody = body
	}
	return command
}

// PostEditResult carries acceptance independently of asynchronously indexed
// content. Never persist a synthetic post in a replayable edit response: it
// would invent engagement and could replay content hidden after acceptance.
type PostEditResult struct {
	Post                *PostResponse              `json:"post"`
	Changed             bool                       `json:"changed"`
	AcceptedCID         syntax.CID                 `json:"acceptedCid"`
	EditedAt            *time.Time                 `json:"editedAt,omitempty"`
	EditTimeSource      *postrecord.EditTimeSource `json:"editTimeSource,omitempty"`
	RecordRevision      syntax.TID                 `json:"recordRevision,omitempty"`
	ObservationCID      syntax.CID                 `json:"observationCid,omitempty"`
	ObservationRevision syntax.TID                 `json:"observationRevision,omitempty"`
	ProjectionPending   bool                       `json:"projectionPending"`
}

func acceptedPostEditResult(record pdscommands.AuthoritativeRecord, expected syntax.CID) (pdscommands.TerminalResult, error) {
	result := PostEditResult{Changed: record.CID != expected, AcceptedCID: record.CID,
		RecordRevision: record.Revision, ObservationCID: record.ObservationCID,
		ObservationRevision: record.ObservationRevision, ProjectionPending: true}
	var metadata struct {
		EditedAt *time.Time `json:"editedAt"`
	}
	if err := json.Unmarshal(record.Record, &metadata); err != nil {
		return pdscommands.TerminalResult{}, err
	}
	result.EditedAt = postEditTimeUTC(metadata.EditedAt)
	if result.EditedAt != nil {
		source := postrecord.EditReported
		if result.Changed {
			source = postrecord.EditConfirmed
		}
		result.EditTimeSource = &source
	}
	body, err := json.Marshal(result)
	return pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: http.StatusOK, ResponseBody: body}, err
}
