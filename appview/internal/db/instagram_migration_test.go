package db_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestInstagramMigrationCreatesPrivateSchemaWithoutMembershipCascades(t *testing.T) {
	t.Parallel()

	sql := readMigration(t, "../../migrations/000023_instagram_migration.up.sql")
	for _, forbidden := range []string{
		"REFERENCES craftsky_profiles",
		"raw_body",
		"message_text",
		"plaintext_challenge",
		"signature_header",
		"profile_response",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("core migration contains forbidden private/lifecycle seam %q", forbidden)
		}
	}

	pool := testdb.WithSchema(t, `CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);`)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatalf("apply core migration: %v", err)
	}

	for _, table := range []string{
		"instagram_verification_attempts",
		"instagram_account_links",
		"instagram_identity_claims",
		"instagram_link_conflicts",
		"instagram_webhook_work",
		"instagram_graph_imports",
		"instagram_graph_handles",
		"instagram_follow_suggestions",
		"instagram_suggestion_sources",
		"instagram_reconciliation_jobs",
		"pds_follow_operations",
		"instagram_rate_limit_buckets",
		"instagram_audit_events",
	} {
		assertTableExists(t, pool, table)
	}

	for _, constraint := range []string{
		"instagram_verification_attempts_state_check",
		"instagram_account_links_state_check",
		"instagram_link_conflicts_state_check",
		"instagram_webhook_work_status_check",
		"instagram_graph_imports_state_check",
		"instagram_follow_suggestions_state_check",
		"instagram_reconciliation_jobs_status_check",
		"pds_follow_operations_status_check",
		"instagram_rate_limit_buckets_count_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}

	for _, index := range []string{
		"instagram_verification_attempts_owner_active_unique",
		"instagram_verification_attempts_challenge_unique",
		"instagram_account_links_owner_current_unique",
		"instagram_identity_claims_active_igsid_unique",
		"instagram_webhook_work_claim_idx",
		"instagram_webhook_work_attempt_unique",
		"instagram_graph_imports_owner_page_idx",
		"instagram_graph_handles_match_idx",
		"instagram_follow_suggestions_owner_page_idx",
		"instagram_reconciliation_jobs_claim_idx",
		"pds_follow_operations_owner_rkey_unique",
		"instagram_rate_limit_buckets_expiry_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}

	var attemptReferenceDeleteAction string
	if err := pool.QueryRow(ctx, `
		SELECT c.confdeltype::text
		FROM pg_constraint c
		JOIN pg_attribute a
		  ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
		WHERE c.conrelid = 'instagram_webhook_work'::regclass
		  AND c.contype = 'f'
		  AND a.attname = 'verification_attempt_id'
	`).Scan(&attemptReferenceDeleteAction); err != nil {
		t.Fatalf("lookup webhook attempt mapping foreign key: %v", err)
	}
	if attemptReferenceDeleteAction != "a" {
		t.Fatalf("webhook attempt mapping delete action = %q, want restrict/no action", attemptReferenceDeleteAction)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:synthetic-owner');
		INSERT INTO instagram_graph_imports
			(id, owner_did, state, source_type, retain_unmatched,
			 following_count, created_at, updated_at)
		VALUES
			('00000000-0000-0000-0000-000000000001', 'did:plc:synthetic-owner',
			 'active', 'manual', false, 0, now(), now());
		DELETE FROM craftsky_profiles WHERE did = 'did:plc:synthetic-owner';
	`); err != nil {
		t.Fatalf("membership loss with private owner row: %v", err)
	}
	var imports int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_graph_imports`).Scan(&imports); err != nil {
		t.Fatalf("count private imports: %v", err)
	}
	if imports != 1 {
		t.Fatalf("membership deletion cascaded to %d imports, want private row retained", imports)
	}
}

func TestInstagramFollowingOnlyMigrationRemovesFollowerStorage(t *testing.T) {
	t.Parallel()

	core := readMigration(t, "../../migrations/000023_instagram_migration.up.sql")
	legacyShape := readMigration(t, "../../migrations/000025_instagram_following_only.down.sql")
	followingOnly := readMigration(t, "../../migrations/000025_instagram_following_only.up.sql")
	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, core); err != nil {
		t.Fatalf("apply core migration: %v", err)
	}
	if _, err := pool.Exec(ctx, legacyShape); err != nil {
		t.Fatalf("restore legacy directional shape: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_imports (
			id, owner_did, state, source_type, retain_unmatched,
			following_count, follower_count, created_at, updated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000025',
			'did:plc:synthetic-owner', 'active', 'instagramJson', true,
			1, 1, now(), now()
		);
		INSERT INTO instagram_graph_handles (
			import_id, username_normalized, direction, matched, created_at
		) VALUES
			('00000000-0000-0000-0000-000000000025', 'synthetic.same', 'following', false, now()),
			('00000000-0000-0000-0000-000000000025', 'synthetic.same', 'follower', false, now());
	`); err != nil {
		t.Fatalf("seed legacy directional import: %v", err)
	}
	if _, err := pool.Exec(ctx, followingOnly); err != nil {
		t.Fatalf("apply following-only migration: %v", err)
	}

	for table, column := range map[string]string{
		"instagram_graph_imports": "follower_count",
		"instagram_graph_handles": "direction",
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = current_schema()
				  AND table_name = $1 AND column_name = $2
			)
		`, table, column).Scan(&exists); err != nil {
			t.Fatalf("inspect %s.%s: %v", table, column, err)
		}
		if exists {
			t.Errorf("legacy follower storage %s.%s still exists", table, column)
		}
	}

	var retainedHandles int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM instagram_graph_handles
		WHERE import_id = '00000000-0000-0000-0000-000000000025'
	`).Scan(&retainedHandles); err != nil {
		t.Fatalf("count migrated handles: %v", err)
	}
	if retainedHandles != 1 {
		t.Fatalf("migrated handles = %d, want only the following entry", retainedHandles)
	}

	freshPool := testdb.WithSchema(t, "")
	if _, err := freshPool.Exec(ctx, core); err != nil {
		t.Fatalf("apply fresh core migration: %v", err)
	}
	if _, err := freshPool.Exec(ctx, followingOnly); err != nil {
		t.Fatalf("apply following-only migration to fresh schema: %v", err)
	}
}

func TestSystemNotificationMigrationCreatesCheckedActorlessUnion(t *testing.T) {
	t.Parallel()

	base := readMigration(t, "../../migrations/000021_appview_notifications.up.sql")
	newness := readMigration(t, "../../migrations/000022_notification_newness.up.sql")
	instagram := readMigration(t, "../../migrations/000023_instagram_migration.up.sql")
	system := readMigration(t, "../../migrations/000024_system_notifications.up.sql")

	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	for _, migration := range []struct {
		name string
		sql  string
	}{
		{name: "000021", sql: base},
		{name: "000022", sql: newness},
		{name: "000023", sql: instagram},
		{name: "000024", sql: system},
	} {
		if _, err := pool.Exec(ctx, migration.sql); err != nil {
			t.Fatalf("apply migration %s: %v", migration.name, err)
		}
	}

	if !constraintExists(t, pool, "notification_events_kind_payload_check") {
		t.Fatal("checked notification union constraint missing")
	}
	if !indexExists(t, pool, "notification_events_system_group_unique") {
		t.Fatal("system notification grouping index missing")
	}
	assertTableExists(t, pool, "instagram_notification_suggestions")

	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, kind, category, subject_key,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			initial_push_evaluated_at, system_count, system_count_capped,
			system_destination, system_group_key, coalesce_until
		) VALUES (
			'00000000-0000-0000-0000-000000000010',
			'did:plc:synthetic-recipient', 'system', 'instagramMatch',
			'instagram:2026-07-19T12:00:00Z', 'everyone', false, true,
			'active', now(), now(), now(), 3, false,
			'instagramMigration', 'synthetic-group', now() + interval '5 minutes'
		)
	`); err != nil {
		t.Fatalf("insert actorless system notification: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, kind, category, subject_key,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			initial_push_evaluated_at, system_count, system_count_capped,
			system_destination, system_group_key, coalesce_until,
			actor_did
		) VALUES (
			'00000000-0000-0000-0000-000000000011',
			'did:plc:synthetic-recipient', 'system', 'instagramMatch',
			'bad', 'everyone', false, true, 'active', now(), now(), now(),
			1, false, 'instagramMigration', 'bad-group', now(),
			'did:plc:synthetic-actor'
		)
	`); err == nil {
		t.Fatal("checked union accepted an actor on a system notification")
	}

	var socialRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_events WHERE kind = 'social'`).Scan(&socialRows); err != nil {
		t.Fatalf("count social rows: %v", err)
	}
	if socialRows != 0 {
		t.Fatalf("migration backfilled unexpected social rows: %d", socialRows)
	}
}

func readMigration(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	return string(contents)
}

func assertTableExists(t *testing.T, pool *pgxpool.Pool, table string) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
		t.Fatalf("lookup table %s: %v", table, err)
	}
	if !exists {
		t.Errorf("table %s missing", table)
	}
}
