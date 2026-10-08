package scheduledposts

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestClaimedScheduledPostCannotAppendAfterAccessLossCommits(t *testing.T) {
	pool := newScheduledPostStoreTestPool(t)
	ctx := context.Background()
	if err := testdb.ApplyMigrations(ctx, pool, "000076_subscription_accounts.up.sql"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE profile_pins(owner_did TEXT, slot TEXT)`); err != nil {
		t.Fatal(err)
	}
	accountID := uuid.MustParse("10000000-0000-4000-8000-000000000091")
	for _, query := range []string{
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000091','did:plc:billing-owner','20000000-0000-4000-8000-000000000091')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000091','10000000-0000-4000-8000-000000000091','project','sub','plus','app','app_store','production','active',true,'plus',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000091','plus','did:plc:alice',now())`,
	} {
		if _, err := pool.Exec(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	access := subscriptions.NewStore(pool)
	store := NewStore(pool)
	now := time.Now().UTC()
	payload, err := EncodePayload(Payload{Kind: PostKindStandard, Text: "claimed during lapse"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Create(ctx, CreateParams{ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(), RequestHash: [32]byte{1}, ScheduledAt: now.Add(-time.Minute), PayloadBytes: payload, PayloadHash: sha256.Sum256(payload), PayloadVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	futureDue := now.Add(2 * time.Hour)
	future, err := store.Create(ctx, CreateParams{ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(), RequestHash: [32]byte{2}, ScheduledAt: futureDue, PayloadBytes: payload, PayloadHash: sha256.Sum256(payload), PayloadVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	claimed, release := make(chan struct{}), make(chan struct{})
	pds := &recordingScheduledPDS{}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{Store: store,
		Sessions:    stubPublicationSessionSelector{wantOwner: "did:plc:alice", sessionID: "owner-session"},
		NewCommands: recordingGuardedFactory(pds, nil), Objects: newMemoryPrivateObjectStore(), Now: func() time.Time { return now },
		CheckPlusAccess: func(ctx context.Context, did syntax.DID) (bool, error) {
			a, err := access.SelfAccess(ctx, did, now)
			return a.AllowsPlus(), err
		},
		WithPlusAccess: func(ctx context.Context, did syntax.DID, effect func(context.Context) error) error {
			err := access.WithPlusAccess(ctx, did, effect)
			if errors.Is(err, subscriptions.ErrFeatureAccessRequired) {
				return ErrSubscriptionRequired
			}
			return err
		},
		Validate: func(context.Context, syntax.DID, Payload) error {
			select {
			case <-claimed:
			default:
				close(claimed)
			}
			<-release
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewWorker(WorkerOptions{Store: store, Processor: processor, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	processed := make(chan error, 1)
	go func() { _, err := worker.ProcessBatch(ctx); processed <- err }()
	select {
	case <-claimed:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not reach claimed validation")
	}
	catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]subscriptions.ProductMapping{"plus": {AppID: "app", Tier: subscriptions.TierPlus}}})
	if err != nil {
		t.Fatal(err)
	}
	token := uuid.New()
	if _, err := pool.Exec(ctx, `UPDATE billing_accounts SET requested_generation=2,claimed_generation=2,lease_token=$2,lease_expires_at=now()+interval '1 hour' WHERE id=$1`, accountID, token); err != nil {
		t.Fatal(err)
	}
	err = access.ApplySnapshot(ctx, subscriptions.SnapshotClaim{BillingAccountID: accountID, Generation: 2, LeaseToken: token}, subscriptions.CompleteSnapshot{Complete: true, Subscriptions: []subscriptions.ProviderSubscriptionSnapshot{{ID: "sub", ProductID: "plus", Store: "app_store", Environment: "production", Status: "expired", GivesAccess: false}}}, catalog)
	if err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-processed; err != nil {
		t.Fatal(err)
	}
	resource, err := store.Get(ctx, created.OwnerDID, created.ID)
	if err != nil || resource.Status != StatusNeedsAttention || resource.LastErrorCode != "subscription_required" || pds.putCalls != 0 {
		t.Fatalf("status=%s code=%s puts=%d err=%v", resource.Status, resource.LastErrorCode, pds.putCalls, err)
	}
	processedCount, err := worker.ProcessBatch(ctx)
	if err != nil || processedCount != 0 || pds.putCalls != 0 {
		t.Fatalf("future item during lapse: processed=%d puts=%d err=%v", processedCount, pds.putCalls, err)
	}
	futureState, err := store.Get(ctx, future.OwnerDID, future.ID)
	if err != nil || futureState.Status != StatusScheduled {
		t.Fatalf("future item after lapse: %+v err=%v", futureState, err)
	}
	token = uuid.New()
	if _, err := pool.Exec(ctx, `UPDATE billing_accounts SET requested_generation=3,claimed_generation=3,lease_token=$2,lease_expires_at=now()+interval '1 hour' WHERE id=$1`, accountID, token); err != nil {
		t.Fatal(err)
	}
	if err := access.ApplySnapshot(ctx, subscriptions.SnapshotClaim{BillingAccountID: accountID, Generation: 3, LeaseToken: token}, subscriptions.CompleteSnapshot{Complete: true, Subscriptions: []subscriptions.ProviderSubscriptionSnapshot{{ID: "sub", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true}}}, catalog); err != nil {
		t.Fatal(err)
	}
	now = futureDue.Add(-time.Second)
	processedCount, err = worker.ProcessBatch(ctx)
	if err != nil || processedCount != 0 || pds.putCalls != 0 {
		t.Fatalf("future item before due after restoration: processed=%d puts=%d err=%v", processedCount, pds.putCalls, err)
	}
	now = futureDue.Add(time.Second)
	processedCount, err = worker.ProcessBatch(ctx)
	if err != nil || processedCount != 1 || pds.putCalls != 1 {
		t.Fatalf("future item due after restoration: processed=%d puts=%d err=%v", processedCount, pds.putCalls, err)
	}
	if _, err := store.Get(ctx, future.OwnerDID, future.ID); !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("published future item still manageable: %v", err)
	}
	resource, err = store.Get(ctx, created.OwnerDID, created.ID)
	if err != nil || resource.Status != StatusNeedsAttention || resource.LastErrorCode != "subscription_required" {
		t.Fatalf("missed item after restoration: %+v err=%v", resource, err)
	}
}

func TestHealthyDueSchedulePublishesExactlyOnceWithoutFlutter(t *testing.T) {
	store := NewStore(newScheduledPostStoreTestPool(t))
	ctx := context.Background()
	due := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	current := due.Add(-time.Second)
	payload, err := EncodePayload(Payload{
		Kind: PostKindStandard, Text: "publish while the app is closed", Langs: []string{"en"},
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Create(ctx, CreateParams{
		ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(),
		RequestHash: [32]byte{1}, ScheduledAt: due, PayloadBytes: payload,
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
		NewCommands: recordingGuardedFactory(pds, nil),
		Objects:     newMemoryPrivateObjectStore(),
		Now:         func() time.Time { return current },
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

	processed, err := worker.ProcessBatch(ctx)
	if err != nil || processed != 0 || pds.putCalls != 0 || pds.uploadCalls != 0 {
		t.Fatalf("before due: processed=%d err=%v puts=%d uploads=%d", processed, err, pds.putCalls, pds.uploadCalls)
	}

	current = due.Add(30 * time.Second)
	processed, err = worker.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("due batch: processed=%d err=%v", processed, err)
	}
	if pds.putCalls != 1 || pds.rkey == "" || pds.record["createdAt"] != current.Format(time.RFC3339) || pds.record["sponsored"] != false {
		t.Fatalf("publication puts=%d rkey=%q record=%#v", pds.putCalls, pds.rkey, pds.record)
	}
	if _, err := store.Get(ctx, "did:plc:alice", created.ID); !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("published schedule remains manageable: %v", err)
	}

	processed, err = worker.ProcessBatch(ctx)
	if err != nil || processed != 0 || pds.putCalls != 1 {
		t.Fatalf("repeated batch: processed=%d err=%v puts=%d", processed, err, pds.putCalls)
	}
}
