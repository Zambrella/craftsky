package instagram

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestEligibilityRestorationHookIsInjectable(t *testing.T) {
	t.Parallel()

	fake := &fakeEligibilityRestorationEnqueuer{}
	importer := syntax.DID("did:plc:synthetic-restored-importer")
	target := syntax.DID("did:plc:synthetic-restored-target")
	if err := enqueueRelationshipSafetyRestoration(context.Background(), fake, importer, target); err != nil {
		t.Fatal(err)
	}
	if fake.importer != importer || fake.target != target || fake.reason != RestorationRelationshipSafe {
		t.Fatalf("restoration call importer=%s target=%s reason=%s", fake.importer, fake.target, fake.reason)
	}
}

func TestReconciliationTriggerPersistsTargetedRestoration(t *testing.T) {
	pool := verificationServicePool(t)
	now := time.Date(2026, 7, 19, 20, 0, 0, 0, time.UTC)
	trigger := NewReconciliationTrigger(pool, func() time.Time { return now })
	importer := syntax.DID("did:plc:synthetic-restored-importer")
	target := syntax.DID("did:plc:synthetic-restored-target")
	if err := trigger.EnqueueEligibilityRestoration(context.Background(), importer, target, RestorationModerationCleared); err != nil {
		t.Fatal(err)
	}
	var owner, storedTarget, reason, status string
	if err := pool.QueryRow(context.Background(), `
		SELECT owner_did,target_did,reason,status FROM instagram_reconciliation_jobs
	`).Scan(&owner, &storedTarget, &reason, &status); err != nil {
		t.Fatal(err)
	}
	if owner != importer.String() || storedTarget != target.String() || reason != string(RestorationModerationCleared) || status != "queued" {
		t.Fatalf("restoration owner=%s target=%s reason=%s status=%s", owner, storedTarget, reason, status)
	}
}

func TestReconciliationTriggerQueuesBothRelationshipDirections(t *testing.T) {
	pool := verificationServicePool(t)
	now := time.Date(2026, 7, 27, 16, 30, 0, 0, time.UTC)
	trigger := NewReconciliationTrigger(pool, func() time.Time { return now })
	left := syntax.DID("did:plc:synthetic-restored-left")
	right := syntax.DID("did:plc:synthetic-restored-right")
	if err := trigger.EnqueueRelationshipSafetyRestoration(
		context.Background(),
		left,
		right,
	); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(context.Background(), `
		SELECT owner_did,target_did,reason
		FROM instagram_reconciliation_jobs
		ORDER BY owner_did
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var pairs [][3]string
	for rows.Next() {
		var pair [3]string
		if err := rows.Scan(&pair[0], &pair[1], &pair[2]); err != nil {
			t.Fatal(err)
		}
		pairs = append(pairs, pair)
	}
	want := [][3]string{
		{left.String(), right.String(), string(RestorationRelationshipSafe)},
		{right.String(), left.String(), string(RestorationRelationshipSafe)},
	}
	if len(pairs) != len(want) ||
		pairs[0] != want[0] ||
		pairs[1] != want[1] {
		t.Fatalf("restoration pairs=%v, want %v", pairs, want)
	}
}

func TestReconciliationTriggerQueuesTargetWideModerationRestoration(t *testing.T) {
	pool := verificationServicePool(t)
	now := time.Date(2026, 7, 27, 16, 45, 0, 0, time.UTC)
	target := syntax.DID("did:plc:synthetic-moderation-target")
	seedSuggestionLink(t, pool, target, "synthetic.moderation", now)
	trigger := NewReconciliationTrigger(pool, func() time.Time { return now })
	if err := trigger.EnqueueModerationRestoration(
		context.Background(),
		target,
	); err != nil {
		t.Fatal(err)
	}
	var owner, reason string
	var linkIDPresent bool
	if err := pool.QueryRow(context.Background(), `
		SELECT owner_did,reason,link_id IS NOT NULL
		FROM instagram_reconciliation_jobs
	`).Scan(&owner, &reason, &linkIDPresent); err != nil {
		t.Fatal(err)
	}
	if owner != target.String() ||
		reason != string(RestorationModerationCleared) ||
		!linkIDPresent {
		t.Fatalf(
			"owner=%s reason=%s link=%t, want target/moderationCleared/true",
			owner,
			reason,
			linkIDPresent,
		)
	}
}

func TestReconciliationTriggerQueuesExpiredModerationOnce(t *testing.T) {
	pool := verificationServicePool(t)
	now := time.Date(2026, 7, 27, 17, 0, 0, 0, time.UTC)
	target := syntax.DID("did:plc:synthetic-expired-moderation-target")
	outputID := uuid.MustParse("00000000-0000-0000-0000-000000000971")
	seedSuggestionLink(t, pool, target, "synthetic.expired", now.Add(-time.Hour))
	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE moderation_outputs (
			id UUID PRIMARY KEY,
			source_did TEXT NOT NULL,
			subject_type TEXT NOT NULL,
			subject_did TEXT NOT NULL,
			value TEXT NOT NULL,
			action TEXT NOT NULL,
			expires_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL
		)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO moderation_outputs (
			id,source_did,subject_type,subject_did,value,action,
			expires_at,created_at,indexed_at
		) VALUES (
			$1,'did:plc:synthetic-labeler','account',$2,'hide','apply',
			$3,$4,$4
		)
	`, outputID, target, now.Add(-time.Minute), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	trigger := NewReconciliationTrigger(pool, func() time.Time { return now })
	first, err := trigger.EnqueueExpiredModerationRestorations(
		context.Background(),
		500,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := trigger.EnqueueExpiredModerationRestorations(
		context.Background(),
		500,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 0 {
		t.Fatalf("expired restoration counts=%d/%d, want 1/0", first, second)
	}
}

// This helper stands in for the future moderation/block/mute owner. Keeping it
// dependent only on the narrow interface proves those features do not need to
// know reconciliation storage or worker details.
func enqueueRelationshipSafetyRestoration(ctx context.Context, enqueuer EligibilityRestorationEnqueuer, importer, target syntax.DID) error {
	return enqueuer.EnqueueEligibilityRestoration(ctx, importer, target, RestorationRelationshipSafe)
}

type fakeEligibilityRestorationEnqueuer struct {
	importer syntax.DID
	target   syntax.DID
	reason   EligibilityRestorationReason
}

func (f *fakeEligibilityRestorationEnqueuer) EnqueueEligibilityRestoration(_ context.Context, importer, target syntax.DID, reason EligibilityRestorationReason) error {
	f.importer = importer
	f.target = target
	f.reason = reason
	return nil
}

var _ EligibilityRestorationEnqueuer = (*fakeEligibilityRestorationEnqueuer)(nil)
