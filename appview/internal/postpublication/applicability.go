package postpublication

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
)

// CurrentApplies is a final check for an already-created publication delivery.
// Source state can be ahead of notification/serving projection. This check can
// only withdraw eligibility; it cannot introduce an event or recipient.
func CurrentApplies(ctx context.Context, reader Reader, uri syntax.ATURI, recipient syntax.DID) (bool, error) {
	original, err := Read(ctx, reader, uri)
	if err != nil {
		return false, err
	}
	if !original.Present {
		// Existing pre-feature events keep their existing delivery policy. New
		// source transitions always establish a known or unknown marker.
		return true, nil
	}
	if !original.Known {
		return false, nil
	}
	var raw json.RawMessage
	err = reader.QueryRow(ctx, `SELECT record FROM tap_source_records WHERE uri=$1 AND action<>'delete'
		AND ordering_status='authoritative' AND structural_validation_status='valid' AND semantic_validation_status='valid'`, uri).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var current craftskylex.FeedPost
	if err := json.Unmarshal(raw, &current); err != nil {
		return false, err
	}
	mentions, err := OriginalMentions(ctx, reader, uri, MentionDIDs(current))
	if err != nil {
		return false, err
	}
	for _, did := range mentions {
		if did == recipient.String() {
			return true, nil
		}
	}
	if reply := current.Reply; original.ReplyEligible && reply != nil && reply.Parent != nil && reply.Root != nil &&
		reply.Parent.Uri == original.ParentURI && reply.Parent.Cid == original.ParentCID &&
		reply.Root.Uri == original.RootURI && reply.Root.Cid == original.RootCID &&
		syntax.ATURI(reply.Parent.Uri).Authority().DID() == recipient {
		return true, nil
	}
	if original.QuoteEligible && current.Embed != nil && current.Embed.FeedPost_QuoteEmbed != nil {
		quote := current.Embed.FeedPost_QuoteEmbed.Record
		if quote != nil && quote.Uri == original.QuotedURI && quote.Cid == original.QuotedCID && syntax.ATURI(quote.Uri).Authority().DID() == recipient {
			return true, nil
		}
	}
	return false, nil
}
