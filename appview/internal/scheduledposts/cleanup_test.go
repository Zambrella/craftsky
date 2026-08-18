package scheduledposts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestStoreCleansEligiblePrivateArtifactsSafely(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

	live, err := store.Create(ctx, capacityCreateParams(owner, 1, now.Add(time.Hour)))
	if err != nil {
		t.Fatalf("create live schedule: %v", err)
	}
	expired, err := store.Create(ctx, capacityCreateParams(owner, 2, now.Add(-time.Hour)))
	if err != nil {
		t.Fatalf("create expiring schedule: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `
		UPDATE scheduled_posts
		SET status='needs_attention',
		    next_attempt_at=$2,
		    needs_attention_at=$2,
		    needs_attention_expires_at=$3
		WHERE id=$1
	`, expired.ID, now.Add(-31*24*time.Hour), now); err != nil {
		t.Fatalf("prepare expired schedule: %v", err)
	}

	mediaRows := []struct {
		id           uuid.UUID
		objectKey    string
		scheduleID   *uuid.UUID
		ordinal      *int
		expiresAt    time.Time
		predictedCID string
	}{
		{id: uuid.MustParse("00000000-0000-4000-8000-000000000101"), scheduleID: &live.ID, ordinal: intPointer(0), expiresAt: now.Add(24 * time.Hour), predictedCID: "bafk-live"},
		{id: uuid.MustParse("00000000-0000-4000-8000-000000000102"), expiresAt: now, predictedCID: "bafk-unclaimed"},
		{id: uuid.MustParse("00000000-0000-4000-8000-000000000103"), scheduleID: &expired.ID, ordinal: intPointer(0), expiresAt: now.Add(24 * time.Hour), predictedCID: "bafk-expired"},
	}
	for index := range mediaRows {
		media := &mediaRows[index]
		media.objectKey = insertReadyPrivateMediaFixture(
			t, store, owner, media.id, media.scheduleID, media.ordinal,
			media.predictedCID, media.expiresAt,
		)
	}

	queuedKeys := []string{
		mediaRows[0].objectKey,
		insertCleanupFixture(t, store, owner, uuid.MustParse("00000000-0000-4000-8000-000000000104"), now, false),
		insertCleanupFixture(t, store, owner, uuid.MustParse("00000000-0000-4000-8000-000000000105"), now, false),
		insertCleanupFixture(t, store, owner, uuid.MustParse("00000000-0000-4000-8000-000000000106"), now, false),
	}
	for index, objectKey := range queuedKeys[:1] {
		if _, err := store.pool.Exec(ctx, `
			INSERT INTO scheduled_post_cleanup_jobs (
				id,object_key,owner_did,owner_generation,upload_generation,
				source_attempt_id,outcome_uncertain,next_attempt_at
			)
			SELECT $1,attempts.object_key,attempts.owner_did,
			       attempts.owner_generation,attempts.upload_generation,
			       attempts.upload_attempt_id,false,$3
			FROM scheduled_post_object_attempts AS attempts
			WHERE attempts.object_key=$2
		`, uuid.MustParse(cleanupUUID(index+1)), objectKey, now); err != nil {
			t.Fatalf("insert cleanup fixture: %v", err)
		}
	}
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO scheduled_post_publication_tombstones (
			schedule_id, owner_did, operation_id, request_hash,
			publication_uri, publication_cid, published_at, expires_at
		) VALUES (
			'00000000-0000-4000-8000-000000000201', $1,
			'00000000-0000-4000-8000-000000000202', decode(repeat('04', 32), 'hex'),
			'at://did:plc:alice/social.craftsky.feed.post/3expired', 'bafk-published',
			$2, $3
		)
	`, owner, now.Add(-30*24*time.Hour), now); err != nil {
		t.Fatalf("insert expired tombstone: %v", err)
	}

	if err := store.SweepExpiredLifecycle(ctx, now); err != nil {
		t.Fatalf("sweep lifecycle: %v", err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE id=$1`, 1, live.ID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE id=$1`, 0, expired.ID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 1, mediaRows[0].id)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id IN ($1, $2)`, 0, mediaRows[1].id, mediaRows[2].id)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_publication_tombstones`, 0)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 0, mediaRows[0].objectKey)

	claims, err := store.ClaimCleanup(ctx, 10, now, time.Minute)
	if err != nil {
		t.Fatalf("claim cleanup: %v", err)
	}
	gotKeys := make([]string, 0, len(claims))
	for _, claim := range claims {
		gotKeys = append(gotKeys, claim.ObjectKey)
	}
	sort.Strings(gotKeys)
	wantKeys := []string{
		mediaRows[1].objectKey,
		mediaRows[2].objectKey,
		queuedKeys[1],
		queuedKeys[2],
		queuedKeys[3],
	}
	sort.Strings(wantKeys)
	if len(gotKeys) != len(wantKeys) {
		t.Fatalf("cleanup claims=%v, want %v", gotKeys, wantKeys)
	}
	for index := range wantKeys {
		if gotKeys[index] != wantKeys[index] {
			t.Fatalf("cleanup claims=%v, want %v", gotKeys, wantKeys)
		}
	}

	retryClaim := claims[0]
	if err := store.RetryCleanup(ctx, retryClaim, now.Add(time.Minute), "object_delete_failed", now); err != nil {
		t.Fatalf("retry cleanup: %v", err)
	}
	for _, claim := range claims[1:] {
		if err := store.CompleteCleanup(ctx, claim); err != nil {
			t.Fatalf("complete cleanup: %v", err)
		}
	}
	if early, err := store.ClaimCleanup(ctx, 10, now, time.Minute); err != nil || len(early) != 0 {
		t.Fatalf("early retry claim=%v err=%v", early, err)
	}
	retried, err := store.ClaimCleanup(ctx, 10, now.Add(time.Minute), time.Minute)
	if err != nil || len(retried) != 1 {
		t.Fatalf("due retry claim=%v err=%v", retried, err)
	}
	if retried[0].ID != retryClaim.ID || retried[0].AttemptCount != 2 {
		t.Fatalf("retried claim=%#v, want same job at attempt 2", retried[0])
	}
	if err := store.CompleteCleanup(ctx, retried[0]); err != nil {
		t.Fatalf("complete retried cleanup: %v", err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs`, 0)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 1, mediaRows[0].id)
}

func TestStoreDeleteEnqueuesAttachedMediaCleanupAtomically(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000107")
	objectKey := insertReadyPrivateMediaFixture(
		t, store, owner, mediaID, nil, nil, "bafk-delete", now.Add(24*time.Hour),
	)
	createParams := capacityCreateParams(owner, 1, now.Add(time.Hour))
	createParams.MediaIDs = []uuid.UUID{mediaID}
	created, err := store.Create(ctx, createParams)
	if err != nil {
		t.Fatalf("create schedule with media: %v", err)
	}

	if err := store.Delete(ctx, owner, created.ID, now); err != nil {
		t.Fatalf("delete schedule: %v", err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE id=$1`, 0, created.ID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 0, mediaID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 1, objectKey)
	claims, err := store.ClaimDue(ctx, 1, now.Add(time.Hour), time.Minute)
	if err != nil || len(claims) != 0 {
		t.Fatalf("claim after delete=%#v err=%v", claims, err)
	}
	if err := store.Delete(ctx, owner, created.ID, now); err != nil {
		t.Fatalf("repeat delete: %v", err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 1, objectKey)
}

func TestStoreUpdateEnqueuesOnlyReplacedMediaCleanupAtomically(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	oldMediaID := uuid.MustParse("00000000-0000-4000-8000-000000000111")
	keptMediaID := uuid.MustParse("00000000-0000-4000-8000-000000000112")
	newMediaID := uuid.MustParse("00000000-0000-4000-8000-000000000113")
	objectKeys := make(map[uuid.UUID]string)
	for _, mediaID := range []uuid.UUID{oldMediaID, keptMediaID, newMediaID} {
		objectKeys[mediaID] = insertReadyPrivateMediaFixture(
			t, store, owner, mediaID, nil, nil,
			"bafk-"+mediaID.String(), now.Add(24*time.Hour),
		)
	}
	createParams := capacityCreateParams(owner, 1, now.Add(time.Hour))
	createParams.MediaIDs = []uuid.UUID{oldMediaID, keptMediaID}
	created, err := store.Create(ctx, createParams)
	if err != nil {
		t.Fatalf("create schedule with media: %v", err)
	}

	payload := []byte(`{"kind":"standard","text":"replacement"}`)
	payloadHash := sha256.Sum256(payload)
	if _, err := store.Update(ctx, UpdateParams{
		ID: created.ID, OwnerDID: owner, ScheduledAt: now.Add(2 * time.Hour),
		PayloadBytes: payload, PayloadHash: payloadHash, Now: now,
		MediaIDs: []uuid.UUID{keptMediaID, newMediaID},
	}); err != nil {
		t.Fatalf("replace scheduled media: %v", err)
	}

	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 0, oldMediaID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 1, objectKeys[oldMediaID])
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key IN ($1, $2)`, 0, objectKeys[keptMediaID], objectKeys[newMediaID])

	rows, err := store.pool.Query(ctx, `
		SELECT id, ordinal
		FROM scheduled_post_media
		WHERE owner_did=$1 AND schedule_id=$2
		ORDER BY ordinal
	`, owner, created.ID)
	if err != nil {
		t.Fatalf("read replacement media order: %v", err)
	}
	defer rows.Close()
	want := []uuid.UUID{keptMediaID, newMediaID}
	var got []uuid.UUID
	for rows.Next() {
		var mediaID uuid.UUID
		var ordinal int
		if err := rows.Scan(&mediaID, &ordinal); err != nil {
			t.Fatalf("scan replacement media: %v", err)
		}
		if ordinal != len(got) {
			t.Fatalf("ordinal=%d, want %d", ordinal, len(got))
		}
		got = append(got, mediaID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read replacement media rows: %v", err)
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("replacement media=%v, want %v", got, want)
	}
}

func TestAccountDeletionRemovesPrivateSchedulesAndQueuesEveryObject(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	otherOwner := syntax.DID("did:plc:bob")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	ownerAttachedID := uuid.MustParse("00000000-0000-4000-8000-000000000114")
	ownerUnclaimedID := uuid.MustParse("00000000-0000-4000-8000-000000000115")
	otherID := uuid.MustParse("00000000-0000-4000-8000-000000000116")
	for _, fixture := range []struct {
		owner syntax.DID
		id    uuid.UUID
	}{
		{owner: owner, id: ownerAttachedID},
		{owner: owner, id: ownerUnclaimedID},
		{owner: otherOwner, id: otherID},
	} {
		insertReadyPrivateMediaFixture(
			t, store, fixture.owner, fixture.id, nil, nil,
			"bafk-"+fixture.id.String(), now.Add(24*time.Hour),
		)
	}
	ownerCreate := capacityCreateParams(owner, 1, now.Add(time.Hour))
	ownerCreate.MediaIDs = []uuid.UUID{ownerAttachedID}
	ownerSchedule, err := store.Create(ctx, ownerCreate)
	if err != nil {
		t.Fatalf("create owner schedule: %v", err)
	}
	otherCreate := capacityCreateParams(otherOwner, 2, now.Add(time.Hour))
	otherCreate.MediaIDs = []uuid.UUID{otherID}
	otherSchedule, err := store.Create(ctx, otherCreate)
	if err != nil {
		t.Fatalf("create other schedule: %v", err)
	}

	deletion := NewAccountDeletion(
		store.pool, func() time.Time { return now },
		newScheduledTestOwnerFencer(t, store.pool),
	)
	if err := deletion.Purge(ctx, owner); err != nil {
		t.Fatalf("delete scheduled account state: %v", err)
	}

	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE owner_did=$1`, 0, owner)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE owner_did=$1`, 0, owner)
	ownerAttachedKey, _, _ := NewGenerationObjectKey(owner, 1, ownerAttachedID)
	ownerUnclaimedKey, _, _ := NewGenerationObjectKey(owner, 1, ownerUnclaimedID)
	otherKey, _, _ := NewGenerationObjectKey(otherOwner, 1, otherID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key IN ($1, $2)`, 2, ownerAttachedKey, ownerUnclaimedKey)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_posts WHERE id=$1`, 1, otherSchedule.ID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_media WHERE id=$1`, 1, otherID)
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 0, otherKey)

	if ownerSchedule.ID == uuid.Nil {
		t.Fatal("owner schedule fixture was not created")
	}
	if err := deletion.Purge(ctx, owner); err != nil {
		t.Fatalf("repeat scheduled account deletion: %v", err)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key IN ($1, $2)`, 2, ownerAttachedKey, ownerUnclaimedKey)
}

func TestCleanupProcessorDeletesObjectsAndRetriesSafely(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	objects := &flakyCleanupObjectStore{objects: map[string][]byte{}}
	service, ownerFence := newScheduledTestMediaService(t, store, objects)
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	currentTime := now
	processor, err := NewCleanupProcessor(CleanupProcessorOptions{
		Store: store, Objects: objects, OwnerFence: ownerFence,
		Now:       func() time.Time { return currentTime },
		BatchSize: 10, LeaseDuration: time.Minute, RetryDelay: time.Minute,
	})
	if err != nil {
		t.Fatalf("construct cleanup processor: %v", err)
	}
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000108")
	created, err := service.Put(ctx, PutPrivateMediaParams{
		ID: mediaID, OwnerDID: owner, OwnerGeneration: 1, MIMEType: "image/jpeg",
		Bytes: []byte("private-image-bytes"), Now: now,
	})
	if err != nil {
		t.Fatalf("stage cleanup fixture: %v", err)
	}
	if err := service.Delete(ctx, owner, mediaID, now, 1); err != nil {
		t.Fatalf("queue cleanup fixture: %v", err)
	}
	objects.failDeletes = 1

	processed, err := processor.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("process failed delete=%d err=%v", processed, err)
	}
	if !objects.has(created.ObjectKey) || objects.deleteCalls != 1 {
		t.Fatalf("failed delete removed object or wrong calls: present=%v calls=%d", objects.has(created.ObjectKey), objects.deleteCalls)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1 AND state='pending' AND attempt_count=1`, 1, created.ObjectKey)

	processed, err = processor.ProcessBatch(ctx)
	if err != nil || processed != 0 {
		t.Fatalf("processed retry before deadline=%d err=%v", processed, err)
	}
	currentTime = now.Add(time.Minute)
	processed, err = processor.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("process due retry=%d err=%v", processed, err)
	}
	if objects.has(created.ObjectKey) || objects.deleteCalls != 2 {
		t.Fatalf("successful retry state: present=%v calls=%d", objects.has(created.ObjectKey), objects.deleteCalls)
	}
	assertRowCount(t, store, `SELECT count(*) FROM scheduled_post_cleanup_jobs WHERE object_key=$1`, 0, created.ObjectKey)
}

func TestCleanupProcessorDoesNotDeleteAConcurrentReupload(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	baseObjects := &flakyCleanupObjectStore{objects: map[string][]byte{}}
	objects := &blockingCleanupObjectStore{
		flakyCleanupObjectStore: baseObjects,
		deleteStarted:           make(chan struct{}),
		continueDelete:          make(chan struct{}),
	}
	service, ownerFence := newScheduledTestMediaService(t, store, objects)
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000109")
	putParams := PutPrivateMediaParams{
		ID: mediaID, OwnerDID: owner, OwnerGeneration: 1, MIMEType: "image/jpeg",
		Bytes: []byte("private-image-bytes"), Now: now,
	}
	created, err := service.Put(ctx, putParams)
	if err != nil {
		t.Fatalf("stage cleanup fixture: %v", err)
	}
	if err := service.Delete(ctx, owner, mediaID, now, 1); err != nil {
		t.Fatalf("queue cleanup fixture: %v", err)
	}
	processor, err := NewCleanupProcessor(CleanupProcessorOptions{
		Store: store, Objects: objects, OwnerFence: ownerFence,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("construct cleanup processor: %v", err)
	}
	processResult := make(chan error, 1)
	go func() {
		_, err := processor.ProcessBatch(ctx)
		processResult <- err
	}()
	<-objects.deleteStarted

	reuploadResult := make(chan error, 1)
	go func() {
		_, err := service.Put(ctx, putParams)
		reuploadResult <- err
	}()
	close(objects.continueDelete)
	if err := <-processResult; err != nil {
		t.Fatalf("complete cleanup processor: %v", err)
	}
	reuploadErr := <-reuploadResult
	if reuploadErr != nil && !errors.Is(reuploadErr, ErrScheduledMediaConflict) {
		t.Fatalf("reupload during deleting: %v", reuploadErr)
	}
	recreated := created
	if errors.Is(reuploadErr, ErrScheduledMediaConflict) {
		recreated, err = service.Put(ctx, putParams)
		if err != nil {
			t.Fatalf("reupload after cleanup: %v", err)
		}
	}
	if recreated.ObjectKey != created.ObjectKey || !objects.has(created.ObjectKey) {
		t.Fatalf("recreated media=%#v object present=%v", recreated, objects.has(created.ObjectKey))
	}
}

func TestCleanupProcessorFencesExpiredLeaseDeleteAcrossReupload(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	objects := &sequencedCleanupObjectStore{
		objects:             map[string][]byte{},
		firstDeleteStarted:  make(chan struct{}),
		continueFirstDelete: make(chan struct{}),
	}
	service, ownerFence := newScheduledTestMediaService(t, store, objects)
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000110")
	putParams := PutPrivateMediaParams{
		ID: mediaID, OwnerDID: owner, OwnerGeneration: 1, MIMEType: "image/jpeg",
		Bytes: []byte("private-image-bytes"), Now: now,
	}
	created, err := service.Put(ctx, putParams)
	if err != nil {
		t.Fatalf("stage cleanup fixture: %v", err)
	}
	if err := service.Delete(ctx, owner, mediaID, now, 1); err != nil {
		t.Fatalf("queue cleanup fixture: %v", err)
	}

	firstProcessor, err := NewCleanupProcessor(CleanupProcessorOptions{
		Store: store, Objects: objects, OwnerFence: ownerFence,
		Now:       func() time.Time { return now },
		BatchSize: 1, LeaseDuration: time.Minute,
	})
	if err != nil {
		t.Fatalf("construct first cleanup processor: %v", err)
	}
	firstResult := make(chan error, 1)
	go func() {
		_, processErr := firstProcessor.ProcessBatch(ctx)
		firstResult <- processErr
	}()
	<-objects.firstDeleteStarted

	secondStore := &cleanupEffectSignalStore{
		Store:         store,
		beforeAcquire: make(chan struct{}),
	}
	secondProcessor, err := NewCleanupProcessor(CleanupProcessorOptions{
		Store: secondStore, Objects: objects, OwnerFence: ownerFence,
		Now:       func() time.Time { return now.Add(time.Minute) },
		BatchSize: 1, LeaseDuration: time.Minute,
	})
	if err != nil {
		t.Fatalf("construct second cleanup processor: %v", err)
	}
	secondResult := make(chan error, 1)
	go func() {
		_, processErr := secondProcessor.ProcessBatch(ctx)
		secondResult <- processErr
	}()
	<-secondStore.beforeAcquire

	reuploadResult := make(chan error, 1)
	go func() {
		_, putErr := service.Put(ctx, putParams)
		reuploadResult <- putErr
	}()
	close(objects.continueFirstDelete)

	if processErr := <-firstResult; processErr == nil {
		t.Fatal("expired first cleanup lease unexpectedly completed")
	}
	if processErr := <-secondResult; processErr != nil {
		t.Fatalf("recovered cleanup processor: %v", processErr)
	}
	if putErr := <-reuploadResult; putErr != nil &&
		!errors.Is(putErr, ErrScheduledMediaConflict) &&
		!errors.Is(putErr, ErrScheduledMediaOutcomeUnknown) {
		t.Fatalf("reupload during recovered cleanup: %v", putErr)
	} else if errors.Is(putErr, ErrScheduledMediaConflict) ||
		errors.Is(putErr, ErrScheduledMediaOutcomeUnknown) {
		if _, err := service.Put(ctx, putParams); err != nil {
			t.Fatalf("retry reupload after recovered cleanup: %v", err)
		}
	}
	opened, err := service.Open(ctx, owner, mediaID)
	if err != nil {
		t.Fatalf("open recreated media: %v", err)
	}
	defer opened.Body.Close()
	payload, err := io.ReadAll(opened.Body)
	if err != nil || !bytes.Equal(payload, putParams.Bytes) || !objects.has(created.ObjectKey) {
		t.Fatalf("recreated media payload=%q read=%v object present=%v", payload, err, objects.has(created.ObjectKey))
	}
}

func TestCleanupProcessorBoundsRemoteWorkByItsLease(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	baseObjects := &flakyCleanupObjectStore{objects: map[string][]byte{}}
	objects := &blockingCleanupObjectStore{
		flakyCleanupObjectStore: baseObjects,
		deleteStarted:           make(chan struct{}),
		continueDelete:          make(chan struct{}),
	}
	service, ownerFence := newScheduledTestMediaService(t, store, objects)
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 14, 13, 0, 0, 0, time.UTC)
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000117")
	if _, err := service.Put(context.Background(), PutPrivateMediaParams{
		ID: mediaID, OwnerDID: owner, OwnerGeneration: 1,
		MIMEType: "image/jpeg", Bytes: []byte("bounded-cleanup"), Now: now,
	}); err != nil {
		t.Fatalf("stage bounded cleanup fixture: %v", err)
	}
	if err := service.Delete(context.Background(), owner, mediaID, now, 1); err != nil {
		t.Fatalf("queue bounded cleanup fixture: %v", err)
	}

	const leaseDuration = 25 * time.Millisecond
	processor, err := NewCleanupProcessor(CleanupProcessorOptions{
		Store: store, Objects: objects, OwnerFence: ownerFence,
		Now: func() time.Time { return now }, BatchSize: 1,
		LeaseDuration: leaseDuration, RetryDelay: time.Minute,
	})
	if err != nil {
		t.Fatalf("construct bounded cleanup processor: %v", err)
	}
	parentCtx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, processErr := processor.ProcessBatch(parentCtx)
	elapsed := time.Since(started)
	if processErr == nil {
		t.Fatal("cleanup whose remote delete exceeded the lease returned nil")
	}
	if elapsed >= 150*time.Millisecond {
		t.Fatalf("cleanup elapsed=%s, want its %s lease to bound remote work", elapsed, leaseDuration)
	}
}

type cleanupEffectSignalStore struct {
	*Store
	beforeAcquire chan struct{}
	once          sync.Once
}

func (store *cleanupEffectSignalStore) AcquireCleanupEffect(
	ctx context.Context,
	claim CleanupClaim,
) (CleanupEffectGuard, error) {
	store.once.Do(func() { close(store.beforeAcquire) })
	return store.Store.AcquireCleanupEffect(ctx, claim)
}

type sequencedCleanupObjectStore struct {
	mu                  sync.Mutex
	objects             map[string][]byte
	deleteCalls         int
	firstDeleteStarted  chan struct{}
	continueFirstDelete chan struct{}
}

func (store *sequencedCleanupObjectStore) Put(
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
	defer store.mu.Unlock()
	store.objects[key] = append([]byte(nil), payload...)
	return nil
}

func (store *sequencedCleanupObjectStore) Open(
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

func (store *sequencedCleanupObjectStore) Delete(ctx context.Context, key string) error {
	store.mu.Lock()
	store.deleteCalls++
	deleteCall := store.deleteCalls
	store.mu.Unlock()
	if deleteCall == 1 {
		close(store.firstDeleteStarted)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-store.continueFirstDelete:
		}
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.objects, key)
	return nil
}

func (store *sequencedCleanupObjectStore) has(key string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	_, ok := store.objects[key]
	return ok
}

func (store *sequencedCleanupObjectStore) Exists(_ context.Context, key string) (bool, error) {
	return store.has(key), nil
}

type flakyCleanupObjectStore struct {
	objects     map[string][]byte
	failDeletes int
	deleteCalls int
}

type blockingCleanupObjectStore struct {
	*flakyCleanupObjectStore
	deleteStarted  chan struct{}
	continueDelete chan struct{}
}

func (store *blockingCleanupObjectStore) Delete(ctx context.Context, key string) error {
	close(store.deleteStarted)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-store.continueDelete:
		return store.flakyCleanupObjectStore.Delete(ctx, key)
	}
}

func (store *flakyCleanupObjectStore) Put(
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
	store.objects[key] = append([]byte(nil), payload...)
	return nil
}

func (store *flakyCleanupObjectStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	payload, ok := store.objects[key]
	if !ok {
		return nil, ErrPrivateObjectStoreUnavailable
	}
	return io.NopCloser(bytes.NewReader(payload)), nil
}

func (store *flakyCleanupObjectStore) Delete(_ context.Context, key string) error {
	store.deleteCalls++
	if store.failDeletes > 0 {
		store.failDeletes--
		return errors.New("injected object delete failure")
	}
	delete(store.objects, key)
	return nil
}

func (store *flakyCleanupObjectStore) has(key string) bool {
	_, ok := store.objects[key]
	return ok
}

func (store *flakyCleanupObjectStore) Exists(_ context.Context, key string) (bool, error) {
	return store.has(key), nil
}

func intPointer(value int) *int { return &value }

func cleanupUUID(suffix int) string {
	return "00000000-0000-4000-8001-" + leftPad12(suffix)
}

func leftPad12(value int) string {
	return fmt.Sprintf("%012d", value)
}

func assertRowCount(t *testing.T, store *Store, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := store.pool.QueryRow(context.Background(), query, args...).Scan(&got); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if got != want {
		t.Fatalf("row count=%d, want %d for %s", got, want, query)
	}
}
