// appview/internal/index/bluesky_follow.go
package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

const blueskyFollowNSID syntax.NSID = "app.bsky.graph.follow"

// BlueskyFollow indexes app.bsky.graph.follow events into atproto_follows.
// Required invariant: idempotent on (URI, CID).
type BlueskyFollow struct {
	pool *pgxpool.Pool
}

var _ Indexer = (*BlueskyFollow)(nil)

func NewBlueskyFollow(pool *pgxpool.Pool) *BlueskyFollow {
	return &BlueskyFollow{pool: pool}
}

type blueskyFollowRecord struct {
	Subject   syntax.DID `json:"subject"`
	CreatedAt string     `json:"createdAt"`
}

func (b *BlueskyFollow) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != blueskyFollowNSID {
		return nil
	}

	switch ev.Action {
	case "create", "update":
		var rec blueskyFollowRecord
		if err := json.Unmarshal(ev.Record, &rec); err != nil {
			return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
		}
		if rec.Subject == "" {
			return fmt.Errorf("follow record missing subject on %s", ev.URI)
		}
		createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
		if err != nil {
			return fmt.Errorf("parse createdAt %q on %s: %w", rec.CreatedAt, ev.URI, err)
		}

		const q = `
			INSERT INTO atproto_follows
				(uri, did, rkey, cid, subject_did, record, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (uri) DO UPDATE SET
				did = EXCLUDED.did,
				rkey = EXCLUDED.rkey,
				cid = EXCLUDED.cid,
				subject_did = EXCLUDED.subject_did,
				record = EXCLUDED.record,
				created_at = EXCLUDED.created_at,
				indexed_at = now()
		`
		if _, err := b.pool.Exec(ctx, q,
			ev.URI, ev.DID, ev.Rkey, ev.CID,
			rec.Subject, ev.Record, createdAt,
		); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		if _, err := b.pool.Exec(ctx,
			`DELETE FROM atproto_follows WHERE uri = $1`, ev.URI); err != nil {
			return fmt.Errorf("delete %s: %w", ev.URI, err)
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}
