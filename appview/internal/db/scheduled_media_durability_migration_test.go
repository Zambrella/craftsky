package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const scheduledMediaDurabilityPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE account_deletion_operations (
    id UUID NOT NULL,
    owner_did TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (id, owner_did)
);
CREATE TABLE owner_lifecycles (
    owner_did TEXT NOT NULL PRIMARY KEY,
    state TEXT NOT NULL,
    generation BIGINT NOT NULL,
    auth_epoch BIGINT NOT NULL,
    transition_reason TEXT NOT NULL,
    transitioned_at TIMESTAMPTZ NOT NULL,
    terminal_at TIMESTAMPTZ,
    purge_completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func TestScheduledMediaDurabilityMigrationsUpDownUp(t *testing.T) {
	up34 := readScheduledMediaMigration(t, "../../migrations/000034_scheduled_posts.up.sql")
	down34 := readScheduledMediaMigration(t, "../../migrations/000034_scheduled_posts.down.sql")
	up40 := readScheduledMediaMigration(t, "../../migrations/000040_scheduled_media_durability.up.sql")
	down40 := readScheduledMediaMigration(t, "../../migrations/000040_scheduled_media_durability.down.sql")
	up41 := readScheduledMediaMigration(t, "../../migrations/000041_account_deletion_safety_tombstones.up.sql")
	down41 := readScheduledMediaMigration(t, "../../migrations/000041_account_deletion_safety_tombstones.down.sql")

	pool := testdb.WithSchema(t, scheduledMediaDurabilityPreStateDDL)
	applyScheduledMediaMigration(t, pool, "34 up", up34)
	applyScheduledMediaMigration(t, pool, "40 up", up40)
	applyScheduledMediaMigration(t, pool, "41 up", up41)
	assertScheduledMediaDurabilitySchema(t, pool)
	assertScheduledMediaDurabilityConstraints(t, pool)

	applyScheduledMediaMigration(t, pool, "41 down", down41)
	applyScheduledMediaMigration(t, pool, "40 down", down40)
	assertRelationAbsent(t, pool, "account_deletion_safety_tombstones")
	assertRelationAbsent(t, pool, "scheduled_post_object_attempts")

	applyScheduledMediaMigration(t, pool, "34 down", down34)
	applyScheduledMediaMigration(t, pool, "34 second up", up34)
	applyScheduledMediaMigration(t, pool, "40 second up", up40)
	applyScheduledMediaMigration(t, pool, "41 second up", up41)
	assertScheduledMediaDurabilitySchema(t, pool)
}

func assertScheduledMediaDurabilityConstraints(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES ('did:plc:migration-owner','deleting',3,2,'fixture',now(),now(),now());
		INSERT INTO account_deletion_operations(id,owner_did,owner_generation)
		VALUES ('10000000-0000-4000-8000-000000000001','did:plc:migration-owner',3);
	`); err != nil {
		t.Fatalf("seed safety tombstone parents: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_safety_tombstones(
			id,operation_id,owner_did,owner_generation,kind,exact_key,
			source_attempt_id,state,next_attempt_at
		) VALUES (
			'20000000-0000-4000-8000-000000000001',
			'10000000-0000-4000-8000-000000000001',
			'did:plc:migration-owner',3,'pds_record',
			'at://did:plc:migration-owner/social.craftsky.feed.post/one',
			'pds-attempt','pending',now()
		);
		INSERT INTO account_deletion_safety_tombstones(
			id,operation_id,owner_did,owner_generation,kind,exact_key,
			upload_generation,source_attempt_id,state,next_attempt_at
		) VALUES (
			'20000000-0000-4000-8000-000000000002',
			'10000000-0000-4000-8000-000000000001',
			'did:plc:migration-owner',3,'scheduled_object',
			'scheduled-media/v2/3/30000000-0000-5000-8000-000000000001',
			3,'30000000-0000-5000-8000-000000000001','pending',now()
		);
	`); err != nil {
		t.Fatalf("insert valid minimized safety tombstones: %v", err)
	}

	invalid := []string{
		`INSERT INTO account_deletion_safety_tombstones(
			id,operation_id,owner_did,owner_generation,kind,exact_key,
			source_attempt_id,state,next_attempt_at
		 ) VALUES (
			gen_random_uuid(),'10000000-0000-4000-8000-000000000001',
			'did:plc:migration-owner',3,'pds_record',
			'at://did:plc:migration-owner/app.bsky.graph.follow/one',
			'follow-attempt','pending',now()
		 )`,
		`INSERT INTO account_deletion_safety_tombstones(
			id,operation_id,owner_did,owner_generation,kind,exact_key,
			source_attempt_id,state,next_attempt_at
		 ) VALUES (
			gen_random_uuid(),'10000000-0000-4000-8000-000000000001',
			'did:plc:migration-owner',3,'pds_record',
			'at://did:plc:other/social.craftsky.feed.post/one',
			'other-owner-attempt','pending',now()
		 )`,
		`INSERT INTO account_deletion_safety_tombstones(
			id,operation_id,owner_did,owner_generation,kind,exact_key,
			upload_generation,source_attempt_id,state,next_attempt_at
		 ) VALUES (
			gen_random_uuid(),'10000000-0000-4000-8000-000000000001',
			'did:plc:migration-owner',3,'scheduled_object',
			'scheduled-media/v2/4/30000000-0000-5000-8000-000000000002',
			3,'wrong-generation','pending',now()
		 )`,
		`INSERT INTO account_deletion_safety_tombstones(
			id,operation_id,owner_did,owner_generation,kind,exact_key,
			upload_generation,source_attempt_id,state,next_attempt_at
		 ) VALUES (
			gen_random_uuid(),'10000000-0000-4000-8000-000000000001',
			'did:plc:migration-owner',3,'scheduled_object',
			'scheduled-media/v2/4/30000000-0000-5000-8000-000000000003',
			4,'30000000-0000-5000-8000-000000000003','pending',now()
		 )`,
		`INSERT INTO account_deletion_safety_tombstones(
			id,operation_id,owner_did,owner_generation,kind,exact_key,
			upload_generation,source_attempt_id,state,next_attempt_at
		 ) VALUES (
			gen_random_uuid(),'10000000-0000-4000-8000-000000000001',
			'did:plc:migration-owner',3,'scheduled_object',
			'scheduled-media/v2/3/30000000-0000-5000-8000-000000000004',
			3,'different-attempt','pending',now()
		 )`,
		`INSERT INTO account_deletion_safety_tombstones(
			id,operation_id,owner_did,owner_generation,kind,exact_key,
			source_attempt_id,state,next_attempt_at,lease_token
		 ) VALUES (
			gen_random_uuid(),'10000000-0000-4000-8000-000000000001',
			'did:plc:migration-owner',3,'pds_record',
			'at://did:plc:migration-owner/social.craftsky.feed.like/two',
			'bad-lease','pending',now(),gen_random_uuid()
		 )`,
	}
	for _, statement := range invalid {
		if _, err := pool.Exec(ctx, statement); err == nil {
			t.Errorf("invalid safety tombstone insert succeeded: %s", statement)
		}
	}

	for _, forbiddenColumn := range []string{
		"handle", "token", "record_body", "media_content", "status_capability", "audit",
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM information_schema.columns
				WHERE table_schema=current_schema()
				  AND table_name='account_deletion_safety_tombstones'
				  AND column_name=$1
			)
		`, forbiddenColumn).Scan(&exists); err != nil {
			t.Fatalf("inspect forbidden column %s: %v", forbiddenColumn, err)
		}
		if exists {
			t.Errorf("forbidden safety tombstone column %s exists", forbiddenColumn)
		}
	}
}

func readScheduledMediaMigration(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return contents
}

func applyScheduledMediaMigration(
	t *testing.T,
	pool *pgxpool.Pool,
	label string,
	sql []byte,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
		t.Fatalf("apply %s: %v", label, err)
	}
}

func assertScheduledMediaDurabilitySchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, column := range []string{
		"owner_generation", "upload_generation", "upload_attempt_id",
	} {
		assertColumnExists(t, pool, "scheduled_post_media", column)
	}
	for _, relation := range []string{
		"scheduled_post_object_attempts",
		"account_deletion_safety_tombstones",
	} {
		var exists bool
		if err := pool.QueryRow(
			context.Background(),
			`SELECT to_regclass($1) IS NOT NULL`,
			relation,
		).Scan(&exists); err != nil {
			t.Fatalf("inspect relation %s: %v", relation, err)
		}
		if !exists {
			t.Errorf("relation %s does not exist", relation)
		}
	}
	for _, index := range []string{
		"scheduled_post_object_attempts_unresolved_idx",
		"scheduled_post_cleanup_jobs_claim_idx",
		"account_deletion_safety_tombstones_claim_idx",
		"account_deletion_safety_tombstones_operation_idx",
	} {
		assertIndexExists(t, pool, index)
	}
}
