// appview/internal/index/craftsky_post.go
package index

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/tap"
)

// CraftskyPost indexes social.craftsky.feed.post events into craftsky_posts.
// Required invariant: idempotent on (URI, CID). Tap delivers at-least-once.
//
// Posts are gated on craftsky_profiles membership: events from non-members
// are dropped silently, matching BlueskyProfile's pattern. A post arriving
// before its author's craftsky_profiles row is dropped permanently for now;
// see the design spec for the post-backfiller follow-up.
type CraftskyPost struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

var _ Indexer = (*CraftskyPost)(nil)

func NewCraftskyPost(pool *pgxpool.Pool, logger *slog.Logger) *CraftskyPost {
	if logger == nil {
		logger = slog.Default()
	}
	return &CraftskyPost{pool: pool, logger: logger}
}

const craftskyPostNSID syntax.NSID = "social.craftsky.feed.post"

func (c *CraftskyPost) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != craftskyPostNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		return c.handleUpsert(ctx, ev)
	case "delete":
		return c.handleDelete(ctx, ev)
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

func (c *CraftskyPost) handleUpsert(ctx context.Context, ev tap.Event) error {
	isMember, err := c.isMember(ctx, ev.DID)
	if err != nil {
		return fmt.Errorf("membership check %s: %w", ev.DID, err)
	}
	if !isMember {
		return nil
	}

	var rec craftskylex.FeedPost
	if err := json.Unmarshal(ev.Record, &rec); err != nil {
		return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
	}
	createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("parse createdAt %q on %s: %w", rec.CreatedAt, ev.URI, err)
	}

	// Materialised columns. Subsequent chunks fill the nil/empty values
	// from rec.Facets, rec.Images, rec.Reply, rec.Embed.
	const q = `
		INSERT INTO craftsky_posts
			(uri, did, rkey, cid, text, facets, images,
			 reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid,
			 quote_uri, quote_cid, tags, record, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (uri) DO UPDATE SET
			cid              = EXCLUDED.cid,
			text             = EXCLUDED.text,
			facets           = EXCLUDED.facets,
			images           = EXCLUDED.images,
			reply_root_uri   = EXCLUDED.reply_root_uri,
			reply_root_cid   = EXCLUDED.reply_root_cid,
			reply_parent_uri = EXCLUDED.reply_parent_uri,
			reply_parent_cid = EXCLUDED.reply_parent_cid,
			quote_uri        = EXCLUDED.quote_uri,
			quote_cid        = EXCLUDED.quote_cid,
			tags             = EXCLUDED.tags,
			record           = EXCLUDED.record,
			created_at       = EXCLUDED.created_at,
			indexed_at       = now()
		WHERE craftsky_posts.cid IS DISTINCT FROM EXCLUDED.cid
	`
	_, err = c.pool.Exec(ctx, q,
		ev.URI, ev.DID, ev.Rkey, ev.CID,
		rec.Text,
		nil, nil, // facets, images — Chunk 4
		nil, nil, // reply_root_*
		nil, nil, // reply_parent_*
		nil, nil, // quote_*       — Chunk 5
		[]string{}, // tags         — Chunk 4
		ev.Record,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("upsert %s: %w", ev.URI, err)
	}
	return nil
}

func (c *CraftskyPost) handleDelete(ctx context.Context, ev tap.Event) error {
	// Real implementation lands in Chunk 7. Returning nil here is fine
	// for now — no test in this chunk exercises the delete path against
	// a populated row.
	_ = ev
	_ = ctx
	return nil
}

func (c *CraftskyPost) isMember(ctx context.Context, did syntax.DID) (bool, error) {
	var exists bool
	err := c.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`, did).
		Scan(&exists)
	return exists, err
}
