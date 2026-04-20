package testpipeline_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testpipeline"
)

// withSchema creates an isolated schema for one test and returns a pool
// whose default search_path points at it. Dropped via t.Cleanup.
// Adapted from appview/internal/index/bluesky_posts_sample_test.go.
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

	ddl := `
		CREATE TABLE ` + schemaName + `.test_posts (
			uri        TEXT PRIMARY KEY,
			cid        TEXT NOT NULL,
			did        TEXT NOT NULL,
			text       TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX ON ` + schemaName + `.test_posts (created_at DESC);
	`
	if _, err := bootstrap.Exec(ctx, ddl); err != nil {
		t.Fatalf("create test table: %v", err)
	}

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

func TestIndexer_CreateInsertsRow(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)

	rec, _ := json.Marshal(map[string]any{
		"text":      "hello pipeline",
		"createdAt": "2026-04-19T10:00:00Z",
	})
	ev := tap.Event{
		URI:        "at://did:plc:abc/social.craftsky.test.post/3kxaaa",
		CID:        "bafyaaa",
		DID:        "did:plc:abc",
		Collection: "social.craftsky.test.post",
		Rkey:       "3kxaaa",
		Action:     "create",
		Record:     rec,
	}

	if err := ix.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var (
		gotCID, gotDID, gotText string
		gotCreatedAt            time.Time
	)
	err := pool.QueryRow(context.Background(),
		`SELECT cid, did, text, created_at FROM test_posts WHERE uri = $1`,
		ev.URI,
	).Scan(&gotCID, &gotDID, &gotText, &gotCreatedAt)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if gotCID != "bafyaaa" || gotDID != "did:plc:abc" || gotText != "hello pipeline" {
		t.Errorf("row mismatch: cid=%s did=%s text=%s", gotCID, gotDID, gotText)
	}
	wantT, _ := time.Parse(time.RFC3339, "2026-04-19T10:00:00Z")
	if !gotCreatedAt.Equal(wantT) {
		t.Errorf("created_at: got %v want %v", gotCreatedAt, wantT)
	}
}

func TestIndexer_UpdateReplacesRow(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)

	rec1, _ := json.Marshal(map[string]any{"text": "v1", "createdAt": "2026-04-19T10:00:00Z"})
	rec2, _ := json.Marshal(map[string]any{"text": "v2", "createdAt": "2026-04-19T10:00:00Z"})
	evBase := tap.Event{
		URI: "at://did:plc:abc/social.craftsky.test.post/3kxaaa",
		DID: "did:plc:abc", Collection: "social.craftsky.test.post", Rkey: "3kxaaa",
	}
	create := evBase
	create.CID = "bafy1"
	create.Action = "create"
	create.Record = rec1
	update := evBase
	update.CID = "bafy2"
	update.Action = "update"
	update.Record = rec2

	if err := ix.Handle(context.Background(), create); err != nil {
		t.Fatal(err)
	}
	if err := ix.Handle(context.Background(), update); err != nil {
		t.Fatal(err)
	}

	var count int
	pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM test_posts WHERE uri = $1`, evBase.URI,
	).Scan(&count)
	if count != 1 {
		t.Errorf("row count: got %d want 1", count)
	}
	var cid, text string
	pool.QueryRow(context.Background(),
		`SELECT cid, text FROM test_posts WHERE uri = $1`, evBase.URI,
	).Scan(&cid, &text)
	if cid != "bafy2" || text != "v2" {
		t.Errorf("post-update: cid=%s text=%s, want bafy2/v2", cid, text)
	}
}
