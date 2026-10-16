package instagram

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/notifications"
)

func TestPendingSuggestionsExcludeFollowedAccountsBeforePagination(t *testing.T) {
	pool := newPrivateSuggestionTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:suggestion-pagination-owner")
	seedPrivateSuggestionLifecycle(t, pool, owner, 1, now)
	store, err := NewPrivateSuggestionStore(pool, newPrivateSuggestionLifecycleStore(t, pool, now), notifications.NewService(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	var ids []uuid.UUID
	for i := range 3 {
		target := syntax.DID(fmt.Sprintf("did:plc:pagination-target-%d", i))
		username := fmt.Sprintf("pagination.target.%d", i)
		createdAt := now.Add(time.Duration(i) * time.Second)
		seedPrivateSuggestionLifecycle(t, pool, target, 1, now)
		importID := uuid.New()
		seedSuggestionImport(t, pool, importID, owner, username, now)
		seedSuggestionLink(t, pool, target, username, now)
		created, err := store.ReconcileCandidate(ctx, ReconcilePrivateSuggestionParams{
			ID: uuid.New(), ImporterDID: owner, TargetDID: target, ImportID: importID, Username: username, Now: createdAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, created.Suggestion.ID)
	}
	// The newest suggestion was followed through another surface. The canonical
	// follow projection must exclude it before the page's LIMIT is applied.
	if _, err := pool.Exec(ctx, `CREATE TABLE pds_set_aggregates (
  kind TEXT NOT NULL, actor_did TEXT NOT NULL, subject_did TEXT
 )`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO pds_set_aggregates (kind,actor_did,subject_did)
 VALUES ('follow',$1,$2)`, owner, "did:plc:pagination-target-2"); err != nil {
		t.Fatal(err)
	}
	first, cursor, err := store.ListPending(ctx, owner, 1, nil)
	if err != nil || len(first) != 1 || first[0].ID != ids[1] || cursor == nil {
		t.Fatalf("first page = %v cursor=%v err=%v", first, cursor, err)
	}
	second, next, err := store.ListPending(ctx, owner, 1, cursor)
	if err != nil || len(second) != 1 || second[0].ID != ids[0] || next != nil {
		t.Fatalf("second page = %v cursor=%v err=%v", second, next, err)
	}
	retained, err := store.GetOwned(ctx, owner, ids[2])
	if err != nil || retained.State != SuggestionPending {
		t.Fatalf("following elsewhere must preserve private suggestion history: %v err=%v", retained, err)
	}
}
