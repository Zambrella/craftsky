package scheduledposts

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

const scheduledPostStorePreStateDDL = `
CREATE TABLE craftsky_profiles (
	 did        TEXT NOT NULL PRIMARY KEY,
	 record_cid TEXT NOT NULL,
	 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
CREATE TABLE owner_lifecycles (
	owner_did TEXT NOT NULL PRIMARY KEY,
	state TEXT NOT NULL,
	generation BIGINT NOT NULL,
	auth_epoch BIGINT NOT NULL,
	transition_reason TEXT NOT NULL,
	transitioned_at TIMESTAMPTZ NOT NULL,
	terminal_at TIMESTAMPTZ,
	purge_completed_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO owner_lifecycles(
	owner_did,state,generation,auth_epoch,transition_reason,
	transitioned_at,created_at,updated_at
) VALUES
	('did:plc:alice','active',1,1,'fixture',now(),now(),now()),
	('did:plc:bob','active',1,1,'fixture',now(),now(),now());
CREATE TABLE account_deletion_operations (
	id UUID NOT NULL,
	owner_did TEXT NOT NULL,
	PRIMARY KEY (id),
	UNIQUE (id,owner_did)
);
`

func TestStoreEnforcesCapacityTransactionally(t *testing.T) {
	pool := newScheduledPostStoreTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	due := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	for index := byte(1); index <= 2; index++ {
		if _, err := store.Create(ctx, capacityCreateParams(owner, index, due)); err != nil {
			t.Fatalf("seed slot %d: %v", index, err)
		}
	}

	start := make(chan struct{})
	results := make(chan createAttempt, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for index := byte(3); index <= 4; index++ {
		params := capacityCreateParams(owner, index, due)
		go func() {
			ready.Done()
			<-start
			created, err := store.Create(ctx, params)
			results <- createAttempt{id: created.ID, err: err}
		}()
	}
	ready.Wait()
	close(start)

	var winner uuid.UUID
	var successes, full int
	for range 2 {
		result := <-results
		switch {
		case result.err == nil:
			successes++
			winner = result.id
		case errors.Is(result.err, ErrCapacityReached):
			full++
		default:
			t.Fatalf("unexpected concurrent create error: %v", result.err)
		}
	}
	if successes != 1 || full != 1 {
		t.Fatalf("concurrent final slot: successes=%d capacityErrors=%d", successes, full)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM scheduled_posts WHERE owner_did=$1`, owner).Scan(&count); err != nil {
		t.Fatalf("count schedules: %v", err)
	}
	if count != MaximumActivePosts {
		t.Fatalf("active count=%d, want %d", count, MaximumActivePosts)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM scheduled_posts WHERE owner_did=$1 AND id=$2`, owner, winner); err != nil {
		t.Fatalf("release completed/deleted slot: %v", err)
	}
	if _, err := store.Create(ctx, capacityCreateParams(owner, 5, due)); err != nil {
		t.Fatalf("create after slot release: %v", err)
	}
}

func TestStoreEditsAreLastWriteWinsAndFenceStaleWorkers(t *testing.T) {
	pool := newScheduledPostStoreTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	due := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	created, err := store.Create(ctx, capacityCreateParams(owner, 1, due))
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	claimedVersion := created.PayloadVersion

	type editAttempt struct {
		payload []byte
		result  UpdateResult
		err     error
	}
	start := make(chan struct{})
	edits := make(chan editAttempt, 2)
	for _, text := range []string{"first client", "second client"} {
		payload := []byte(fmt.Sprintf(`{"kind":"standard","text":%q}`, text))
		hash := sha256.Sum256(payload)
		go func() {
			<-start
			result, err := store.Update(ctx, UpdateParams{
				ID:           created.ID,
				OwnerDID:     owner,
				ScheduledAt:  due.Add(time.Hour),
				PayloadBytes: payload,
				PayloadHash:  hash,
				Now:          due.Add(-time.Hour),
			})
			edits <- editAttempt{payload: payload, result: result, err: err}
		}()
	}
	close(start)

	committedPayloads := make(map[int64][]byte, 2)
	for range 2 {
		edit := <-edits
		if edit.err != nil {
			t.Fatalf("concurrent edit: %v", edit.err)
		}
		committedPayloads[edit.result.PayloadVersion] = edit.payload
	}

	var storedPayload []byte
	var currentVersion int64
	if err := pool.QueryRow(ctx, `
		SELECT payload_bytes, payload_version
		FROM scheduled_posts
		WHERE owner_did=$1 AND id=$2
	`, owner, created.ID).Scan(&storedPayload, &currentVersion); err != nil {
		t.Fatalf("read edited schedule: %v", err)
	}
	if currentVersion != claimedVersion+2 {
		t.Fatalf("payload version=%d, want %d", currentVersion, claimedVersion+2)
	}
	if string(storedPayload) != string(committedPayloads[currentVersion]) {
		t.Fatalf("stored payload is not the last committed edit: got %q want %q", storedPayload, committedPayloads[currentVersion])
	}

	lease := uuid.New()
	if _, err := pool.Exec(ctx, `
		UPDATE scheduled_posts
		SET status='publishing', lease_token=$3, lease_expires_at=$4,
		    publication_rkey='3mtest', publication_created_at=$5
		WHERE owner_did=$1 AND id=$2
	`, owner, created.ID, lease, due.Add(time.Minute), due); err != nil {
		t.Fatalf("prepare publishing fixture: %v", err)
	}
	record := []byte(`{"$type":"social.craftsky.feed.post"}`)
	recordHash := sha256.Sum256(record)
	stale := FrozenRecordParams{
		ID:              created.ID,
		OwnerDID:        owner,
		OwnerGeneration: created.OwnerGeneration,
		LeaseToken:      lease,
		PayloadVersion:  claimedVersion,
		RecordBytes:     record,
		RecordHash:      recordHash,
		Now:             due,
	}
	if err := store.SaveFrozenRecord(ctx, stale); !errors.Is(err, ErrStaleWorkerVersion) {
		t.Fatalf("stale worker error=%v, want %v", err, ErrStaleWorkerVersion)
	}
	var frozen bool
	if err := pool.QueryRow(ctx, `
		SELECT publication_record_bytes IS NOT NULL
		FROM scheduled_posts
		WHERE owner_did=$1 AND id=$2
	`, owner, created.ID).Scan(&frozen); err != nil {
		t.Fatalf("inspect stale worker write: %v", err)
	}
	if frozen {
		t.Fatal("stale worker persisted a publication record")
	}

	stale.PayloadVersion = currentVersion
	if err := store.SaveFrozenRecord(ctx, stale); err != nil {
		t.Fatalf("current worker save: %v", err)
	}
}

func TestStoreSerializesMutationsAgainstPublishing(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	due := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	t.Run("member edit commits before Publishing", func(t *testing.T) {
		store := NewStore(newScheduledPostStoreTestPool(t))
		ctx := context.Background()
		created, err := store.Create(ctx, capacityCreateParams(owner, 1, due))
		if err != nil {
			t.Fatalf("create schedule: %v", err)
		}
		payload := []byte(`{"kind":"standard","text":"latest"}`)
		payloadHash := sha256.Sum256(payload)
		updated, err := store.Update(ctx, UpdateParams{
			ID: created.ID, OwnerDID: owner, ScheduledAt: due,
			PayloadBytes: payload, PayloadHash: payloadHash, Now: due.Add(-time.Minute),
		})
		if err != nil {
			t.Fatalf("edit before Publishing: %v", err)
		}
		claims, err := store.ClaimDue(ctx, 1, due, time.Minute)
		if err != nil || len(claims) != 1 {
			t.Fatalf("claim after edit: claims=%v err=%v", claims, err)
		}
		claim := claims[0]
		guard, err := store.AcquirePublishingEffect(ctx, claim)
		if err != nil {
			t.Fatalf("acquire Publishing effect after edit: %v", err)
		}
		defer guard.Release(context.Background())
		if claim.PayloadVersion != updated.PayloadVersion {
			t.Fatalf("worker claimed version=%d, want edited version=%d", claim.PayloadVersion, updated.PayloadVersion)
		}
	})

	t.Run("Publishing commits before member edit and delete", func(t *testing.T) {
		store := NewStore(newScheduledPostStoreTestPool(t))
		ctx := context.Background()
		created, err := store.Create(ctx, capacityCreateParams(owner, 2, due))
		if err != nil {
			t.Fatalf("create schedule: %v", err)
		}
		claims, err := store.ClaimDue(ctx, 1, due, time.Minute)
		if err != nil || len(claims) != 1 {
			t.Fatalf("claim Publishing: claims=%v err=%v", claims, err)
		}
		guard, err := store.AcquirePublishingEffect(ctx, claims[0])
		if err != nil {
			t.Fatalf("acquire Publishing effect: %v", err)
		}

		started := make(chan struct{}, 2)
		results := make(chan error, 2)
		payload := []byte(`{"kind":"standard","text":"too late"}`)
		payloadHash := sha256.Sum256(payload)
		go func() {
			started <- struct{}{}
			_, err := store.Update(ctx, UpdateParams{
				ID: created.ID, OwnerDID: owner, ScheduledAt: due,
				PayloadBytes: payload, PayloadHash: payloadHash, Now: due,
			})
			results <- err
		}()
		go func() {
			started <- struct{}{}
			results <- store.Delete(ctx, owner, created.ID, due)
		}()
		<-started
		<-started
		select {
		case err := <-results:
			t.Fatalf("member mutation crossed held publication effect lock: %v", err)
		default:
		}
		if err := guard.Release(ctx); err != nil {
			t.Fatalf("release publication effect: %v", err)
		}
		for range 2 {
			if err := <-results; !errors.Is(err, ErrMutationLocked) {
				t.Fatalf("mutation after Publishing error=%v, want %v", err, ErrMutationLocked)
			}
		}
	})

	t.Run("delete commits before Publishing", func(t *testing.T) {
		store := NewStore(newScheduledPostStoreTestPool(t))
		ctx := context.Background()
		created, err := store.Create(ctx, capacityCreateParams(owner, 3, due))
		if err != nil {
			t.Fatalf("create schedule: %v", err)
		}
		if err := store.Delete(ctx, owner, created.ID, due.Add(-time.Minute)); err != nil {
			t.Fatalf("delete before Publishing: %v", err)
		}
		claims, err := store.ClaimDue(ctx, 1, due, time.Minute)
		if err != nil {
			t.Fatalf("claim after delete: %v", err)
		}
		if len(claims) != 0 {
			t.Fatalf("claimed deleted schedule: %#v", claims)
		}
	})
}

func TestStoreClaimsDueWorkWithExclusiveRecoverableLeases(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	t.Run("concurrent workers claim due rows once", func(t *testing.T) {
		store := NewStore(newScheduledPostStoreTestPool(t))
		ctx := context.Background()
		for index, scheduledAt := range []time.Time{now.Add(-time.Minute), now, now.Add(time.Minute)} {
			if _, err := store.Create(ctx, capacityCreateParams(owner, byte(index+1), scheduledAt)); err != nil {
				t.Fatalf("create fixture %d: %v", index, err)
			}
		}
		start := make(chan struct{})
		results := make(chan []PublishingClaim, 2)
		errs := make(chan error, 2)
		for range 2 {
			go func() {
				<-start
				claims, err := store.ClaimDue(ctx, 2, now, time.Minute)
				results <- claims
				errs <- err
			}()
		}
		close(start)
		claimed := map[uuid.UUID]bool{}
		for range 2 {
			claims := <-results
			if err := <-errs; err != nil {
				t.Fatalf("claim due: %v", err)
			}
			for _, claim := range claims {
				if claimed[claim.ID] {
					t.Fatalf("schedule %s was claimed twice", claim.ID)
				}
				claimed[claim.ID] = true
			}
		}
		if len(claimed) != 2 {
			t.Fatalf("claimed %d due schedules, want 2", len(claimed))
		}
		var futureStatus Status
		if err := store.pool.QueryRow(ctx, `
			SELECT status FROM scheduled_posts WHERE id=$1
		`, capacityCreateParams(owner, 3, now).ID).Scan(&futureStatus); err != nil {
			t.Fatalf("read future row: %v", err)
		}
		if futureStatus != StatusScheduled {
			t.Fatalf("future status=%s, want %s", futureStatus, StatusScheduled)
		}
	})

	t.Run("expired lease recovers identity and fences stale completion", func(t *testing.T) {
		store := NewStore(newScheduledPostStoreTestPool(t))
		ctx := context.Background()
		created, err := store.Create(ctx, capacityCreateParams(owner, 1, now))
		if err != nil {
			t.Fatal(err)
		}
		first, err := store.ClaimDue(ctx, 1, now, time.Minute)
		if err != nil || len(first) != 1 {
			t.Fatalf("first claim=%v err=%v", first, err)
		}
		if claims, err := store.ClaimDue(ctx, 1, now.Add(59*time.Second), time.Minute); err != nil || len(claims) != 0 {
			t.Fatalf("claim before expiry=%v err=%v", claims, err)
		}
		recovered, err := store.ClaimDue(ctx, 1, now.Add(time.Minute), time.Minute)
		if err != nil || len(recovered) != 1 {
			t.Fatalf("recovered claim=%v err=%v", recovered, err)
		}
		if recovered[0].ID != created.ID || recovered[0].Rkey != first[0].Rkey ||
			!recovered[0].CreatedAt.Equal(first[0].CreatedAt) || recovered[0].LeaseToken == first[0].LeaseToken {
			t.Fatalf("recovery changed identity or reused lease: first=%#v recovered=%#v", first[0], recovered[0])
		}
		record := []byte(`{"$type":"social.craftsky.feed.post"}`)
		hash := sha256.Sum256(record)
		if err := store.SaveFrozenRecord(ctx, FrozenRecordParams{
			ID: created.ID, OwnerDID: owner, OwnerGeneration: first[0].OwnerGeneration,
			LeaseToken:     first[0].LeaseToken,
			PayloadVersion: first[0].PayloadVersion, RecordBytes: record,
			RecordHash: hash, Now: now.Add(time.Minute),
		}); !errors.Is(err, ErrWorkerLeaseLost) {
			t.Fatalf("stale completion error=%v, want %v", err, ErrWorkerLeaseLost)
		}
	})

	t.Run("expired sixth attempt remains claimable for reconciliation", func(t *testing.T) {
		store := NewStore(newScheduledPostStoreTestPool(t))
		ctx := context.Background()
		created, err := store.Create(ctx, capacityCreateParams(owner, 1, now))
		if err != nil {
			t.Fatal(err)
		}
		first, err := store.ClaimDue(ctx, 1, now, time.Minute)
		if err != nil || len(first) != 1 {
			t.Fatalf("first claim=%v err=%v", first, err)
		}
		if _, err := store.pool.Exec(ctx, `
			UPDATE scheduled_posts
			SET attempt_count=6, lease_expires_at=$2
			WHERE id=$1
		`, created.ID, now.Add(time.Minute)); err != nil {
			t.Fatalf("prepare expired sixth attempt: %v", err)
		}

		recovered, err := store.ClaimDue(ctx, 1, now.Add(time.Minute), time.Minute)
		if err != nil {
			t.Fatalf("recover sixth attempt: %v", err)
		}
		if len(recovered) != 1 {
			t.Fatalf("recovered claims=%d, want 1", len(recovered))
		}
		if recovered[0].ID != created.ID || recovered[0].Rkey != first[0].Rkey ||
			!recovered[0].CreatedAt.Equal(first[0].CreatedAt) ||
			recovered[0].LeaseToken == first[0].LeaseToken {
			t.Fatalf("sixth-attempt recovery changed identity or lease: first=%#v recovered=%#v", first[0], recovered[0])
		}
		var status Status
		var attemptCount int
		if err := store.pool.QueryRow(ctx, `
			SELECT status, attempt_count FROM scheduled_posts WHERE id=$1
		`, created.ID).Scan(&status, &attemptCount); err != nil {
			t.Fatalf("inspect sixth-attempt recovery: %v", err)
		}
		if status != StatusPublishing || attemptCount != 6 {
			t.Fatalf("recovered status/attempt=%s/%d, want publishing/6", status, attemptCount)
		}
	})
}

func TestClaimedScheduleAcquiresEffectFenceBeforeExternalWrite(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	created, err := store.Create(ctx, capacityCreateParams(owner, 1, now))
	if err != nil {
		t.Fatalf("create due schedule: %v", err)
	}
	claims, err := store.ClaimDue(ctx, 1, now, time.Minute)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim due schedule: claims=%v err=%v", claims, err)
	}
	claim := claims[0]

	wrongLease := claim
	wrongLease.LeaseToken = uuid.New()
	if guard, err := store.AcquirePublishingEffect(ctx, wrongLease); guard != nil ||
		!errors.Is(err, ErrWorkerLeaseLost) {
		t.Fatalf("wrong lease guard=%v error=%v, want lease lost", guard, err)
	}

	guard, err := store.AcquirePublishingEffect(ctx, claim)
	if err != nil {
		t.Fatalf("acquire claimed publication effect: %v", err)
	}
	externalWrites := 0
	externalWrites++

	mutation := make(chan error, 1)
	go func() {
		mutation <- store.Delete(ctx, owner, created.ID, now)
	}()
	select {
	case err := <-mutation:
		t.Fatalf("mutation crossed held publication effect lock: %v", err)
	default:
	}
	if err := guard.Release(ctx); err != nil {
		t.Fatalf("release publication effect: %v", err)
	}
	if err := <-mutation; !errors.Is(err, ErrMutationLocked) {
		t.Fatalf("delete after Publishing error=%v, want %v", err, ErrMutationLocked)
	}
	if externalWrites != 1 {
		t.Fatalf("external writes=%d, want 1 after fence", externalWrites)
	}
}

func TestPublishingEffectReleaseUnlocksWhenCallerIsCancelled(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	created, err := store.Create(ctx, capacityCreateParams(owner, 1, now))
	if err != nil {
		t.Fatalf("create due schedule: %v", err)
	}
	claims, err := store.ClaimDue(ctx, 1, now, time.Minute)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim due schedule: claims=%v err=%v", claims, err)
	}
	guard, err := store.AcquirePublishingEffect(ctx, claims[0])
	if err != nil {
		t.Fatalf("acquire publication effect: %v", err)
	}

	other, err := store.pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire independent connection: %v", err)
	}
	defer other.Release()

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := guard.Release(cancelled); err != nil {
		t.Fatalf("release with cancelled caller: %v", err)
	}

	var locked bool
	if err := other.QueryRow(ctx, `
		SELECT pg_try_advisory_lock(
			hashtextextended($1::text || ':' || $2::uuid::text, 0)
		)
	`, owner, created.ID).Scan(&locked); err != nil {
		t.Fatalf("try matching effect lock: %v", err)
	}
	if !locked {
		t.Fatal("released pool connection still owns the session advisory lock")
	}
	if _, err := other.Exec(ctx, unlockScheduleEffectForSessionSQL, owner, created.ID); err != nil {
		t.Fatalf("unlock independent connection: %v", err)
	}
}

func TestStoreAllocatesDistinctRkeysWhenClaimClockIDsCollide(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	first := capacityCreateParams(owner, 1, now)
	second := capacityCreateParams(owner, 2, now)
	second.ID = uuid.MustParse("11111111-1111-4111-8111-000000000001")

	for _, params := range []CreateParams{first, second} {
		if _, err := store.Create(ctx, params); err != nil {
			t.Fatalf("create due schedule %s: %v", params.ID, err)
		}
	}
	claims, err := store.ClaimDue(ctx, 2, now, time.Minute)
	if err != nil {
		t.Fatalf("claim colliding clock IDs: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("claims=%d, want 2", len(claims))
	}
	if claims[0].Rkey == claims[1].Rkey {
		t.Fatalf("colliding schedules received rkey %q", claims[0].Rkey)
	}

	var distinct int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(DISTINCT publication_rkey)
		FROM scheduled_posts
		WHERE owner_did=$1 AND status='publishing'
	`, owner).Scan(&distinct); err != nil {
		t.Fatalf("count persisted rkeys: %v", err)
	}
	if distinct != 2 {
		t.Fatalf("persisted distinct rkeys=%d, want 2", distinct)
	}
}

func TestStoreCreateRejectsMismatchedScheduledMediaReferences(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	payload := []byte(`{"kind":"standard","text":"external","external":{"sourceUri":"https://source.example/pattern","uri":"https://final.example/pattern","title":"Pattern","description":"Description","thumbMediaId":"55555555-5555-4555-8555-555555555555"}}`)
	params := capacityCreateParams("did:plc:alice", 9, now)
	params.PayloadBytes = payload
	params.PayloadHash = sha256.Sum256(payload)
	params.MediaIDs = []uuid.UUID{uuid.MustParse("66666666-6666-4666-8666-666666666666")}

	_, err := store.Create(t.Context(), params)
	if !errors.Is(err, ErrScheduledMediaInvalid) {
		t.Fatalf("Create error=%v, want ErrScheduledMediaInvalid", err)
	}
}

type createAttempt struct {
	id  uuid.UUID
	err error
}

func capacityCreateParams(owner syntax.DID, suffix byte, due time.Time) CreateParams {
	payload := []byte(`{"kind":"standard","text":"private"}`)
	payloadHash := sha256.Sum256(payload)
	requestHash := sha256.Sum256([]byte{suffix})
	return CreateParams{
		ID:             uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", suffix)),
		OwnerDID:       owner,
		OperationID:    uuid.MustParse(fmt.Sprintf("10000000-0000-4000-8000-%012d", suffix)),
		RequestHash:    requestHash,
		ScheduledAt:    due,
		PayloadBytes:   payload,
		PayloadHash:    payloadHash,
		PayloadVersion: 1,
	}
}

func withPayloadMedia(params CreateParams, mediaIDs ...uuid.UUID) CreateParams {
	media := make([]PayloadMedia, 0, len(mediaIDs))
	for _, mediaID := range mediaIDs {
		media = append(media, PayloadMedia{ID: mediaID.String()})
	}
	payload, err := EncodePayload(Payload{Kind: PostKindStandard, Text: "private", Media: media})
	if err != nil {
		panic(err)
	}
	params.PayloadBytes = payload
	params.PayloadHash = sha256.Sum256(payload)
	params.MediaIDs = append([]uuid.UUID(nil), mediaIDs...)
	return params
}

func newScheduledPostStoreTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ddl := scheduledPostStorePreStateDDL
	for _, path := range []string{
		"../../migrations/000034_scheduled_posts.up.sql",
		"../../migrations/000039_owner_effects_terminal_purge.up.sql",
		"../../migrations/000040_scheduled_media_durability.up.sql",
		"../../migrations/000041_account_deletion_safety_tombstones.up.sql",
		"../../migrations/000045_tap_ingestion_durability.up.sql",
		"../../migrations/000048_scheduled_post_owner_generation.up.sql",
		"../../migrations/000049_pds_effect_action.up.sql",
		"../../migrations/000050_pds_effect_source_reconciliation.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read scheduled-post migration %s: %v", path, err)
		}
		ddl += string(migration)
	}
	return testdb.WithSchema(t, ddl)
}

func newScheduledTestOwnerFencer(t *testing.T, pool *pgxpool.Pool) *ownerlifecycle.Fencer {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("construct scheduled-test owner fencer: %v", err)
	}
	return fencer
}

func newScheduledTestMediaService(
	t *testing.T,
	store *Store,
	objects PrivateObjectStore,
) (*PrivateMediaService, *ownerlifecycle.Fencer) {
	t.Helper()
	fencer := newScheduledTestOwnerFencer(t, store.pool)
	lifecycle, err := ownerlifecycle.NewStore(store.pool, fencer, time.Now)
	if err != nil {
		t.Fatalf("construct scheduled-test lifecycle store: %v", err)
	}
	return NewPrivateMediaService(store, objects, PrivateMediaServiceOptions{
		Lifecycle: lifecycle, PutTimeout: time.Minute,
	}), fencer
}

func insertReadyPrivateMediaFixture(
	t *testing.T,
	store *Store,
	owner syntax.DID,
	mediaID uuid.UUID,
	scheduleID *uuid.UUID,
	ordinal *int,
	blobCID string,
	expiresAt time.Time,
) string {
	t.Helper()
	objectKey, attemptID, err := NewGenerationObjectKey(owner, 1, mediaID)
	if err != nil {
		t.Fatalf("derive media fixture key: %v", err)
	}
	startedAt := expiresAt.Add(-2 * time.Hour)
	dispatchedAt := startedAt.Add(time.Second)
	completedAt := dispatchedAt.Add(time.Second)
	ctx := context.Background()
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin ready media fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO scheduled_post_object_attempts (
			upload_attempt_id,media_id,owner_did,owner_generation,
			upload_generation,object_key,request_fingerprint,remote_outcome,
			remote_started_at,remote_deadline,dispatched_at,completed_at,
			created_at,updated_at
		) VALUES (
			$1,$2,$3,1,1,$4,decode(repeat('03',32),'hex'),'accepted',
			$5,$6,$7,$8,$5,$8
		)
	`, attemptID, mediaID, owner, objectKey, startedAt, startedAt.Add(time.Minute),
		dispatchedAt, completedAt); err != nil {
		t.Fatalf("insert ready media attempt %s: %v", mediaID, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO scheduled_post_media (
			id,owner_did,owner_generation,upload_generation,upload_attempt_id,
			object_key,state,schedule_id,ordinal,mime_type,size_bytes,sha256,
			blob_cid,unclaimed_expires_at
		) VALUES (
			$2,$3,1,1,$1,$4,'ready',$5,$6,'image/jpeg',4,
			decode(repeat('03',32),'hex'),$7,$8
		)
	`, attemptID, mediaID, owner, objectKey, scheduleID, ordinal, blobCID, expiresAt); err != nil {
		t.Fatalf("insert ready media fixture %s: %v", mediaID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit ready media fixture %s: %v", mediaID, err)
	}
	return objectKey
}

func insertCleanupFixture(
	t *testing.T,
	store *Store,
	owner syntax.DID,
	mediaID uuid.UUID,
	now time.Time,
	outcomeUncertain bool,
) string {
	t.Helper()
	objectKey, attemptID, err := NewGenerationObjectKey(owner, 1, mediaID)
	if err != nil {
		t.Fatalf("derive cleanup fixture key: %v", err)
	}
	outcome := "accepted"
	var completedAt *time.Time
	dispatchedAt := now.Add(-time.Minute)
	if outcomeUncertain {
		outcome = "dispatched"
	} else {
		completion := dispatchedAt.Add(time.Second)
		completedAt = &completion
	}
	if _, err := store.pool.Exec(context.Background(), `
		INSERT INTO scheduled_post_object_attempts (
			upload_attempt_id,media_id,owner_did,owner_generation,
			upload_generation,object_key,request_fingerprint,remote_outcome,
			remote_started_at,remote_deadline,dispatched_at,completed_at,
			created_at,updated_at
		) VALUES (
			$1,$2,$3,1,1,$4,decode(repeat('05',32),'hex'),$5,
			$6,$7,$8::timestamptz,$9::timestamptz,$6,
			COALESCE($9::timestamptz,$8::timestamptz)
		)
	`, attemptID, mediaID, owner, objectKey, outcome,
		dispatchedAt.Add(-time.Second), now.Add(time.Minute), dispatchedAt, completedAt); err != nil {
		t.Fatalf("insert cleanup attempt fixture: %v", err)
	}
	if _, err := store.pool.Exec(
		context.Background(), insertCleanupJobSQL, uuid.New(), objectKey, now,
	); err != nil {
		t.Fatalf("insert cleanup job fixture: %v", err)
	}
	return objectKey
}
