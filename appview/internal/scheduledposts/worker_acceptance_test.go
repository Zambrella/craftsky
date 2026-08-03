package scheduledposts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
)

func TestHealthyDueSchedulePublishesExactlyOnceWithoutFlutter(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	due := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	current := due.Add(-time.Second)
	payload, err := EncodePayload(Payload{
		Kind: PostKindStandard, Text: "publish while the app is closed", Langs: []string{"en"},
	})
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
	pds := &recordingScheduledPDS{}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			wantOwner: "did:plc:alice", sessionID: "owner-session",
		},
		NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return pds, nil
		},
		Objects: newMemoryPrivateObjectStore(),
		Now:     func() time.Time { return current },
	})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewWorker(WorkerOptions{
		Store: store, Processor: processor, Now: func() time.Time { return current },
	})
	if err != nil {
		t.Fatal(err)
	}

	processed, err := worker.ProcessBatch(ctx)
	if err != nil || processed != 0 || pds.putCalls != 0 || pds.uploadCalls != 0 {
		t.Fatalf("before due: processed=%d err=%v puts=%d uploads=%d", processed, err, pds.putCalls, pds.uploadCalls)
	}

	current = due.Add(30 * time.Second)
	processed, err = worker.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("due batch: processed=%d err=%v", processed, err)
	}
	if pds.putCalls != 1 || pds.rkey == "" || pds.record["createdAt"] != current.Format(time.RFC3339) {
		t.Fatalf("publication puts=%d rkey=%q record=%#v", pds.putCalls, pds.rkey, pds.record)
	}
	if _, err := store.Get(ctx, "did:plc:alice", created.ID); !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("published schedule remains manageable: %v", err)
	}

	processed, err = worker.ProcessBatch(ctx)
	if err != nil || processed != 0 || pds.putCalls != 1 {
		t.Fatalf("repeated batch: processed=%d err=%v puts=%d", processed, err, pds.putCalls)
	}
}
