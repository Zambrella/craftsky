package index_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
)

// withSchema creates an isolated schema for one test and returns a pool
// whose default search_path points at it. Dropped via t.Cleanup.
func withSchema(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("TEST_DATABASE_URL and DATABASE_URL both unset; skipping real-pg test")
	}

	ctx := context.Background()
	bootstrap, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("bootstrap pool: %v", err)
	}
	schemaName := fmt.Sprintf("test_%d", rand.Uint32())
	if _, err := bootstrap.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = bootstrap.Exec(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE")
		bootstrap.Close()
	})

	// Copy the table into the fresh schema so we don't pollute public.
	ddl := `
		CREATE TABLE ` + schemaName + `.bluesky_posts_sample (
			uri        TEXT PRIMARY KEY,
			cid        TEXT NOT NULL,
			did        TEXT NOT NULL,
			rkey       TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			record     JSONB NOT NULL
		);
		CREATE INDEX ON ` + schemaName + `.bluesky_posts_sample (did);
	`
	if _, err := bootstrap.Exec(ctx, ddl); err != nil {
		t.Fatalf("create test table: %v", err)
	}

	// Return a pool whose search_path is scoped to the test schema.
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestBlueskyPostsSample_Create(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	ev := tap.Event{
		URI:        "at://did:plc:abc/app.bsky.feed.post/3k1",
		CID:        "bafy1",
		DID:        "did:plc:abc",
		Rkey:       "3k1",
		Collection: "app.bsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"hello"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM bluesky_posts_sample WHERE uri = $1", ev.URI).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestBlueskyPostsSample_CreateTwiceIsIdempotent(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	ev := tap.Event{
		URI:        "at://did:plc:abc/app.bsky.feed.post/3k1",
		CID:        "bafy1",
		DID:        "did:plc:abc",
		Rkey:       "3k1",
		Collection: "app.bsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"v1"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	// Second delivery with same URI+CID — should not error.
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("second Handle: %v", err)
	}
	var count int
	_ = pool.QueryRow(context.Background(),
		"SELECT count(*) FROM bluesky_posts_sample WHERE uri = $1", ev.URI).Scan(&count)
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestBlueskyPostsSample_Update(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	create := tap.Event{
		URI: "at://did:plc:x/app.bsky.feed.post/k", CID: "c1", DID: "did:plc:x", Rkey: "k",
		Collection: "app.bsky.feed.post", Action: "create",
		Record: json.RawMessage(`{"text":"old"}`),
	}
	update := create
	update.CID = "c2"
	update.Action = "update"
	update.Record = json.RawMessage(`{"text":"new"}`)

	if err := idx.Handle(context.Background(), create); err != nil {
		t.Fatal(err)
	}
	if err := idx.Handle(context.Background(), update); err != nil {
		t.Fatal(err)
	}

	var cid string
	var record []byte
	err := pool.QueryRow(context.Background(),
		"SELECT cid, record FROM bluesky_posts_sample WHERE uri = $1", create.URI).
		Scan(&cid, &record)
	if err != nil {
		t.Fatal(err)
	}
	if cid != "c2" {
		t.Errorf("cid = %q, want c2", cid)
	}
	// JSONB normalizes whitespace on read, so compare parsed values.
	var got map[string]any
	if err := json.Unmarshal(record, &got); err != nil {
		t.Fatalf("unmarshal record: %v", err)
	}
	if got["text"] != "new" {
		t.Errorf("record.text = %v, want new", got["text"])
	}
}

func TestBlueskyPostsSample_Delete(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	create := tap.Event{
		URI: "at://did:plc:x/app.bsky.feed.post/k", CID: "c1", DID: "did:plc:x", Rkey: "k",
		Collection: "app.bsky.feed.post", Action: "create",
		Record: json.RawMessage(`{"text":"bye"}`),
	}
	if err := idx.Handle(context.Background(), create); err != nil {
		t.Fatal(err)
	}

	del := tap.Event{URI: create.URI, DID: create.DID, Rkey: create.Rkey,
		Collection: "app.bsky.feed.post", Action: "delete"}
	if err := idx.Handle(context.Background(), del); err != nil {
		t.Fatalf("delete Handle: %v", err)
	}

	var count int
	_ = pool.QueryRow(context.Background(),
		"SELECT count(*) FROM bluesky_posts_sample WHERE uri = $1", create.URI).Scan(&count)
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestBlueskyPostsSample_DeleteMissingIsNoop(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	del := tap.Event{
		URI:        "at://did:plc:z/app.bsky.feed.post/nothing",
		Collection: "app.bsky.feed.post", Action: "delete",
	}
	if err := idx.Handle(context.Background(), del); err != nil {
		t.Errorf("delete-missing Handle: %v", err)
	}
}

func TestBlueskyPostsSample_UnknownCollectionIgnored(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	ev := tap.Event{
		URI: "at://did:plc:y/app.bsky.graph.follow/k", CID: "c", DID: "did:plc:y", Rkey: "k",
		Collection: "app.bsky.graph.follow", Action: "create",
		Record: json.RawMessage(`{"subject":"x"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("unknown-collection Handle: %v", err)
	}
	var count int
	_ = pool.QueryRow(context.Background(), "SELECT count(*) FROM bluesky_posts_sample").Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}
