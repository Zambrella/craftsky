package push

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

func TestDispatcherSendsDueInstagramMatchWithoutSocialFacts(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 5, 1, 0, time.UTC)
	pool := instagramDispatcherPool(t)
	seedInstagramDelivery(t, pool, now.Add(-time.Second), true)
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
	if request.Category != notifications.InstagramMatch || request.ActorDisplayName != "" || request.RoutingFacts.ActorDID != "" || request.RoutingFacts.SourceURI != "" || request.RoutingFacts.SubjectURI != "" || request.RoutingFacts.RootURI != "" {
		t.Fatalf("system request contains social facts: %+v", request)
	}
	if request.RoutingFacts.NotificationID != "00000000-0000-0000-0000-000000000701" || request.RoutingFacts.SystemCount != 99 || !request.RoutingFacts.SystemCountCapped || request.RoutingFacts.SystemDestination != "instagramMigration" {
		t.Fatalf("system routing facts=%+v", request.RoutingFacts)
	}
	payload := BuildPayload(request.Category, request.AccountSubscriptionID, request.ActorDisplayName, request.RoutingFacts)
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"did:plc:", "at://", "private-instagram-handle", "igsid"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("provider payload leaked %q: %s", forbidden, raw)
		}
	}
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "succeeded" {
		t.Fatalf("delivery status=%q", status)
	}
}

func TestDispatcherCancelsInstagramMatchWhenRetractedOrPushDisabled(t *testing.T) {
	tests := []struct {
		name        string
		pushEnabled bool
		retract     bool
	}{
		{name: "push disabled", pushEnabled: false},
		{name: "event retracted", pushEnabled: true, retract: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			now := time.Date(2026, 7, 19, 12, 5, 1, 0, time.UTC)
			pool := instagramDispatcherPool(t)
			seedInstagramDelivery(t, pool, now.Add(-time.Second), test.pushEnabled)
			if test.retract {
				if _, err := pool.Exec(context.Background(), `
					UPDATE notification_events
					SET state='retracted', retracted_at=$1, retraction_reason='suggestions_invalidated'
				`, now); err != nil {
					t.Fatal(err)
				}
			}
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
			if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&status); err != nil {
				t.Fatal(err)
			}
			if status != "cancelled" {
				t.Fatalf("delivery status=%q, want cancelled", status)
			}
		})
	}
}

func TestDispatcherRechecksInstagramPreferenceAfterLease(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 5, 1, 0, time.UTC)
	pool := instagramDispatcherPool(t)
	seedInstagramDelivery(t, pool, now.Add(-time.Second), true)
	sender := &scriptedSender{}
	claimed := make(chan struct{})
	release := make(chan struct{})
	var clockCalls atomic.Int32
	dispatcher := NewDispatcher(pool, sender, DispatcherOptions{
		BatchSize: 1,
		Now: func() time.Time {
			if clockCalls.Add(1) == 2 {
				close(claimed)
				<-release
			}
			return now
		},
	})
	result := make(chan error, 1)
	go func() {
		_, err := dispatcher.ProcessBatch(context.Background(), "instagram-worker")
		result <- err
	}()
	<-claimed
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_preferences (account_did, category, scope, push_enabled)
		VALUES ('did:plc:viewer', 'instagramMatch', 'everyone', false)
	`); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if len(sender.requests) != 0 {
		t.Fatalf("preference changed after lease but provider received %d requests", len(sender.requests))
	}
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "cancelled" {
		t.Fatalf("delivery status=%q, want cancelled", status)
	}
}

func instagramDispatcherPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, `
		CREATE TABLE bluesky_profiles(did TEXT PRIMARY KEY,display_name TEXT,avatar_cid TEXT);
		CREATE TABLE craftsky_posts(uri TEXT PRIMARY KEY,reply_root_uri TEXT,reply_parent_uri TEXT);
		CREATE TABLE instagram_follow_suggestions(id UUID PRIMARY KEY);
	`)
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000024_system_notifications.up.sql",
	} {
		if _, err := pool.Exec(context.Background(), readPushMigration(t, path)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
	return pool
}

func seedInstagramDelivery(t *testing.T, pool *pgxpool.Pool, coalesceUntil time.Time, pushEnabled bool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, kind, category, subject_key,
			eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
			state, first_activity_at, activity_at, indexed_at,
			initial_push_evaluated_at, system_count, system_count_capped,
			system_destination, system_group_key, coalesce_until
		) VALUES (
			'00000000-0000-0000-0000-000000000701', 'did:plc:viewer',
			'system', 'instagramMatch', 'instagram-system', 'everyone', false, true,
			'active', $1::timestamptz - interval '5 minutes', $1, $1, $1,
			99, true, 'instagramMigration', 'instagram-group', $1
		)
	`, coalesceUntil); err != nil {
		t.Fatalf("seed Instagram event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_follow_suggestions (id)
		VALUES ('50000000-0000-0000-0000-000000000701');
		INSERT INTO instagram_notification_suggestions (notification_id, suggestion_id)
		VALUES (
			'00000000-0000-0000-0000-000000000701',
			'50000000-0000-0000-0000-000000000701'
		)
	`); err != nil {
		t.Fatalf("seed Instagram notification support: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_installations (id, device_id, platform, fcm_token)
		VALUES ('10000000-0000-0000-0000-000000000701', 'instagram-device', 'ios', 'synthetic-token')
	`); err != nil {
		t.Fatalf("seed Instagram installation: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_account_subscriptions (id, installation_id, account_did, routing_id)
		VALUES (
			'20000000-0000-0000-0000-000000000701',
			'10000000-0000-0000-0000-000000000701',
			'did:plc:viewer', '30000000-0000-0000-0000-000000000701'
		)
	`); err != nil {
		t.Fatalf("seed Instagram subscription: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_deliveries (
			id, notification_id, account_subscription_id, status,
			next_attempt_at, deadline_at
		) VALUES (
			'40000000-0000-0000-0000-000000000701',
			'00000000-0000-0000-0000-000000000701',
			'20000000-0000-0000-0000-000000000701',
			'pending', $1, $1::timestamptz + interval '6 hours'
		)
	`, coalesceUntil); err != nil {
		t.Fatalf("seed Instagram delivery: %v", err)
	}
	if !pushEnabled {
		if _, err := pool.Exec(ctx, `
			INSERT INTO notification_preferences (account_did, category, scope, push_enabled)
			VALUES ('did:plc:viewer', 'instagramMatch', 'everyone', false)
		`); err != nil {
			t.Fatalf("disable Instagram push: %v", err)
		}
	}
}

func readPushMigration(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	return string(contents)
}
