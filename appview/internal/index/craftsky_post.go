// appview/internal/index/craftsky_post.go
package index

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
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

	var facetsJSON []byte
	if len(rec.Facets) > 0 {
		facetsJSON, err = json.Marshal(rec.Facets)
		if err != nil {
			return fmt.Errorf("marshal facets %s: %w", ev.URI, err)
		}
	}

	var imagesJSON []byte
	if flat := flattenImages(rec.Images); flat != nil {
		imagesJSON, err = json.Marshal(flat)
		if err != nil {
			return fmt.Errorf("marshal images %s: %w", ev.URI, err)
		}
	}

	tags := extractTags(rec.Facets)

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
		facetsJSON, imagesJSON,
		nil, nil, // reply_root_*
		nil, nil, // reply_parent_*
		nil, nil, // quote_*       — Chunk 5
		tags,
		ev.Record,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("upsert %s: %w", ev.URI, err)
	}
	return nil
}

func (c *CraftskyPost) handleDelete(ctx context.Context, ev tap.Event) error {
	// Real implementation lands in Chunk 7. Until then, delete events for
	// existing posts are silently acked but the row is left in place — Tap
	// considers the event delivered. Acceptable because no production traffic
	// reaches this code path before Chunk 7 ships on the same branch.
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

// extractTags walks facets and pulls hashtag-feature tags. Lowercase,
// trim, drop empties, dedupe (preserve first-seen order). Always returns
// a non-nil slice — the column is NOT NULL DEFAULT '{}'.
func extractTags(facets []*appbsky.RichtextFacet) []string {
	if len(facets) == 0 {
		return []string{}
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, facet := range facets {
		if facet == nil {
			continue
		}
		for _, feat := range facet.Features {
			if feat == nil || feat.RichtextFacet_Tag == nil {
				continue
			}
			t := strings.ToLower(strings.TrimSpace(feat.RichtextFacet_Tag.Tag))
			if t == "" {
				continue
			}
			if _, dup := seen[t]; dup {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

// flattenImages turns the lexicon's [{image: LexBlob, alt}, ...] array
// into the storage shape [{cid, mime, alt}, ...]. Returns nil when there
// are no images, so the caller can pass nil to the JSONB column for SQL NULL.
func flattenImages(images []*craftskylex.FeedPost_Image) []map[string]string {
	if len(images) == 0 {
		return nil
	}
	out := make([]map[string]string, 0, len(images))
	for _, img := range images {
		if img == nil || img.Image == nil {
			continue
		}
		out = append(out, map[string]string{
			"cid":  img.Image.Ref.String(),
			"mime": img.Image.MimeType,
			"alt":  img.Alt,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
