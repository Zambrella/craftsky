package push

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

func TestDispatcherSendsActorfulInstagramMatchWithIdentityFreePayload(t *testing.T) {
	now := time.Date(2026, 7, 27, 15, 0, 0, 0, time.UTC)
	pool := instagramDispatcherPool(t)
	seedInstagramDelivery(t, pool, now, true)
	sender := &scriptedSender{}
	dispatcher := NewDispatcher(pool, sender, DispatcherOptions{
		BatchSize: 1,
		Now:       func() time.Time { return now },
	})
	processed, err := dispatcher.ProcessBatch(context.Background(), "instagram-worker")
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || len(sender.requests) != 1 {
		t.Fatalf("processed=%d requests=%d", processed, len(sender.requests))
	}
	request := sender.requests[0]
	payload := BuildPayload(
		request.Category,
		request.AccountSubscriptionID,
		request.ActorDisplayName,
		request.RoutingFacts,
	)
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"did:plc:",
		"at://",
		"synthetic.private.actor",
		"handle",
		"igsid",
	} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("provider payload leaked %q: %s", forbidden, raw)
		}
	}
	if payload.Data["notificationId"] != "00000000-0000-0000-0000-000000000701" ||
		payload.Data["type"] != string(notifications.InstagramMatch) {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestDispatcherCancelsInstagramMatchWhenPushDisabled(t *testing.T) {
	now := time.Date(2026, 7, 27, 15, 0, 0, 0, time.UTC)
	pool := instagramDispatcherPool(t)
	seedInstagramDelivery(t, pool, now, false)
	sender := &scriptedSender{}
	dispatcher := NewDispatcher(pool, sender, DispatcherOptions{
		BatchSize: 1,
		Now:       func() time.Time { return now },
	})
	processed, err := dispatcher.ProcessBatch(context.Background(), "instagram-worker")
	if err != nil {
		t.Fatal(err)
	}
	if processed != 0 || len(sender.requests) != 0 {
		t.Fatalf("processed=%d requests=%d", processed, len(sender.requests))
	}
	var status string
	if err := pool.QueryRow(context.Background(), `
		SELECT status FROM push_deliveries
	`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "cancelled" {
		t.Fatalf("status=%q", status)
	}
}

func instagramDispatcherPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, `
		CREATE TABLE bluesky_profiles(
			did TEXT PRIMARY KEY,
			display_name TEXT,
			avatar_cid TEXT
		);
		CREATE TABLE craftsky_posts(
			uri TEXT PRIMARY KEY,
			reply_root_uri TEXT,
			reply_parent_uri TEXT
		);
		CREATE TABLE actor_mutes(
			owner_did TEXT NOT NULL,
			subject_did TEXT NOT NULL,
			PRIMARY KEY(owner_did, subject_did)
		);
		CREATE TABLE atproto_blocks(
			uri TEXT PRIMARY KEY,
			blocker_did TEXT NOT NULL,
			subject_did TEXT NOT NULL
		);
	`)
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000025_instagram_migration.up.sql",
		"../../migrations/000026_system_notifications.up.sql",
		"../../migrations/000029_notification_client_owned_destination.up.sql",
		"../../migrations/000030_instagram_automatic_follows.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	return pool
}

func seedInstagramDelivery(
	t *testing.T,
	pool *pgxpool.Pool,
	now time.Time,
	pushEnabled bool,
) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			indexed_at, initial_push_evaluated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000701',
			'did:plc:viewer',
			'did:plc:synthetic-private-actor',
			'instagramMatch',
			'00000000-0000-0000-0000-000000000702',
			'everyone',true,true,'active',$1,$1,$1,$1
		)
	`, now); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO push_installations (id, device_id, platform, fcm_token)
		 VALUES ('10000000-0000-0000-0000-000000000701','instagram-device','ios','synthetic-token')`,
		`INSERT INTO push_account_subscriptions (
			id, installation_id, account_did, routing_id
		 ) VALUES (
			'20000000-0000-0000-0000-000000000701',
			'10000000-0000-0000-0000-000000000701',
			'did:plc:viewer',
			'30000000-0000-0000-0000-000000000701'
		 )`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_deliveries (
			id, notification_id, account_subscription_id, status,
			next_attempt_at, deadline_at
		) VALUES (
			'40000000-0000-0000-0000-000000000701',
			'00000000-0000-0000-0000-000000000701',
			'20000000-0000-0000-0000-000000000701',
			'pending',$1,$1::timestamptz + interval '6 hours'
		)
	`, now); err != nil {
		t.Fatal(err)
	}
	if !pushEnabled {
		if _, err := pool.Exec(ctx, `
			INSERT INTO notification_preferences (
				account_did, category, scope, push_enabled
			) VALUES (
				'did:plc:viewer','instagramMatch','everyone',false
			)
		`); err != nil {
			t.Fatal(err)
		}
	}
}
