package api

import (
	"context"
	"os"
	"strings"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

const languageQueryPlanPreStateDDL = `
CREATE TABLE craftsky_posts (
    uri        TEXT        NOT NULL PRIMARY KEY,
    did        TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX craftsky_posts_did_created_idx
    ON craftsky_posts (did, created_at DESC);
`

func TestIT025RepresentativeLanguagePlansUsePreferencePKAndGINIndexes(t *testing.T) {
	pool := testdb.WithSchema(t, languageQueryPlanPreStateDDL)
	ctx := context.Background()
	up, err := os.ReadFile("../../migrations/000033_post_languages.up.sql")
	if err != nil {
		t.Fatalf("read language migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply language migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_language_preferences (
			account_did,
			primary_language,
			content_languages
		) VALUES (
			'did:plc:viewer',
			'en',
			ARRAY['en']::text[]
		);

		INSERT INTO craftsky_posts (uri, did, created_at, langs)
		SELECT
			'at://did:plc:author/social.craftsky.feed.post/' || n,
			CASE
				WHEN n % 1000 = 0 THEN 'did:plc:viewer'
				ELSE 'did:plc:author'
			END,
			now() - n * interval '1 second',
			CASE
				WHEN n % 20 = 0 THEN ARRAY['en']::text[]
				ELSE ARRAY['fr']::text[]
			END
		FROM generate_series(1, 20000) AS n;
		ANALYZE account_language_preferences;
		ANALYZE craftsky_posts;
		SET enable_seqscan = off;
	`); err != nil {
		t.Fatalf("seed representative posts: %v", err)
	}

	var preferenceRaw []byte
	if err := pool.QueryRow(ctx, `
		EXPLAIN (FORMAT JSON, COSTS OFF)
		SELECT primary_language, content_languages
		FROM account_language_preferences
		WHERE account_did = 'did:plc:viewer'
	`).Scan(&preferenceRaw); err != nil {
		t.Fatalf("explain language preference lookup: %v", err)
	}
	preferencePlan := string(preferenceRaw)
	if !strings.Contains(
		preferencePlan,
		`"Index Name": "account_language_preferences_pkey"`,
	) {
		t.Fatalf(
			"preference lookup does not use its primary-key index:\n%s",
			preferencePlan,
		)
	}

	var raw []byte
	query := `
		EXPLAIN (FORMAT JSON, COSTS OFF)
		SELECT uri
		FROM craftsky_posts p
		WHERE TRUE
		` + languageVisibilityPredicate("p", "$1", "$2") + `
		ORDER BY created_at DESC, uri DESC
		LIMIT 20
	`
	if err := pool.QueryRow(
		ctx,
		query,
		"did:plc:viewer",
		[]string{"en"},
	).Scan(&raw); err != nil {
		t.Fatalf("explain language overlap: %v", err)
	}
	plan := string(raw)
	if !strings.Contains(plan, "craftsky_posts_langs_gin") {
		t.Fatalf("query plan does not use language GIN index:\n%s", plan)
	}
	if strings.Contains(plan, `"Node Type": "Seq Scan"`) {
		t.Fatalf("query plan uses a sequential post scan:\n%s", plan)
	}
}
