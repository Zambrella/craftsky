package index_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

func TestLikeIngestionPersistsFansOutAndReplaysIdempotently(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyInteractionsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply notification migration: %v", err)
	}
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_installations (id, device_id, platform, fcm_token) VALUES
		('10000000-0000-0000-0000-000000000001', 'device-1', 'ios', 'token-1'),
		('10000000-0000-0000-0000-000000000002', 'device-2', 'android', 'token-2');
		INSERT INTO push_account_subscriptions (id, installation_id, account_did, routing_id) VALUES
		('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'did:plc:author', '30000000-0000-0000-0000-000000000001'),
		('20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000002', 'did:plc:author', '30000000-0000-0000-0000-000000000002')
	`); err != nil {
		t.Fatalf("seed subscriptions: %v", err)
	}

	idx := index.NewCraftskyLike(pool, testLogger(), notifications.NewService())
	ev := interactionEvent(interactionIndexerCases()[0], "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle create: %v", err)
	}
	var firstID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM notification_events`).Scan(&firstID); err != nil {
		t.Fatalf("select notification: %v", err)
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle replay: %v", err)
	}

	var notificationCount, deliveryCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM notification_events`).Scan(&notificationCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_deliveries`).Scan(&deliveryCount); err != nil {
		t.Fatal(err)
	}
	if notificationCount != 1 || deliveryCount != 2 {
		t.Fatalf("notifications=%d deliveries=%d, want 1 and 2", notificationCount, deliveryCount)
	}
	var replayID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM notification_events`).Scan(&replayID); err != nil {
		t.Fatal(err)
	}
	if replayID != firstID {
		t.Fatalf("replay changed stable ID from %s to %s", firstID, replayID)
	}
}

func TestPushPreferenceIsProspectiveAndDoesNotBackfill(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyInteractionsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
	if _, err := pool.Exec(context.Background(), `INSERT INTO push_installations(id,device_id,platform,fcm_token)VALUES('10000000-0000-0000-0000-000000000001','d','ios','t');INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id)VALUES('20000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000001','did:plc:author','30000000-0000-0000-0000-000000000001');INSERT INTO notification_preferences(account_did,category,scope,push_enabled)VALUES('did:plc:author','like','everyone',false)`); err != nil {
		t.Fatal(err)
	}
	idx := index.NewCraftskyLike(pool, testLogger(), notifications.NewService())
	ev := interactionEvent(interactionIndexerCases()[0], "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE notification_preferences SET push_enabled=true WHERE account_did='did:plc:author' AND category='like'`); err != nil {
		t.Fatal(err)
	}
	var events, deliveries int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM notification_events`).Scan(&events)
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM push_deliveries`).Scan(&deliveries)
	if events != 1 || deliveries != 0 {
		t.Fatalf("events=%d deliveries=%d", events, deliveries)
	}
}
