package instagram

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestFutureMatchReconciliationQueuesOnePrivateAutomaticFollow(t *testing.T) {
	pool := newReconciliationTestPool(t)
	applyAutomaticFollowMigration(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 13, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-future-importer")
	target := syntax.DID("did:plc:synthetic-future-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001201")
	seedSuggestionImport(t, pool, importID, importer, "synthetic.future", now)
	seedSuggestionLink(t, pool, target, "synthetic.future", now)
	queueLinkReconciliation(t, pool, target, now)
	queueLinkReconciliation(t, pool, target, now)

	worker, err := NewReconciliationWorker(ReconciliationWorkerOptions{
		Pool:             pool,
		AutomaticFollows: NewAutomaticFollowStore(pool),
		Policy:           newReconciliationPolicy(),
		Now:              func() time.Time { return now },
		NewID: func() uuid.UUID {
			return uuid.MustParse("00000000-0000-0000-0000-000000001202")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if claimed, err := worker.ProcessBatch(ctx, 2); err != nil || claimed != 2 {
		t.Fatalf("claimed=%d err=%v", claimed, err)
	}

	var ledgers, operations, notifications int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM instagram_follow_suggestions
			  WHERE importer_did=$1 AND target_did=$2),
			(SELECT count(*) FROM pds_follow_operations
			  WHERE owner_did=$1 AND target_did=$2 AND status='pending'),
			(SELECT count(*) FROM notification_events
			  WHERE recipient_did=$1 AND category='instagramMatch')
	`, importer, target).Scan(&ledgers, &operations, &notifications); err != nil {
		t.Fatal(err)
	}
	if ledgers != 1 || operations != 1 || notifications != 0 {
		t.Fatalf(
			"ledgers=%d operations=%d notifications=%d",
			ledgers,
			operations,
			notifications,
		)
	}
}

func TestInitialImportMatchingQueuesOnePrivateAutomaticFollow(t *testing.T) {
	pool := newReconciliationTestPool(t)
	applyAutomaticFollowMigration(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 13, 30, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-initial-auto-importer")
	target := syntax.DID("did:plc:synthetic-initial-auto-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001211")
	seedSuggestionImport(t, pool, importID, importer, "synthetic.initial.auto", now)
	seedSuggestionLink(t, pool, target, "synthetic.initial.auto", now)

	matcher := NewAutomaticFollowMatcher(
		pool,
		NewAutomaticFollowStore(pool),
		newReconciliationPolicy(),
		func() time.Time { return now },
	)
	for range 2 {
		if _, err := matcher.MatchImport(ctx, importer, importID); err != nil {
			t.Fatal(err)
		}
	}
	var operations, notifications int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM pds_follow_operations
			  WHERE owner_did=$1 AND target_did=$2 AND status='pending'),
			(SELECT count(*) FROM notification_events
			  WHERE recipient_did=$1 AND category='instagramMatch')
	`, importer, target).Scan(&operations, &notifications); err != nil {
		t.Fatal(err)
	}
	if operations != 1 || notifications != 0 {
		t.Fatalf("operations=%d notifications=%d", operations, notifications)
	}
}
