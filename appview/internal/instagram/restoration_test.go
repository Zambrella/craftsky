package instagram

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
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
