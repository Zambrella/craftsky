package instagram

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestSuggestionStoreDeduplicatesSupportsListsPrivatelyAndDismisses(t *testing.T) {
	store, pool := newSuggestionTestStore(t)
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	carol := syntax.DID("did:plc:synthetic-carol")
	now := time.Date(2026, 7, 19, 16, 0, 0, 0, time.UTC)
	importOne := uuid.MustParse("00000000-0000-0000-0000-000000000241")
	importTwo := uuid.MustParse("00000000-0000-0000-0000-000000000242")
	seedSuggestionImport(t, pool, importOne, alice, "synthetic.bob", now)
	seedSuggestionImport(t, pool, importTwo, alice, "synthetic.bob", now.Add(time.Minute))
	seedSuggestionLink(t, pool, bob, "synthetic.bob", now)

	suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000243")
	created, err := store.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{
		ID: suggestionID, ImporterDID: alice, TargetDID: bob,
		ImportID: importOne, Username: "synthetic.bob", Now: now,
	})
	if err != nil {
		t.Fatalf("upsert first support: %v", err)
	}
	if !created || suggestionID == uuid.Nil {
		t.Fatalf("created=%t id=%s", created, suggestionID)
	}
	created, err = store.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{
		ID:          uuid.MustParse("00000000-0000-0000-0000-000000000244"),
		ImporterDID: alice, TargetDID: bob, ImportID: importTwo,
		Username: "synthetic.bob", Now: now.Add(time.Minute),
	})
	if err != nil || created {
		t.Fatalf("upsert second support created=%t err=%v", created, err)
	}
	var suggestions, supports int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_follow_suggestions`).Scan(&suggestions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_suggestion_sources`).Scan(&supports); err != nil {
		t.Fatal(err)
	}
	if suggestions != 1 || supports != 2 {
		t.Fatalf("suggestions=%d supports=%d", suggestions, supports)
	}

	items, cursor, err := store.ListPendingSuggestions(ctx, alice, 20, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || cursor != nil || items[0].Suggestion.ID != suggestionID || items[0].ImportedUsername != "synthetic.bob" || items[0].Direction != DirectionFollowing {
		t.Fatalf("items=%+v cursor=%+v", items, cursor)
	}
	foreign, _, err := store.ListPendingSuggestions(ctx, carol, 20, nil)
	if err != nil || len(foreign) != 0 {
		t.Fatalf("foreign list=%+v err=%v", foreign, err)
	}

	if err := store.DismissSuggestion(ctx, carol, suggestionID, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("foreign dismiss: %v", err)
	}
	if err := store.DismissSuggestion(ctx, alice, suggestionID, now.Add(3*time.Minute)); err != nil {
		t.Fatalf("owner dismiss: %v", err)
	}
	if err := store.DismissSuggestion(ctx, alice, suggestionID, now.Add(4*time.Minute)); err != nil {
		t.Fatalf("dismiss replay: %v", err)
	}
	items, _, err = store.ListPendingSuggestions(ctx, alice, 20, nil)
	if err != nil || len(items) != 0 {
		t.Fatalf("post-dismiss items=%+v err=%v", items, err)
	}
}

func TestSuggestionStoreClaimsStableFollowOperationAndFinalizes(t *testing.T) {
	store, pool := newSuggestionTestStore(t)
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	now := time.Date(2026, 7, 19, 16, 30, 0, 0, time.UTC)
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000245")
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000246")
	seedSuggestionImport(t, pool, importID, alice, "synthetic.bob", now)
	seedSuggestionLink(t, pool, bob, "synthetic.bob", now)
	if _, err := store.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{
		ID: suggestionID, ImporterDID: alice, TargetDID: bob,
		ImportID: importID, Username: "synthetic.bob", Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	claim, err := store.ClaimSuggestionAcceptance(ctx, alice, suggestionID, "3kabcde234s2z", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claim.Suggestion.State != SuggestionAccepting || claim.Operation.Rkey != "3kabcde234s2z" || claim.ImportedUsername != "synthetic.bob" {
		t.Fatalf("claim=%+v", claim)
	}
	replay, err := store.ClaimSuggestionAcceptance(ctx, alice, suggestionID, "3kdifferent2z", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("claim replay: %v", err)
	}
	if replay.Operation.Rkey != claim.Operation.Rkey {
		t.Fatalf("replay rkey=%q want=%q", replay.Operation.Rkey, claim.Operation.Rkey)
	}
	if _, err := store.ClaimSuggestionAcceptance(ctx, syntax.DID("did:plc:synthetic-carol"), suggestionID, "3kforeign222z", now); !errors.Is(err, ErrInstagramResourceNotFound) {
		t.Fatalf("foreign claim error=%v", err)
	}

	completed, err := store.CompleteSuggestionAcceptance(ctx, alice, suggestionID, SuggestionAccepted, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completed.State != SuggestionAccepted {
		t.Fatalf("completed=%+v", completed)
	}
	stable, err := store.ClaimSuggestionAcceptance(ctx, alice, suggestionID, "3kignored222z", now.Add(4*time.Minute))
	if err != nil || stable.Suggestion.State != SuggestionAccepted || stable.Operation.Rkey != claim.Operation.Rkey {
		t.Fatalf("terminal replay=%+v err=%v", stable, err)
	}
}

func TestImportDeletionPreservesMultiSourceSuggestionUntilLastSupport(t *testing.T) {
	suggestionStore, pool := newSuggestionTestStore(t)
	importStore := NewImportStore(pool)
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	now := time.Date(2026, 7, 19, 16, 45, 0, 0, time.UTC)
	first := uuid.MustParse("00000000-0000-0000-0000-000000000247")
	second := uuid.MustParse("00000000-0000-0000-0000-000000000248")
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000249")
	seedSuggestionImport(t, pool, first, alice, "synthetic.bob", now)
	seedSuggestionImport(t, pool, second, alice, "synthetic.bob", now.Add(time.Minute))
	seedSuggestionLink(t, pool, bob, "synthetic.bob", now)
	for index, importID := range []uuid.UUID{first, second} {
		if _, err := suggestionStore.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{
			ID: suggestionID, ImporterDID: alice, TargetDID: bob,
			ImportID: importID, Username: "synthetic.bob", Now: now.Add(time.Duration(index) * time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := importStore.DeleteImport(ctx, alice, first, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var state InstagramSuggestionState
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_follow_suggestions WHERE id = $1`, suggestionID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != SuggestionPending {
		t.Fatalf("state after first support deletion=%s", state)
	}
	if err := importStore.DeleteImport(ctx, alice, second, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_follow_suggestions WHERE id = $1`, suggestionID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != SuggestionInvalidated {
		t.Fatalf("state after last support deletion=%s", state)
	}
}

func newSuggestionTestStore(t *testing.T) (*SuggestionStore, *pgxpool.Pool) {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/000023_instagram_migration.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	return NewSuggestionStore(pool), pool
}

func seedSuggestionImport(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, owner syntax.DID, username string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_imports (
			id, owner_did, state, source_type, retain_unmatched,
			retention_expires_at, following_count, follower_count,
			created_at, updated_at
		) VALUES ($1, $2, 'active', 'manual', true, $3, 1, 0, $4, $4)
	`, id, owner, now.AddDate(1, 0, 0), now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_handles (
			import_id, username_normalized, direction, matched, retain_until, created_at
		) VALUES ($1, $2, 'following', false, $3, $4)
	`, id, username, now.AddDate(1, 0, 0), now); err != nil {
		t.Fatal(err)
	}
}

func seedSuggestionLink(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, username string, now time.Time) {
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
