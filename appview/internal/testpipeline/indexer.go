package testpipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

// Indexer writes social.craftsky.test.post records into test_posts.
// Delete me with the rest of the package; see doc.go.
type Indexer struct {
	pool *pgxpool.Pool
}

// NewIndexer returns an indexer backed by pool.
func NewIndexer(pool *pgxpool.Pool) *Indexer { return &Indexer{pool: pool} }

const testPostNSID = "social.craftsky.test.post"

// testPostRecord is the decoded shape of a social.craftsky.test.post.
// Fields not defined in the lexicon are ignored.
type testPostRecord struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

// Handle upserts on create/update and deletes on delete. Events for any
// other collection are ignored (the Dispatcher already routes by NSID,
// so this is belt-and-braces — same idiom as BlueskyPostsSample).
//
// createdAt is required because test_posts.created_at is NOT NULL. Empty
// text is allowed through (the lexicon treats it as valid; the table
// column is NOT NULL but empty string satisfies it).
func (i *Indexer) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != testPostNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		var rec testPostRecord
		if err := json.Unmarshal(ev.Record, &rec); err != nil {
			return fmt.Errorf("unmarshal test post %s: %w", ev.URI, err)
		}
		if rec.CreatedAt.IsZero() {
			return fmt.Errorf("test post %s: missing createdAt", ev.URI)
		}
		const q = `
			INSERT INTO test_posts (uri, cid, did, text, created_at, indexed_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (uri) DO UPDATE SET
				cid        = EXCLUDED.cid,
				text       = EXCLUDED.text,
				created_at = EXCLUDED.created_at,
				indexed_at = now()
		`
		if _, err := i.pool.Exec(ctx, q, ev.URI, ev.CID, ev.DID, rec.Text, rec.CreatedAt); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		// Implemented in Task 3.4.
		return fmt.Errorf("delete not yet implemented")
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}
