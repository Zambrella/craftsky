package instagram

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestFutureMatchReconciliationPersistsOneGenerationBoundPrivateSuggestion(t *testing.T) {
	pool := newPrivateSuggestionTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 13, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-future-importer")
	target := syntax.DID("did:plc:synthetic-future-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001201")
	seedPrivateSuggestionLifecycle(t, pool, importer, 3, now)
	seedPrivateSuggestionLifecycle(t, pool, target, 8, now)
	seedSuggestionImport(t, pool, importID, importer, "synthetic.future", now)
	seedSuggestionLink(t, pool, target, "synthetic.future", now)
	queueLinkReconciliation(t, pool, target, now)
	queueLinkReconciliation(t, pool, target, now)
	store, err := NewPrivateSuggestionStore(
		pool,
		newPrivateSuggestionLifecycleStore(t, pool, now),
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	ids := []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000001202"),
		uuid.MustParse("00000000-0000-0000-0000-000000001203"),
		uuid.MustParse("00000000-0000-0000-0000-000000001204"),
		uuid.MustParse("00000000-0000-0000-0000-000000001205"),
	}

	worker, err := NewReconciliationWorker(ReconciliationWorkerOptions{
		Pool:               pool,
		PrivateSuggestions: store,
		Policy:             newReconciliationPolicy(),
		Now:                func() time.Time { return now },
		NewID: func() uuid.UUID {
			id := ids[0]
			ids = ids[1:]
			return id
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if claimed, err := worker.ProcessBatch(ctx, 2); err != nil || claimed != 2 {
		t.Fatalf("claimed=%d err=%v", claimed, err)
	}

	var suggestions, sources, operations, notifications int
	var importerGeneration, targetGeneration int64
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM instagram_private_suggestions
			  WHERE importer_did=$1 AND target_did=$2),
			(SELECT count(*) FROM instagram_private_suggestion_sources),
			(SELECT count(*) FROM pds_follow_operations
			  WHERE owner_did=$1 AND target_did=$2),
			(SELECT count(*) FROM notification_events
			  WHERE recipient_did=$1 AND category='instagramMatch'),
			(SELECT importer_generation FROM instagram_private_suggestions
			  WHERE importer_did=$1 AND target_did=$2),
			(SELECT target_generation FROM instagram_private_suggestions
			  WHERE importer_did=$1 AND target_did=$2)
	`, importer, target).Scan(
		&suggestions,
		&sources,
		&operations,
		&notifications,
		&importerGeneration,
		&targetGeneration,
	); err != nil {
		t.Fatal(err)
	}
	if suggestions != 1 || sources != 1 || operations != 0 || notifications != 0 ||
		importerGeneration != 3 || targetGeneration != 8 {
		t.Fatalf(
			"suggestions=%d sources=%d operations=%d notifications=%d generations=%d/%d",
			suggestions,
			sources,
			operations,
			notifications,
			importerGeneration,
			targetGeneration,
		)
	}
}

func TestInitialImportMatchingPersistsOneGenerationBoundPrivateSuggestion(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 13, 30, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-initial-auto-importer")
	target := syntax.DID("did:plc:synthetic-initial-auto-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001211")
	seedPrivateSuggestionLifecycle(t, pool, importer, 5, now)
	seedPrivateSuggestionLifecycle(t, pool, target, 11, now)
	seedSuggestionImport(t, pool, importID, importer, "synthetic.initial.auto", now)
	seedSuggestionLink(t, pool, target, "synthetic.initial.auto", now)

	matcher := NewPrivateSuggestionMatcher(
		pool,
		newPrivateSuggestionStoreForReconciliationTest(t, pool, func() time.Time { return now }),
		newReconciliationPolicy(),
		func() time.Time { return now },
	)
	for range 2 {
		if _, err := matcher.MatchImport(ctx, importer, importID); err != nil {
			t.Fatal(err)
		}
	}
	var suggestions, sources, operations, notifications int
	var importerGeneration, targetGeneration int64
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM instagram_private_suggestions
			  WHERE importer_did=$1 AND target_did=$2),
			(SELECT count(*) FROM instagram_private_suggestion_sources),
			(SELECT count(*) FROM pds_follow_operations
			  WHERE owner_did=$1 AND target_did=$2),
			(SELECT count(*) FROM notification_events
			  WHERE recipient_did=$1 AND category='instagramMatch'),
			(SELECT importer_generation FROM instagram_private_suggestions
			  WHERE importer_did=$1 AND target_did=$2),
			(SELECT target_generation FROM instagram_private_suggestions
			  WHERE importer_did=$1 AND target_did=$2)
	`, importer, target).Scan(
		&suggestions,
		&sources,
		&operations,
		&notifications,
		&importerGeneration,
		&targetGeneration,
	); err != nil {
		t.Fatal(err)
	}
	if suggestions != 1 || sources != 1 || operations != 0 || notifications != 0 ||
		importerGeneration != 5 || targetGeneration != 11 {
		t.Fatalf(
			"suggestions=%d sources=%d operations=%d notifications=%d generations=%d/%d",
			suggestions,
			sources,
			operations,
			notifications,
			importerGeneration,
			targetGeneration,
		)
	}
}
