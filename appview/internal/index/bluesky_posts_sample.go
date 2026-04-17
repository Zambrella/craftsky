package index

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

// BlueskyPostsSample is a throwaway indexer that writes Bluesky posts
// into the bluesky_posts_sample table. It exists to validate the
// end-to-end Tap → appview → Postgres pipe end-to-end and MUST be
// deleted when the first social.craftsky.* indexer lands.
//
// See docs/superpowers/specs/2026-04-17-tap-integration-design.md.
type BlueskyPostsSample struct {
	pool *pgxpool.Pool
}

var _ Indexer = (*BlueskyPostsSample)(nil)

// NewBlueskyPostsSample builds an indexer backed by the given pool.
func NewBlueskyPostsSample(pool *pgxpool.Pool) *BlueskyPostsSample {
	return &BlueskyPostsSample{pool: pool}
}

const blueskyPostNSID = "app.bsky.feed.post"

// Handle indexes create/update/delete on app.bsky.feed.post into
// bluesky_posts_sample. Other collections are ignored.
func (b *BlueskyPostsSample) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != blueskyPostNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		const q = `
			INSERT INTO bluesky_posts_sample (uri, cid, did, rkey, record)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (uri) DO UPDATE SET
				cid = EXCLUDED.cid,
				record = EXCLUDED.record
		`
		if _, err := b.pool.Exec(ctx, q, ev.URI, ev.CID, ev.DID, ev.Rkey, ev.Record); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		if _, err := b.pool.Exec(ctx,
			`DELETE FROM bluesky_posts_sample WHERE uri = $1`, ev.URI); err != nil {
			return fmt.Errorf("delete %s: %w", ev.URI, err)
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}
