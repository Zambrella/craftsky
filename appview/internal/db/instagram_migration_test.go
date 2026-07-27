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

	sql := readMigration(t, "../../migrations/000025_instagram_migration.up.sql")
	storageNames := readMigration(
		t,
		"../../migrations/000031_instagram_automatic_follow_storage_names.up.sql",
	)
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
	if _, err := pool.Exec(ctx, storageNames); err != nil {
		t.Fatalf("apply automatic-follow storage-name migration: %v", err)
	}

	for _, table := range []string{
		"instagram_verification_attempts",
		"instagram_account_links",
		"instagram_identity_claims",
		"instagram_link_conflicts",
		"instagram_webhook_work",
		"instagram_graph_imports",
		"instagram_graph_handles",
		"instagram_automatic_follow_ledger",
		"instagram_automatic_follow_sources",
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
		"instagram_automatic_follow_ledger_state_check",
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
		"instagram_automatic_follow_ledger_owner_page_idx",
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
			(id, owner_did, state, source_type,
			 following_count, created_at, updated_at)
		VALUES
			('00000000-0000-0000-0000-000000000001', 'did:plc:synthetic-owner',
			 'active', 'manual', 0, now(), now());
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

func TestInstagramAutomaticFollowStorageNamesMigrationIsReversible(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	for _, path := range []string{
		"../../migrations/000025_instagram_migration.up.sql",
		"../../migrations/000031_instagram_automatic_follow_storage_names.up.sql",
	} {
		if _, err := pool.Exec(ctx, readMigration(t, path)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}

	for _, table := range []string{
		"instagram_automatic_follow_ledger",
		"instagram_automatic_follow_sources",
	} {
		assertTableExists(t, pool, table)
	}
	if !columnExists(
		t,
		pool,
		"instagram_automatic_follow_sources",
		"automatic_follow_id",
	) {
		t.Error("renamed source reference column missing")
	}
	if !columnExists(t, pool, "pds_follow_operations", "automatic_follow_id") {
		t.Error("renamed PDS operation reference column missing")
	}

	down := readMigration(
		t,
		"../../migrations/000031_instagram_automatic_follow_storage_names.down.sql",
	)
	if _, err := pool.Exec(ctx, down); err != nil {
		t.Fatalf("roll back automatic-follow storage-name migration: %v", err)
	}
	for _, table := range []string{
		"instagram_follow_suggestions",
		"instagram_suggestion_sources",
	} {
		assertTableExists(t, pool, table)
	}
	if !columnExists(t, pool, "instagram_suggestion_sources", "suggestion_id") {
		t.Error("rolled-back source reference column missing")
	}
	if !columnExists(t, pool, "pds_follow_operations", "suggestion_id") {
		t.Error("rolled-back PDS operation reference column missing")
	}
}

func TestInstagramFollowingOnlyMigrationRemovesFollowerStorage(t *testing.T) {
	t.Parallel()

	core := readMigration(t, "../../migrations/000025_instagram_migration.up.sql")
	legacyShape := readMigration(t, "../../migrations/000027_instagram_following_only.down.sql")
	followingOnly := readMigration(t, "../../migrations/000027_instagram_following_only.up.sql")
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
			id, owner_did, state, source_type,
			following_count, follower_count, created_at, updated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000025',
			'did:plc:synthetic-owner', 'active', 'instagramJson',
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

func TestInstagramImportLifetimeMigrationRemovesRetentionStorage(t *testing.T) {
	t.Parallel()

	core := readMigration(t, "../../migrations/000025_instagram_migration.up.sql")
	lifetime := readMigration(t, "../../migrations/000028_instagram_import_lifetime.up.sql")
	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, core); err != nil {
		t.Fatalf("apply core migration: %v", err)
	}
	if _, err := pool.Exec(ctx, lifetime); err != nil {
		t.Fatalf("apply import lifetime migration: %v", err)
	}

	for table, columns := range map[string][]string{
		"instagram_graph_imports": {
			"retain_unmatched",
			"retention_expires_at",
			"final_terminal_at",
			"aggregate_purge_at",
		},
		"instagram_graph_handles": {"retain_until"},
	} {
		for _, column := range columns {
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
				t.Errorf("legacy retention storage %s.%s still exists", table, column)
			}
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_imports (
			id, owner_did, state, source_type, following_count,
			created_at, updated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000026',
			'did:plc:synthetic-owner', 'active', 'manual', 1, now(), now()
		);
		INSERT INTO instagram_graph_handles (
			import_id, username_normalized, matched, created_at
		) VALUES (
			'00000000-0000-0000-0000-000000000026',
			'synthetic.retained', false, now()
		);
	`); err != nil {
		t.Fatalf("insert permanent import: %v", err)
	}
}

func TestInstagramImportLifetimeMigrationKeepsOnlyVerifiedLinkImports(t *testing.T) {
	t.Parallel()

	core := readMigration(t, "../../migrations/000025_instagram_migration.up.sql")
	legacyLifetime := readMigration(t, "../../migrations/000028_instagram_import_lifetime.down.sql")
	lifetime := readMigration(t, "../../migrations/000028_instagram_import_lifetime.up.sql")
	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, core); err != nil {
		t.Fatalf("apply core migration: %v", err)
	}
	if _, err := pool.Exec(ctx, legacyLifetime); err != nil {
		t.Fatalf("restore legacy lifetime shape: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_account_links (
			id, owner_did, state, igsid, igsid_digest_version, igsid_digest,
			username, username_normalized, discoverable, verified_at,
			created_at, updated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000261',
			'did:plc:synthetic-verified-owner', 'active',
			'synthetic-verified-igsid', 1, decode(repeat('26', 32), 'hex'),
			'synthetic.verified', 'synthetic.verified', true, now(), now(), now()
		);
		INSERT INTO instagram_graph_imports (
			id, owner_did, state, source_type, retain_unmatched,
			retention_expires_at, following_count, created_at, updated_at
		) VALUES
			('00000000-0000-0000-0000-000000000262',
			 'did:plc:synthetic-verified-owner', 'active', 'manual', true,
			 now() + interval '1 year', 1, now(), now()),
			('00000000-0000-0000-0000-000000000263',
			 'did:plc:synthetic-unverified-owner', 'active', 'manual', true,
			 now() + interval '1 year', 1, now(), now()),
			('00000000-0000-0000-0000-000000000264',
			 'did:plc:synthetic-verified-owner', 'expired', 'manual', true,
			 now() - interval '1 day', 1, now() - interval '1 year', now());
		INSERT INTO instagram_graph_handles (
			import_id, username_normalized, matched, retain_until, created_at
		) VALUES
			('00000000-0000-0000-0000-000000000262',
			 'synthetic.verified.following', false, now() + interval '1 year', now()),
			('00000000-0000-0000-0000-000000000263',
			 'synthetic.unverified.following', false, now() + interval '1 year', now()),
			('00000000-0000-0000-0000-000000000264',
			 'synthetic.expired.following', false, now() - interval '1 day', now());
		INSERT INTO instagram_follow_suggestions (
			id, importer_did, target_did, state, reason, created_at, updated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000265',
			'did:plc:synthetic-unverified-owner',
			'did:plc:synthetic-target', 'pending',
			'verifiedInstagramFollow', now(), now()
		);
		INSERT INTO instagram_suggestion_sources (
			suggestion_id, import_id, created_at
		) VALUES (
			'00000000-0000-0000-0000-000000000265',
			'00000000-0000-0000-0000-000000000263', now()
		);
		INSERT INTO pds_follow_operations (
			id, suggestion_id, owner_did, target_did, rkey, status,
			created_at, updated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000266',
			'00000000-0000-0000-0000-000000000265',
			'did:plc:synthetic-unverified-owner',
			'did:plc:synthetic-target',
			'3kylegacyremoved', 'pending', now(), now()
		);
	`); err != nil {
		t.Fatalf("seed legacy imports: %v", err)
	}
	if _, err := pool.Exec(ctx, lifetime); err != nil {
		t.Fatalf("apply import lifetime migration: %v", err)
	}

	var importIDs, handles []string
	rows, err := pool.Query(ctx, `
		SELECT id::text
		FROM instagram_graph_imports
		ORDER BY id
	`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		importIDs = append(importIDs, id)
	}
	rows.Close()
	handleRows, err := pool.Query(ctx, `
		SELECT username_normalized
		FROM instagram_graph_handles
		ORDER BY username_normalized
	`)
	if err != nil {
		t.Fatal(err)
	}
	for handleRows.Next() {
		var username string
		if err := handleRows.Scan(&username); err != nil {
			t.Fatal(err)
		}
		handles = append(handles, username)
	}
	handleRows.Close()
	if len(importIDs) != 1 || importIDs[0] != "00000000-0000-0000-0000-000000000262" {
		t.Fatalf("retained import IDs = %v", importIDs)
	}
	if len(handles) != 1 || handles[0] != "synthetic.verified.following" {
		t.Fatalf("retained handles = %v", handles)
	}
	var suggestionState, operationStatus, operationError string
	if err := pool.QueryRow(ctx, `
		SELECT suggestion.state, operation.status, operation.last_error_code
		FROM instagram_follow_suggestions suggestion
		JOIN pds_follow_operations operation
		  ON operation.suggestion_id = suggestion.id
		WHERE suggestion.id = '00000000-0000-0000-0000-000000000265'
	`).Scan(&suggestionState, &operationStatus, &operationError); err != nil {
		t.Fatal(err)
	}
	if suggestionState != "invalidated" ||
		operationStatus != "failed" ||
		operationError != "legacyImportRemoved" {
		t.Fatalf(
			"legacy dependent state = suggestion %q operation %q error %q",
			suggestionState,
			operationStatus,
			operationError,
		)
	}
}

func TestNotificationMigrationUsesCategoryAsTheOnlyPayloadDiscriminator(t *testing.T) {
	t.Parallel()

	base := readMigration(t, "../../migrations/000021_appview_notifications.up.sql")
	newness := readMigration(t, "../../migrations/000022_notification_newness.up.sql")
	instagram := readMigration(t, "../../migrations/000025_instagram_migration.up.sql")
	system := readMigration(t, "../../migrations/000026_system_notifications.up.sql")
	clientDestination := readMigration(t, "../../migrations/000029_notification_client_owned_destination.up.sql")
	if strings.Contains(system, "ADD COLUMN kind") ||
		strings.Contains(system, "notification_events_kind_payload_check") {
		t.Fatal("notification migration still defines redundant kind storage")
	}
	if !strings.Contains(system, "notification_events_type_payload_check") {
		t.Fatal("notification migration omits type-derived payload constraint")
	}
	if !strings.Contains(clientDestination, "DROP COLUMN IF EXISTS kind") ||
		!strings.Contains(clientDestination, "DROP COLUMN IF EXISTS system_destination") {
		t.Fatal("notification migration does not remove legacy discriminators and client destinations")
	}

	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	for _, migration := range []struct {
		name string
		sql  string
	}{
		{name: "000021", sql: base},
		{name: "000022", sql: newness},
		{name: "000025", sql: instagram},
		{name: "000026", sql: system},
	} {
		if _, err := pool.Exec(ctx, migration.sql); err != nil {
			t.Fatalf("apply migration %s: %v", migration.name, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		ALTER TABLE notification_events
			DROP CONSTRAINT notification_events_type_payload_check;
		ALTER TABLE notification_events
			ADD COLUMN kind TEXT NOT NULL DEFAULT 'social';
		ALTER TABLE notification_events
			ALTER COLUMN kind DROP DEFAULT;
		ALTER TABLE notification_events
			ADD CONSTRAINT notification_events_kind_check
				CHECK (kind IN ('social','system')),
			ADD CONSTRAINT notification_events_kind_payload_check CHECK (true);
	`); err != nil {
		t.Fatalf("simulate legacy notification schema: %v", err)
	}
	if _, err := pool.Exec(ctx, clientDestination); err != nil {
		t.Fatalf("apply migration 000029 over legacy notification schema: %v", err)
	}

	var kindColumnExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'notification_events'
			  AND column_name = 'kind'
		)
	`).Scan(&kindColumnExists); err != nil {
		t.Fatalf("inspect notification discriminator columns: %v", err)
	}
	if kindColumnExists {
		t.Fatal("notification_events still stores redundant kind")
	}
	var destinationColumnExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'notification_events'
			  AND column_name = 'system_destination'
		)
	`).Scan(&destinationColumnExists); err != nil {
		t.Fatalf("inspect notification destination column: %v", err)
	}
	if destinationColumnExists {
		t.Fatal("notification_events still stores a client navigation destination")
	}
	if !constraintExists(t, pool, "notification_events_type_payload_check") {
		t.Fatal("type-derived notification payload constraint missing")
	}
	if !indexExists(t, pool, "notification_events_system_group_unique") {
		t.Fatal("system notification grouping index missing")
	}
	assertTableExists(t, pool, "instagram_notification_suggestions")

	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, category, subject_key,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			initial_push_evaluated_at, system_count, system_count_capped,
			system_group_key, coalesce_until
		) VALUES (
			'00000000-0000-0000-0000-000000000010',
			'did:plc:synthetic-recipient', 'instagramMatch',
			'instagram:2026-07-19T12:00:00Z', 'everyone', false, true,
			'active', now(), now(), now(), 3, false,
			'synthetic-group', now() + interval '5 minutes'
		)
	`); err != nil {
		t.Fatalf("insert actorless system notification: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, category, subject_key,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			initial_push_evaluated_at, system_count, system_count_capped,
			system_group_key, coalesce_until,
			actor_did
		) VALUES (
			'00000000-0000-0000-0000-000000000011',
			'did:plc:synthetic-recipient', 'instagramMatch',
			'bad', 'everyone', false, true, 'active', now(), now(), now(),
			1, false, 'bad-group', now(),
			'did:plc:synthetic-actor'
		)
	`); err == nil {
		t.Fatal("type-derived payload constraint accepted an actor on instagramMatch")
	}

	var socialRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_events WHERE category <> 'instagramMatch'`).Scan(&socialRows); err != nil {
		t.Fatalf("count social rows: %v", err)
	}
	if socialRows != 0 {
		t.Fatalf("migration backfilled unexpected social rows: %d", socialRows)
	}
}

func TestInstagramAutomaticFollowMigrationEnforcesWorkerAndActorfulNotificationShape(t *testing.T) {
	t.Parallel()

	automaticFollow := readMigration(t, "../../migrations/000030_instagram_automatic_follows.up.sql")
	for _, required := range []string{
		"pds_follow_operations_lease_shape_check",
		"pds_follow_operations_owner_target_unique",
		"notification_events_instagram_operation_unique",
		"DROP TABLE instagram_notification_suggestions",
		"DROP COLUMN system_count",
		"DROP COLUMN system_group_key",
		"category = 'instagramMatch'",
		"actor_did IS NOT NULL",
		"source_uri IS NULL",
	} {
		if !strings.Contains(automaticFollow, required) {
			t.Errorf("automatic-follow migration missing %q", required)
		}
	}

	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000025_instagram_migration.up.sql",
		"../../migrations/000026_system_notifications.up.sql",
		"../../migrations/000029_notification_client_owned_destination.up.sql",
		"../../migrations/000030_instagram_automatic_follows.up.sql",
		"../../migrations/000031_instagram_automatic_follow_storage_names.up.sql",
	} {
		if _, err := pool.Exec(ctx, readMigration(t, path)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}

	for _, column := range []string{
		"system_count",
		"system_count_capped",
		"system_group_key",
		"coalesce_until",
		"system_push_released_at",
	} {
		if columnExists(t, pool, "notification_events", column) {
			t.Errorf("obsolete actorless notification column %q still exists", column)
		}
	}
	if instagramTableExists(t, pool, "instagram_notification_suggestions") {
		t.Fatal("obsolete notification/suggestion grouping table still exists")
	}
	for _, index := range []string{
		"pds_follow_operations_claim_idx",
		"pds_follow_operations_expired_lease_idx",
		"pds_follow_operations_owner_target_unique",
		"notification_events_instagram_operation_unique",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("automatic-follow index %q missing", index)
		}
	}

	for _, legacyTable := range []string{
		"instagram_follow_suggestions",
		"instagram_suggestion_sources",
	} {
		if instagramTableExists(t, pool, legacyTable) {
			t.Errorf("legacy table %s remains after rename migration", legacyTable)
		}
	}
	if !columnExists(
		t,
		pool,
		"instagram_automatic_follow_sources",
		"automatic_follow_id",
	) {
		t.Error("automatic-follow source reference column missing")
	}
	if !columnExists(t, pool, "pds_follow_operations", "automatic_follow_id") {
		t.Error("PDS follow operation reference column missing")
	}
	for _, constraint := range []string{
		"pds_follow_operations_status_check",
		"pds_follow_operations_lease_shape_check",
		"notification_events_type_payload_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("automatic-follow constraint %q missing", constraint)
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			initial_push_evaluated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000030',
			'did:plc:synthetic-recipient',
			'did:plc:synthetic-actor',
			'instagramMatch',
			'00000000-0000-0000-0000-000000000031',
			'everyone', true, true, 'active', now(), now(), now()
		)
	`); err != nil {
		t.Fatalf("insert actorful Instagram match: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, category, subject_key,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			initial_push_evaluated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000032',
			'did:plc:synthetic-recipient',
			'instagramMatch',
			'00000000-0000-0000-0000-000000000033',
			'everyone', false, true, 'active', now(), now(), now()
		)
	`); err == nil {
		t.Fatal("actorless Instagram match passed the actorful payload constraint")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key, source_uri,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			initial_push_evaluated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000034',
			'did:plc:synthetic-recipient',
			'did:plc:synthetic-actor',
			'instagramMatch',
			'00000000-0000-0000-0000-000000000035',
			'at://did:plc:synthetic-actor/app.bsky.graph.follow/3kbadsource',
			'everyone', false, true, 'active', now(), now(), now()
		)
	`); err == nil {
		t.Fatal("Instagram match with a source record passed the payload constraint")
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

func instagramTableExists(t *testing.T, pool *pgxpool.Pool, table string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `
		SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL
	`, table).Scan(&exists); err != nil {
		t.Fatalf("lookup table %s: %v", table, err)
	}
	return exists
}

func columnExists(t *testing.T, pool *pgxpool.Pool, table, column string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema=current_schema()
			  AND table_name=$1
			  AND column_name=$2
		)
	`, table, column).Scan(&exists); err != nil {
		t.Fatalf("lookup column %s.%s: %v", table, column, err)
	}
	return exists
}

func assertTableExists(t *testing.T, pool *pgxpool.Pool, table string) {
	t.Helper()
	if !instagramTableExists(t, pool, table) {
		t.Errorf("table %s missing", table)
	}
}
