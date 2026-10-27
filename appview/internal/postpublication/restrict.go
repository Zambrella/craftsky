package postpublication

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/postutil"
)

// RestrictCurrent retires removed original reasons permanently, before source
// coalescing or delayed projection can forget an intervening removal. Existing
// candidates may shrink; subsequent edits can never introduce/re-enable one.
func RestrictCurrent(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, raw json.RawMessage, now time.Time) (int64, error) {
	var current craftskylex.FeedPost
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &current); err != nil {
			return 0, err
		}
	}
	mentions := MentionDIDs(current)
	result, err := tx.Exec(ctx, `UPDATE post_publication_mentions SET eligible=false
 WHERE post_uri=$1 AND eligible AND NOT (recipient_did=ANY($2::text[]))`, uri, mentions)
	if err != nil {
		return 0, err
	}
	retired := result.RowsAffected()
	var parent, parentCID, root, rootCID, quote, quoteCID string
	if current.Reply != nil {
		if current.Reply.Parent != nil {
			parent = current.Reply.Parent.Uri
			parentCID = current.Reply.Parent.Cid
		}
		if current.Reply.Root != nil {
			root = current.Reply.Root.Uri
			rootCID = current.Reply.Root.Cid
		}
	}
	if current.Embed != nil && current.Embed.FeedPost_QuoteEmbed != nil && current.Embed.FeedPost_QuoteEmbed.Record != nil {
		quote = current.Embed.FeedPost_QuoteEmbed.Record.Uri
		quoteCID = current.Embed.FeedPost_QuoteEmbed.Record.Cid
	}
	result, err = tx.Exec(ctx, `UPDATE post_publication_state SET reply_eligible=false WHERE uri=$1 AND reply_eligible AND parent_uri IS NOT NULL
 AND (parent_uri IS DISTINCT FROM NULLIF($2,'') OR parent_cid IS DISTINCT FROM NULLIF($3,'') OR root_uri IS DISTINCT FROM NULLIF($4,'') OR root_cid IS DISTINCT FROM NULLIF($5,''))`, uri, parent, parentCID, root, rootCID)
	if err != nil {
		return 0, err
	}
	retired += result.RowsAffected()
	result, err = tx.Exec(ctx, `UPDATE post_publication_state SET quote_eligible=false WHERE uri=$1 AND quote_eligible AND quoted_uri IS NOT NULL
 AND (quoted_uri IS DISTINCT FROM NULLIF($2,'') OR quoted_cid IS DISTINCT FROM NULLIF($3,''))`, uri, quote, quoteCID)
	if err != nil {
		return 0, err
	}
	retired += result.RowsAffected()
	restrictPending := retired > 0
	if len(raw) == 0 && !restrictPending {
		original, err := Read(ctx, tx, uri)
		if err != nil {
			return 0, err
		}
		restrictPending = original.Present
	}
	if restrictPending {
		allowed, err := eligibleRecipients(ctx, tx, uri, current)
		if err != nil {
			return 0, err
		}
		if err := notifications.RestrictStoredPublication(ctx, tx, uri, allowed, now); err != nil {
			return 0, err
		}
	}
	if len(raw) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM post_publication_mentions WHERE post_uri=$1`, uri); err != nil {
			return 0, err
		}
		// A current deletion marker has no original dates/recipients/refs and
		// cannot be promoted into catch-up work by a later re-added record.
		if _, err := tx.Exec(ctx, `INSERT INTO post_publication_state(uri,owner_did,origin_status,reply_eligible,quote_eligible)
   VALUES($1,$2,'deleted',false,false)
   ON CONFLICT(uri) DO UPDATE SET origin_status='deleted',original_creation_time=NULL,
   parent_uri=NULL,parent_cid=NULL,root_uri=NULL,root_cid=NULL,quoted_uri=NULL,quoted_cid=NULL,
   reply_eligible=false,quote_eligible=false`, uri, uri.Authority().DID()); err != nil {
			return 0, err
		}
	}
	return retired, nil
}

func eligibleRecipients(ctx context.Context, reader Reader, uri syntax.ATURI, current craftskylex.FeedPost) ([]string, error) {
	original, err := Read(ctx, reader, uri)
	if err != nil {
		return nil, err
	}
	if !original.Known {
		return []string{}, nil
	}
	mentions, err := OriginalMentions(ctx, reader, uri, MentionDIDs(current))
	if err != nil {
		return nil, err
	}
	recipients := append([]string{}, mentions...)
	if reply := current.Reply; original.ReplyEligible && reply != nil && reply.Parent != nil && reply.Root != nil &&
		reply.Parent.Uri == original.ParentURI && reply.Parent.Cid == original.ParentCID && reply.Root.Uri == original.RootURI && reply.Root.Cid == original.RootCID {
		recipients = append(recipients, syntax.ATURI(reply.Parent.Uri).Authority().DID().String())
	}
	if original.QuoteEligible && current.Embed != nil && current.Embed.FeedPost_QuoteEmbed != nil {
		quote := current.Embed.FeedPost_QuoteEmbed.Record
		if quote != nil && quote.Uri == original.QuotedURI && quote.Cid == original.QuotedCID {
			recipients = append(recipients, syntax.ATURI(quote.Uri).Authority().DID().String())
		}
	}
	return postutil.MergeMentionDIDs(recipients), nil
}
