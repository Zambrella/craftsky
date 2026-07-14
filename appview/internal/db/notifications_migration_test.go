package db_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

const notificationMigrationPreStateDDL = `
CREATE TABLE craftsky_likes (
    uri TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    subject_uri TEXT NOT NULL
);
INSERT INTO craftsky_likes (uri, did, subject_uri)
VALUES ('at://did:plc:actor/social.craftsky.feed.like/old', 'did:plc:actor', 'at://did:plc:recipient/social.craftsky.feed.post/old');
`

func TestNotificationsMigrationCreatesPrivateDurableSchemaWithoutBackfill(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	for _, forbidden := range []string{"notification_blocks", "notification_mutes"} {
		if strings.Contains(string(sql), forbidden) {
			t.Fatalf("migration introduced out-of-scope %s", forbidden)
		}
	}

	pool := testdb.WithSchema(t, notificationMigrationPreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	for _, table := range []string{
		"notification_events",
		"notification_preferences",
		"push_installations",
		"push_account_subscriptions",
		"push_deliveries",
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("lookup table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s missing", table)
		}
	}

	for _, constraint := range []string{
		"notification_events_recipient_actor_category_subject_key_key",
		"notification_events_category_check",
		"notification_events_state_check",
		"notification_preferences_pkey",
		"notification_preferences_scope_check",
		"push_installations_platform_check",
		"push_account_subscriptions_routing_id_key",
		"push_deliveries_notification_id_account_subscription_id_key",
		"push_deliveries_status_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}

	for _, index := range []string{
		"notification_events_active_feed_idx",
		"notification_events_source_uri_idx",
		"notification_events_actor_did_idx",
		"push_installations_active_token_unique",
		"push_account_subscriptions_active_account_idx",
		"push_deliveries_claim_idx",
		"push_deliveries_expired_lease_idx",
		"push_deliveries_queue_age_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}

	for _, table := range []string{"notification_events", "push_deliveries"} {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s backfilled %d rows, want 0", table, count)
		}
	}
}

func TestNotificationNewnessMigrationAddsAccountRevisionState(t *testing.T) {
	base, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatalf("read base migration: %v", err)
	}
	newness, err := os.ReadFile("../../migrations/000022_notification_newness.up.sql")
	if err != nil {
		t.Fatalf("read newness migration: %v", err)
	}

	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(base)); err != nil {
		t.Fatalf("apply base migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(newness)); err != nil {
		t.Fatalf("apply newness migration: %v", err)
	}

	var columnNotNull, tableExists, sequenceExists bool
	if err := pool.QueryRow(ctx, `
		SELECT attnotnull
		FROM pg_attribute
		WHERE attrelid = 'notification_events'::regclass
		  AND attname = 'newness_revision'
	`).Scan(&columnNotNull); err != nil {
		t.Fatalf("lookup revision column: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.notification_seen_state') IS NOT NULL`).Scan(&tableExists); err != nil {
		t.Fatalf("lookup seen table: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.notification_newness_revision_seq') IS NOT NULL`).Scan(&sequenceExists); err != nil {
		t.Fatalf("lookup revision sequence: %v", err)
	}
	if !columnNotNull || !tableExists || !sequenceExists {
		t.Fatalf("newness schema columnNotNull=%v table=%v sequence=%v", columnNotNull, tableExists, sequenceExists)
	}
	if !indexExists(t, pool, "notification_events_active_newness_idx") {
		t.Fatal("notification_events_active_newness_idx missing")
	}
	if !indexExists(t, pool, "notification_events_recipient_newness_idx") {
		t.Fatal("notification_events_recipient_newness_idx missing")
	}
}
