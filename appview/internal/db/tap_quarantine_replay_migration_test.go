package db_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestTapQuarantineReplayPayloadMigrationUpDownUp(t *testing.T) {
	base, err := os.ReadFile("../../migrations/000045_tap_ingestion_durability.up.sql")
	if err != nil {
		t.Fatalf("read Tap durability migration: %v", err)
	}
	up, err := os.ReadFile("../../migrations/000051_tap_quarantine_replay_payload.up.sql")
	if err != nil {
		t.Fatalf("read quarantine replay up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000051_tap_quarantine_replay_payload.down.sql")
	if err != nil {
		t.Fatalf("read quarantine replay down migration: %v", err)
	}

	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(base)); err != nil {
		t.Fatalf("apply Tap durability migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tap_quarantined_events(
			event_fingerprint,tap_event_id,event_type,reason_code,envelope,envelope_bytes
		) VALUES(
			decode(repeat('51',32),'hex'),51,'record','invalid_envelope','{}'::jsonb,2
		)
	`); err != nil {
		t.Fatalf("seed legacy quarantine: %v", err)
	}

	applyTapIngestionMigration(t, pool, "quarantine replay up", up)
	if !columnExists(t, pool, "tap_quarantined_events", "replay_envelope") {
		t.Fatal("replay_envelope missing after up migration")
	}
	var legacyReplayEnvelope []byte
	if err := pool.QueryRow(ctx, `
		SELECT replay_envelope
		FROM tap_quarantined_events
		WHERE tap_event_id=51
	`).Scan(&legacyReplayEnvelope); err != nil {
		t.Fatalf("read migrated legacy quarantine: %v", err)
	}
	if legacyReplayEnvelope != nil {
		t.Fatal("migration fabricated exact bytes for a legacy JSONB quarantine row")
	}

	maximumFrame := bytes.Repeat([]byte{0x51}, 2<<20)
	if _, err := pool.Exec(ctx, `
		UPDATE tap_quarantined_events SET replay_envelope=$2 WHERE tap_event_id=$1
	`, 51, maximumFrame); err != nil {
		t.Fatalf("store maximum Tap replay frame: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE tap_quarantined_events SET replay_envelope=$2 WHERE tap_event_id=$1
	`, 51, append(maximumFrame, 0x52)); !isCheckViolation(err) {
		t.Fatalf("oversized replay frame error=%v, want check violation", err)
	}

	applyTapIngestionMigration(t, pool, "quarantine replay down", down)
	if columnExists(t, pool, "tap_quarantined_events", "replay_envelope") {
		t.Fatal("replay_envelope remained after down migration")
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tap_quarantined_events WHERE tap_event_id=51`).Scan(&rows); err != nil {
		t.Fatalf("count quarantine rows after down: %v", err)
	}
	if rows != 1 {
		t.Fatalf("quarantine rows after down=%d, want legacy row preserved", rows)
	}

	applyTapIngestionMigration(t, pool, "quarantine replay second up", up)
	if !columnExists(t, pool, "tap_quarantined_events", "replay_envelope") {
		t.Fatal("replay_envelope missing after second up migration")
	}
}
