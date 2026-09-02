package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/tap"
)

type blueskyProfileEventHandler interface {
	Handle(context.Context, tap.Event) error
}

type oauthBlueskyProfileProjection struct {
	handler blueskyProfileEventHandler
}

type oauthCraftskyProfileProjection struct {
	handler blueskyProfileEventHandler
}

func (p oauthBlueskyProfileProjection) ProjectBlueskyProfile(
	ctx context.Context,
	did syntax.DID,
	cid syntax.CID,
	record map[string]any,
) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal Bluesky profile: %w", err)
	}
	return p.handler.Handle(ctx, tap.Event{
		URI:        syntax.ATURI("at://" + did.String() + "/app.bsky.actor.profile/self"),
		CID:        cid,
		DID:        did,
		Collection: syntax.NSID("app.bsky.actor.profile"),
		Rkey:       syntax.RecordKey("self"),
		Action:     "create",
		Record:     raw,
	})
}

func (p oauthCraftskyProfileProjection) ProjectCraftskyProfile(
	ctx context.Context,
	did syntax.DID,
	cid syntax.CID,
	record map[string]any,
) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal Craftsky profile: %w", err)
	}
	return p.handler.Handle(ctx, tap.Event{
		URI:        syntax.ATURI("at://" + did.String() + "/social.craftsky.actor.profile/self"),
		CID:        cid,
		DID:        did,
		Collection: syntax.NSID("social.craftsky.actor.profile"),
		Rkey:       syntax.RecordKey("self"),
		Action:     "create",
		Record:     raw,
	})
}
