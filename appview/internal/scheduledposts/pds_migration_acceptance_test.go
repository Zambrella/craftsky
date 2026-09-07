package scheduledposts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/pdseffects"
)

func TestPDSMigrationScheduledPublicationCutoff(t *testing.T) {
	tests := []struct {
		name          string
		offset        time.Duration
		sessionErr    error
		wantStatus    Status
		wantPDSWrites int
	}{
		{name: "authorization at 29:59 publishes", offset: 29*time.Minute + 59*time.Second, wantStatus: StatusPublished, wantPDSWrites: 1},
		{name: "authorization at exactly 30:00 publishes", offset: 30 * time.Minute, wantStatus: StatusPublished, wantPDSWrites: 1},
		{name: "failed final attempt needs attention", offset: 30 * time.Minute, sessionErr: auth.ErrNoUsableBackgroundSession, wantStatus: StatusNeedsAttention},
		{name: "authorization after 30:00 cannot publish", offset: 30*time.Minute + time.Second, wantStatus: StatusNeedsAttention},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewStore(newScheduledPostStoreTestPool(t))
			ctx := context.Background()
			due := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
			current := due.Add(test.offset)
			payload, err := EncodePayload(Payload{Kind: PostKindStandard, Text: "publish after reauthorization"})
			if err != nil {
				t.Fatal(err)
			}
			created, err := store.Create(ctx, CreateParams{
				ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
				RequestHash: [32]byte{1}, ScheduledAt: due, PayloadBytes: payload,
				PayloadHash: [32]byte{2}, PayloadVersion: 1,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.pool.Exec(ctx, `
				UPDATE scheduled_posts
				SET status='retrying', next_attempt_at=$2, attempt_count=5
				WHERE id=$1
			`, created.ID, current); err != nil {
				t.Fatal(err)
			}

			pds := &recordingScheduledPDS{}
			processor, err := NewPublicationProcessor(PublicationProcessorOptions{
				Store: store,
				Sessions: stubPublicationSessionSelector{
					wantOwner: "did:plc:alice", sessionID: "current-parent", err: test.sessionErr,
				},
				NewEffects: func(context.Context, syntax.DID, string) (pdseffects.GuardedEffectCoordinator, error) {
					if test.sessionErr != nil {
						return nil, errors.New("effect factory must not run without a session")
					}
					return recordingGuardedFactory(pds, nil)(ctx, "did:plc:alice", "current-parent")
				},
				Objects: newMemoryPrivateObjectStore(),
				Now:     func() time.Time { return current },
			})
			if err != nil {
				t.Fatal(err)
			}
			worker, err := NewWorker(WorkerOptions{Store: store, Processor: processor, Now: func() time.Time { return current }})
			if err != nil {
				t.Fatal(err)
			}
			processed, err := worker.ProcessBatch(ctx)
			if err != nil || processed != 1 {
				t.Fatalf("ProcessBatch() = %d, %v", processed, err)
			}
			resource, err := store.Get(ctx, "did:plc:alice", created.ID)
			if test.wantStatus == StatusPublished {
				if !errors.Is(err, ErrScheduleNotFound) || pds.putCalls != test.wantPDSWrites {
					t.Fatalf("published schedule err=%v writes=%d, want removed and %d", err, pds.putCalls, test.wantPDSWrites)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if resource.Status != test.wantStatus || pds.putCalls != test.wantPDSWrites {
				t.Fatalf("status=%s writes=%d, want %s and %d", resource.Status, pds.putCalls, test.wantStatus, test.wantPDSWrites)
			}
		})
	}
}
