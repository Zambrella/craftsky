package db_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const instagramPrivateSuggestionsPreStateDDL = `
CREATE TABLE notification_events (
    id UUID PRIMARY KEY,
    recipient_did TEXT NOT NULL,
    actor_did TEXT,
    category TEXT NOT NULL,
    subject_key TEXT NOT NULL,
    source_uri TEXT,
    source_cid TEXT,
    source_rkey TEXT,
    subject_uri TEXT,
    subject_cid TEXT,
    parent_uri TEXT,
    parent_cid TEXT,
    root_uri TEXT,
    root_cid TEXT,
    quoted_uri TEXT,
    quoted_cid TEXT,
    eligibility_scope TEXT NOT NULL,
    CONSTRAINT notification_events_category_check CHECK (category IN (
        'like','follow','reply','mention','quote','repost','everythingElse','instagramMatch'
    )),
    CONSTRAINT notification_events_type_payload_check CHECK (
        category <> 'instagramMatch' OR actor_did IS NOT NULL
    )
);
CREATE UNIQUE INDEX notification_events_instagram_operation_unique
    ON notification_events(recipient_did, category, subject_key)
    WHERE category='instagramMatch';
CREATE TABLE notification_preferences (
    account_did TEXT NOT NULL,
    category TEXT NOT NULL,
    scope TEXT NOT NULL,
    push_enabled BOOLEAN NOT NULL,
    PRIMARY KEY(account_did, category),
    CONSTRAINT notification_preferences_category_check CHECK (category IN (
        'like','follow','reply','mention','quote','repost','everythingElse','instagramMatch'
    )),
    CONSTRAINT notification_preferences_instagram_match_scope_check CHECK (
        category <> 'instagramMatch' OR scope='everyone'
    )
);
CREATE TABLE instagram_account_links (
    id UUID PRIMARY KEY
);
CREATE TABLE instagram_graph_imports (
    id UUID PRIMARY KEY
);
CREATE TABLE instagram_automatic_follow_ledger (
    id UUID PRIMARY KEY
);
CREATE TABLE instagram_automatic_follow_sources (
    automatic_follow_id UUID NOT NULL REFERENCES instagram_automatic_follow_ledger(id) ON DELETE CASCADE,
    import_id UUID NOT NULL REFERENCES instagram_graph_imports(id) ON DELETE CASCADE,
    PRIMARY KEY(automatic_follow_id, import_id)
);
CREATE TABLE pds_follow_operations (
    id UUID PRIMARY KEY,
    automatic_follow_id UUID NOT NULL REFERENCES instagram_automatic_follow_ledger(id) ON DELETE CASCADE
);
INSERT INTO notification_events(
    id,recipient_did,actor_did,category,subject_key,eligibility_scope
) VALUES (
    '10000000-0000-4000-8000-000000000001','did:plc:owner','did:plc:target',
    'instagramMatch','legacy-operation','everyone'
);
INSERT INTO notification_preferences(account_did,category,scope,push_enabled)
VALUES ('did:plc:owner','instagramMatch','everyone',true);
INSERT INTO instagram_graph_imports(id)
VALUES ('20000000-0000-4000-8000-000000000001');
INSERT INTO instagram_automatic_follow_ledger(id)
VALUES ('30000000-0000-4000-8000-000000000001');
INSERT INTO instagram_automatic_follow_sources(automatic_follow_id,import_id)
VALUES (
    '30000000-0000-4000-8000-000000000001',
    '20000000-0000-4000-8000-000000000001'
);
INSERT INTO pds_follow_operations(id,automatic_follow_id)
VALUES (
    '40000000-0000-4000-8000-000000000001',
    '30000000-0000-4000-8000-000000000001'
);
`

func TestInstagramPrivateSuggestionsMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000042_instagram_private_suggestions.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000042_instagram_private_suggestions.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, instagramPrivateSuggestionsPreStateDDL)
	applyInstagramPrivateSuggestionMigration(t, pool, "up", up)
	assertInstagramPrivateSuggestionSchema(t, pool)
	assertInstagramAutomaticStateReset(t, pool)
	assertInstagramPrivateSuggestionConstraints(t, pool)

	applyInstagramPrivateSuggestionMigration(t, pool, "down", down)
	if tableExists(t, pool, "instagram_private_suggestions") ||
		tableExists(t, pool, "instagram_private_suggestion_sources") {
		t.Fatal("private suggestion relations remained after down migration")
	}

	applyInstagramPrivateSuggestionMigration(t, pool, "second up", up)
	assertInstagramPrivateSuggestionSchema(t, pool)
}

func applyInstagramPrivateSuggestionMigration(
	t *testing.T,
	pool *pgxpool.Pool,
	label string,
	sql []byte,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
		t.Fatalf("apply %s migration: %v", label, err)
	}
}

func assertInstagramPrivateSuggestionSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range []string{"instagram_private_suggestions", "instagram_private_suggestion_sources"} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s is missing", table)
		}
	}
	for _, index := range []string{
		"instagram_private_suggestions_owner_page_idx",
		"instagram_private_suggestions_target_idx",
		"instagram_private_suggestions_terminal_retention_idx",
		"instagram_private_suggestions_link_idx",
		"instagram_private_suggestion_sources_import_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s is missing", index)
		}
	}
}

func assertInstagramAutomaticStateReset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range []string{
		"notification_events",
		"notification_preferences",
		"pds_follow_operations",
		"instagram_automatic_follow_sources",
		"instagram_automatic_follow_ledger",
	} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT count(*)::int FROM "+table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s count = %d, want pre-production reset", table, count)
		}
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_preferences(account_did,category,scope,push_enabled)
		VALUES ('did:plc:owner','instagramMatch','everyone',true)
	`); err == nil {
		t.Fatal("retired instagramMatch preference category was accepted")
	}
}

func assertInstagramPrivateSuggestionConstraints(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_account_links(id)
		VALUES ('50000000-0000-4000-8000-000000000001');
		INSERT INTO instagram_private_suggestions(
			id,importer_did,target_did,importer_generation,target_generation,
			evidence_link_id,state,reason,created_at,updated_at
		) VALUES (
			'60000000-0000-4000-8000-000000000001',
			'did:plc:owner','did:plc:target',2,4,
			'50000000-0000-4000-8000-000000000001','pending',
			'verifiedInstagramFollow',now(),now()
		)
	`); err != nil {
		t.Fatalf("insert valid private suggestion: %v", err)
	}
	for _, statement := range []string{
		`INSERT INTO instagram_private_suggestions(
			id,importer_did,target_did,importer_generation,target_generation,
			evidence_link_id,state,reason,created_at,updated_at
		 ) VALUES (
			'60000000-0000-4000-8000-000000000002',
			'did:plc:owner','did:plc:target',0,4,
			'50000000-0000-4000-8000-000000000001','pending',
			'verifiedInstagramFollow',now(),now()
		 )`,
		`UPDATE instagram_private_suggestions
		 SET state='dismissed',updated_at=now()
		 WHERE id='60000000-0000-4000-8000-000000000001'`,
	} {
		_, err := pool.Exec(ctx, statement)
		var postgresError *pgconn.PgError
		if !errors.As(err, &postgresError) || postgresError.Code != "23514" {
			t.Errorf("constraint statement error = %v, want check violation", err)
		}
	}
}
