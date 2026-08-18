package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestTapIngestionDurabilityMigrationUpDownUp(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000045_tap_ingestion_durability.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000045_tap_ingestion_durability.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, "")
	applyTapIngestionMigration(t, pool, "up", up)
	assertTapIngestionSchema(t, pool)
	assertTapIngestionConstraints(t, pool)

	applyTapIngestionMigration(t, pool, "down", down)
	for _, table := range []string{
		"tap_repository_jobs",
		"tap_quarantined_events",
		"tap_projection_jobs",
		"tap_source_records",
		"tap_ingestion_receipts",
	} {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}

	applyTapIngestionMigration(t, pool, "second up", up)
	assertTapIngestionSchema(t, pool)
}

func applyTapIngestionMigration(t *testing.T, pool *pgxpool.Pool, label string, sql []byte) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
		t.Fatalf("apply %s migration: %v", label, err)
	}
}

func assertTapIngestionSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range []string{
		"tap_ingestion_receipts",
		"tap_source_records",
		"tap_projection_jobs",
		"tap_quarantined_events",
		"tap_repository_jobs",
	} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, index := range []string{
		"tap_ingestion_receipts_event_id_idx",
		"tap_source_records_owner_idx",
		"tap_source_records_projection_idx",
		"tap_projection_jobs_claim_idx",
		"tap_projection_jobs_dependency_idx",
		"tap_quarantined_events_claim_idx",
		"tap_repository_jobs_claim_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}
}

func assertTapIngestionConstraints(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO tap_source_records(
			uri,did,collection,rkey,source_event_id,source_fingerprint,
			revision,cid,action,record,record_bytes,live,ordering_status,projection_disposition
		) VALUES (
			'at://did:plc:alice/social.craftsky.feed.post/one','did:plc:alice',
			'social.craftsky.feed.post','one',1,decode(repeat('11',32),'hex'),
			'3mabc','bafy-one','create','{}'::jsonb,2,true,'authoritative','eligible'
		);
		INSERT INTO tap_projection_jobs(
			source_uri,projection_kind,source_event_id,state,next_attempt_at
		) VALUES (
			'at://did:plc:alice/social.craftsky.feed.post/one','craftsky_post',1,
			'pending',now()
		);
	`); err != nil {
		t.Fatalf("insert valid source/job: %v", err)
	}

	invalid := []string{
		`INSERT INTO tap_source_records(
			uri,did,collection,rkey,source_event_id,source_fingerprint,
			revision,cid,action,record,record_bytes,live,ordering_status,projection_disposition
		 ) VALUES (
			'at://did:plc:alice/social.craftsky.feed.post/bad','did:plc:alice',
			'social.craftsky.feed.post','bad',2,decode('00','hex'),'3mabd',NULL,
			'tombstone','{}'::jsonb,2,true,'authoritative','eligible'
		 )`,
		`INSERT INTO tap_source_records(
			uri,did,collection,rkey,source_event_id,source_fingerprint,
			revision,cid,action,record,record_bytes,live,ordering_status,projection_disposition
		 ) VALUES (
			'at://did:plc:alice/social.craftsky.feed.post/bad-size','did:plc:alice',
			'social.craftsky.feed.post','bad-size',3,decode(repeat('33',32),'hex'),
			'3mabe','bafy-bad-size','create','{}'::jsonb,0,true,'authoritative','eligible'
		 )`,
		`INSERT INTO tap_projection_jobs(
			source_uri,projection_kind,source_event_id,state,dependency_kind,
			dependency_key,next_attempt_at
		 ) VALUES (
			'at://did:plc:alice/social.craftsky.feed.post/one','other',1,
			'blocked',NULL,NULL,now()
		 )`,
		`INSERT INTO tap_quarantined_events(
			event_fingerprint,tap_event_id,event_type,reason_code,envelope,
			envelope_bytes,replay_state
		 ) VALUES (
			decode(repeat('22',32),'hex'),9,'record','raw_remote_error',
			'{}'::jsonb,2,'quarantined'
		 )`,
	}
	for _, statement := range invalid {
		if _, err := pool.Exec(ctx, statement); err == nil {
			t.Errorf("invalid constrained insert succeeded: %s", statement)
		}
	}
}
