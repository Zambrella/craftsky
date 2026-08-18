package db_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestScheduledPostGenerationMigrationUpDownUp(t *testing.T) {
	up34 := readScheduledMediaMigration(t, "../../migrations/000034_scheduled_posts.up.sql")
	up40 := readScheduledMediaMigration(t, "../../migrations/000040_scheduled_media_durability.up.sql")
	up48 := readScheduledMediaMigration(t, "../../migrations/000048_scheduled_post_owner_generation.up.sql")
	down48 := readScheduledMediaMigration(t, "../../migrations/000048_scheduled_post_owner_generation.down.sql")

	pool := testdb.WithSchema(t, scheduledMediaDurabilityPreStateDDL)
	applyScheduledMediaMigration(t, pool, "34 up", up34)
	applyScheduledMediaMigration(t, pool, "40 up", up40)
	seedPreGenerationScheduledPost(t, pool)

	applyScheduledMediaMigration(t, pool, "48 up", up48)
	assertScheduledPostGenerationSchema(t, pool, true)
	assertPreGenerationScheduleSafelyCancelled(t, pool)
	assertScheduledPostGenerationConstraint(t, pool)

	applyScheduledMediaMigration(t, pool, "48 down", down48)
	assertScheduledPostGenerationSchema(t, pool, false)

	applyScheduledMediaMigration(t, pool, "48 second up", up48)
	assertScheduledPostGenerationSchema(t, pool, true)
}

func seedPreGenerationScheduledPost(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did) VALUES('did:plc:generation-owner');
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES('did:plc:generation-owner','active',7,1,'fixture',now(),now(),now());
		INSERT INTO scheduled_posts(
			id,owner_did,operation_id,request_hash,status,scheduled_at,
			next_attempt_at,payload_bytes,payload_hash,payload_version
		) VALUES(
			'10000000-0000-4000-8000-000000000001','did:plc:generation-owner',
			'20000000-0000-4000-8000-000000000001',decode(repeat('01',32),'hex'),
			'scheduled',now(),now(),decode('01','hex'),decode(repeat('02',32),'hex'),1
		);
		INSERT INTO scheduled_post_object_attempts(
			upload_attempt_id,media_id,owner_did,owner_generation,upload_generation,
			object_key,request_fingerprint,remote_started_at,remote_deadline
		) VALUES(
			'30000000-0000-5000-8000-000000000001',
			'30000000-0000-5000-8000-000000000001',
			'did:plc:generation-owner',7,7,
			'scheduled-media/v2/7/30000000-0000-5000-8000-000000000001',
			decode(repeat('03',32),'hex'),now(),now()+interval '1 minute'
		);
		INSERT INTO scheduled_post_media(
			id,owner_did,owner_generation,upload_generation,upload_attempt_id,
			object_key,state,schedule_id,ordinal,mime_type,size_bytes,sha256,
			blob_cid,unclaimed_expires_at
		) VALUES(
			'30000000-0000-5000-8000-000000000001','did:plc:generation-owner',7,7,
			'30000000-0000-5000-8000-000000000001',
			'scheduled-media/v2/7/30000000-0000-5000-8000-000000000001',
			'ready','10000000-0000-4000-8000-000000000001',0,'image/jpeg',1,
			decode(repeat('04',32),'hex'),'bafkfixture',now()+interval '1 hour'
		);
	`); err != nil {
		t.Fatalf("seed pre-generation schedule: %v", err)
	}
}

func assertPreGenerationScheduleSafelyCancelled(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var schedules, media, cleanup int
	if err := pool.QueryRow(context.Background(), `
		SELECT (SELECT count(*)::int FROM scheduled_posts),
		       (SELECT count(*)::int FROM scheduled_post_media),
		       (SELECT count(*)::int FROM scheduled_post_cleanup_jobs
		        WHERE object_key='scheduled-media/v2/7/30000000-0000-5000-8000-000000000001')
	`).Scan(&schedules, &media, &cleanup); err != nil {
		t.Fatalf("inspect pre-generation cancellation: %v", err)
	}
	if schedules != 0 || media != 0 || cleanup != 1 {
		t.Fatalf("pre-generation schedules/media/cleanup=%d/%d/%d, want 0/0/1", schedules, media, cleanup)
	}
}

func assertScheduledPostGenerationSchema(t *testing.T, pool *pgxpool.Pool, want bool) {
	t.Helper()
	for _, table := range []string{"scheduled_posts", "scheduled_post_publication_tombstones"} {
		var exists bool
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS(
				SELECT 1 FROM information_schema.columns
				WHERE table_schema=current_schema() AND table_name=$1
				  AND column_name='owner_generation' AND is_nullable='NO'
			)
		`, table).Scan(&exists); err != nil {
			t.Fatalf("inspect %s owner_generation: %v", table, err)
		}
		if exists != want {
			t.Errorf("%s owner_generation exists=%v, want %v", table, exists, want)
		}
	}
}

func assertScheduledPostGenerationConstraint(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_posts(
			id,owner_did,owner_generation,operation_id,request_hash,status,scheduled_at,
			next_attempt_at,payload_bytes,payload_hash,payload_version
		) VALUES(
			'10000000-0000-4000-8000-000000000002','did:plc:generation-owner',7,
			'20000000-0000-4000-8000-000000000002',decode(repeat('05',32),'hex'),
			'scheduled',now(),now(),decode('01','hex'),decode(repeat('06',32),'hex'),1
		);
		INSERT INTO scheduled_post_object_attempts(
			upload_attempt_id,media_id,owner_did,owner_generation,upload_generation,
			object_key,request_fingerprint,remote_started_at,remote_deadline
		) VALUES(
			'30000000-0000-5000-8000-000000000002',
			'30000000-0000-5000-8000-000000000002','did:plc:generation-owner',8,8,
			'scheduled-media/v2/8/30000000-0000-5000-8000-000000000002',
			decode(repeat('07',32),'hex'),now(),now()+interval '1 minute'
		)
	`); err != nil {
		t.Fatalf("seed mismatched-generation media attempt: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_post_media(
			id,owner_did,owner_generation,upload_generation,upload_attempt_id,
			object_key,state,schedule_id,ordinal,mime_type,size_bytes,sha256,
			blob_cid,unclaimed_expires_at
		) VALUES(
			'30000000-0000-5000-8000-000000000002','did:plc:generation-owner',8,8,
			'30000000-0000-5000-8000-000000000002',
			'scheduled-media/v2/8/30000000-0000-5000-8000-000000000002',
			'ready','10000000-0000-4000-8000-000000000002',0,'image/jpeg',1,
			decode(repeat('08',32),'hex'),'bafkmismatch',now()+interval '1 hour'
		)
	`); err == nil {
		t.Fatal("media from generation 8 attached to generation 7 schedule")
	}
}
