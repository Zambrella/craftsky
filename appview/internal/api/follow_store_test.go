package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const followStoreDDL = `
CREATE TABLE tap_source_records (
    uri TEXT PRIMARY KEY, rkey TEXT NOT NULL, cid TEXT, updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE pds_set_sources (
    source_uri TEXT PRIMARY KEY, kind TEXT NOT NULL, actor_did TEXT NOT NULL,
    scope_key TEXT NOT NULL, subject_did TEXT, activity_at TIMESTAMPTZ NOT NULL,
    eligible BOOLEAN NOT NULL
);
CREATE TABLE pds_set_aggregates (
    kind TEXT NOT NULL, actor_did TEXT NOT NULL, scope_key TEXT NOT NULL,
    subject_did TEXT, representative_source_uri TEXT NOT NULL, activated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (kind, actor_did, scope_key)
);
`

func seedFollowReadModel(t *testing.T, pool *pgxpool.Pool, row api.FollowRow, representative bool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO tap_source_records(uri,rkey,cid,updated_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT(uri) DO UPDATE SET cid=EXCLUDED.cid
	`, row.URI, row.Rkey, row.CID, row.CreatedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_set_sources(source_uri,kind,actor_did,scope_key,subject_did,activity_at,eligible)
		VALUES ($1,'follow',$2,$3,$3,$4,true)
		ON CONFLICT(source_uri) DO NOTHING
	`, row.URI, row.DID, row.SubjectDID, row.CreatedAt); err != nil {
		t.Fatal(err)
	}
	if representative {
		if _, err := pool.Exec(ctx, `
			INSERT INTO pds_set_aggregates(kind,actor_did,scope_key,subject_did,representative_source_uri,activated_at)
			VALUES ('follow',$1,$2,$2,$3,$4)
			ON CONFLICT(kind,actor_did,scope_key) DO NOTHING
		`, row.DID, row.SubjectDID, row.URI, row.CreatedAt); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFollowStoreReadsOnlyCurrentAggregate(t *testing.T) {
	pool := testdb.WithSchema(t, followStoreDDL)
	store := api.NewFollowStore(pool)
	ctx := context.Background()
	row := api.FollowRow{
		URI: "at://did:plc:alice/app.bsky.graph.follow/3aaaaaaaaaaa2",
		DID: "did:plc:alice", Rkey: "3aaaaaaaaaaa2", CID: "bafy-follow",
		SubjectDID: "did:plc:bob", CreatedAt: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC),
	}
	seedFollowReadModel(t, pool, row, true)
	seedFollowReadModel(t, pool, row, true) // duplicate source delivery
	active, err := store.FindActiveFollow(ctx, row.DID, row.SubjectDID)
	if err != nil || active == nil || active.URI != row.URI || active.CID != row.CID || !active.CreatedAt.Equal(row.CreatedAt) {
		t.Fatalf("active follow=%+v err=%v", active, err)
	}
	followed, err := store.ListActiveFollowedDIDs(ctx, row.DID)
	if err != nil || len(followed) != 1 || followed[0] != row.SubjectDID {
		t.Fatalf("followed=%v err=%v", followed, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM pds_set_aggregates WHERE kind='follow' AND actor_did=$1 AND scope_key=$2`, row.DID, row.SubjectDID); err != nil {
		t.Fatal(err)
	}
	active, err = store.FindActiveFollow(ctx, row.DID, row.SubjectDID)
	if err != nil || active != nil {
		t.Fatalf("inactive follow=%+v err=%v", active, err)
	}
}

func TestFollowStoreDeduplicatesPhysicalSourcesThroughAggregates(t *testing.T) {
	pool := testdb.WithSchema(t, followStoreDDL)
	store := api.NewFollowStore(pool)
	created := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	for index, subject := range []string{"did:plc:bob", "did:plc:carol", "did:plc:bob"} {
		rkey := []string{"3aaaaaaaaaaa2", "3aaaaaaaaaaa3", "3aaaaaaaaaaa4"}[index]
		seedFollowReadModel(t, pool, api.FollowRow{
			URI: "at://did:plc:alice/app.bsky.graph.follow/" + rkey,
			DID: "did:plc:alice", Rkey: rkey, CID: "bafy-" + rkey,
			SubjectDID: subject, CreatedAt: created,
		}, true)
	}
	followed, err := store.ListActiveFollowedDIDs(context.Background(), "did:plc:alice")
	if err != nil || len(followed) != 2 || followed[0] != "did:plc:bob" || followed[1] != "did:plc:carol" {
		t.Fatalf("followed=%v err=%v", followed, err)
	}
}
