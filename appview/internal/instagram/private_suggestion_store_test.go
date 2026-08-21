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

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestPrivateSuggestionMatcherStopsAtPrivatePersistence(t *testing.T) {
	pool := newPrivateSuggestionTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 13, 30, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:matcher-importer")
	target := syntax.DID("did:plc:matcher-target")
	importID := uuid.MustParse("01000000-0000-4000-8000-000000000001")
	seedPrivateSuggestionLifecycle(t, pool, importer, 2, now)
	seedPrivateSuggestionLifecycle(t, pool, target, 6, now)
	seedSuggestionImport(t, pool, importID, importer, "matcher.target", now)
	seedSuggestionLink(t, pool, target, "matcher.target", now)

	store, err := NewPrivateSuggestionStore(
		pool,
		newPrivateSuggestionLifecycleStore(t, pool, now),
		notifications.NewService(),
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	matcher := NewPrivateSuggestionMatcher(
		pool,
		store,
		privateSuggestionAllowPolicy{},
		func() time.Time { return now },
	)
	ids := []uuid.UUID{
		uuid.MustParse("02000000-0000-4000-8000-000000000001"),
		uuid.MustParse("02000000-0000-4000-8000-000000000002"),
	}
	matcher.newID = func() uuid.UUID {
		id := ids[0]
		ids = ids[1:]
		return id
	}

	first, err := matcher.MatchImport(ctx, importer, importID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := matcher.MatchImport(ctx, importer, importID)
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 0 {
		t.Fatalf("created suggestions = %d/%d, want 1/0", first, second)
	}
	for _, table := range []string{"pds_follow_operations"} {
		var count int
		query := "SELECT count(*)::int FROM " + table
		if err := pool.QueryRow(ctx, query).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("forbidden effect table %s has %d rows", table, count)
		}
	}
	var (
		notifications int
		actor         string
		sourceURI     *string
	)
	if err := pool.QueryRow(ctx, `
		SELECT count(*)::int,min(actor_did),min(source_uri)
		FROM notification_events
		WHERE recipient_did=$1 AND category='instagramMatch'
	`, importer).Scan(&notifications, &actor, &sourceURI); err != nil {
		t.Fatal(err)
	}
	if notifications != 1 || actor != target.String() || sourceURI != nil {
		t.Fatalf("instagram matches = %d actor=%q source=%v, want one source-less target event", notifications, actor, sourceURI)
	}
}

func TestPrivateSuggestionStoreReconcilesOneGenerationBoundSuggestion(t *testing.T) {
	pool := newPrivateSuggestionTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 14, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:private-importer")
	target := syntax.DID("did:plc:private-target")
	importID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	seedPrivateSuggestionLifecycle(t, pool, importer, 4, now)
	seedPrivateSuggestionLifecycle(t, pool, target, 7, now)
	seedSuggestionImport(t, pool, importID, importer, "private.target", now)
	seedSuggestionLink(t, pool, target, "private.target", now)

	lifecycles := newPrivateSuggestionLifecycleStore(t, pool, now)
	store, err := NewPrivateSuggestionStore(pool, lifecycles, notifications.NewService(), func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewPrivateSuggestionStore: %v", err)
	}
	firstID := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	secondID := uuid.MustParse("20000000-0000-4000-8000-000000000002")
	first, err := store.ReconcileCandidate(ctx, ReconcilePrivateSuggestionParams{
		ID: firstID, ImporterDID: importer, TargetDID: target,
		ImportID: importID, Username: "private.target", Now: now,
	})
	if err != nil {
		t.Fatalf("first ReconcileCandidate: %v", err)
	}
	second, err := store.ReconcileCandidate(ctx, ReconcilePrivateSuggestionParams{
		ID: secondID, ImporterDID: importer, TargetDID: target,
		ImportID: importID, Username: "@PRIVATE.TARGET", Now: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("second ReconcileCandidate: %v", err)
	}
	if !first.Created || second.Created {
		t.Fatalf("created flags = %t/%t, want true/false", first.Created, second.Created)
	}
	if first.Suggestion.ID != firstID || second.Suggestion.ID != firstID {
		t.Fatalf("canonical suggestion IDs = %s/%s, want %s", first.Suggestion.ID, second.Suggestion.ID, firstID)
	}
	if first.Suggestion.ImporterGeneration != 4 || first.Suggestion.TargetGeneration != 7 {
		t.Fatalf("suggestion generations = %d/%d, want 4/7", first.Suggestion.ImporterGeneration, first.Suggestion.TargetGeneration)
	}

	var (
		suggestions   int
		sources       int
		operations    int
		notifications int
	)
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM instagram_private_suggestions`).Scan(&suggestions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM instagram_private_suggestion_sources`).Scan(&sources); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM pds_follow_operations`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM notification_events WHERE category='instagramMatch'`).Scan(&notifications); err != nil {
		t.Fatal(err)
	}
	if suggestions != 1 || sources != 1 || operations != 0 || notifications != 1 {
		t.Fatalf(
			"rows suggestions/sources/operations/notifications = %d/%d/%d/%d, want 1/1/0/1",
			suggestions, sources, operations, notifications,
		)
	}
}

func TestPrivateSuggestionStoreDismissIsPrivateIdempotentAndTerminal(t *testing.T) {
	pool := newPrivateSuggestionTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 14, 30, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:dismiss-importer")
	target := syntax.DID("did:plc:dismiss-target")
	foreign := syntax.DID("did:plc:dismiss-foreign")
	importID := uuid.MustParse("30000000-0000-4000-8000-000000000001")
	seedPrivateSuggestionLifecycle(t, pool, importer, 2, now)
	seedPrivateSuggestionLifecycle(t, pool, target, 3, now)
	seedPrivateSuggestionLifecycle(t, pool, foreign, 5, now)
	seedSuggestionImport(t, pool, importID, importer, "dismiss.target", now)
	seedSuggestionLink(t, pool, target, "dismiss.target", now)

	store, err := NewPrivateSuggestionStore(
		pool,
		newPrivateSuggestionLifecycleStore(t, pool, now),
		notifications.NewService(),
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.ReconcileCandidate(ctx, ReconcilePrivateSuggestionParams{
		ID:          uuid.MustParse("40000000-0000-4000-8000-000000000001"),
		ImporterDID: importer, TargetDID: target, ImportID: importID,
		Username: "dismiss.target", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	changed, err := store.Dismiss(ctx, foreign, created.Suggestion.ID)
	if err != nil || changed {
		t.Fatalf("foreign dismiss = %t, %v; want false, nil", changed, err)
	}
	changed, err = store.Dismiss(ctx, importer, created.Suggestion.ID)
	if err != nil || !changed {
		t.Fatalf("owner dismiss = %t, %v; want true, nil", changed, err)
	}
	changed, err = store.Dismiss(ctx, importer, created.Suggestion.ID)
	if err != nil || changed {
		t.Fatalf("replayed dismiss = %t, %v; want false, nil", changed, err)
	}

	var state SuggestionState
	var terminalAt *time.Time
	if err := pool.QueryRow(ctx, `
		SELECT state,terminal_at
		FROM instagram_private_suggestions
		WHERE id=$1
	`, created.Suggestion.ID).Scan(&state, &terminalAt); err != nil {
		t.Fatal(err)
	}
	if state != SuggestionDismissed || terminalAt == nil {
		t.Fatalf("dismissed suggestion = %q terminal:%v", state, terminalAt)
	}
}

func newPrivateSuggestionTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, "")
	ensureInstagramOwnerLifecyclePreState(t, pool)
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000025_instagram_migration.up.sql",
		"../../migrations/000026_system_notifications.up.sql",
		"../../migrations/000029_notification_client_owned_destination.up.sql",
		"../../migrations/000030_instagram_automatic_follows.up.sql",
		"../../migrations/000031_instagram_automatic_follow_storage_names.up.sql",
		"../../migrations/000038_owner_auth_lifecycle.up.sql",
		"../../migrations/000039_owner_effects_terminal_purge.up.sql",
		"../../migrations/000042_instagram_private_suggestions.up.sql",
		"../../migrations/000055_instagram_match_notifications.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	return pool
}

func newPrivateSuggestionLifecycleStore(
	t *testing.T,
	pool *pgxpool.Pool,
	now time.Time,
) *ownerlifecycle.Store {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	store, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func seedPrivateSuggestionLifecycle(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	generation int64,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES ($1,'active',$2,1,'testActive',$3,$3,$3)
	`, owner, generation, now); err != nil {
		t.Fatalf("seed lifecycle %s: %v", owner, err)
	}
}

type privateSuggestionAllowPolicy struct{}

func (privateSuggestionAllowPolicy) Evaluate(
	_ context.Context,
	stage EligibilityStage,
	_ SuggestionEligibilityRequest,
) (EligibilityDecision, error) {
	if stage != EligibilityAtMatch && stage != EligibilityAtPersist && stage != EligibilityAtAccept {
		return EligibilityDecision{}, errors.New("unexpected private suggestion policy stage")
	}
	return EligibilityDecision{Eligible: true, Reason: EligibilityAllowed}, nil
}
