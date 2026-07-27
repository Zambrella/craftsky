package instagram

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedSuggestionImport(
	t *testing.T,
	pool *pgxpool.Pool,
	id uuid.UUID,
	owner syntax.DID,
	username string,
	now time.Time,
) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_imports (
			id, owner_did, state, source_type, following_count,
			created_at, updated_at
		) VALUES ($1, $2, 'active', 'manual', 1, $3, $3)
	`, id, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_handles (
			import_id, username_normalized, matched, created_at
		) VALUES ($1, $2, false, $3)
	`, id, username, now); err != nil {
		t.Fatal(err)
	}
}

func seedSuggestionLink(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	username string,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO instagram_account_links (
			id, owner_did, state, igsid, igsid_digest_version, igsid_digest,
			username, username_normalized, discoverable, conflict_pending,
			verified_at, created_at, updated_at
		) VALUES ($1, $2, 'active', 'synthetic-igsid', 1, decode(repeat('01', 32), 'hex'),
		          $3, $3, true, false, $4, $4, $4)
	`, uuid.New(), owner, username, now); err != nil {
		t.Fatal(err)
	}
}
