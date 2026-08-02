package scheduledposts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
)

type acceptanceOperationalObserver struct {
	queues       []string
	operations   []string
	publications []string
	cleanups     []string
}

func (o *acceptanceOperationalObserver) ObserveScheduledQueue(status string, count, due, overdue int, oldest time.Duration) {
	o.queues = append(o.queues, fmt.Sprintf("%s:%d:%d:%d:%s", status, count, due, overdue, oldest))
}

func (o *acceptanceOperationalObserver) ObserveScheduledOperation(operation, result, errorClass string, duration time.Duration) {
	o.operations = append(o.operations, fmt.Sprintf("%s:%s:%s:%s", operation, result, errorClass, duration))
}

func (o *acceptanceOperationalObserver) ObserveScheduledPublication(attempt int, latency, duration time.Duration) {
	o.publications = append(o.publications, fmt.Sprintf("%d:%s:%s", attempt, latency, duration))
}

func (o *acceptanceOperationalObserver) ObserveScheduledCleanupQueue(pending int, oldest time.Duration) {
	o.cleanups = append(o.cleanups, fmt.Sprintf("%d:%s", pending, oldest))
}

func TestScheduledOperationsEmitBoundedContentFreeSignals(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	observer := &acceptanceOperationalObserver{}
	store.SetOperationalObserver(observer)
	ctx := context.Background()
	due := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	current := due.Add(90 * time.Second)
	payload, _ := EncodePayload(Payload{Kind: PostKindStandard, Text: "private-observability-canary"})
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
		NewPDS:  func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return nil, ErrAuthUnavailable },
		Objects: newMemoryPrivateObjectStore(), Now: func() time.Time { return current }, Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	worker, _ := NewWorker(WorkerOptions{Store: store, Processor: processor, Now: func() time.Time { return current }, Observer: observer})
	if processed, err := worker.ProcessBatch(ctx); err != nil || processed != 1 {
		t.Fatalf("retry batch=%d err=%v", processed, err)
	}

	// An expired Publishing lease is reclaimed and the prior worker becomes stale.
	current = due.Add(time.Minute)
	if _, err := store.pool.Exec(ctx, `
		UPDATE scheduled_posts SET status='publishing', lease_token=$2,
		lease_expires_at=$3, publication_rkey='3jzfcijpj2z2a',
		publication_created_at=$4 WHERE id=$1
	`, created.ID, uuid.New(), current.Add(-time.Second), due); err != nil {
		t.Fatal(err)
	}
	claims, err := store.ClaimDue(ctx, 1, current, time.Minute)
	if err != nil || len(claims) != 1 {
		t.Fatalf("recovery claims=%v err=%v", claims, err)
	}
	stale := claims[0]
	stale.LeaseToken = uuid.New()
	if err := processor.Process(ctx, WorkItem{
		ID: stale.ID, OwnerDID: stale.OwnerDID, LeaseToken: stale.LeaseToken,
		PayloadVersion: stale.PayloadVersion, Rkey: stale.Rkey, CreatedAt: stale.CreatedAt,
	}); err != nil {
		t.Fatal(err)
	}
	validRecovery := claims[0]
	if err := processor.Process(ctx, WorkItem{
		ID: validRecovery.ID, OwnerDID: validRecovery.OwnerDID,
		LeaseToken:     validRecovery.LeaseToken,
		PayloadVersion: validRecovery.PayloadVersion,
		Rkey:           validRecovery.Rkey, CreatedAt: validRecovery.CreatedAt,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, "did:plc:alice", created.ID, current); err != nil {
		t.Fatal(err)
	}

	// A healthy due item publishes and reports its start/completion timing.
	publishPayload, _ := EncodePayload(Payload{
		Kind: PostKindStandard, Text: "private-publish-canary",
	})
	if _, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:bob", OperationID: uuid.New(),
		RequestHash: [32]byte{3}, ScheduledAt: current,
		PayloadBytes: publishPayload, PayloadHash: [32]byte{4}, PayloadVersion: 1,
	}); err != nil {
		t.Fatal(err)
	}
	pds := &recordingScheduledPDS{}
	publishProcessor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			wantOwner: "did:plc:bob", sessionID: "private-owner-session-canary",
		},
		NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return pds, nil
		},
		Objects: newMemoryPrivateObjectStore(),
		Now:     func() time.Time { return current }, Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	publishWorker, _ := NewWorker(WorkerOptions{
		Store: store, Processor: publishProcessor,
		Now: func() time.Time { return current }, Observer: observer,
	})
	if processed, err := publishWorker.ProcessBatch(ctx); err != nil || processed != 1 {
		t.Fatalf("publish batch=%d err=%v", processed, err)
	}

	// Six authentication failures emit retries followed by Needs attention.
	exhaustDue := current.Add(time.Minute)
	exhaustPayload, _ := EncodePayload(Payload{
		Kind: PostKindStandard, Text: "private-exhaustion-canary",
	})
	if _, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:bob", OperationID: uuid.New(),
		RequestHash: [32]byte{5}, ScheduledAt: exhaustDue,
		PayloadBytes: exhaustPayload, PayloadHash: [32]byte{6}, PayloadVersion: 1,
	}); err != nil {
		t.Fatal(err)
	}
	exhaustProcessor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			wantOwner: "did:plc:bob", err: auth.ErrNoUsableBackgroundSession,
		},
		NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return nil, errors.New("private-provider-response-canary")
		},
		Objects: newMemoryPrivateObjectStore(),
		Now:     func() time.Time { return current }, Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	exhaustWorker, _ := NewWorker(WorkerOptions{
		Store: store, Processor: exhaustProcessor,
		Now: func() time.Time { return current }, Observer: observer,
	})
	for _, offset := range []time.Duration{
		0, time.Minute, 3 * time.Minute, 7 * time.Minute,
		15 * time.Minute, 30 * time.Minute,
	} {
		current = exhaustDue.Add(offset)
		if processed, err := exhaustWorker.ProcessBatch(ctx); err != nil || processed != 1 {
			t.Fatalf("auth exhaustion at %s: processed=%d err=%v", offset, processed, err)
		}
	}

	objects := newMemoryPrivateObjectStore()
	mediaService := NewPrivateMediaService(store, objects)
	mediaID := uuid.New()
	if _, err := mediaService.Put(ctx, PutPrivateMediaParams{
		ID: mediaID, OwnerDID: "did:plc:alice", MIMEType: "image/jpeg",
		Bytes: []byte("private-cleanup-canary"), Now: current,
	}); err != nil {
		t.Fatal(err)
	}
	if err := mediaService.Delete(ctx, "did:plc:alice", mediaID, current); err != nil {
		t.Fatal(err)
	}
	cleanup, err := NewCleanupProcessor(CleanupProcessorOptions{
		Store: store, Objects: objects, Now: func() time.Time { return current }, Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := cleanup.ProcessBatch(ctx); err != nil || processed != 1 {
		t.Fatalf("cleanup batch=%d err=%v", processed, err)
	}

	failingObjects := &flakyCleanupObjectStore{objects: map[string][]byte{}}
	failingMedia := NewPrivateMediaService(store, failingObjects)
	failingMediaID := uuid.New()
	if _, err := failingMedia.Put(ctx, PutPrivateMediaParams{
		ID: failingMediaID, OwnerDID: "did:plc:alice", MIMEType: "image/jpeg",
		Bytes: []byte("private-cleanup-failure-canary"), Now: current,
	}); err != nil {
		t.Fatal(err)
	}
	if err := failingMedia.Delete(ctx, "did:plc:alice", failingMediaID, current); err != nil {
		t.Fatal(err)
	}
	failingObjects.failDeletes = 1
	failingCleanup, err := NewCleanupProcessor(CleanupProcessorOptions{
		Store: store, Objects: failingObjects,
		Now: func() time.Time { return current }, Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := failingCleanup.ProcessBatch(ctx); err != nil || processed != 1 {
		t.Fatalf("failing cleanup batch=%d err=%v", processed, err)
	}

	joined := strings.Join(append(append(append(observer.queues, observer.operations...), observer.publications...), observer.cleanups...), "|")
	for _, want := range []string{
		"scheduled:",
		"publish:success:none",
		"retry:failure:auth_unavailable",
		"needs_attention:failure:auth_unavailable",
		"recover:success:lease_expired",
		"stale_worker:stale:stale_worker",
		"cleanup:success:none",
		"cleanup:failure:object_delete_failed",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("operational signals missing %q: %s", want, joined)
		}
	}
	if len(observer.publications) == 0 || len(observer.cleanups) == 0 {
		t.Fatalf("publication/cleanup measurements missing: %s", joined)
	}
	for _, canary := range []string{
		"private-observability-canary",
		"private-publish-canary",
		"private-owner-session-canary",
		"private-exhaustion-canary",
		"private-provider-response-canary",
		"private-cleanup-canary",
		"private-cleanup-failure-canary",
		"did:plc:alice",
		"did:plc:bob",
		mediaID.String(),
		failingMediaID.String(),
	} {
		if strings.Contains(joined, canary) {
			t.Fatalf("signal output leaked %q: %s", canary, joined)
		}
	}
}
