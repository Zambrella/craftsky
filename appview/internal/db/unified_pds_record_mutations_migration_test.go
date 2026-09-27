package db_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestUnifiedPDSRecordMutationsMigrationFilesExist(t *testing.T) {
	for _, name := range []string{
		"000073_unified_pds_record_mutations.up.sql",
		"000073_unified_pds_record_mutations.down.sql",
	} {
		if _, err := testdb.ReadMigration(name); err != nil {
			t.Errorf("read migration %s: %v", name, err)
		}
	}
}

func TestUnifiedPDSRecordMutationsMigrationRejectsPublicRecordEffectsAndKeepsObjectEffects(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	const owner = "did:plc:unifiedcommands"
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES ($1,'active',1,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatalf("insert owner lifecycle: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,
			mutation_key,deterministic_key,request_fingerprint,remote_deadline
		) VALUES (
			'legacy-public-command',$1,1,'pds_record','delete_record',
			'legacy-public-command','at://did:plc:unifiedcommands/social.craftsky.feed.post/one',
			decode(repeat('11',32),'hex'),now() + interval '1 hour'
		)
	`, owner); err == nil {
		t.Fatal("owner_effect_attempts accepted a new public PDS record effect")
	}

	for _, effect := range []struct {
		operationID string
		kind        string
		action      string
	}{
		{operationID: "retained-object-put", kind: "object_put", action: "upload_blob"},
		{operationID: "retained-object-delete", kind: "object_delete", action: "delete_object"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO owner_effect_attempts(
				operation_id,owner_did,owner_generation,effect_kind,effect_action,
				mutation_key,deterministic_key,request_fingerprint,remote_deadline
			) VALUES ($1,$2,1,$3,$4,$1,$1,decode(repeat('22',32),'hex'),now() + interval '1 hour')
		`, effect.operationID, owner, effect.kind, effect.action); err != nil {
			t.Errorf("owner_effect_attempts rejected retained %s/%s effect: %v", effect.kind, effect.action, err)
		}
	}
}

func TestUnifiedPDSRecordMutationsMigrationCreatesOwnershipBaseline(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)

	for _, table := range []string{
		"pds_set_sources",
		"pds_set_aggregates",
		"pds_commands",
		"pds_command_steps",
		"pds_command_dispatches",
		"pds_command_tombstones",
	} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, column := range []string{
		"validation_version",
		"structural_validation_status",
		"semantic_validation_status",
		"validation_reason",
	} {
		if !columnExists(t, pool, "tap_source_records", column) {
			t.Errorf("tap_source_records column %s missing", column)
		}
	}
}

func TestUnifiedPDSRecordMutationsMigrationConstrainsAggregateRepresentativeMembership(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	if !constraintExists(t, pool, "pds_set_aggregates_representative_fkey") {
		t.Fatal("pds_set_aggregates representative is not constrained to a matching normalized source")
	}
}

func TestUnifiedPDSRecordMutationsMigrationBoundsSetSourceIneligibilityData(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	const sourceURI = "at://did:plc:unifiedfacts/app.bsky.graph.follow/one"
	if _, err := pool.Exec(ctx, `
		INSERT INTO tap_source_records(
			uri,did,collection,rkey,source_event_id,source_fingerprint,
			revision,cid,action,record,record_bytes,live,ordering_status,
			projection_disposition
		) VALUES ($1,'did:plc:unifiedfacts','app.bsky.graph.follow','one',1,
			decode(repeat('22',32),'hex'),'1','bafyfact','create','{}',2,true,
			'authoritative','eligible')
	`, sourceURI); err != nil {
		t.Fatalf("insert source record: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_set_sources(
			source_uri,kind,actor_did,scope_key,subject_did,activity_at,
			eligible,ineligibility_reason
		) VALUES ($1,'follow','did:plc:unifiedfacts','did:plc:target',
			'did:plc:target',now(),false,$2)
	`, sourceURI, strings.Repeat("x", 129)); err == nil {
		t.Fatal("pds_set_sources accepted an unbounded ineligibility reason")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_set_sources(
			source_uri,kind,actor_did,scope_key,subject_did,activity_at,
			eligible,ineligibility_reason,dependency_kind,dependency_key
		) VALUES ($1,'follow','did:plc:unifiedfacts','did:plc:target',
			'did:plc:target',now(),false,'missing_dependency','member_did',$2)
	`, sourceURI, strings.Repeat("x", 2049)); err == nil {
		t.Fatal("pds_set_sources accepted an unbounded dependency key")
	}
}

func TestUnifiedPDSRecordMutationsMigrationKeepsRawSourceStateOutOfSetTables(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	for _, column := range []string{
		"revision",
		"action",
		"record",
		"record_bytes",
		"source_event_id",
		"source_fingerprint",
		"ordering_status",
		"validation_version",
		"structural_validation_status",
		"semantic_validation_status",
		"validation_reason",
	} {
		if !columnExists(t, pool, "tap_source_records", column) {
			t.Errorf("tap_source_records does not own raw source column %s", column)
		}
		for _, table := range []string{"pds_set_sources", "pds_set_aggregates"} {
			if columnExists(t, pool, table, column) {
				t.Errorf("%s duplicates raw source column %s", table, column)
			}
		}
	}
}

func TestUnifiedPDSRecordMutationsMigrationCreatesRequiredConstraintsAndIndexes(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	for _, constraint := range []string{
		"pds_set_sources_representative_key",
		"pds_set_sources_subject_check",
		"pds_set_sources_eligibility_check",
		"pds_set_sources_dependency_check",
		"pds_set_aggregates_pkey",
		"pds_set_aggregates_representative_fkey",
		"pds_commands_owner_did_operation_kind_operation_key_key",
		"pds_commands_terminal_check",
		"pds_commands_ambiguity_check",
		"pds_command_steps_pkey",
		"pds_command_steps_record_check",
		"pds_command_dispatches_command_id_attempt_ordinal_key",
		"pds_command_dispatches_completion_check",
		"pds_command_tombstones_pkey",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	for _, index := range []string{
		"pds_set_sources_scope_idx",
		"pds_set_sources_dependency_idx",
		"pds_set_sources_actor_purge_idx",
		"pds_set_sources_subject_did_purge_idx",
		"pds_set_aggregates_actor_purge_idx",
		"pds_set_aggregates_subject_did_purge_idx",
		"pds_set_aggregates_subject_uri_idx",
		"pds_set_aggregates_follow_subject_pagination_idx",
		"pds_set_aggregates_follow_actor_pagination_idx",
		"pds_set_aggregates_block_actor_pagination_idx",
		"pds_commands_owner_purge_idx",
		"pds_commands_expected_owner_purge_idx",
		"pds_commands_expected_target_purge_idx",
		"pds_commands_recovery_idx",
		"pds_commands_retention_idx",
		"pds_command_steps_uri_idx",
		"pds_command_steps_repo_purge_idx",
		"pds_command_dispatches_command_idx",
		"pds_command_tombstones_owner_purge_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}
}

func TestUnifiedPDSRecordMutationsMigrationCreatesBlockListPaginationIndex(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	var definition, predicate string
	if err := pool.QueryRow(context.Background(), `
		SELECT pg_get_indexdef(i.indexrelid), pg_get_expr(i.indpred, i.indrelid)
		FROM pg_index i
		JOIN pg_class index_class ON index_class.oid = i.indexrelid
		JOIN pg_namespace namespace ON namespace.oid = index_class.relnamespace
		WHERE namespace.nspname = current_schema()
		  AND index_class.relname = 'pds_set_aggregates_block_actor_pagination_idx'
	`).Scan(&definition, &predicate); err != nil {
		t.Fatalf("read block pagination index: %v", err)
	}
	if !strings.Contains(definition, "(actor_did, activated_at DESC, subject_did DESC)") {
		t.Fatalf("block pagination index definition = %s", definition)
	}
	if !strings.Contains(predicate, "kind") || !strings.Contains(predicate, "'block'") {
		t.Fatalf("block pagination index predicate = %s", predicate)
	}
}

func TestUnifiedPDSRecordMutationsMigrationMovesFollowReadStructuresToAggregates(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	definition := viewDefinition(t, pool, "craftsky_profile_follower_counts")
	if !strings.Contains(definition, "pds_set_aggregates") || !strings.Contains(definition, "kind = 'follow'") {
		t.Fatalf("follower-count view does not read follow aggregates: %s", definition)
	}
	if strings.Contains(definition, "atproto_follows") {
		t.Fatalf("follower-count view still reads legacy follows: %s", definition)
	}

	for index, columns := range map[string]string{
		"pds_set_aggregates_follow_subject_pagination_idx": "(subject_did, activated_at DESC, representative_source_uri DESC)",
		"pds_set_aggregates_follow_actor_pagination_idx":   "(actor_did, activated_at DESC, representative_source_uri DESC)",
	} {
		var definition, predicate string
		if err := pool.QueryRow(context.Background(), `
			SELECT pg_get_indexdef(i.indexrelid), pg_get_expr(i.indpred, i.indrelid)
			FROM pg_index i
			JOIN pg_class index_class ON index_class.oid = i.indexrelid
			JOIN pg_namespace namespace ON namespace.oid = index_class.relnamespace
			WHERE namespace.nspname = current_schema() AND index_class.relname = $1
		`, index).Scan(&definition, &predicate); err != nil {
			t.Fatalf("read index %s: %v", index, err)
		}
		if !strings.Contains(definition, columns) {
			t.Errorf("index %s definition = %s, want columns %s", index, definition, columns)
		}
		if !strings.Contains(predicate, "kind") || !strings.Contains(predicate, "'follow'") {
			t.Errorf("index %s predicate = %s, want follow-only predicate", index, predicate)
		}
	}
}

func TestUnifiedPDSRecordMutationsMigrationSeparatesCommandPersistenceBoundaries(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	wantColumns := map[string][]string{
		"pds_commands": {
			"id", "owner_did", "owner_generation", "operation_kind", "operation_key",
			"request_fingerprint_version", "request_fingerprint", "immutable_request",
			"selected_uri", "selected_rkey", "expected_owner_did", "expected_owner_generation",
			"expected_target_did", "active_plan_version", "state", "terminal_http_status",
			"terminal_response_body", "terminal_response_headers", "retry_after_seconds",
			"replay_expires_at", "created_at", "updated_at", "terminal_at",
		},
		"pds_command_steps": {
			"command_id", "plan_version", "ordinal", "action", "repo_did", "collection",
			"rkey", "record", "expected_cid", "selected_uri", "generated_values",
		},
		"pds_command_dispatches": {
			"id", "command_id", "attempt_ordinal", "plan_version", "dispatch_fingerprint_version",
			"dispatch_fingerprint", "repository_cid", "repository_revision", "remote_deadline", "outcome",
			"error_class", "result_commit_cid", "result_records", "started_at", "completed_at",
		},
		"pds_command_tombstones": {
			"owner_did", "operation_kind", "scoped_key_hash", "compacted_at",
		},
	}
	for table, want := range wantColumns {
		if got := tableColumns(t, pool, table); !slices.Equal(got, want) {
			t.Errorf("%s columns = %v, want %v", table, got, want)
		}
	}
}

func TestUnifiedPDSRecordMutationsMigrationUpDownUp(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	down, err := testdb.ReadMigration("000073_unified_pds_record_mutations.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := testdb.ReadMigration("000073_unified_pds_record_mutations.up.sql")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply migration 73 down: %v", err)
	}
	if !tableExists(t, pool, "pds_follow_operations") {
		t.Fatal("migration 73 down did not restore pds_follow_operations")
	}
	if columnExists(t, pool, "tap_source_records", "validation_version") {
		t.Fatal("migration 73 down retained validation columns")
	}
	legacyView := viewDefinition(t, pool, "craftsky_profile_follower_counts")
	if !strings.Contains(legacyView, "atproto_follows") || strings.Contains(legacyView, "pds_set_aggregates") {
		t.Fatalf("migration 73 down did not restore legacy follower-count view: %s", legacyView)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("reapply migration 73: %v", err)
	}
	if tableExists(t, pool, "pds_follow_operations") {
		t.Fatal("reapplied migration 73 retained pds_follow_operations")
	}
	aggregateView := viewDefinition(t, pool, "craftsky_profile_follower_counts")
	if !strings.Contains(aggregateView, "pds_set_aggregates") || strings.Contains(aggregateView, "atproto_follows") {
		t.Fatalf("reapplied migration 73 did not restore aggregate follower-count view: %s", aggregateView)
	}
}

func TestUnifiedPDSRecordMutationsMigrationRemovesRelationshipIntentTable(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	if tableExists(t, pool, "pds_follow_operations") {
		t.Fatal("legacy pds_follow_operations table still exists")
	}
}

func tableColumns(t *testing.T, pool *pgxpool.Pool, table string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema=current_schema() AND table_name=$1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		t.Fatalf("query %s columns: %v", table, err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("scan %s column: %v", table, err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate %s columns: %v", table, err)
	}
	return columns
}

func viewDefinition(t *testing.T, pool *pgxpool.Pool, view string) string {
	t.Helper()
	var definition string
	if err := pool.QueryRow(context.Background(), `
		SELECT pg_get_viewdef(to_regclass(current_schema() || '.' || $1), true)
	`, view).Scan(&definition); err != nil {
		t.Fatalf("read view %s definition: %v", view, err)
	}
	return definition
}
