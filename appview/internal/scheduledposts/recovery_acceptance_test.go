package scheduledposts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestScheduledPublicationRecoversTheSameFrozenRecordAcrossCrashBoundaries(t *testing.T) {
	t.Run("before any PDS media upload", func(t *testing.T) {
		fixture := newPublicationRecoveryFixture(t, 1)
		fixture.pds.panicBeforeUploadAt = 1

		stopped := fixture.stop(t)
		if fixture.pds.uploadCalls != 0 || fixture.pds.putCalls != 0 {
			t.Fatalf("uploads=%d puts=%d, want zero before the stopped upload boundary", fixture.pds.uploadCalls, fixture.pds.putCalls)
		}
		frozen := fixture.frozenRecord(t)
		if len(frozen) == 0 {
			t.Fatal("record was not frozen before the stopped-upload boundary")
		}
		fixture.recover(t, stopped, frozen)
		fixture.assertFinalizedWithRecord(t, frozen)
	})

	t.Run("after a partial PDS media upload", func(t *testing.T) {
		fixture := newPublicationRecoveryFixture(t, 2)
		fixture.pds.panicBeforeUploadAt = 2

		stopped := fixture.stop(t)
		if fixture.pds.uploadCalls != 1 || fixture.pds.putCalls != 0 {
			t.Fatalf("uploads=%d puts=%d, want partial upload and no record", fixture.pds.uploadCalls, fixture.pds.putCalls)
		}
		frozen := fixture.frozenRecord(t)
		fixture.recover(t, stopped, frozen)
		fixture.assertFinalizedWithRecord(t, frozen)
	})

	t.Run("after media but before the record write", func(t *testing.T) {
		fixture := newPublicationRecoveryFixture(t, 1)
		fixture.pds.panicBeforeGet = true

		stopped := fixture.stop(t)
		if fixture.pds.uploadCalls != 1 || fixture.pds.putCalls != 0 {
			t.Fatalf("uploads=%d puts=%d, want uploads complete before interrupted lookup", fixture.pds.uploadCalls, fixture.pds.putCalls)
		}
		frozen := fixture.frozenRecord(t)
		fixture.recover(t, stopped, frozen)
		fixture.assertFinalizedWithRecord(t, frozen)
	})

	t.Run("after PDS record success before local completion", func(t *testing.T) {
		fixture := newPublicationRecoveryFixture(t, 1)
		fixture.pds.panicAfterPut = true

		stopped := fixture.stop(t)
		if fixture.pds.putCalls != 1 || fixture.pds.record == nil {
			t.Fatalf("puts=%d record=%#v, want record committed before worker stop", fixture.pds.putCalls, fixture.pds.record)
		}
		frozen := fixture.frozenRecord(t)
		fixture.recover(t, stopped, frozen)
		if fixture.pds.putCalls != 1 {
			t.Fatalf("record writes=%d, want recovery to reconcile without a second write", fixture.pds.putCalls)
		}
		fixture.assertFinalizedWithRecord(t, frozen)
	})
}

type publicationRecoveryFixture struct {
	store   *Store
	objects *memoryPrivateObjectStore
	pds     *recordingScheduledPDS
	owner   syntax.DID
	id      uuid.UUID
	now     time.Time
	clock   *time.Time
	media   []PrivateMedia
	bodies  [][]byte
	worker  *Worker
}

func newPublicationRecoveryFixture(t *testing.T, mediaCount int) *publicationRecoveryFixture {
	t.Helper()
	store := NewStore(newScheduledPostStoreTestPool(t))
	objects := newMemoryPrivateObjectStore()
	mediaService, _ := newScheduledTestMediaService(t, store, objects)
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	current := now
	media := make([]PrivateMedia, 0, mediaCount)
	bodies := make([][]byte, 0, mediaCount)
	payloadMedia := make([]PayloadMedia, 0, mediaCount)
	mediaIDs := make([]uuid.UUID, 0, mediaCount)
	for index := range mediaCount {
		id := uuid.New()
		body := []byte{byte(index + 1), 'p', 'r', 'i', 'v', 'a', 't', 'e'}
		staged, err := mediaService.Put(context.Background(), PutPrivateMediaParams{
			ID: id, OwnerDID: owner, OwnerGeneration: 1,
			MIMEType: "image/jpeg", Bytes: body, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		media = append(media, staged)
		bodies = append(bodies, body)
		payloadMedia = append(payloadMedia, PayloadMedia{ID: id.String(), Alt: "private"})
		mediaIDs = append(mediaIDs, id)
	}
	payload, err := EncodePayload(Payload{
		Kind: PostKindStandard, Text: "recover exactly once", Media: payloadMedia,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	if _, err := store.Create(context.Background(), CreateParams{
		ID: id, OwnerDID: owner, OperationID: uuid.New(), RequestHash: [32]byte{1},
		ScheduledAt: now, PayloadBytes: payload, PayloadHash: sha256.Sum256(payload),
		PayloadVersion: 1, MediaIDs: mediaIDs,
	}); err != nil {
		t.Fatal(err)
	}
	pds := &recordingScheduledPDS{}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			wantOwner: owner, sessionID: "owner-session",
		},
		NewEffects: recordingGuardedFactory(pds, nil),
		Objects:    objects,
		Now:        func() time.Time { return current },
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
	return &publicationRecoveryFixture{
		store: store, objects: objects, pds: pds, owner: owner, id: id,
		now: now, clock: &current, media: media, bodies: bodies, worker: worker,
	}
}

func (f *publicationRecoveryFixture) frozenRecord(t *testing.T) []byte {
	t.Helper()
	var frozen []byte
	if err := f.store.pool.QueryRow(context.Background(), `
		SELECT publication_record_bytes
		FROM scheduled_posts
		WHERE owner_did=$1 AND id=$2
	`, f.owner, f.id).Scan(&frozen); err != nil {
		t.Fatalf("read frozen publication: %v", err)
	}
	return frozen
}

func (f *publicationRecoveryFixture) stop(t *testing.T) PublishingClaim {
	t.Helper()
	stopped := false
	func() {
		defer func() {
			if recover() != nil {
				stopped = true
			}
		}()
		_, _ = f.worker.ProcessBatch(context.Background())
	}()
	if !stopped {
		t.Fatal("publication worker did not stop at the injected crash boundary")
	}
	if status := f.status(t); status != StatusPublishing {
		t.Fatalf("status after stopped worker = %s, want %s until lease recovery", status, StatusPublishing)
	}
	var claim PublishingClaim
	var rkey string
	if err := f.store.pool.QueryRow(context.Background(), `
		SELECT id, owner_did, lease_token, payload_version,
		       publication_rkey, publication_created_at
		FROM scheduled_posts
		WHERE owner_did=$1 AND id=$2
	`, f.owner, f.id).Scan(
		&claim.ID, &claim.OwnerDID, &claim.LeaseToken, &claim.PayloadVersion,
		&rkey, &claim.CreatedAt,
	); err != nil {
		t.Fatalf("read stopped publication claim: %v", err)
	}
	parsed, err := syntax.ParseRecordKey(rkey)
	if err != nil {
		t.Fatalf("parse stopped publication rkey: %v", err)
	}
	claim.Rkey = parsed
	return claim
}

func (f *publicationRecoveryFixture) recover(
	t *testing.T,
	stopped PublishingClaim,
	frozen []byte,
) {
	t.Helper()
	*f.clock = f.now.Add(DefaultPublicationLeaseDuration)
	claims, err := f.store.ClaimDue(
		context.Background(),
		1,
		*f.clock,
		DefaultPublicationLeaseDuration,
	)
	if err != nil || len(claims) != 1 {
		t.Fatalf("recover expired publication claim=%#v error=%v", claims, err)
	}
	recovered := claims[0]
	if recovered.ID != stopped.ID || recovered.Rkey != stopped.Rkey ||
		!recovered.CreatedAt.Equal(stopped.CreatedAt) ||
		recovered.PayloadVersion != stopped.PayloadVersion ||
		recovered.LeaseToken == stopped.LeaseToken {
		t.Fatalf("recovery changed frozen identity or reused lease: stopped=%#v recovered=%#v", stopped, recovered)
	}
	if err := f.store.SaveFrozenRecord(context.Background(), FrozenRecordParams{
		ID: stopped.ID, OwnerDID: stopped.OwnerDID, OwnerGeneration: stopped.OwnerGeneration,
		LeaseToken: stopped.LeaseToken, PayloadVersion: stopped.PayloadVersion,
		RecordBytes: frozen, RecordHash: sha256.Sum256(frozen), Now: *f.clock,
	}); !errors.Is(err, ErrWorkerLeaseLost) {
		t.Fatalf("stale worker completion error=%v, want %v", err, ErrWorkerLeaseLost)
	}
	claimStore := &recordingClaimStore{items: []WorkItem{{
		ID: recovered.ID, OwnerDID: recovered.OwnerDID,
		OwnerGeneration: recovered.OwnerGeneration,
		LeaseToken:      recovered.LeaseToken, PayloadVersion: recovered.PayloadVersion,
		Rkey: recovered.Rkey, CreatedAt: recovered.CreatedAt,
	}}}
	restarted, err := NewWorker(WorkerOptions{
		Store: claimStore, Processor: f.worker.processor,
		Now: func() time.Time { return *f.clock },
	})
	if err != nil {
		t.Fatalf("construct restarted worker: %v", err)
	}
	processed, err := restarted.ProcessBatch(context.Background())
	if err != nil || processed != 1 {
		t.Fatalf("restarted worker processed=%d error=%v", processed, err)
	}
}

func (f *publicationRecoveryFixture) status(t *testing.T) Status {
	t.Helper()
	var status Status
	if err := f.store.pool.QueryRow(context.Background(), `
		SELECT status
		FROM scheduled_posts
		WHERE owner_did=$1 AND id=$2
	`, f.owner, f.id).Scan(&status); err != nil {
		t.Fatalf("read scheduled publication status: %v", err)
	}
	return status
}

func (f *publicationRecoveryFixture) assertFinalizedWithRecord(t *testing.T, frozen []byte) {
	t.Helper()
	if _, err := f.store.Get(context.Background(), f.owner, f.id); !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("finalized schedule get error = %v", err)
	}
	actual, err := json.Marshal(f.pds.record)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, frozen) {
		t.Fatalf("published record changed across recovery:\nactual: %s\nfrozen: %s", actual, frozen)
	}
	if f.pds.putCalls != 1 {
		t.Fatalf("record writes = %d, want exactly one visible record write", f.pds.putCalls)
	}
	for _, staged := range f.media {
		if !bytes.Contains(frozen, []byte(staged.BlobCID.String())) {
			t.Fatalf("frozen record does not contain predicted blob CID %q", staged.BlobCID)
		}
	}
}
