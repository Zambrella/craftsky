package scheduledposts

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/pdseffects"
)

func TestTransientFailureUsesAllSixAttemptsThenRequiresRecovery(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	due := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	current := due
	payload, _ := EncodePayload(Payload{Kind: PostKindStandard, Text: "retain this exact payload"})
	created, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
		RequestHash: [32]byte{1}, ScheduledAt: due, PayloadBytes: payload,
		PayloadHash: [32]byte{2}, PayloadVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			wantOwner: "did:plc:alice", err: auth.ErrNoUsableBackgroundSession,
		},
		NewEffects: func(context.Context, syntax.DID, string) (pdseffects.GuardedEffectCoordinator, error) {
			t.Fatal("PDS factory called without a usable owner session")
			return nil, errors.New("unreachable")
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

	offsets := []time.Duration{0, time.Minute, 3 * time.Minute, 7 * time.Minute, 15 * time.Minute, 30 * time.Minute}
	for index, offset := range offsets {
		current = due.Add(offset)
		processed, err := worker.ProcessBatch(ctx)
		if err != nil || processed != 1 {
			t.Fatalf("attempt %d at %s: processed=%d err=%v", index+1, offset, processed, err)
		}
		resource, err := store.Get(ctx, "did:plc:alice", created.ID)
		if err != nil {
			t.Fatal(err)
		}
		wantStatus := StatusRetrying
		if index == len(offsets)-1 {
			wantStatus = StatusNeedsAttention
		}
		if resource.Status != wantStatus || !bytes.Equal(resource.PayloadBytes, payload) {
			t.Fatalf("attempt %d status=%s payload=%s", index+1, resource.Status, resource.PayloadBytes)
		}
		var attempts int
		if err := store.pool.QueryRow(ctx, `SELECT attempt_count FROM scheduled_posts WHERE id=$1`, created.ID).Scan(&attempts); err != nil {
			t.Fatal(err)
		}
		if attempts != index+1 {
			t.Fatalf("attempt_count=%d, want %d", attempts, index+1)
		}
	}

	current = due.Add(31 * time.Minute)
	processed, err := worker.ProcessBatch(ctx)
	if err != nil || processed != 0 {
		t.Fatalf("automatic attempt after exhaustion: processed=%d err=%v", processed, err)
	}
}

func TestPermanentPolicyFailurePreservesMemberContent(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	payload, _ := EncodePayload(Payload{Kind: PostKindStandard, Text: "policy can change, content must not"})
	created, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
		RequestHash: [32]byte{1}, ScheduledAt: now, PayloadBytes: payload,
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
		NewEffects: recordingGuardedFactory(pds, nil),
		Objects:    newMemoryPrivateObjectStore(),
		Now:        func() time.Time { return now },
		Validate: func(context.Context, syntax.DID, Payload) error {
			return ErrPolicyInvalid
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	worker, _ := NewWorker(WorkerOptions{Store: store, Processor: processor, Now: func() time.Time { return now }})
	processed, err := worker.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("policy batch: processed=%d err=%v", processed, err)
	}
	resource, err := store.Get(ctx, "did:plc:alice", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resource.Status != StatusNeedsAttention || !bytes.Equal(resource.PayloadBytes, payload) || pds.putCalls != 0 {
		t.Fatalf("status=%s payload=%s puts=%d", resource.Status, resource.PayloadBytes, pds.putCalls)
	}
}
