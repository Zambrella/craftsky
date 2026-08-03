package scheduledposts

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
)

func TestScheduledLifecycleLeavesNotificationStateUnchanged(t *testing.T) {
	pool := newScheduledPostStoreTestPool(t)
	seedNotificationBoundarySentinels(t, pool)
	before := readNotificationBoundarySnapshot(t, pool)
	store := NewStore(pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

	create := func(suffix byte) ScheduledPost {
		t.Helper()
		payload, err := EncodePayload(Payload{Kind: PostKindStandard, Text: "private"})
		if err != nil {
			t.Fatal(err)
		}
		created, err := store.Create(ctx, CreateParams{
			ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
			RequestHash: [32]byte{suffix}, ScheduledAt: now,
			PayloadBytes: payload, PayloadHash: [32]byte{suffix + 10}, PayloadVersion: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		return created
	}
	newWorker := func(validate func(context.Context, syntax.DID, Payload) error, sessions stubPublicationSessionSelector, pds *recordingScheduledPDS) *Worker {
		t.Helper()
		processor, err := NewPublicationProcessor(PublicationProcessorOptions{
			Store: store, Sessions: sessions,
			NewPDS: func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
				if pds == nil {
					return nil, ErrAuthUnavailable
				}
				return pds, nil
			},
			Objects: newMemoryPrivateObjectStore(), Now: func() time.Time { return now },
			Validate: validate,
		})
		if err != nil {
			t.Fatal(err)
		}
		worker, err := NewWorker(WorkerOptions{
			Store: store, Processor: processor, Now: func() time.Time { return now }, BatchSize: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		return worker
	}

	retrying := create(1)
	retryWorker := newWorker(nil, stubPublicationSessionSelector{
		wantOwner: "did:plc:alice", err: auth.ErrNoUsableBackgroundSession,
	}, nil)
	if processed, err := retryWorker.ProcessBatch(ctx); err != nil || processed != 1 {
		t.Fatalf("retry transition: processed=%d err=%v", processed, err)
	}
	assertScheduledStatus(t, store, retrying.ID, StatusRetrying)

	needsAttention := create(2)
	policyWorker := newWorker(
		func(context.Context, syntax.DID, Payload) error { return ErrPolicyInvalid },
		stubPublicationSessionSelector{wantOwner: "did:plc:alice", sessionID: "owner-session"},
		&recordingScheduledPDS{},
	)
	if processed, err := policyWorker.ProcessBatch(ctx); err != nil || processed != 1 {
		t.Fatalf("Needs attention transition: processed=%d err=%v", processed, err)
	}
	assertScheduledStatus(t, store, needsAttention.ID, StatusNeedsAttention)

	published := create(3)
	pds := &recordingScheduledPDS{}
	publishWorker := newWorker(nil,
		stubPublicationSessionSelector{wantOwner: "did:plc:alice", sessionID: "owner-session"},
		pds,
	)
	if processed, err := publishWorker.ProcessBatch(ctx); err != nil || processed != 1 {
		t.Fatalf("publish transition: processed=%d err=%v", processed, err)
	}
	if pds.putCalls != 1 {
		t.Fatalf("PDS puts=%d, want 1", pds.putCalls)
	}
	if _, err := store.Get(ctx, "did:plc:alice", published.ID); !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("published schedule remains visible: %v", err)
	}

	after := readNotificationBoundarySnapshot(t, pool)
	if before != after {
		t.Fatalf("notification state changed across scheduled lifecycle: before=%q after=%q", before, after)
	}
}

func assertScheduledStatus(t *testing.T, store *Store, id uuid.UUID, want Status) {
	t.Helper()
	resource, err := store.Get(context.Background(), "did:plc:alice", id)
	if err != nil {
		t.Fatal(err)
	}
	if resource.Status != want {
		t.Fatalf("schedule %s status=%s, want %s", id, resource.Status, want)
	}
}

func seedNotificationBoundarySentinels(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE notification_events (marker TEXT PRIMARY KEY);
		CREATE TABLE notification_preferences (marker TEXT PRIMARY KEY);
		CREATE TABLE push_deliveries (marker TEXT PRIMARY KEY);
		CREATE TABLE push_account_subscriptions (marker TEXT PRIMARY KEY);
		INSERT INTO notification_events VALUES ('event-before');
		INSERT INTO notification_preferences VALUES ('preference-before');
		INSERT INTO push_deliveries VALUES ('delivery-before');
		INSERT INTO push_account_subscriptions VALUES ('subscription-before');
	`); err != nil {
		t.Fatal(err)
	}
}

func readNotificationBoundarySnapshot(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var snapshot string
	if err := pool.QueryRow(context.Background(), `
		SELECT concat_ws('|',
			(SELECT string_agg(marker, ',' ORDER BY marker) FROM notification_events),
			(SELECT string_agg(marker, ',' ORDER BY marker) FROM notification_preferences),
			(SELECT string_agg(marker, ',' ORDER BY marker) FROM push_deliveries),
			(SELECT string_agg(marker, ',' ORDER BY marker) FROM push_account_subscriptions)
		)
	`).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestScheduledLifecycleHasNoNotificationProducerBoundary(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(".", entry.Name())
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, spec := range file.Imports {
			if strings.Contains(spec.Path.Value, "notifications") {
				t.Fatalf("%s imports notification producer %s", path, spec.Path.Value)
			}
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, notificationStore := range []string{
			"notification_events",
			"notification_preferences",
			"push_deliveries",
			"push_account_subscriptions",
		} {
			if strings.Contains(string(source), notificationStore) {
				t.Fatalf("%s writes notification boundary %q", path, notificationStore)
			}
		}
	}
}
