// appview/internal/index/craftsky_profile.go
package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
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
	pool       *pgxpool.Pool
	backfiller BlueskyBackfiller
	logger     *slog.Logger
}

var _ Indexer = (*CraftskyProfile)(nil)

// NewCraftskyProfile builds an indexer. The backfiller is invoked when
// Handle commits a genuinely new membership row; it may be a no-op in
// tests. A nil logger defaults to slog.Default() to keep call sites that
// don't care about structured logging concise.
func NewCraftskyProfile(pool *pgxpool.Pool, backfiller BlueskyBackfiller, logger *slog.Logger) *CraftskyProfile {
	if logger == nil {
		logger = slog.Default()
	}
	return &CraftskyProfile{pool: pool, backfiller: backfiller, logger: logger}
}

const craftskyProfileNSID syntax.NSID = "social.craftsky.actor.profile"

func (c *CraftskyProfile) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != craftskyProfileNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		var rec craftskylex.ActorProfile
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
			RETURNING xmax = 0 AS created
		`
		var created bool
		err := c.pool.QueryRow(ctx, q, ev.DID, rec.Crafts, ev.CID).Scan(&created)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// Replay of an existing row: ON CONFLICT ... WHERE IS DISTINCT FROM
			// filtered the update, so no row came back. Not an error.
			return nil
		case err != nil:
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		if !created {
			// UPDATE branch — membership row already existed; no backfill needed.
			return nil
		}
		// Genuine new-row INSERT. Trigger one-shot Bluesky backfill;
		// errors are logged and swallowed so the craftsky event is still
		// acked by Tap. ev.DID is already validated at the WS boundary.
		if bfErr := c.backfiller.Backfill(ctx, ev.DID); bfErr != nil {
			c.logger.Warn("craftsky profile: bluesky backfill failed",
				slog.String("did", ev.DID.String()), slog.String("err", bfErr.Error()))
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
func (c *CraftskyProfile) handleDelete(ctx context.Context, did syntax.DID) error {
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
