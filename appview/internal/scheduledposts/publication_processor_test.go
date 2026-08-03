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
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"

	"social.craftsky/appview/internal/auth"
)

func TestPublicationWorkerPublishesDuePostWithStablePutAndFinalizes(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	payload, _ := EncodePayload(Payload{Kind: PostKindStandard, Text: "publish me", Langs: []string{"en"}})
	created, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
		RequestHash: [32]byte{1}, ScheduledAt: now, PayloadBytes: payload,
		PayloadHash: [32]byte{2}, PayloadVersion: 1,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	pds := &recordingScheduledPDS{}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store:    store,
		Sessions: stubPublicationSessionSelector{wantOwner: "did:plc:alice", sessionID: "owner-session"},
		NewPDS:   func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil },
		Objects:  newMemoryPrivateObjectStore(),
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	worker, err := NewWorker(WorkerOptions{Store: store, Processor: processor, Now: func() time.Time { return now }, BatchSize: 10})
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}
	processed, err := worker.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("process batch = %d, %v", processed, err)
	}
	if pds.putCalls != 1 || pds.rkey == "" || pds.record["createdAt"] != now.Format(time.RFC3339) {
		t.Fatalf("PDS writes=%d rkey=%q record=%#v", pds.putCalls, pds.rkey, pds.record)
	}
	if _, err := store.Get(ctx, "did:plc:alice", created.ID); !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("finalized schedule get error=%v", err)
	}
}

func TestPublicationWorkerUploadsThePrivateCopyBeforeWritingImageRecord(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	objects := newMemoryPrivateObjectStore()
	mediaService := NewPrivateMediaService(store, objects)
	ctx := context.Background()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.New()
	privateMarker := []byte("private-scheduled-marker")
	if _, err := mediaService.Put(ctx, PutPrivateMediaParams{ID: mediaID, OwnerDID: "did:plc:alice", MIMEType: "image/jpeg", Bytes: privateMarker, Now: now}); err != nil {
		t.Fatal(err)
	}
	payload, _ := EncodePayload(Payload{Kind: PostKindStandard, Text: "with image", Media: []PayloadMedia{{ID: mediaID.String(), Alt: "private alt", Width: 4, Height: 3}}})
	if _, err := store.Create(ctx, CreateParams{ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(), RequestHash: [32]byte{1}, ScheduledAt: now, PayloadBytes: payload, PayloadHash: [32]byte{2}, PayloadVersion: 1, MediaIDs: []uuid.UUID{mediaID}}); err != nil {
		t.Fatal(err)
	}
	pds := &recordingScheduledPDS{}
	processor, _ := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store, Sessions: stubPublicationSessionSelector{wantOwner: "did:plc:alice", sessionID: "owner-session"},
		NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil }, Objects: objects, Now: func() time.Time { return now },
	})
	worker, _ := NewWorker(WorkerOptions{Store: store, Processor: processor, Now: func() time.Time { return now }})
	if _, err := worker.ProcessBatch(ctx); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pds.uploadedBytes, privateMarker) {
		t.Fatalf("uploaded bytes=%q, want private marker", pds.uploadedBytes)
	}
	images := pds.record["images"].([]any)
	image := images[0].(map[string]any)
	if image["alt"] != "private alt" {
		t.Fatalf("image record=%#v", image)
	}
}

func TestPublicationWorkerFreezesPredictedMediaBeforeFirstPDSUpload(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	objects := newMemoryPrivateObjectStore()
	mediaService := NewPrivateMediaService(store, objects)
	ctx := context.Background()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.New()
	privateMarker := []byte("freeze-before-upload-private-marker")
	staged, err := mediaService.Put(ctx, PutPrivateMediaParams{
		ID: mediaID, OwnerDID: "did:plc:alice", MIMEType: "image/jpeg",
		Bytes: privateMarker, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := EncodePayload(Payload{
		Kind:  PostKindStandard,
		Text:  "frozen before upload",
		Media: []PayloadMedia{{ID: mediaID.String(), Alt: "private alt"}},
	})
	created, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
		RequestHash: [32]byte{1}, ScheduledAt: now, PayloadBytes: payload,
		PayloadHash: sha256.Sum256(payload), PayloadVersion: 1,
		MediaIDs: []uuid.UUID{mediaID},
	})
	if err != nil {
		t.Fatal(err)
	}

	var frozenBeforeUpload []byte
	pds := &recordingScheduledPDS{onUpload: func() {
		if err := store.pool.QueryRow(ctx, `
			SELECT publication_record_bytes
			FROM scheduled_posts
			WHERE owner_did=$1 AND id=$2
		`, syntax.DID("did:plc:alice"), created.ID).Scan(&frozenBeforeUpload); err != nil {
			t.Fatalf("read frozen record at upload boundary: %v", err)
		}
	}}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			wantOwner: "did:plc:alice", sessionID: "owner-session",
		},
		NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return pds, nil
		},
		Objects: objects,
		Now:     func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewWorker(WorkerOptions{
		Store: store, Processor: processor, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := worker.ProcessBatch(ctx); err != nil {
		t.Fatal(err)
	}
	if len(frozenBeforeUpload) == 0 {
		t.Fatal("publication record was not frozen before the first PDS upload")
	}
	if !bytes.Contains(frozenBeforeUpload, []byte(staged.BlobCID.String())) {
		t.Fatalf("frozen record %s does not contain predicted blob CID %q", frozenBeforeUpload, staged.BlobCID)
	}
}

func TestPublicationWorkerRejectsPDSBlobThatDiffersFromFrozenPrediction(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	objects := newMemoryPrivateObjectStore()
	mediaService := NewPrivateMediaService(store, objects)
	ctx := context.Background()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.New()
	privateMarker := []byte("private-media-with-predicted-identity")
	if _, err := mediaService.Put(ctx, PutPrivateMediaParams{
		ID: mediaID, OwnerDID: "did:plc:alice", MIMEType: "image/jpeg",
		Bytes: privateMarker, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	payload, _ := EncodePayload(Payload{
		Kind:  PostKindStandard,
		Text:  "reject mismatched PDS blob",
		Media: []PayloadMedia{{ID: mediaID.String()}},
	})
	created, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
		RequestHash: [32]byte{1}, ScheduledAt: now, PayloadBytes: payload,
		PayloadHash: sha256.Sum256(payload), PayloadVersion: 1,
		MediaIDs: []uuid.UUID{mediaID},
	})
	if err != nil {
		t.Fatal(err)
	}
	pds := &recordingScheduledPDS{uploadedBlob: &auth.UploadedBlob{
		Raw: map[string]any{
			"$type": "blob", "ref": map[string]any{"$link": "bafk-wrong"},
			"mimeType": "image/png", "size": len(privateMarker) + 1,
		},
		CID: "bafk-wrong", MIME: "image/png", Size: int64(len(privateMarker) + 1),
	}}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			wantOwner: "did:plc:alice", sessionID: "owner-session",
		},
		NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return pds, nil
		},
		Objects: objects,
		Now:     func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	worker, _ := NewWorker(WorkerOptions{
		Store: store, Processor: processor, Now: func() time.Time { return now },
	})
	if _, err := worker.ProcessBatch(ctx); err != nil {
		t.Fatal(err)
	}
	resource, err := store.Get(ctx, "did:plc:alice", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resource.Status != StatusNeedsAttention || pds.putCalls != 0 {
		t.Fatalf("status=%s record writes=%d, want needs_attention and zero", resource.Status, pds.putCalls)
	}
}

func TestPublicationWorkerRechecksCurrentMediaSizePolicy(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	objects := newMemoryPrivateObjectStore()
	mediaService := NewPrivateMediaService(store, objects)
	ctx := context.Background()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.New()
	if _, err := mediaService.Put(ctx, PutPrivateMediaParams{
		ID: mediaID, OwnerDID: "did:plc:alice", MIMEType: "image/jpeg",
		Bytes: []byte("larger-than-current-policy"), Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	payload, _ := EncodePayload(Payload{
		Kind:  PostKindStandard,
		Text:  "with image",
		Media: []PayloadMedia{{ID: mediaID.String()}},
	})
	created, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
		RequestHash: [32]byte{1}, ScheduledAt: now, PayloadBytes: payload,
		PayloadHash: [32]byte{2}, PayloadVersion: 1, MediaIDs: []uuid.UUID{mediaID},
	})
	if err != nil {
		t.Fatal(err)
	}
	pds := &recordingScheduledPDS{}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store, Sessions: stubPublicationSessionSelector{
			wantOwner: "did:plc:alice", sessionID: "owner-session",
		},
		NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return pds, nil
		},
		Objects: objects, Now: func() time.Time { return now }, MaxMediaBytes: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	worker, _ := NewWorker(WorkerOptions{
		Store: store, Processor: processor, Now: func() time.Time { return now },
	})
	if _, err := worker.ProcessBatch(ctx); err != nil {
		t.Fatal(err)
	}
	resource, err := store.Get(ctx, "did:plc:alice", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resource.Status != StatusNeedsAttention || len(pds.uploadedBytes) != 0 {
		t.Fatalf("status=%s uploaded=%d, want needs_attention and no upload", resource.Status, len(pds.uploadedBytes))
	}
}

func TestManualPublicationPublishesImmediatelyAndRetainsDefiniteFailure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store, now, created, payload := newManualPublicationFixture(t)
		pds := &recordingScheduledPDS{}
		processor, err := NewPublicationProcessor(PublicationProcessorOptions{
			Store: store,
			Sessions: stubPublicationSessionSelector{
				wantOwner: "did:plc:alice", sessionID: "owner-session",
			},
			NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
				return pds, nil
			},
			Objects: newMemoryPrivateObjectStore(), Now: func() time.Time { return now },
		})
		if err != nil {
			t.Fatal(err)
		}
		service, _ := NewManualPublicationService(store, processor)
		outcome, err := service.PublishManual(context.Background(), UpdateParams{
			ID: created.ID, OwnerDID: "did:plc:alice", ScheduledAt: now,
			PayloadBytes: payload, PayloadHash: sha256.Sum256(payload), Now: now,
		})
		if err != nil || outcome != ManualPublicationPublished {
			t.Fatalf("PublishManual = %s, %v", outcome, err)
		}
		if _, err := store.Get(context.Background(), "did:plc:alice", created.ID); !errors.Is(err, ErrScheduleNotFound) {
			t.Fatalf("published schedule remains: %v", err)
		}
	})

	t.Run("definite auth failure", func(t *testing.T) {
		store, now, created, payload := newManualPublicationFixture(t)
		processor, err := NewPublicationProcessor(PublicationProcessorOptions{
			Store: store,
			Sessions: stubPublicationSessionSelector{
				wantOwner: "did:plc:alice", err: auth.ErrNoUsableBackgroundSession,
			},
			NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
				return &recordingScheduledPDS{}, nil
			},
			Objects: newMemoryPrivateObjectStore(), Now: func() time.Time { return now },
		})
		if err != nil {
			t.Fatal(err)
		}
		service, _ := NewManualPublicationService(store, processor)
		_, err = service.PublishManual(context.Background(), UpdateParams{
			ID: created.ID, OwnerDID: "did:plc:alice", ScheduledAt: now,
			PayloadBytes: payload, PayloadHash: sha256.Sum256(payload), Now: now,
		})
		if !errors.Is(err, ErrManualPublicationFailed) {
			t.Fatalf("PublishManual error = %v", err)
		}
		resource, getErr := store.Get(context.Background(), "did:plc:alice", created.ID)
		if getErr != nil || resource.Status != StatusNeedsAttention {
			t.Fatalf("retained resource=%+v error=%v", resource, getErr)
		}
	})

	t.Run("ambiguous PDS write", func(t *testing.T) {
		store, now, created, payload := newManualPublicationFixture(t)
		pds := &recordingScheduledPDS{putErr: errors.New("connection reset")}
		processor, err := NewPublicationProcessor(PublicationProcessorOptions{
			Store: store,
			Sessions: stubPublicationSessionSelector{
				wantOwner: "did:plc:alice", sessionID: "owner-session",
			},
			NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
				return pds, nil
			},
			Objects: newMemoryPrivateObjectStore(), Now: func() time.Time { return now },
		})
		if err != nil {
			t.Fatal(err)
		}
		service, _ := NewManualPublicationService(store, processor)
		outcome, err := service.PublishManual(context.Background(), UpdateParams{
			ID: created.ID, OwnerDID: "did:plc:alice", ScheduledAt: now,
			PayloadBytes: payload, PayloadHash: sha256.Sum256(payload), Now: now,
		})
		if err != nil || outcome != ManualPublicationReconciling {
			t.Fatalf("PublishManual = %s, %v", outcome, err)
		}
		resource, getErr := store.Get(context.Background(), "did:plc:alice", created.ID)
		if getErr != nil || resource.Status != StatusPublishing {
			t.Fatalf("reconciling resource=%+v error=%v", resource, getErr)
		}
	})
}

func newManualPublicationFixture(
	t *testing.T,
) (*Store, time.Time, ScheduledPost, []byte) {
	t.Helper()
	store := NewStore(newScheduledPostStoreTestPool(t))
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	payload, _ := EncodePayload(Payload{Kind: PostKindStandard, Text: "post now"})
	created, err := store.Create(context.Background(), CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
		RequestHash: [32]byte{1}, ScheduledAt: now.Add(time.Hour),
		PayloadBytes: payload, PayloadHash: sha256.Sum256(payload), PayloadVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return store, now, created, payload
}

type recordingScheduledPDS struct {
	putCalls            int
	rkey                string
	record              map[string]any
	uploadedBytes       []byte
	uploadCalls         int
	uploadErrAt         int
	putErr              error
	commitThenErr       bool
	getErrOnce          error
	onUpload            func()
	uploadedBlob        *auth.UploadedBlob
	panicBeforeUploadAt int
	panicBeforeGet      bool
	panicAfterPut       bool
}

func (p *recordingScheduledPDS) GetRecord(_ context.Context, repo syntax.DID, collection, rkey string, out any) (string, error) {
	if p.panicBeforeGet {
		p.panicBeforeGet = false
		panic("synthetic worker stop before PDS record lookup")
	}
	if p.getErrOnce != nil {
		err := p.getErrOnce
		p.getErrOnce = nil
		return "", err
	}
	if p.record == nil || rkey != p.rkey {
		return "", auth.ErrRecordNotFound
	}
	encoded, _ := json.Marshal(p.record)
	_ = json.Unmarshal(encoded, out)
	return "bafyreischeduled", nil
}
func (p *recordingScheduledPDS) PutRecord(_ context.Context, _ syntax.DID, _ string, rkey string, record any) error {
	p.putCalls++
	if p.putErr != nil && !p.commitThenErr {
		return p.putErr
	}
	p.rkey = rkey
	encoded, _ := json.Marshal(record)
	if err := json.Unmarshal(encoded, &p.record); err != nil {
		return err
	}
	if p.panicAfterPut {
		p.panicAfterPut = false
		panic("synthetic worker stop after committed PDS record")
	}
	if p.putErr != nil {
		return p.putErr
	}
	return nil
}
func (*recordingScheduledPDS) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", errors.New("unexpected CreateRecord")
}
func (*recordingScheduledPDS) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return errors.New("unexpected DeleteRecord")
}
func (p *recordingScheduledPDS) UploadBlob(_ context.Context, contentType string, body []byte) (*auth.UploadedBlob, error) {
	if p.panicBeforeUploadAt > 0 && p.uploadCalls+1 == p.panicBeforeUploadAt {
		p.panicBeforeUploadAt = 0
		panic("synthetic worker stop before PDS blob upload")
	}
	p.uploadCalls++
	p.uploadedBytes = append([]byte(nil), body...)
	if p.onUpload != nil {
		p.onUpload()
	}
	if p.uploadErrAt > 0 && p.uploadCalls == p.uploadErrAt {
		return nil, errors.New("synthetic PDS blob upload failure")
	}
	if p.uploadedBlob != nil {
		return p.uploadedBlob, nil
	}
	digest, err := multihash.Sum(body, multihash.SHA2_256, -1)
	if err != nil {
		return nil, err
	}
	blobCID := cid.NewCidV1(cid.Raw, digest).String()
	return &auth.UploadedBlob{
		Raw: map[string]any{
			"$type": "blob", "ref": map[string]any{"$link": blobCID},
			"mimeType": contentType, "size": len(body),
		},
		CID: blobCID, MIME: contentType, Size: int64(len(body)),
	}, nil
}
