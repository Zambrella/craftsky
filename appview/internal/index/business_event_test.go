package index_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestBusinessEventProjectionConvergesByRevision(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000062_business_records.up.sql")
	if err != nil {
		t.Fatalf("read business records migration: %v", err)
	}
	pool := testdb.WithSchema(t, string(migration))
	projector := index.NewCraftskyBusinessEvent()
	base := tap.Event{
		URI:        syntax.ATURI("at://did:plc:alice/social.craftsky.business.event/3meventrecord"),
		DID:        syntax.DID("did:plc:alice"),
		Collection: syntax.NSID("social.craftsky.business.event"),
		Rkey:       syntax.RecordKey("3meventrecord"),
	}
	r1 := businessEventRecord("First", "2026-09-10T10:00:00Z", map[string]any{"version": 1})
	r2 := businessEventRecord("Updated", "2026-09-11T10:00:00Z", map[string]any{"version": 2})
	r4 := businessEventRecord("Recreated", "2026-09-12T10:00:00Z", map[string]any{"version": 4, "future": true})

	event := base
	event.Action, event.CID, event.Rev, event.Record = "create", syntax.CID("bafyreieventv1"), syntax.TID("3meventrev001"), r1
	projectBusinessEvent(t, pool, projector, event)
	assertBusinessEventState(t, pool, event.URI, "3meventrev001", "", r1)
	projectBusinessEvent(t, pool, projector, event)
	assertBusinessEventState(t, pool, event.URI, "3meventrev001", "", r1)

	event.Action, event.CID, event.Rev, event.Record = "update", syntax.CID("bafyreieventv2"), syntax.TID("3meventrev002"), r2
	projectBusinessEvent(t, pool, projector, event)
	assertBusinessEventState(t, pool, event.URI, "3meventrev002", "", r2)

	event.Action, event.CID, event.Rev, event.Record = "update", syntax.CID("bafyreistalev1"), syntax.TID("3meventrev001"), r1
	projectBusinessEvent(t, pool, projector, event)
	assertBusinessEventState(t, pool, event.URI, "3meventrev002", "", r2)

	event.Action, event.CID, event.Rev, event.Record = "delete", "", syntax.TID("3meventrev003"), nil
	projectBusinessEvent(t, pool, projector, event)
	assertBusinessEventState(t, pool, event.URI, "", "3meventrev003", nil)

	event.Action, event.CID, event.Rev, event.Record = "update", syntax.CID("bafyreistalev2"), syntax.TID("3meventrev002"), r2
	projectBusinessEvent(t, pool, projector, event)
	assertBusinessEventState(t, pool, event.URI, "", "3meventrev003", nil)

	event.Action, event.CID, event.Rev, event.Record = "create", syntax.CID("bafyreieventv4"), syntax.TID("3meventrev004"), r4
	projectBusinessEvent(t, pool, projector, event)
	assertBusinessEventState(t, pool, event.URI, "3meventrev004", "", r4)
}

func businessEventRecord(name, startsAt string, extension map[string]any) json.RawMessage {
	record := map[string]any{
		"$type":           "social.craftsky.business.event",
		"name":            name,
		"startsAt":        startsAt,
		"endsAt":          "2026-09-13T10:00:00Z",
		"roles":           []string{"vendor"},
		"createdAt":       "2026-09-01T00:00:00Z",
		"futureExtension": extension,
	}
	encoded, _ := json.Marshal(record)
	return encoded
}

func projectBusinessEvent(t *testing.T, pool *pgxpool.Pool, projector *index.CraftskyBusinessEvent, event tap.Event) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin event projection: %v", err)
	}
	outcome, err := projector.Project(ctx, tx, event)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("project event %s at %s: %v", event.Action, event.Rev, err)
	}
	if outcome.Kind != tap.OutcomeApplied {
		_ = tx.Rollback(ctx)
		t.Fatalf("event outcome = %+v", outcome)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit event projection: %v", err)
	}
}

func assertBusinessEventState(t *testing.T, pool *pgxpool.Pool, uri syntax.ATURI, wantRowRevision, wantTombstoneRevision string, wantRaw json.RawMessage) {
	t.Helper()
	ctx := context.Background()
	var rowRevision string
	var raw json.RawMessage
	err := pool.QueryRow(ctx, `SELECT source_revision, raw_record FROM craftsky_business_events WHERE uri=$1`, uri).Scan(&rowRevision, &raw)
	if wantRowRevision == "" {
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("event row error = %v, want no row", err)
		}
	} else {
		if err != nil || rowRevision != wantRowRevision {
			t.Fatalf("event row revision = %q, error %v, want %q", rowRevision, err, wantRowRevision)
		}
		assertJSONEquivalent(t, raw, wantRaw)
	}
	var tombstoneRevision string
	err = pool.QueryRow(ctx, `SELECT source_revision FROM craftsky_business_record_tombstones WHERE uri=$1`, uri).Scan(&tombstoneRevision)
	if wantTombstoneRevision == "" {
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("event tombstone error = %v, want no row", err)
		}
	} else if err != nil || tombstoneRevision != wantTombstoneRevision {
		t.Fatalf("event tombstone revision = %q, error %v, want %q", tombstoneRevision, err, wantTombstoneRevision)
	}
}
