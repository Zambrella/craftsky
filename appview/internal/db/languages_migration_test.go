package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"social.craftsky/appview/internal/testdb"
)

const languagesMigrationPreStateDDL = `
CREATE TABLE craftsky_posts (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    tags        TEXT[]      NOT NULL DEFAULT '{}',
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey)
);
INSERT INTO craftsky_posts (uri, did, rkey, cid, record, created_at)
VALUES (
    'at://did:plc:alice/social.craftsky.feed.post/one',
    'did:plc:alice',
    'one',
    'post-cid',
    '{}',
    '2026-07-29T08:00:00Z'
);
`

func TestLanguagesMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000033_post_languages.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000033_post_languages.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, languagesMigrationPreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}

	var existingLangs []string
	if err := pool.QueryRow(ctx, `
		SELECT langs
		FROM craftsky_posts
		WHERE uri = 'at://did:plc:alice/social.craftsky.feed.post/one'
	`).Scan(&existingLangs); err != nil {
		t.Fatalf("read existing post languages: %v", err)
	}
	if len(existingLangs) != 0 {
		t.Fatalf("existing post languages = %v, want empty", existingLangs)
	}

	if !tableExists(t, pool, "account_language_preferences") {
		t.Fatal("account_language_preferences table missing")
	}
	if !constraintExists(t, pool, "account_language_preferences_pkey") {
		t.Fatal("account_language_preferences primary key missing")
	}
	if !indexExists(t, pool, "craftsky_posts_langs_gin") {
		t.Fatal("craftsky_posts language GIN index missing")
	}

	beforeInsert := time.Now().Add(-time.Second)
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_language_preferences (
			account_did,
			primary_language,
			content_languages
		) VALUES (
			'did:plc:alice',
			'en',
			ARRAY['en', 'cy']::text[]
		)
	`); err != nil {
		t.Fatalf("insert language preferences: %v", err)
	}

	var (
		primary   string
		content   []string
		createdAt time.Time
		updatedAt time.Time
	)
	if err := pool.QueryRow(ctx, `
		SELECT primary_language, content_languages, created_at, updated_at
		FROM account_language_preferences
		WHERE account_did = 'did:plc:alice'
	`).Scan(&primary, &content, &createdAt, &updatedAt); err != nil {
		t.Fatalf("read language preferences: %v", err)
	}
	if primary != "en" {
		t.Fatalf("primary language = %q, want en", primary)
	}
	if len(content) != 2 || content[0] != "en" || content[1] != "cy" {
		t.Fatalf("content languages = %v, want [en cy]", content)
	}
	if createdAt.Before(beforeInsert) || updatedAt.Before(beforeInsert) {
		t.Fatalf("timestamps were not defaulted: created=%s updated=%s", createdAt, updatedAt)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO account_language_preferences (
			account_did,
			primary_language,
			content_languages
		) VALUES (
			'did:plc:alice',
			'fr',
			ARRAY['fr']::text[]
		)
	`); err == nil {
		t.Fatal("duplicate account preference row succeeded")
	}

	if _, err := pool.Exec(ctx, `
		DELETE FROM account_language_preferences
		WHERE account_did = 'did:plc:alice'
	`); err != nil {
		t.Fatalf("delete language preferences: %v", err)
	}
	assertRowCount(t, pool, "account_language_preferences", 0)

	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}
	if tableExists(t, pool, "account_language_preferences") {
		t.Fatal("account_language_preferences table remained after down")
	}
	if indexExists(t, pool, "craftsky_posts_langs_gin") {
		t.Fatal("craftsky_posts language GIN index remained after down")
	}

	var langsColumnExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'craftsky_posts'
			  AND column_name = 'langs'
		)
	`).Scan(&langsColumnExists); err != nil {
		t.Fatalf("inspect langs column after down: %v", err)
	}
	if langsColumnExists {
		t.Fatal("craftsky_posts.langs remained after down")
	}
}
