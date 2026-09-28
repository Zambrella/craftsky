package index_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

type failingRetractionLifecycle struct{ service *notifications.Service }

func (f failingRetractionLifecycle) Activate(ctx context.Context, tx pgx.Tx, activation notifications.Activation) error {
	return f.service.Activate(ctx, tx, activation)
}

func (f failingRetractionLifecycle) Retract(ctx context.Context, tx pgx.Tx, retraction notifications.Retraction) error {
	if err := f.service.Retract(ctx, tx, retraction); err != nil {
		return err
	}
	return errors.New("forced retraction failure")
}

func TestPostDeletionRollsBackSourceNotificationAndDeliveryTogether(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	applyNotificationMigration(t, pool)
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskyMember(t, pool, "did:plc:recipient")
	if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES('at://did:plc:recipient/social.craftsky.feed.post/root','did:plc:recipient','root','rootcid','root','{}',now())`); err != nil {
		t.Fatal(err)
	}
	seedNotificationSubscription(t, pool, "did:plc:recipient")
	service := notifications.NewService()
	ev := tap.Event{URI: "at://did:plc:actor/social.craftsky.feed.post/reply", CID: "replycid", DID: "did:plc:actor", Rkey: "reply", Collection: "social.craftsky.feed.post", Action: "create", Record: json.RawMessage(`{"text":"reply","createdAt":"2026-05-04T12:00:00Z","reply":{"root":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/root","cid":"rootcid"},"parent":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/root","cid":"rootcid"}}}`)}
	if err := index.NewCraftskyPost(pool, testLogger(), service).Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	ev.Action, ev.Record = "delete", nil
	if err := index.NewCraftskyPost(pool, testLogger(), failingRetractionLifecycle{service}).Handle(context.Background(), ev); err == nil {
		t.Fatal("delete succeeded, want forced rollback")
	}
	assertDeletionRollbackState(t, pool, `SELECT count(*) FROM craftsky_posts WHERE uri='at://did:plc:actor/social.craftsky.feed.post/reply'`)
}

func applyNotificationMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	migration, err := testdb.ReadMigration("000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
}

func seedNotificationSubscription(t *testing.T, pool *pgxpool.Pool, accountDID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES('10000000-0000-0000-0000-000000000001','device','ios','token')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id) VALUES('20000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000001',$1,'30000000-0000-0000-0000-000000000001')`, accountDID); err != nil {
		t.Fatal(err)
	}
}

func assertDeletionRollbackState(t *testing.T, pool *pgxpool.Pool, sourceCountSQL string) {
	t.Helper()
	var sourceCount int
	if err := pool.QueryRow(context.Background(), sourceCountSQL).Scan(&sourceCount); err != nil {
		t.Fatal(err)
	}
	var state, deliveryStatus string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM notification_events`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&deliveryStatus); err != nil {
		t.Fatal(err)
	}
	if sourceCount != 1 || state != "active" || deliveryStatus != "pending" {
		t.Fatalf("rollback source=%d state=%s delivery=%s", sourceCount, state, deliveryStatus)
	}
}
