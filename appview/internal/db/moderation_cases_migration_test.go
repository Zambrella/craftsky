package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const moderationCasesPreStateDDL = `
CREATE TABLE moderation_reports (
    id TEXT PRIMARY KEY,
    reporter_did TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    subject_collection TEXT,
    subject_rkey TEXT,
    subject_uri TEXT,
    subject_cid_snapshot TEXT,
    submitted_handle_snapshot TEXT,
    reason_type TEXT NOT NULL,
    details TEXT,
    device_id TEXT,
    forwarding_status TEXT NOT NULL,
    forwarding_schema_version TEXT,
    forwarding_prepared_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE moderation_outputs (
    id TEXT PRIMARY KEY,
    source_did TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    subject_collection TEXT,
    subject_rkey TEXT,
    subject_uri TEXT,
    value TEXT NOT NULL,
    action TEXT NOT NULL,
    internal_reason TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE notification_events (
    id UUID PRIMARY KEY,
    recipient_did TEXT NOT NULL,
    actor_did TEXT NOT NULL,
    category TEXT NOT NULL,
    subject_key TEXT NOT NULL,
    source_uri TEXT,
    source_cid TEXT,
    source_rkey TEXT,
    eligibility_scope TEXT NOT NULL,
    recipient_followed_actor BOOLEAN NOT NULL,
    push_enabled_snapshot BOOLEAN NOT NULL,
    state TEXT NOT NULL,
    first_activity_at TIMESTAMPTZ NOT NULL,
    activity_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    initial_push_evaluated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT notification_events_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch'
    )),
    CONSTRAINT notification_events_type_payload_check CHECK (
        actor_did IS NOT NULL
        AND (
            (category = 'instagramMatch'
                AND source_uri IS NULL AND source_cid IS NULL AND source_rkey IS NULL)
            OR
            (category <> 'instagramMatch'
                AND source_uri IS NOT NULL AND source_cid IS NOT NULL AND source_rkey IS NOT NULL)
        )
    ),
    CONSTRAINT notification_events_recipient_actor_category_subject_key_key
        UNIQUE (recipient_did, actor_did, category, subject_key)
);
CREATE TABLE notification_preferences (
    account_did TEXT NOT NULL,
    category TEXT NOT NULL,
    scope TEXT NOT NULL,
    push_enabled BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_did, category),
    CONSTRAINT notification_preferences_category_check CHECK (category IN (
        'like', 'follow', 'reply', 'mention', 'quote', 'repost',
        'everythingElse', 'instagramMatch'
    )),
    CONSTRAINT notification_preferences_instagram_match_scope_check CHECK (
        category <> 'instagramMatch' OR scope = 'everyone'
    )
);
INSERT INTO moderation_reports(
    id,reporter_did,subject_type,subject_did,subject_collection,subject_rkey,
    subject_uri,subject_cid_snapshot,reason_type,forwarding_status,
    forwarding_prepared_at
) VALUES (
    'legacy-report','did:plc:reporter','post','did:plc:owner',
    'social.craftsky.feed.post','legacy','at://did:plc:owner/social.craftsky.feed.post/legacy',
    'bafylegacy','spam','prepared_not_submitted',now()
);
INSERT INTO moderation_outputs(
    id,source_did,subject_type,subject_did,subject_collection,subject_rkey,
    subject_uri,value,action
) VALUES (
    'legacy-output','did:plc:moderator','post','did:plc:owner',
    'social.craftsky.feed.post','legacy','at://did:plc:owner/social.craftsky.feed.post/legacy',
    'hide','apply'
);
INSERT INTO notification_events(
    id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,
    source_rkey,eligibility_scope,recipient_followed_actor,push_enabled_snapshot,
    state,first_activity_at,activity_at,initial_push_evaluated_at
) VALUES (
    '10000000-0000-4000-8000-000000000001','did:plc:owner','did:plc:actor',
    'instagramMatch','legacy-suggestion',NULL,NULL,NULL,'everyone',false,true,
    'active',now(),now(),now()
);
`

func TestModerationCasesMigrationPreservesLegacyDataAndEnforcesInvariants(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000069_moderation_cases.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000069_moderation_cases.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, moderationCasesPreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertModerationCaseSchema(t, pool)
	assertModerationLegacyState(t, pool)
	assertModerationCaseConstraints(t, pool)

	apply("down", down)
	for _, table := range moderationCaseTables() {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	assertTableCount(t, pool, "moderation_reports", 2)
	assertTableCount(t, pool, "moderation_outputs", 1)
	assertTableCount(t, pool, "notification_events", 1)

	apply("second up", up)
	assertModerationCaseSchema(t, pool)
}

func moderationCaseTables() []string {
	return []string{
		"moderation_case_reports",
		"moderation_decisions",
		"moderation_effect_events",
		"moderation_active_case_effects",
		"moderation_case_strikes",
		"moderation_appeal_correspondence",
		"moderation_appeals",
		"moderation_case_events",
		"moderation_account_standings",
		"moderation_cases",
	}
}

func assertModerationCaseSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range moderationCaseTables() {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, index := range []string{
		"moderation_cases_one_open_subject_idx",
		"moderation_cases_queue_idx",
		"moderation_case_events_replay_idx",
		"moderation_case_strikes_due_idx",
		"moderation_effect_events_case_created_idx",
		"notification_events_moderation_event_unique",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}
}

func assertModerationLegacyState(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	assertTableCount(t, pool, "moderation_reports", 1)
	assertTableCount(t, pool, "moderation_outputs", 1)
	assertTableCount(t, pool, "notification_events", 1)
	for _, table := range moderationCaseTables() {
		assertTableCount(t, pool, table, 0)
	}
}

func assertModerationCaseConstraints(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_reports(
			id,reporter_did,subject_type,subject_did,reason_type,
			forwarding_status,forwarding_prepared_at
		) VALUES (
			'new-report','did:plc:reporter','account','did:plc:owner','spam',
			'prepared_not_submitted',now()
		);
		INSERT INTO moderation_cases(
			id,subject_key,subject_type,subject_did,owner_did,safe_snapshot
		) VALUES (
			'20000000-0000-4000-8000-000000000001','account:did:plc:owner',
			'account','did:plc:owner','did:plc:owner','{}'
		);
		INSERT INTO moderation_case_reports(case_id,report_id)
		VALUES ('20000000-0000-4000-8000-000000000001','new-report');
	`); err != nil {
		t.Fatalf("seed valid moderation case: %v", err)
	}

	assertModerationStatementFails(t, pool, `
		INSERT INTO moderation_cases(
			id,subject_key,subject_type,subject_did,owner_did,safe_snapshot
		) VALUES (
			'20000000-0000-4000-8000-000000000002','account:did:plc:owner',
			'account','did:plc:owner','did:plc:owner','{}'
		)
	`)
	if _, err := pool.Exec(ctx, `
		UPDATE moderation_cases SET state='resolved',resolved_at=now()
		WHERE id='20000000-0000-4000-8000-000000000001';
		INSERT INTO moderation_cases(
			id,subject_key,subject_type,subject_did,owner_did,safe_snapshot
		) VALUES (
			'20000000-0000-4000-8000-000000000002','account:did:plc:owner',
			'account','did:plc:owner','did:plc:owner','{}'
		)
	`); err != nil {
		t.Fatalf("create new case after resolution: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_case_events(
			id,case_id,event_type,actor_id,source_system,replay_id,
			request_fingerprint,expected_revision,result_revision
		) VALUES (
			'30000000-0000-4000-8000-000000000001',
			'20000000-0000-4000-8000-000000000001','decision','moderator-one',
			'retool','replay-one',decode(repeat('11',32),'hex'),0,1
		)
	`); err != nil {
		t.Fatalf("insert valid case event: %v", err)
	}
	assertModerationStatementFails(t, pool, `
		INSERT INTO moderation_case_events(
			id,case_id,event_type,actor_id,source_system,replay_id,
			request_fingerprint,expected_revision,result_revision
		) VALUES (
			'30000000-0000-4000-8000-000000000002',
			'20000000-0000-4000-8000-000000000002','decision','moderator-one',
			'retool','replay-one',decode(repeat('22',32),'hex'),0,1
		)
	`)

	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_case_strikes(
			case_id,logical_effect_id,owner_did,issued_at,due_at
		) VALUES (
			'20000000-0000-4000-8000-000000000001',
			'40000000-0000-4000-8000-000000000001','did:plc:owner',now(),now()+interval '12 months'
		);
		INSERT INTO moderation_appeals(case_id,confirmed_event_id,status,confirmed_at)
		VALUES (
			'20000000-0000-4000-8000-000000000001',
			'30000000-0000-4000-8000-000000000001','pending',now()
		);
		INSERT INTO notification_events(
			id,recipient_did,actor_did,category,subject_key,moderation_case_reference,moderation_event_id,source_uri,source_cid,
			source_rkey,eligibility_scope,recipient_followed_actor,push_enabled_snapshot,
			state,first_activity_at,activity_at,initial_push_evaluated_at
		) VALUES (
			'50000000-0000-4000-8000-000000000001','did:plc:owner',NULL,
			'moderation','event-one','MOD-20000000-0000-4000-8000-000000000001',
			'30000000-0000-4000-8000-000000000001',NULL,NULL,NULL,
			'everyone',false,true,'active',now(),now(),now()
		);
		INSERT INTO notification_preferences(account_did,category,scope,push_enabled)
		VALUES ('did:plc:owner','moderation','everyone',true)
	`); err != nil {
		t.Fatalf("insert valid strike, appeal, and moderation notification: %v", err)
	}
	assertModerationStatementFails(t, pool, `
		INSERT INTO notification_preferences(account_did,category,scope,push_enabled)
		VALUES ('did:plc:other','moderation','peopleIFollow',true)
	`)
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_case_events(
			id,case_id,event_type,actor_id,source_system,replay_id,
			request_fingerprint,expected_revision,result_revision
		) VALUES (
			'30000000-0000-4000-8000-000000000003',
			'20000000-0000-4000-8000-000000000001','effectsChanged','moderator-one',
			'retool','replay-two',decode(repeat('33',32),'hex'),1,2
		);
		INSERT INTO notification_events(
			id,recipient_did,category,subject_key,moderation_case_reference,moderation_event_id,
			eligibility_scope,recipient_followed_actor,push_enabled_snapshot,
			state,first_activity_at,activity_at,initial_push_evaluated_at
		) VALUES (
			'50000000-0000-4000-8000-000000000002','did:plc:owner','moderation',
			'event-two','MOD-20000000-0000-4000-8000-000000000001',
			'30000000-0000-4000-8000-000000000003','everyone',false,true,
			'active',now(),now(),now()
		)
	`); err != nil {
		t.Fatalf("second event for moderation case must be supported: %v", err)
	}
}

func assertModerationStatementFails(t *testing.T, pool *pgxpool.Pool, statement string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), statement); err == nil {
		t.Fatalf("constrained statement succeeded: %s", statement)
	}
}
