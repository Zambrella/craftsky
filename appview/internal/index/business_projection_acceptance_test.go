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

	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestBusinessProjectionPreservesFederatedSourceAndConverges(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000062_business_records.up.sql")
	if err != nil {
		t.Fatalf("read business records migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY, state TEXT NOT NULL, generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL, transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL, terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
	`+string(migration))
	dispatcher := index.NewTransactionalDispatcher()
	dispatcher.Register("social.craftsky.business.profile", index.NewCraftskyBusinessProfile())
	dispatcher.Register("social.craftsky.business.event", index.NewCraftskyBusinessEvent())

	profileR1 := json.RawMessage(`{"tagline":"First"}`)
	profileR2 := json.RawMessage(`{
		"tagline":"Newest","businessTypes":["future-business-type","dyer"],
		"primaryAction":{"type":"shop","destination":"http://unsafe.example"},
		"products":[{"title":"Kit","uri":"https://example.com/kit","price":{"amount":"12.00","currency":"ZZZ"}}],
		"futureExtension":{"version":2}
	}`)
	eventR1 := json.RawMessage(`{"name":"First market","startsAt":"2026-09-10T10:00:00Z","endsAt":"2026-09-10T12:00:00Z","roles":["vendor"],"createdAt":"2026-09-01T00:00:00Z"}`)
	eventR2 := json.RawMessage(`{"name":"Newest market","startsAt":"2026-09-11T10:00:00Z","endsAt":"2026-09-11T12:00:00Z","roles":["future-role"],"status":"future-status","createdAt":"2026-09-01T00:00:00Z","futureExtension":{"version":2}}`)
	cases := []struct {
		collection syntax.NSID
		rkey       syntax.RecordKey
		table      string
		r1         json.RawMessage
		r2         json.RawMessage
	}{
		{"social.craftsky.business.profile", "self", "craftsky_business_profiles", profileR1, profileR2},
		{"social.craftsky.business.event", "3meventrecord", "craftsky_business_events", eventR1, eventR2},
	}
	for _, tc := range cases {
		uri := syntax.ATURI("at://did:plc:independent/" + tc.collection.String() + "/" + tc.rkey.String())
		source := ingestion.SourceRecord{
			URI: uri, DID: "did:plc:independent", Collection: tc.collection, Rkey: tc.rkey,
			Revision: "3mbusinessr001", CID: "bafy-r1", Action: "create", Record: tc.r1,
		}
		projectBusinessSource(t, pool, dispatcher, source)
		projectBusinessSource(t, pool, dispatcher, source)
		source.Revision, source.CID, source.Action, source.Record = "3mbusinessr002", "bafy-r2", "update", tc.r2
		projectBusinessSource(t, pool, dispatcher, source)
		source.Revision, source.CID, source.Action, source.Record = "3mbusinessr001", "bafy-r1", "create", tc.r1
		projectBusinessSource(t, pool, dispatcher, source)

		var revision string
		var raw json.RawMessage
		if err := pool.QueryRow(context.Background(), "SELECT source_revision, raw_record FROM "+tc.table+" WHERE uri=$1", uri).Scan(&revision, &raw); err != nil {
			t.Fatalf("read %s projection: %v", tc.collection, err)
		}
		if revision != "3mbusinessr002" {
			t.Fatalf("%s revision = %q, want R2", tc.collection, revision)
		}
		assertJSONEquivalent(t, raw, tc.r2)
		if tc.collection == "social.craftsky.business.profile" {
			view, err := business.HydrateProfile(raw)
			if err != nil {
				t.Fatalf("hydrate profile: %v", err)
			}
			if view.PrimaryAction != nil || len(view.Products) != 1 || view.Products[0].Price != nil || len(view.BusinessTypes) != 2 || view.BusinessTypes[1].Value != "future-business-type" || view.BusinessTypes[1].Known {
				t.Fatalf("unsafe or unknown profile hydration = %+v", view)
			}
		}

		source.Revision, source.CID, source.Action, source.Record = "3mbusinessr003", "", "delete", nil
		projectBusinessSource(t, pool, dispatcher, source)
		source.Revision, source.CID, source.Action, source.Record = "3mbusinessr002", "bafy-r2", "update", tc.r2
		projectBusinessSource(t, pool, dispatcher, source)
		if err := pool.QueryRow(context.Background(), "SELECT source_revision FROM "+tc.table+" WHERE uri=$1", uri).Scan(&revision); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("stale %s resurrected projection: %v", tc.collection, err)
		}
		if err := pool.QueryRow(context.Background(), `SELECT source_revision FROM craftsky_business_record_tombstones WHERE uri=$1`, uri).Scan(&revision); err != nil || revision != "3mbusinessr003" {
			t.Fatalf("%s tombstone revision = %q, error %v", tc.collection, revision, err)
		}
	}
}

func projectBusinessSource(t *testing.T, pool *pgxpool.Pool, dispatcher *index.TransactionalDispatcher, source ingestion.SourceRecord) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin business source projection: %v", err)
	}
	outcome, err := dispatcher.Project(ctx, tx, source)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("project %s %s at %s: %v", source.Collection, source.Action, source.Revision, err)
	}
	if outcome.Kind != tap.OutcomeApplied {
		_ = tx.Rollback(ctx)
		t.Fatalf("projection outcome = %+v", outcome)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit business source projection: %v", err)
	}
}
