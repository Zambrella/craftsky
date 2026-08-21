package scheduledposts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestDelayedAcceptedPutRemainsTrackedUntilRepeatedCleanup(t *testing.T) {
	pool := newScheduledMediaDurabilityTestPool(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, time.August, 14, 10, 0, 0, 0, time.UTC)
	operationID := uuid.MustParse("10000000-0000-4000-8000-000000000901")
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(id,owner_did,owner_generation) VALUES($1,$2,1)
	`, operationID, owner); err != nil {
		t.Fatalf("seed account-deletion operation: %v", err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("construct owner fencer: %v", err)
	}
	lifecycle, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatalf("construct owner lifecycle store: %v", err)
	}
	store := NewStore(pool)
	objects := newDelayedAcceptedObjectStore()
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000901")
	payload := []byte("private-delayed-payload")
	digest := sha256.Sum256(payload)
	objectKey, uploadAttemptID, err := NewGenerationObjectKey(owner, 1, mediaID)
	if err != nil {
		t.Fatalf("derive upload identity: %v", err)
	}
	err = lifecycle.WithActiveEffects(ctx, []ownerlifecycle.ExpectedOwner{{
		Owner: owner, Generation: 1,
	}}, func(effectCtx context.Context) (effectErr error) {
		objectFence, err := acquirePrivateObjectFence(effectCtx, pool, objectKey)
		if err != nil {
			return err
		}
		defer func() { effectErr = errors.Join(effectErr, objectFence.Release()) }()
		reserved, err := store.reservePrivateMedia(
			effectCtx, mediaID, owner, 1, uploadAttemptID, objectKey,
			"image/jpeg", int64(len(payload)), digest, now.Add(time.Second), nil, now,
		)
		if err != nil {
			return err
		}
		if err := store.markPrivateMediaDispatched(effectCtx, reserved, now); err != nil {
			return err
		}
		// The remote accepted the Put, then the AppView process disappeared
		// before it could record success or enqueue cleanup.
		_ = objects.Put(
			effectCtx, objectKey, bytes.NewReader(payload), int64(len(payload)), "image/jpeg",
		)
		return nil
	})
	if err != nil {
		t.Fatalf("persist crash-boundary upload: %v", err)
	}
	if objects.acceptedKey() == "" {
		t.Fatal("fake object store did not accept an exact key")
	}
	assertRowCount(t, store, `
		SELECT count(*) FROM scheduled_post_media
		WHERE object_key=$1 AND state='uploading'
	`, 1, objectKey)
	deletion := NewAccountDeletion(pool, func() time.Time { return now }, fencer)
	if err := deletion.PurgeAccepted(ctx, operationID, owner, 1); err != nil {
		t.Fatalf("adopt crashed upload during account deletion: %v", err)
	}
	assertRowCount(t, store, `
		SELECT count(*) FROM account_deletion_safety_tombstones
		WHERE operation_id=$1 AND owner_did=$2 AND owner_generation=1
		  AND kind='scheduled_object' AND exact_key=$3
		  AND upload_generation=1 AND source_attempt_id=$4
	`, 1, operationID, owner, objectKey, uploadAttemptID.String())
	assertRowCount(t, store, `
		SELECT count(*) FROM scheduled_post_cleanup_jobs
		WHERE object_key=$1 AND outcome_uncertain
	`, 1, objectKey)

	currentTime := now
	cleanup, err := NewCleanupProcessor(CleanupProcessorOptions{
		Store: store, Objects: objects, OwnerFence: fencer,
		Now: func() time.Time { return currentTime }, BatchSize: 1,
		LeaseDuration: time.Minute, RetryDelay: time.Minute,
	})
	if err != nil {
		t.Fatalf("construct cleanup processor: %v", err)
	}
	processed, err := cleanup.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("early cleanup processed=%d error=%v", processed, err)
	}
	if objects.has(objectKey) {
		t.Fatal("delayed object materialized before release")
	}
	assertRowCount(t, store, `
		SELECT count(*) FROM scheduled_post_cleanup_jobs
		WHERE object_key=$1 AND state='pending'
	`, 1, objectKey)

	objects.materialize()
	if !objects.has(objectKey) {
		t.Fatal("accepted Put did not materialize after early absent cleanup")
	}
	currentTime = now.Add(time.Minute)
	processed, err = cleanup.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("repeated cleanup processed=%d error=%v", processed, err)
	}
	if objects.has(objectKey) {
		t.Fatal("repeated exact-key cleanup did not remove delayed object")
	}
	assertRowCount(t, store, `
		SELECT count(*) FROM scheduled_post_cleanup_jobs
		WHERE object_key=$1 AND state='pending' AND outcome_uncertain
	`, 1, objectKey)
}

func TestAccountDeletionWaitsForLiveUploadReadyBoundary(t *testing.T) {
	pool := newScheduledMediaDurabilityTestPool(t)
	store := NewStore(pool)
	baseObjects := newMemoryPrivateObjectStore()
	objects := &blockingPutObjectStore{
		memoryPrivateObjectStore: baseObjects,
		started:                  make(chan struct{}),
		continuePut:              make(chan struct{}),
	}
	media, ownerFence := newScheduledTestMediaService(t, store, objects)
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, time.August, 14, 11, 0, 0, 0, time.UTC)
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000902")
	ctx := context.Background()

	uploadDone := make(chan error, 1)
	go func() {
		_, err := media.Put(ctx, PutPrivateMediaParams{
			ID: mediaID, OwnerDID: owner, OwnerGeneration: 1,
			MIMEType: "image/jpeg", Bytes: []byte("live-private-upload"), Now: now,
		})
		uploadDone <- err
	}()
	<-objects.started

	deletion := NewAccountDeletion(pool, func() time.Time { return now }, ownerFence)
	deletionDone := make(chan error, 1)
	go func() { deletionDone <- deletion.Purge(ctx, owner) }()
	select {
	case err := <-deletionDone:
		t.Fatalf("account deletion crossed live upload fence early: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(objects.continuePut)
	if err := <-uploadDone; err != nil {
		t.Fatalf("complete live upload: %v", err)
	}
	if err := <-deletionDone; err != nil {
		t.Fatalf("complete fenced account deletion: %v", err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 0, mediaID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs`, 1)
}

func TestTestedSettlementBoundRequiresPostBoundaryAbsence(t *testing.T) {
	pool := newScheduledMediaDurabilityTestPool(t)
	store := NewStore(pool)
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, time.August, 14, 12, 0, 0, 0, time.UTC)
	operationID := uuid.MustParse("10000000-0000-4000-8000-000000000904")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO account_deletion_operations(id,owner_did,owner_generation)
		VALUES($1,$2,1)
	`, operationID, owner); err != nil {
		t.Fatalf("seed account-deletion operation: %v", err)
	}
	fencer := newScheduledTestOwnerFencer(t, pool)
	lifecycle, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatalf("construct owner lifecycle store: %v", err)
	}
	objects := newDelayedAcceptedObjectStore()
	media := NewPrivateMediaService(store, objects, PrivateMediaServiceOptions{
		Lifecycle: lifecycle, PutTimeout: time.Second,
		TestedSettlementBound: 10 * time.Second, SettlementMargin: time.Second,
	})
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000903")
	if _, err := media.Put(context.Background(), PutPrivateMediaParams{
		ID: mediaID, OwnerDID: owner, OwnerGeneration: 1,
		MIMEType: "image/jpeg", Bytes: []byte("bounded-delayed-payload"), Now: now,
	}); !errors.Is(err, ErrPrivateObjectStoreUnavailable) {
		t.Fatalf("outcome-unknown bounded Put error=%v", err)
	}
	objectKey := objects.acceptedKey()
	deletion := NewAccountDeletion(pool, func() time.Time { return now }, fencer)
	if err := deletion.PurgeAccepted(context.Background(), operationID, owner, 1); err != nil {
		t.Fatalf("adopt bounded object attempt: %v", err)
	}
	assertRowCount(t, store, `
		SELECT count(*) FROM account_deletion_safety_tombstones
		WHERE operation_id=$1 AND kind='scheduled_object' AND state='pending'
	`, 1, operationID)
	currentTime := now
	cleanup, err := NewCleanupProcessor(CleanupProcessorOptions{
		Store: store, Objects: objects, OwnerFence: fencer,
		Now: func() time.Time { return currentTime }, BatchSize: 1,
		LeaseDuration: time.Minute, RetryDelay: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("construct bounded cleanup processor: %v", err)
	}
	if processed, err := cleanup.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("early bounded cleanup=%d error=%v", processed, err)
	}
	objects.materialize()
	currentTime = now.Add(5 * time.Second)
	if processed, err := cleanup.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("pre-boundary cleanup=%d error=%v", processed, err)
	}
	if objects.has(objectKey) {
		t.Fatal("pre-boundary cleanup did not remove materialized object")
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 1, objectKey)

	currentTime = now.Add(13 * time.Second)
	if processed, err := cleanup.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("post-boundary cleanup=%d error=%v", processed, err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 0, objectKey)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_object_attempts WHERE object_key=$1`, 0, objectKey)
	assertRowCount(t, store, `
		SELECT count(*) FROM account_deletion_safety_tombstones
		WHERE operation_id=$1 AND kind='scheduled_object' AND state='settled'
	`, 1, operationID)
}

type delayedAcceptedObjectStore struct {
	mu              sync.Mutex
	objects         map[string][]byte
	key             string
	payload         []byte
	materializeOnce sync.Once
}

type blockingPutObjectStore struct {
	*memoryPrivateObjectStore
	started     chan struct{}
	continuePut chan struct{}
}

func (store *blockingPutObjectStore) Put(
	ctx context.Context,
	key string,
	body io.Reader,
	size int64,
	contentType string,
) error {
	payload, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	close(store.started)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-store.continuePut:
		return store.memoryPrivateObjectStore.Put(
			ctx, key, bytes.NewReader(payload), size, contentType,
		)
	}
}

func newDelayedAcceptedObjectStore() *delayedAcceptedObjectStore {
	return &delayedAcceptedObjectStore{objects: make(map[string][]byte)}
}

func (store *delayedAcceptedObjectStore) Put(
	_ context.Context,
	key string,
	body io.Reader,
	_ int64,
	_ string,
) error {
	payload, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	store.mu.Lock()
	store.key = key
	store.payload = append([]byte(nil), payload...)
	store.mu.Unlock()
	// The remote service accepted the bytes, but the client lost the response.
	return context.DeadlineExceeded
}

func (store *delayedAcceptedObjectStore) Open(
	_ context.Context,
	key string,
) (io.ReadCloser, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	payload, ok := store.objects[key]
	if !ok {
		return nil, ErrPrivateObjectStoreUnavailable
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), payload...))), nil
}

func (store *delayedAcceptedObjectStore) Delete(_ context.Context, key string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.objects, key)
	return nil
}

func (store *delayedAcceptedObjectStore) Exists(_ context.Context, key string) (bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	_, ok := store.objects[key]
	return ok, nil
}

func (store *delayedAcceptedObjectStore) acceptedKey() string {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.key
}

func (store *delayedAcceptedObjectStore) materialize() {
	store.materializeOnce.Do(func() {
		store.mu.Lock()
		defer store.mu.Unlock()
		store.objects[store.key] = append([]byte(nil), store.payload...)
	})
}

func (store *delayedAcceptedObjectStore) has(key string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	_, ok := store.objects[key]
	return ok
}

func newScheduledMediaDurabilityTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return newScheduledPostStoreTestPool(t)
}
