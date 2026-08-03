package db_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

const scheduledPostsMigrationPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did        TEXT NOT NULL PRIMARY KEY,
    record_cid TEXT NOT NULL
);
CREATE TABLE migration_sentinel (
    id INTEGER NOT NULL PRIMARY KEY
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
INSERT INTO migration_sentinel (id) VALUES (1);
`

func TestScheduledPostsMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000034_scheduled_posts.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000034_scheduled_posts.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, scheduledPostsMigrationPreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}

	for _, table := range []string{
		"scheduled_posts",
		"scheduled_post_media",
		"scheduled_post_publication_tombstones",
		"scheduled_post_cleanup_jobs",
	} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, constraint := range []string{
		"scheduled_posts_pkey",
		"scheduled_posts_owner_did_fkey",
		"scheduled_posts_owner_id_key",
		"scheduled_posts_owner_operation_key",
		"scheduled_posts_status_check",
		"scheduled_posts_lease_shape_check",
		"scheduled_posts_publication_identity_check",
		"scheduled_posts_publication_record_check",
		"scheduled_posts_needs_attention_shape_check",
		"scheduled_post_media_pkey",
		"scheduled_post_media_owner_id_key",
		"scheduled_post_media_schedule_owner_fkey",
		"scheduled_post_media_state_check",
		"scheduled_post_media_attachment_check",
		"scheduled_post_media_lifecycle_check",
		"scheduled_post_media_schedule_ordinal_key",
		"scheduled_post_publication_tombstones_pkey",
		"scheduled_post_publication_tombstones_owner_operation_key",
		"scheduled_post_cleanup_jobs_pkey",
		"scheduled_post_cleanup_jobs_object_key_key",
		"scheduled_post_cleanup_jobs_state_check",
		"scheduled_post_cleanup_jobs_lease_shape_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	for _, index := range []string{
		"scheduled_posts_due_claim_idx",
		"scheduled_posts_expired_lease_idx",
		"scheduled_posts_owner_scheduled_idx",
		"scheduled_posts_needs_attention_expiry_idx",
		"scheduled_posts_owner_publication_rkey_unique",
		"scheduled_post_media_unclaimed_cleanup_idx",
		"scheduled_post_publication_tombstones_expiry_idx",
		"scheduled_post_cleanup_jobs_pending_idx",
		"scheduled_post_cleanup_jobs_expired_lease_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_posts (
			id, owner_did, operation_id, request_hash, status,
			scheduled_at, next_attempt_at, payload_bytes, payload_hash
		) VALUES (
			'00000000-0000-4000-8000-000000000001',
			'did:plc:alice',
			'00000000-0000-4000-8000-000000000011',
			decode(repeat('01', 32), 'hex'),
			'scheduled',
			'2026-08-01T12:00:00Z',
			'2026-08-01T12:00:00Z',
			convert_to('{"text":"private"}', 'UTF8'),
			decode(repeat('02', 32), 'hex')
		)
	`); err != nil {
		t.Fatalf("insert valid scheduled post: %v", err)
	}

	t.Run("rejects invalid status", func(t *testing.T) {
		_, err := pool.Exec(ctx, `
			INSERT INTO scheduled_posts (
				id, owner_did, operation_id, request_hash, status,
				scheduled_at, next_attempt_at, payload_bytes, payload_hash
			) VALUES (
				'00000000-0000-4000-8000-000000000002', 'did:plc:alice',
				'00000000-0000-4000-8000-000000000012', decode(repeat('01', 32), 'hex'),
				'published', '2026-08-01T12:00:00Z', '2026-08-01T12:00:00Z',
				convert_to('{}', 'UTF8'), decode(repeat('02', 32), 'hex')
			)
		`)
		if err == nil {
			t.Fatal("invalid active status was accepted")
		}
	})

	t.Run("rejects Publishing without lease and stable identity", func(t *testing.T) {
		_, err := pool.Exec(ctx, `
			INSERT INTO scheduled_posts (
				id, owner_did, operation_id, request_hash, status,
				scheduled_at, next_attempt_at, payload_bytes, payload_hash
			) VALUES (
				'00000000-0000-4000-8000-000000000003', 'did:plc:alice',
				'00000000-0000-4000-8000-000000000013', decode(repeat('01', 32), 'hex'),
				'publishing', '2026-08-01T12:00:00Z', '2026-08-01T12:00:00Z',
				convert_to('{}', 'UTF8'), decode(repeat('02', 32), 'hex')
			)
		`)
		if err == nil {
			t.Fatal("Publishing without its lease/identity was accepted")
		}
	})

	t.Run("rejects incomplete and cross-owner media attachments", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO scheduled_post_media (
				id, owner_did, object_key, state, schedule_id, mime_type,
				size_bytes, sha256, blob_cid, unclaimed_expires_at
			) VALUES (
				'00000000-0000-4000-8000-000000000021', 'did:plc:alice',
				'scheduled-media/one', 'ready', '00000000-0000-4000-8000-000000000001',
				'image/jpeg', 4, decode(repeat('03', 32), 'hex'), 'bafk-one',
				'2026-08-01T12:00:00Z'
			)
		`); err == nil {
			t.Fatal("attachment without ordinal was accepted")
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO scheduled_post_media (
				id, owner_did, object_key, state, schedule_id, ordinal, mime_type,
				size_bytes, sha256, blob_cid, unclaimed_expires_at
			) VALUES (
				'00000000-0000-4000-8000-000000000022', 'did:plc:bob',
				'scheduled-media/two', 'ready', '00000000-0000-4000-8000-000000000001', 0,
				'image/jpeg', 4, decode(repeat('03', 32), 'hex'), 'bafk-two',
				'2026-08-01T12:00:00Z'
			)
		`); err == nil {
			t.Fatal("cross-owner media attachment was accepted")
		}
	})

	t.Run("requires predicted CID before media is ready or attached", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO scheduled_post_media (
				id, owner_did, object_key, state, mime_type,
				size_bytes, sha256, unclaimed_expires_at
			) VALUES (
				'00000000-0000-4000-8000-000000000023', 'did:plc:alice',
				'scheduled-media/uploading', 'uploading', 'image/jpeg',
				4, decode(repeat('03', 32), 'hex'), '2026-08-01T12:00:00Z'
			)
		`); err != nil {
			t.Fatalf("resumable uploading metadata was rejected: %v", err)
		}
		for _, testCase := range []struct {
			name      string
			id        string
			objectKey string
			blobCID   *string
			ordinal   int
		}{
			{name: "null CID", id: "00000000-0000-4000-8000-000000000024", objectKey: "scheduled-media/null", ordinal: 0},
			{name: "empty CID", id: "00000000-0000-4000-8000-000000000025", objectKey: "scheduled-media/empty", blobCID: new(string), ordinal: 1},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				_, err := pool.Exec(ctx, `
					INSERT INTO scheduled_post_media (
						id, owner_did, object_key, state, schedule_id, ordinal,
						mime_type, size_bytes, sha256, blob_cid, unclaimed_expires_at
					) VALUES (
						$1, 'did:plc:alice', $2, 'ready',
						'00000000-0000-4000-8000-000000000001', $4,
						'image/jpeg', 4, decode(repeat('03', 32), 'hex'), $3,
						'2026-08-01T12:00:00Z'
					)
				`, testCase.id, testCase.objectKey, testCase.blobCID, testCase.ordinal)
				if err == nil {
					t.Fatal("ready attached media without a predicted CID was accepted")
				}
			})
		}
	})

	t.Run("tombstones contain no private content columns", func(t *testing.T) {
		for _, column := range []string{"payload_bytes", "preview", "alt", "facets", "project", "object_key", "media_id"} {
			var exists bool
			if err := pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_schema = current_schema()
					  AND table_name = 'scheduled_post_publication_tombstones'
					  AND column_name = $1
				)
			`, column).Scan(&exists); err != nil {
				t.Fatalf("inspect tombstone column %s: %v", column, err)
			}
			if exists {
				t.Errorf("private tombstone column %s exists", column)
			}
		}
	})

	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}
	for _, table := range []string{
		"scheduled_posts",
		"scheduled_post_media",
		"scheduled_post_publication_tombstones",
		"scheduled_post_cleanup_jobs",
	} {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	var sentinel int
	if err := pool.QueryRow(ctx, `SELECT id FROM migration_sentinel`).Scan(&sentinel); err != nil || sentinel != 1 {
		t.Fatalf("down migration changed unrelated sentinel data: id=%d err=%v", sentinel, err)
	}
}
