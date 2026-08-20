// appview/internal/api/post_response_shape.go
package api

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

type quoteViewReader interface {
	QuoteViewRows(context.Context, []ResponseStrongRef) (map[string]*QuoteViewRow, error)
}

type relationshipStatesReader interface {
	RelationshipStates(context.Context, syntax.DID, []syntax.DID) (map[syntax.DID]relationships.State, error)
}

type blockedPairsReader interface {
	BlockedPairs(context.Context, []RelationshipPair) (map[RelationshipPair]bool, error)
}

type postQuoteHydrationStore interface {
	quoteViewReader
	relationshipStatesReader
	blockedPairsReader
}

func attachQuoteView(ctx context.Context, store quoteViewReader, resolver HandleResolver, resp *PostResponse) error {
	return attachQuoteViews(ctx, store, resolver, []*PostResponse{resp})
}

func attachQuoteViews(ctx context.Context, store quoteViewReader, resolver HandleResolver, responses []*PostResponse) error {
	refs := make([]ResponseStrongRef, 0, len(responses))
	for _, resp := range responses {
		if resp == nil || resp.Quote == nil {
			continue
		}
		refs = append(refs, *resp.Quote)
	}
	if len(refs) == 0 {
		return nil
	}
	views, err := store.QuoteViewRows(ctx, refs)
	if err != nil {
		return err
	}
	viewerDID, hasViewer := middleware.GetDID(ctx)
	subjects := make([]syntax.DID, 0, len(views))
	for _, viewRow := range views {
		if viewRow == nil || viewRow.Post == nil {
			continue
		}
		did, err := syntax.ParseDID(viewRow.Post.DID)
		if err != nil {
			return err
		}
		subjects = append(subjects, did)
	}
	states := map[syntax.DID]relationships.State{}
	if relationshipReader, ok := store.(interface {
		RelationshipStates(context.Context, syntax.DID, []syntax.DID) (map[syntax.DID]relationships.State, error)
	}); ok && hasViewer && len(subjects) > 0 {
		states, err = relationshipReader.RelationshipStates(ctx, viewerDID, subjects)
		if err != nil {
			return err
		}
	}
	pairsByResponse := make(map[*PostResponse]RelationshipPair)
	pairs := make([]RelationshipPair, 0, len(responses))
	for _, resp := range responses {
		if resp == nil || resp.Quote == nil {
			continue
		}
		viewRow := views[resp.Quote.URI]
		if viewRow == nil || viewRow.Post == nil {
			continue
		}
		first, firstErr := syntax.ParseDID(resp.Author.DID)
		second, secondErr := syntax.ParseDID(viewRow.Post.DID)
		if firstErr != nil || secondErr != nil {
			continue
		}
		pair := RelationshipPair{First: first, Second: second}
		pairsByResponse[resp] = pair
		pairs = append(pairs, pair)
	}
	blockedPairs := map[RelationshipPair]bool{}
	if pairReader, ok := store.(interface {
		BlockedPairs(context.Context, []RelationshipPair) (map[RelationshipPair]bool, error)
	}); ok && len(pairs) > 0 {
		blockedPairs, err = pairReader.BlockedPairs(ctx, pairs)
		if err != nil {
			return err
		}
	}
	handles := map[string]syntax.Handle{}
	for _, resp := range responses {
		if resp == nil || resp.Quote == nil {
			continue
		}
		viewRow := views[resp.Quote.URI]
		if viewRow == nil {
			resp.QuoteView = &QuoteView{State: "unavailable"}
			continue
		}
		var handle syntax.Handle
		if viewRow.Post != nil {
			if blockedPairs[pairsByResponse[resp]] {
				resp.QuoteView = &QuoteView{State: "blocked"}
				continue
			}
			postDID, _ := syntax.ParseDID(viewRow.Post.DID)
			state := states[postDID]
			if decision := relationships.Decide(relationships.ModerationVisible, state, relationships.SurfaceQuote); decision == relationships.DecisionMutedPlaceholder || decision == relationships.DecisionBlockedPlaceholder {
				resp.QuoteView = &QuoteView{State: "visible"}
				ApplyQuoteRelationshipPolicy(resp.QuoteView, state)
				continue
			}
			cached, ok := handles[viewRow.Post.DID]
			if !ok {
				did, err := syntax.ParseDID(viewRow.Post.DID)
				if err != nil {
					return err
				}
				cached, err = resolver.ResolveHandle(ctx, did)
				if err != nil {
					return err
				}
				handles[viewRow.Post.DID] = cached
			}
			handle = cached
		}
		resp.QuoteView = BuildQuoteView(viewRow, handle)
	}
	return nil
}

func resolveHandlesForRows(ctx context.Context, rows []*PostRow, resolver HandleResolver) (map[string]syntax.Handle, error) {
	handles := make(map[string]syntax.Handle)
	for _, row := range rows {
		if _, ok := handles[row.DID]; ok {
			continue
		}
		did, err := syntax.ParseDID(row.DID)
		if err != nil {
			return nil, err
		}
		handle, err := resolver.ResolveHandle(ctx, did)
		if err != nil {
			return nil, err
		}
		handles[row.DID] = handle
	}
	return handles, nil
}
