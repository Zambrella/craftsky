// appview/internal/index/craftsky_profile.go
package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

// CraftskyProfile indexes social.craftsky.actor.profile events into the
// craftsky_profiles table. Required invariant: idempotent on (DID, CID).
// Tap delivers events at least once.
//
// A delete cascades into bluesky_profiles — the user leaving Craftsky
// (by deleting their social.craftsky.actor.profile record) removes their
// Bluesky mirror, since membership is defined by presence in
// craftsky_profiles.
type CraftskyProfile struct {
	pool *pgxpool.Pool
}

var _ Indexer = (*CraftskyProfile)(nil)

// NewCraftskyProfile builds an indexer backed by the given pool.
func NewCraftskyProfile(pool *pgxpool.Pool) *CraftskyProfile {
	return &CraftskyProfile{pool: pool}
}

const craftskyProfileNSID = "social.craftsky.actor.profile"

// craftskyProfileRecord mirrors the subset of social.craftsky.actor.profile
// that the indexer cares about.
type craftskyProfileRecord struct {
	Crafts []string `json:"crafts"`
}

func (c *CraftskyProfile) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != craftskyProfileNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		var rec craftskyProfileRecord
		if err := json.Unmarshal(ev.Record, &rec); err != nil {
			return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
		}
		// Defensive: TEXT[] NOT NULL in the column; never let a nil slice go in.
		if rec.Crafts == nil {
			rec.Crafts = []string{}
		}
		const q = `
			INSERT INTO craftsky_profiles (did, crafts, record_cid)
			VALUES ($1, $2, $3)
			ON CONFLICT (did) DO UPDATE SET
				crafts = EXCLUDED.crafts,
				record_cid = EXCLUDED.record_cid,
				indexed_at = now()
			WHERE craftsky_profiles.record_cid IS DISTINCT FROM EXCLUDED.record_cid
		`
		if _, err := c.pool.Exec(ctx, q, ev.DID, rec.Crafts, ev.CID); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		return c.handleDelete(ctx, ev.DID)
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

// handleDelete removes the craftsky_profiles row and its bluesky_profiles
// mirror in a single transaction. See spec §3.1.
func (c *CraftskyProfile) handleDelete(ctx context.Context, did string) error {
	return pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`DELETE FROM craftsky_profiles WHERE did = $1`, did); err != nil {
			return fmt.Errorf("delete craftsky_profiles %s: %w", did, err)
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM bluesky_profiles WHERE did = $1`, did); err != nil {
			return fmt.Errorf("delete bluesky_profiles %s: %w", did, err)
		}
		return nil
	})
}
