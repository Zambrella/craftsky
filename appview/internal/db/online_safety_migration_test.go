package db_test

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const imageSafetyMigrationPreStateDDL = `
CREATE TABLE tap_source_records (
    uri TEXT PRIMARY KEY
);
CREATE TABLE tap_projection_jobs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_uri TEXT NOT NULL REFERENCES tap_source_records(uri) ON DELETE CASCADE,
    projection_kind TEXT NOT NULL,
    source_event_id BIGINT NOT NULL CHECK (source_event_id > 0),
    state TEXT NOT NULL CHECK (state IN ('pending', 'blocked', 'processing', 'complete', 'permanent_denied')),
    dependency_kind TEXT,
    dependency_key TEXT,
    CONSTRAINT tap_projection_jobs_dependency_check CHECK (
        (state = 'blocked'
            AND dependency_kind IN ('member_did', 'subject_uri', 'repository_did')
            AND dependency_key IS NOT NULL
            AND btrim(dependency_key) <> '' AND char_length(dependency_key) <= 2048)
        OR
        (state <> 'blocked' AND dependency_kind IS NULL AND dependency_key IS NULL)
    )
);
INSERT INTO tap_source_records(uri) VALUES
    ('at://did:plc:owner/social.craftsky.feed.post/one'),
    ('at://did:plc:owner/social.craftsky.feed.post/two');
`

func TestImageSafetyMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000081_image_safety.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000081_image_safety.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, imageSafetyMigrationPreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertImageSafetySchema(t, pool)
	assertImageSafetyClosedStates(t, pool)
	assertImageSubjectDependencyKind(t, pool, "at://did:plc:owner/social.craftsky.feed.post/one")

	apply("down", down)
	for _, table := range imageSafetyTables() {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	assertImageSafetyStatementFails(t, pool, `
		UPDATE tap_projection_jobs
		SET state = 'blocked', dependency_kind = 'image_subject_uri', dependency_key = source_uri
	`)

	apply("second up", up)
	assertImageSafetySchema(t, pool)
	assertImageSubjectDependencyKind(t, pool, "at://did:plc:owner/social.craftsky.feed.post/two")
}

func TestSafetyIncidentMigration(t *testing.T) {
	imageSafety, err := os.ReadFile("../../migrations/000081_image_safety.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/000082_safety_incidents.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000082_safety_incidents.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, imageSafetyMigrationPreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(imageSafety)); err != nil {
		t.Fatalf("apply image safety prerequisite: %v", err)
	}
	apply := func(label string, migration []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}
	apply("up", up)
	assertSafetyIncidentSchema(t, pool)
	assertTriggerExists(t, pool, "safety_incident_events_append_only")
	assertAppendOnlyTable(t, pool, "safety_incident_events")
	apply("down", down)
	for _, table := range []string{"safety_incident_events", "safety_incident_subjects", "safety_incidents"} {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	apply("second up", up)
	assertSafetyIncidentSchema(t, pool)
	assertTriggerExists(t, pool, "safety_incident_events_append_only")
}

func TestExternalSafetyIntakeMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000085_external_safety_intake.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000085_external_safety_intake.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE moderation_cases (
			id UUID PRIMARY KEY,
			owner_did TEXT NOT NULL
		);
		CREATE TABLE moderation_appeal_correspondence (
			id UUID PRIMARY KEY,
			case_id UUID NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
			source_system TEXT NOT NULL,
			replay_id TEXT NOT NULL,
			sender_reference_hash BYTEA,
			received_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			UNIQUE(source_system, replay_id)
		);
	`)
	ctx := context.Background()
	apply := func(label string, migration []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	for _, table := range externalSafetyIntakeTables() {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, constraint := range []string{
		"external_safety_intakes_values_check",
		"external_safety_correspondence_values_check",
		"external_safety_appeal_links_values_check",
		"safety_workflows_deadline_check",
		"safety_workflows_values_check",
		"safety_workflow_events_values_check",
		"safety_authority_requests_values_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	assertTriggerExists(t, pool, "safety_workflow_events_append_only")
	assertAppendOnlyTable(t, pool, "safety_workflow_events")
	assertImageSafetyStatementFails(t, pool, `
		INSERT INTO external_safety_intakes(
			id,reference,provider_message_ref,intake_kind,urgency,canonical_subject,
			owner_actor_id,attachment_status,received_at,created_at
		) VALUES (
			'10000000-0000-4000-8000-000000000001','CS-SAF-TEST','message-1',
			'allegation','routine','at://did:plc:test/social.craftsky.feed.post/one',
			'operator','rendered',now(),now()
		)
	`)

	apply("down", down)
	for _, table := range externalSafetyIntakeTables() {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	apply("second up", up)
}

func TestAgeEligibilityMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000086_age_eligibility.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000086_age_eligibility.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	apply := func(label string, migration []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	for _, table := range []string{"account_policy_acceptances", "account_age_eligibility", "account_age_eligibility_events"} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO account_policy_acceptances(account_did,policy_version) VALUES('did:plc:alice','safety-2026-09')`); err != nil {
		t.Fatalf("insert minimal policy acceptance: %v", err)
	}
	assertImageSafetyStatementFails(t, pool, `INSERT INTO account_policy_acceptances(account_did,policy_version) VALUES('did:plc:bob','')`)
	assertImageSafetyStatementFails(t, pool, `INSERT INTO account_age_eligibility(account_did,state,evidence_kind) VALUES('did:plc:bob','restricted','self_disclosure')`)

	apply("down", down)
	for _, table := range []string{"account_policy_acceptances", "account_age_eligibility", "account_age_eligibility_events"} {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	apply("second up", up)
}

func TestSafetyOperatorsMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000087_safety_operators.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000087_safety_operators.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `CREATE TABLE safety_incidents(id UUID PRIMARY KEY);`)
	ctx := context.Background()
	apply := func(label string, migration []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertSafetyOperatorSchema(t, pool)
	if _, err := pool.Exec(ctx, `
		INSERT INTO safety_incidents(id) VALUES ('10000000-0000-4000-8000-000000000001');
		INSERT INTO safety_operators(id,actor_id,role,created_at,updated_at)
		VALUES ('20000000-0000-4000-8000-000000000001','operator-1','helper',now(),now());
		INSERT INTO safety_operator_tokens(id,operator_id,token_digest,expires_at,created_at)
		VALUES (
			'30000000-0000-4000-8000-000000000001',
			'20000000-0000-4000-8000-000000000001',decode(repeat('ab',32),'hex'),
			now()+interval '1 hour',now()
		);
		INSERT INTO safety_operator_permissions(operator_id,permission,granted_by,granted_at)
		VALUES ('20000000-0000-4000-8000-000000000001','incident.readSafe','admin',now());
		INSERT INTO safety_operator_assignments(
			id,incident_id,operator_id,assignment_role,assigned_by,assigned_at
		) VALUES (
			'40000000-0000-4000-8000-000000000001',
			'10000000-0000-4000-8000-000000000001',
			'20000000-0000-4000-8000-000000000001','helper','admin',now()
		)
	`); err != nil {
		t.Fatalf("insert valid operator authorization state: %v", err)
	}
	assertImageSafetyStatementFails(t, pool, `
		INSERT INTO safety_operator_tokens(id,operator_id,token_digest,expires_at,created_at)
		VALUES (
			'30000000-0000-4000-8000-000000000002',
			'20000000-0000-4000-8000-000000000001',decode('abcd','hex'),
			now()+interval '1 hour',now()
		)
	`)
	assertImageSafetyStatementFails(t, pool, `
		INSERT INTO safety_operator_permissions(operator_id,permission,granted_by,granted_at)
		VALUES ('20000000-0000-4000-8000-000000000001','evidence.delete','admin',now())
	`)
	assertImageSafetyStatementFails(t, pool, `
		UPDATE safety_operators SET active=false WHERE id='20000000-0000-4000-8000-000000000001'
	`)
	assertImageSafetyStatementFails(t, pool, `
		DELETE FROM safety_operators WHERE id='20000000-0000-4000-8000-000000000001'
	`)

	apply("down", down)
	for _, table := range safetyOperatorTables() {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	apply("second up", up)
	assertSafetyOperatorSchema(t, pool)
}

func TestSafetyRetentionMigration(t *testing.T) {
	evidenceUp, err := os.ReadFile("../../migrations/000083_safety_evidence_holds.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/000088_safety_retention.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000088_safety_retention.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `CREATE TABLE safety_incidents(id UUID PRIMARY KEY);`)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(evidenceUp)); err != nil {
		t.Fatalf("apply safety evidence prerequisite: %v", err)
	}
	assertTriggerExists(t, pool, "safety_evidence_accesses_append_only")
	assertTriggerExists(t, pool, "safety_evidence_exports_append_only")
	assertAppendOnlyTable(t, pool, "safety_evidence_accesses")
	assertAppendOnlyTable(t, pool, "safety_evidence_exports")
	apply := func(label string, migration []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertSafetyRetentionSchema(t, pool)
	if _, err := pool.Exec(ctx, `
		INSERT INTO safety_retention_jobs(
			id,data_class,item_reference,state,attempt_count,max_attempts,created_at,updated_at
		) VALUES (
			'50000000-0000-4000-8000-000000000001','restrictedEvidence',
			'60000000-0000-4000-8000-000000000001','completed',1,5,now(),now()
		);
		INSERT INTO safety_retention_attempts(
			id,job_id,attempt_number,outcome,started_at,completed_at
		) VALUES (
			'70000000-0000-4000-8000-000000000001',
			'50000000-0000-4000-8000-000000000001',1,'succeeded',now(),now()
		);
		INSERT INTO safety_retention_events(id,data_class,item_reference,outcome,occurred_at)
		VALUES (
			'80000000-0000-4000-8000-000000000001','restrictedEvidence',
			'60000000-0000-4000-8000-000000000001','deleted',now()
		)
	`); err != nil {
		t.Fatalf("insert valid retention state: %v", err)
	}
	assertImageSafetyStatementFails(t, pool, `
		INSERT INTO safety_retention_jobs(
			id,data_class,item_reference,state,attempt_count,max_attempts,created_at,updated_at
		) VALUES (
			'50000000-0000-4000-8000-000000000002','restrictedEvidence',
			'60000000-0000-4000-8000-000000000002','retry',5,5,now(),now()
		)
	`)
	assertImageSafetyStatementFails(t, pool, `
		INSERT INTO safety_retention_attempts(
			id,job_id,attempt_number,outcome,started_at,completed_at
		) VALUES (
			'70000000-0000-4000-8000-000000000002',
			'50000000-0000-4000-8000-000000000001',11,'succeeded',now(),now()
		)
	`)
	assertImageSafetyStatementFails(t, pool, `
		UPDATE safety_retention_events SET outcome='failed'
		WHERE id='80000000-0000-4000-8000-000000000001'
	`)
	assertImageSafetyStatementFails(t, pool, `
		DELETE FROM safety_retention_events
		WHERE id='80000000-0000-4000-8000-000000000001'
	`)
	assertImageSafetyStatementFails(t, pool, `TRUNCATE safety_retention_events`)

	apply("down", down)
	for _, table := range []string{"safety_retention_attempts", "safety_retention_jobs"} {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	if _, err := pool.Exec(ctx, `
		UPDATE safety_retention_events SET outcome='failed'
		WHERE id='80000000-0000-4000-8000-000000000001'
	`); err != nil {
		t.Fatalf("append-only trigger remained after down migration: %v", err)
	}
	apply("second up", up)
	assertSafetyRetentionSchema(t, pool)
}

func safetyOperatorTables() []string {
	return []string{
		"safety_operators",
		"safety_operator_tokens",
		"safety_operator_permissions",
		"safety_operator_assignments",
	}
}

func assertSafetyOperatorSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range safetyOperatorTables() {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, constraint := range []string{
		"safety_operators_role_check",
		"safety_operators_values_check",
		"safety_operator_tokens_token_digest_check",
		"safety_operator_tokens_lifecycle_check",
		"safety_operator_permissions_permission_check",
		"safety_operator_permissions_values_check",
		"safety_operator_assignments_assignment_role_check",
		"safety_operator_assignments_values_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	var columns []string
	rows, err := pool.Query(context.Background(), `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema=current_schema() AND table_name='safety_operator_tokens'
		ORDER BY ordinal_position
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"token", "plaintext_token", "token_value"} {
		if slices.Contains(columns, forbidden) {
			t.Errorf("plaintext token column %q exists", forbidden)
		}
	}
	if !slices.Contains(columns, "token_digest") {
		t.Error("token_digest column missing")
	}
}

func assertSafetyRetentionSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range []string{"safety_retention_jobs", "safety_retention_attempts", "safety_retention_events"} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, constraint := range []string{
		"safety_retention_jobs_state_check",
		"safety_retention_jobs_lifecycle_check",
		"safety_retention_jobs_values_check",
		"safety_retention_attempts_attempt_number_check",
		"safety_retention_attempts_outcome_check",
		"safety_retention_attempts_values_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	for _, index := range []string{
		"safety_retention_jobs_claim_idx",
		"safety_retention_jobs_lease_idx",
		"safety_retention_jobs_dead_letter_idx",
		"safety_retention_attempts_job_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}
}

func assertSafetyIncidentSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range []string{"safety_incidents", "safety_incident_subjects", "safety_incident_events"} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, constraint := range []string{
		"safety_incidents_reference_check",
		"safety_incident_subjects_values_check",
		"safety_incident_events_event_type_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
}

func imageSafetyTables() []string {
	return []string{
		"image_scan_results",
		"image_scan_jobs",
		"image_blob_sources",
		"image_subject_requirements",
		"image_subject_states",
		"profile_image_candidates",
		"image_scan_events",
	}
}

func externalSafetyIntakeTables() []string {
	return []string{
		"external_safety_intakes",
		"external_safety_correspondence",
		"external_safety_appeal_links",
		"safety_workflows",
		"safety_workflow_events",
		"safety_authority_requests",
	}
}

func assertImageSafetySchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range imageSafetyTables() {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, constraint := range []string{
		"image_scan_results_state_check",
		"image_scan_jobs_state_check",
		"image_scan_jobs_version_check",
		"image_subject_states_visibility_state_check",
		"profile_image_candidates_slot_check",
		"image_scan_events_event_type_check",
		"tap_projection_jobs_dependency_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	for _, index := range []string{
		"image_scan_results_cache_key_idx",
		"image_scan_jobs_claim_idx",
		"image_scan_jobs_dead_letter_idx",
		"image_blob_sources_blob_idx",
		"image_subject_requirements_result_idx",
		"image_subject_states_visibility_idx",
		"profile_image_candidates_result_idx",
		"image_scan_events_result_created_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}
}

func assertImageSafetyClosedStates(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_scan_results(
			id,blob_cid,scanner_id,policy_version,corpus_version,state
		) VALUES (
			'10000000-0000-4000-8000-000000000001','bafy-image','stub','policy-1','corpus-1','pending'
		);
		INSERT INTO image_scan_jobs(id,scan_result_id,state)
		VALUES ('20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001','queued');
		INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
		VALUES ('at://did:plc:owner/social.craftsky.feed.post/safe','post','bafy-record','blocked');
		INSERT INTO profile_image_candidates(
			profile_did,slot,source_cid,candidate_blob_cid,declared_mime,scan_result_id
		) VALUES (
			'did:plc:owner','avatar','bafy-profile','bafy-image','image/jpeg',
			'10000000-0000-4000-8000-000000000001'
		);
		INSERT INTO image_scan_events(
			id,scan_result_id,scan_job_id,event_type,to_state
		) VALUES (
			'30000000-0000-4000-8000-000000000001',
			'10000000-0000-4000-8000-000000000001',
			'20000000-0000-4000-8000-000000000001','queued','pending'
		)
	`); err != nil {
		t.Fatalf("insert valid image safety states: %v", err)
	}

	assertImageSafetyStatementFails(t, pool, `
		INSERT INTO image_scan_results(
			id,blob_cid,scanner_id,policy_version,corpus_version,state
		) VALUES (
			'10000000-0000-4000-8000-000000000002','bafy-other','stub','policy-1','corpus-1','suspected'
		)
	`)
	assertImageSafetyStatementFails(t, pool, `
		INSERT INTO image_scan_jobs(id,scan_result_id,state)
		VALUES ('20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001','running')
	`)
	assertImageSafetyStatementFails(t, pool, `
		UPDATE image_scan_jobs SET scan_version=0
	`)
	assertImageSafetyStatementFails(t, pool, `
		INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
		VALUES ('at://did:plc:owner/social.craftsky.feed.post/unsafe','post','bafy-record','visible')
	`)
	assertImageSafetyStatementFails(t, pool, `
		INSERT INTO profile_image_candidates(
			profile_did,slot,source_cid,candidate_blob_cid,declared_mime,scan_result_id
		) VALUES (
			'did:plc:other','thumbnail','bafy-profile','bafy-image','image/jpeg',
			'10000000-0000-4000-8000-000000000001'
		)
	`)
}

func assertImageSubjectDependencyKind(t *testing.T, pool *pgxpool.Pool, sourceURI string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO tap_projection_jobs(
			source_uri,projection_kind,source_event_id,state,dependency_kind,dependency_key
		) VALUES ($1,'craftsky_post',1,'blocked','image_subject_uri',$1)
	`, sourceURI); err != nil {
		t.Fatalf("insert image subject dependency: %v", err)
	}
	assertImageSafetyStatementFails(t, pool, `
		UPDATE tap_projection_jobs SET dependency_kind = 'unknown_dependency'
	`)
}

func assertImageSafetyStatementFails(t *testing.T, pool *pgxpool.Pool, statement string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), statement); err == nil {
		t.Fatalf("constrained statement succeeded: %s", statement)
	}
}

func assertTriggerExists(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM pg_trigger WHERE tgname=$1 AND NOT tgisinternal)
	`, name).Scan(&exists); err != nil {
		t.Fatalf("inspect trigger %s: %v", name, err)
	}
	if !exists {
		t.Errorf("trigger %s missing", name)
	}
}

func assertAppendOnlyTable(t *testing.T, pool *pgxpool.Pool, table string) {
	t.Helper()
	for _, statement := range []string{
		"UPDATE " + table + " SET id=id WHERE false",
		"DELETE FROM " + table + " WHERE false",
		"TRUNCATE " + table,
	} {
		if _, err := pool.Exec(context.Background(), statement); err == nil {
			t.Errorf("append-only table %s allowed %q", table, statement)
		}
	}
}
