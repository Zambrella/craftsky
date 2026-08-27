package scheduledposts

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestPrivateMediaServiceIsIdempotentAndOwnerScoped(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	objects := newMemoryPrivateObjectStore()
	service, _ := newScheduledTestMediaService(t, store, objects)
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000601")
	now := time.Now().UTC()
	body := []byte("private-image-bytes")
	params := PutPrivateMediaParams{
		ID: mediaID, OwnerDID: alice, OwnerGeneration: 1,
		MIMEType: "image/jpeg", Bytes: body, Now: now,
	}

	created, err := service.Put(ctx, params)
	if err != nil {
		t.Fatalf("put private media: %v", err)
	}
	if created.State != "ready" || created.BlobCID == "" || created.SizeBytes != int64(len(body)) {
		t.Fatalf("created media=%#v", created)
	}
	if created.RemoteDeadline.Nanosecond()%1_000 != 0 {
		t.Fatalf("remote deadline=%s, want database-canonical microsecond precision", created.RemoteDeadline)
	}
	repeated, err := service.Put(ctx, params)
	if err != nil {
		t.Fatalf("repeat private media: %v", err)
	}
	if repeated != created {
		t.Fatalf("repeated media=%#v, want %#v", repeated, created)
	}
	changed := params
	changed.Bytes = []byte("different-private-image")
	if _, err := service.Put(ctx, changed); !errors.Is(err, ErrScheduledMediaConflict) {
		t.Fatalf("changed retry error=%v, want %v", err, ErrScheduledMediaConflict)
	} else if stage := ScheduledMediaConflictStage(err); stage != "existing_size" {
		t.Fatalf("changed retry stage=%q, want existing_size", stage)
	}

	if _, err := service.Open(ctx, bob, mediaID); !errors.Is(err, ErrScheduledMediaNotFound) {
		t.Fatalf("foreign read error=%v, want %v", err, ErrScheduledMediaNotFound)
	}
	opened, err := service.Open(ctx, alice, mediaID)
	if err != nil {
		t.Fatalf("owner read: %v", err)
	}
	got, err := io.ReadAll(opened.Body)
	closeErr := opened.Body.Close()
	if err != nil || closeErr != nil || !bytes.Equal(got, body) {
		t.Fatalf("owner bytes=%q read=%v close=%v", got, err, closeErr)
	}
	if err := service.Delete(ctx, bob, mediaID, now, 1); err != nil {
		t.Fatalf("foreign idempotent delete: %v", err)
	}
	if _, err := service.Open(ctx, alice, mediaID); err != nil {
		t.Fatalf("foreign delete changed owner media: %v", err)
	}
	if err := service.Delete(ctx, alice, mediaID, now, 1); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
	if err := service.Delete(ctx, alice, mediaID, now, 1); err != nil {
		t.Fatalf("repeat owner delete: %v", err)
	}
	if _, err := service.Open(ctx, alice, mediaID); !errors.Is(err, ErrScheduledMediaNotFound) {
		t.Fatalf("deleted read error=%v, want %v", err, ErrScheduledMediaNotFound)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 1, created.ObjectKey)
}

func TestScheduledMediaConflictStageRejectsUnlistedValues(t *testing.T) {
	err := NewScheduledMediaConflict(MediaConflictStage("private-value-canary"))
	if stage := ScheduledMediaConflictStage(err); stage != "unspecified" {
		t.Fatalf("unlisted conflict stage=%q, want unspecified", stage)
	}
}

type memoryPrivateObjectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newMemoryPrivateObjectStore() *memoryPrivateObjectStore {
	return &memoryPrivateObjectStore{objects: map[string][]byte{}}
}

func (s *memoryPrivateObjectStore) Put(
	_ context.Context,
	key string,
	body io.Reader,
	_ int64,
	_ string,
) error {
	value, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = value
	return nil
}

func (s *memoryPrivateObjectStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.objects[key]
	if !ok {
		return nil, ErrPrivateObjectStoreUnavailable
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), value...))), nil
}

func (s *memoryPrivateObjectStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

func (s *memoryPrivateObjectStore) Exists(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objects[key]
	return ok, nil
}
