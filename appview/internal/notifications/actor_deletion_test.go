package notifications

import (
	"context"
	"os"
	"social.craftsky/appview/internal/testdb"
	"testing"
)

func TestActorDeletionHardDeletesCausedNotificationsAndDeliveries(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(), `INSERT INTO notification_events(id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,first_activity_at,activity_at,indexed_at,initial_push_evaluated_at)VALUES('00000000-0000-0000-0000-000000000001','did:plc:viewer','did:plc:actor','follow','s','u','c','r','everyone',false,true,'active',now(),now(),now(),now());INSERT INTO push_installations(id,device_id,platform,fcm_token)VALUES('10000000-0000-0000-0000-000000000001','d','ios','t');INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id)VALUES('20000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000001','did:plc:viewer','30000000-0000-0000-0000-000000000001');INSERT INTO push_deliveries(id,notification_id,account_subscription_id,status,next_attempt_at,deadline_at)VALUES('40000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','pending',now(),now()+interval '6 hours')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewActorDeletionService(pool).HandleIdentityDeleted(context.Background(), "did:plc:actor"); err != nil {
		t.Fatal(err)
	}
	var events, deliveries int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM notification_events`).Scan(&events)
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM push_deliveries`).Scan(&deliveries)
	if events != 0 || deliveries != 0 {
		t.Fatalf("events=%d deliveries=%d", events, deliveries)
	}
}
