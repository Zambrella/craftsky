package ingestion_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestQuarantineIsBoundedIdempotentAndReplayable(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	largeEnvelope := append([]byte(`{"id":81,"type":"record","padding":"`), bytes.Repeat([]byte("x"), 70_000)...)
	largeEnvelope = append(largeEnvelope, []byte(`"}`)...)
	invalid := tap.InvalidEvent{
		ID: 81, Type: "record", Reason: tap.ReasonInvalidCollection, Envelope: largeEnvelope,
	}
	for attempt := 0; attempt < 2; attempt++ {
		outcome, err := store.Quarantine(ctx, invalid)
		if err != nil || outcome.Kind != tap.OutcomePermanentInvalid {
			t.Fatalf("quarantine attempt %d outcome=%+v err=%v", attempt, outcome, err)
		}
	}

	items, err := store.ListQuarantine(ctx, 10)
	if err != nil {
		t.Fatalf("list quarantine: %v", err)
	}
	if len(items) != 1 || items[0].OccurrenceCount != 2 || len(items[0].Envelope) > 65_536 {
		t.Fatalf("quarantine items=%+v", items)
	}
	if bytes.Contains(items[0].Envelope, bytes.Repeat([]byte("x"), 1024)) {
		t.Fatal("bounded quarantine retained the oversized raw payload")
	}

	if err := store.RequestQuarantineReplay(ctx, items[0].Fingerprint); err != nil {
		t.Fatalf("request replay: %v", err)
	}
	claims, err := store.ClaimQuarantine(ctx, ingestion.QuarantineClaimRequest{
		Worker: "quarantine-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim quarantine=%+v err=%v", claims, err)
	}
	replayed := 0
	if err := store.ReplayQuarantine(ctx, claims[0], func(_ context.Context, envelope []byte) (tap.Outcome, error) {
		replayed++
		if len(envelope) == 0 {
			t.Fatal("replay received empty bounded evidence")
		}
		return tap.PermanentInvalid(tap.ReasonInvalidCollection), nil
	}); err != nil {
		t.Fatalf("replay quarantine: %v", err)
	}
	if replayed != 1 {
		t.Fatalf("replay calls=%d", replayed)
	}
	items, err = store.ListQuarantine(ctx, 10)
	if err != nil || len(items) != 1 || items[0].ReplayState != "resolved" {
		t.Fatalf("resolved quarantine=%+v err=%v", items, err)
	}
}

func TestOversizedQuarantineReplayPreservesExactWireEnvelopeAcrossRestart(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	ctx := context.Background()

	wireEnvelope := append([]byte(`{"id":181,"type":"record","record":{"live":true,"rev":"3m00000000181","did":"did:plc:quarantine-replay","collection":"social.craftsky.feed.post","rkey":"large","action":"create","cid":"bafy-quarantine-replay","record":{"text":"`), bytes.Repeat([]byte("x"), 70_000)...)
	wireEnvelope = append(wireEnvelope, []byte(`","createdAt":"2026-08-20T12:00:00Z"}}}`)...)
	if len(wireEnvelope) <= 64<<10 {
		t.Fatalf("wire envelope bytes=%d, want oversized diagnostic evidence", len(wireEnvelope))
	}

	firstStore, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new first ingestion store: %v", err)
	}
	invalid := tap.InvalidEvent{
		ID:       181,
		Type:     "record",
		Reason:   tap.ReasonMalformedRecord,
		Envelope: wireEnvelope,
	}
	if outcome, err := firstStore.Quarantine(ctx, invalid); err != nil || !outcome.Acknowledgable() {
		t.Fatalf("first quarantine outcome=%+v err=%v", outcome, err)
	}

	// A new Store models an AppView restart. Redelivery of the same frame must
	// remain idempotent while retaining the exact replay payload.
	restartedStore, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new restarted ingestion store: %v", err)
	}
	if outcome, err := restartedStore.Quarantine(ctx, invalid); err != nil || !outcome.Acknowledgable() {
		t.Fatalf("redelivered quarantine outcome=%+v err=%v", outcome, err)
	}

	items, err := restartedStore.ListQuarantine(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("list quarantine=%+v err=%v", items, err)
	}
	if items[0].OccurrenceCount != 2 {
		t.Fatalf("occurrence count=%d, want 2", items[0].OccurrenceCount)
	}
	if len(items[0].Envelope) > 64<<10 || bytes.Equal(items[0].Envelope, wireEnvelope) ||
		bytes.Contains(items[0].Envelope, bytes.Repeat([]byte("x"), 1024)) {
		t.Fatal("operator listing exposed the oversized wire payload instead of bounded evidence")
	}

	if err := restartedStore.RequestQuarantineReplay(ctx, items[0].Fingerprint); err != nil {
		t.Fatalf("request replay: %v", err)
	}
	claims, err := restartedStore.ClaimQuarantine(ctx, ingestion.QuarantineClaimRequest{
		Worker: "quarantine-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim quarantine=%+v err=%v", claims, err)
	}
	if err := restartedStore.ReplayQuarantine(ctx, claims[0], func(operationCtx context.Context, envelope []byte) (tap.Outcome, error) {
		if !bytes.Equal(envelope, wireEnvelope) {
			t.Fatalf("replay envelope bytes=%d, want exact %d-byte wire frame", len(envelope), len(wireEnvelope))
		}
		return tap.ReplayEnvelope(operationCtx, envelope, &replayDurableIngestor{})
	}); err != nil {
		t.Fatalf("replay quarantine: %v", err)
	}

	items, err = restartedStore.ListQuarantine(ctx, 1)
	if err != nil || len(items) != 1 || items[0].ReplayState != "resolved" {
		t.Fatalf("resolved quarantine=%+v err=%v", items, err)
	}
}

func TestMalformedQuarantineReplayPreservesExactWireEnvelopeAndReschedules(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	wireEnvelope := []byte("  {\n\t\"id\":182,\"type\":\"record\",\"record\": [not-json]  ")
	outcome, err := store.Quarantine(ctx, tap.InvalidEvent{
		Type: "unknown", Reason: tap.ReasonInvalidEnvelope, Envelope: wireEnvelope,
	})
	if err != nil || !outcome.Acknowledgable() {
		t.Fatalf("quarantine outcome=%+v err=%v", outcome, err)
	}
	items, err := store.ListQuarantine(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("list quarantine=%+v err=%v", items, err)
	}
	if bytes.Equal(items[0].Envelope, wireEnvelope) {
		t.Fatal("operator listing exposed a malformed raw wire payload")
	}
	if err := store.RequestQuarantineReplay(ctx, items[0].Fingerprint); err != nil {
		t.Fatalf("request replay: %v", err)
	}
	claims, err := store.ClaimQuarantine(ctx, ingestion.QuarantineClaimRequest{
		Worker: "quarantine-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim quarantine=%+v err=%v", claims, err)
	}
	replayed := false
	err = store.ReplayQuarantine(ctx, claims[0], func(operationCtx context.Context, envelope []byte) (tap.Outcome, error) {
		replayed = true
		if !bytes.Equal(envelope, wireEnvelope) {
			t.Fatalf("replay envelope=%q, want exact malformed wire frame %q", envelope, wireEnvelope)
		}
		return tap.ReplayEnvelope(operationCtx, envelope, &replayDurableIngestor{})
	})
	if err == nil || !replayed {
		t.Fatalf("malformed replay called=%t err=%v, want retryable decode failure", replayed, err)
	}
	items, listErr := store.ListQuarantine(ctx, 1)
	if listErr != nil || len(items) != 1 || items[0].ReplayState != "pending" {
		t.Fatalf("rescheduled malformed quarantine=%+v err=%v", items, listErr)
	}
}

func TestOversizedInvalidQuarantineReplayPreservesExactWireEnvelope(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	wireEnvelope := append([]byte(`{"id":183,"type":"record","record":{"live":true,"rev":"3m00000000183","did":"did:plc:quarantine-replay","collection":"invalid collection","rkey":"large-invalid","action":"create","cid":"bafy-quarantine-invalid","record":{"text":"`), bytes.Repeat([]byte("y"), 70_000)...)
	wireEnvelope = append(wireEnvelope, []byte(`"}}}`)...)
	if len(wireEnvelope) <= 64<<10 {
		t.Fatalf("wire envelope bytes=%d, want oversized diagnostic evidence", len(wireEnvelope))
	}
	if outcome, err := store.Quarantine(ctx, tap.InvalidEvent{
		ID: 183, Type: "record", Reason: tap.ReasonInvalidCollection, Envelope: wireEnvelope,
	}); err != nil || !outcome.Acknowledgable() {
		t.Fatalf("quarantine outcome=%+v err=%v", outcome, err)
	}
	items, err := store.ListQuarantine(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("list quarantine=%+v err=%v", items, err)
	}
	if len(items[0].Envelope) > 64<<10 || bytes.Contains(items[0].Envelope, bytes.Repeat([]byte("y"), 1024)) {
		t.Fatal("operator listing exposed oversized invalid wire content")
	}
	if err := store.RequestQuarantineReplay(ctx, items[0].Fingerprint); err != nil {
		t.Fatalf("request replay: %v", err)
	}
	claims, err := store.ClaimQuarantine(ctx, ingestion.QuarantineClaimRequest{
		Worker: "quarantine-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim quarantine=%+v err=%v", claims, err)
	}
	replayed := false
	err = store.ReplayQuarantine(ctx, claims[0], func(operationCtx context.Context, envelope []byte) (tap.Outcome, error) {
		replayed = true
		if !bytes.Equal(envelope, wireEnvelope) {
			t.Fatalf("replay envelope bytes=%d, want exact %d-byte invalid frame", len(envelope), len(wireEnvelope))
		}
		return tap.ReplayEnvelope(operationCtx, envelope, &replayDurableIngestor{})
	})
	if err == nil || !replayed {
		t.Fatalf("invalid replay called=%t err=%v, want retryable validation failure", replayed, err)
	}
	items, listErr := store.ListQuarantine(ctx, 1)
	if listErr != nil || len(items) != 1 || items[0].ReplayState != "pending" {
		t.Fatalf("rescheduled invalid quarantine=%+v err=%v", items, listErr)
	}
}

func TestQuarantineReplayWorkerConsumesOperatorQueuedWork(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	outcome, err := store.Quarantine(ctx, tap.InvalidEvent{
		ID: 91, Type: "record", Reason: tap.ReasonInvalidCollection,
		Envelope: []byte(`{"id":91,"type":"record"}`),
	})
	if err != nil || !outcome.Acknowledgable() {
		t.Fatalf("quarantine outcome=%+v err=%v", outcome, err)
	}
	items, err := store.ListQuarantine(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("list quarantine=%+v err=%v", items, err)
	}
	if err := store.RequestQuarantineReplay(ctx, items[0].Fingerprint); err != nil {
		t.Fatalf("request replay: %v", err)
	}

	replayed := 0
	worker, err := ingestion.NewQuarantineReplayWorker(ingestion.QuarantineReplayWorkerConfig{
		Store: store, WorkerID: "quarantine-worker", PollInterval: time.Second,
		LeaseDuration: 30 * time.Second, OperationTimeout: 10 * time.Second, BatchSize: 8,
		Handler: func(_ context.Context, envelope []byte) (tap.Outcome, error) {
			replayed++
			if len(envelope) == 0 {
				t.Fatal("empty replay envelope")
			}
			return tap.Applied(), nil
		},
	})
	if err != nil {
		t.Fatalf("new replay worker: %v", err)
	}
	processed, err := worker.RunOnce(ctx)
	if err != nil || processed != 1 || replayed != 1 {
		t.Fatalf("processed=%d replayed=%d err=%v", processed, replayed, err)
	}
	items, err = store.ListQuarantine(ctx, 1)
	if err != nil || len(items) != 1 || items[0].ReplayState != "resolved" {
		t.Fatalf("resolved quarantine=%+v err=%v", items, err)
	}
}

func TestQuarantineReplayWorkerTimeoutReschedulesWithLiveFinalizationContext(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	if _, err := store.Quarantine(ctx, tap.InvalidEvent{
		ID: 93, Type: "record", Reason: tap.ReasonInvalidCollection,
		Envelope: []byte(`{"id":93,"type":"record"}`),
	}); err != nil {
		t.Fatalf("quarantine: %v", err)
	}
	items, err := store.ListQuarantine(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("list quarantine=%+v err=%v", items, err)
	}
	if err := store.RequestQuarantineReplay(ctx, items[0].Fingerprint); err != nil {
		t.Fatalf("request replay: %v", err)
	}
	worker, err := ingestion.NewQuarantineReplayWorker(ingestion.QuarantineReplayWorkerConfig{
		Store: store, WorkerID: "quarantine-worker", PollInterval: time.Millisecond,
		LeaseDuration: time.Second, OperationTimeout: 5 * time.Millisecond, BatchSize: 1,
		Handler: func(ctx context.Context, _ []byte) (tap.Outcome, error) {
			<-ctx.Done()
			return tap.Retryable(tap.ReasonStorageUnavailable), ctx.Err()
		},
	})
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}
	processed, err := worker.RunOnce(ctx)
	if processed != 1 || err == nil {
		t.Fatalf("processed=%d err=%v", processed, err)
	}
	items, err = store.ListQuarantine(ctx, 1)
	if err != nil || len(items) != 1 || items[0].ReplayState != "pending" {
		t.Fatalf("rescheduled quarantine=%+v err=%v", items, err)
	}
}

func TestReplayEnvelopeRequeuesCurrentPermanentlyDeniedSource(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	uri := "at://did:plc:replay-source/social.craftsky.actor.profile/self"
	if _, err := store.IngestRecord(ctx, tap.Event{
		ID: 92, URI: "at://did:plc:replay-source/social.craftsky.actor.profile/self",
		DID: "did:plc:replay-source", Collection: "social.craftsky.actor.profile", Rkey: "self",
		Rev: "3mreplay-source", CID: "bafy-replay-source", Action: "create",
		Record: []byte(`{"crafts":["sewing"]}`),
	}); err != nil {
		t.Fatalf("ingest source: %v", err)
	}
	claims, err := store.ClaimProjectionJobs(ctx, ingestion.ProjectionClaimRequest{
		Worker: "projection-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim projection=%+v err=%v", claims, err)
	}
	if err := store.Project(ctx, claims[0], func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}); err != nil {
		t.Fatalf("permanently deny source: %v", err)
	}
	items, err := store.ListQuarantine(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("list quarantine=%+v err=%v", items, err)
	}
	outcome, err := store.ReplayEnvelope(ctx, items[0].Envelope, &replayDurableIngestor{})
	if err != nil || !outcome.Acknowledgable() {
		t.Fatalf("replay outcome=%+v err=%v", outcome, err)
	}
	job, err := store.ProjectionJob(ctx, syntax.ATURI(uri))
	if err != nil || job.State != "pending" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
}

type replayDurableIngestor struct{}

func (*replayDurableIngestor) IngestRecord(context.Context, tap.Event) (tap.Outcome, error) {
	return tap.Applied(), nil
}

func (*replayDurableIngestor) IngestIdentity(context.Context, tap.IdentityEvent) (tap.Outcome, error) {
	return tap.Applied(), nil
}

func (*replayDurableIngestor) Quarantine(context.Context, tap.InvalidEvent) (tap.Outcome, error) {
	return tap.Retryable(tap.ReasonStorageUnavailable), nil
}
