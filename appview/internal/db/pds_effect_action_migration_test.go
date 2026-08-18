package db_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"social.craftsky/appview/internal/testdb"
)

func TestPDSEffectActionMigrationSupportsRecordVersionsUpDownUp(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000049_pds_effect_action.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000049_pds_effect_action.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_effect_attempts (
			operation_id TEXT PRIMARY KEY,
			owner_did TEXT NOT NULL,
			owner_generation BIGINT NOT NULL,
			effect_kind TEXT NOT NULL CHECK(effect_kind IN ('pds_record','object_put','object_delete')),
			deterministic_key TEXT NOT NULL,
			request_fingerprint BYTEA NOT NULL CHECK(octet_length(request_fingerprint)=32),
			CONSTRAINT owner_effect_attempts_remote_identity_key
				UNIQUE(owner_did,owner_generation,effect_kind,deterministic_key)
		);
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,deterministic_key,request_fingerprint
		) VALUES(
			'legacy-put','did:plc:effect-action',2,'pds_record',
			'at://did:plc:effect-action/social.craftsky.actor.profile/self',
			decode(repeat('01',32),'hex')
		);
	`)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply action up: %v", err)
	}
	var action, mutationKey string
	if err := pool.QueryRow(ctx, `
		SELECT effect_action,mutation_key
		FROM owner_effect_attempts WHERE operation_id='legacy-put'
	`).Scan(&action, &mutationKey); err != nil {
		t.Fatal(err)
	}
	if action != "put_record" || mutationKey != "legacy-put" {
		t.Fatalf("legacy action/key = %q/%q, want put_record/legacy-put", action, mutationKey)
	}
	exactURI := "at://did:plc:effect-action/social.craftsky.actor.profile/self"
	for _, insert := range []struct {
		operationID string
		action      string
		mutationKey string
		fingerprint string
	}{
		{operationID: "updated-put", action: "put_record", mutationKey: "profile-v2", fingerprint: "02"},
		{operationID: "returned-put", action: "put_record", mutationKey: "profile-v3", fingerprint: "01"},
		{operationID: "later-delete", action: "delete_record", mutationKey: "profile-v4", fingerprint: "03"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO owner_effect_attempts(
				operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,
				deterministic_key,request_fingerprint
			) VALUES($1,'did:plc:effect-action',2,'pds_record',$2,$3,$4,decode(repeat($5,32),'hex'))
		`, insert.operationID, insert.action, insert.mutationKey, exactURI, insert.fingerprint); err != nil {
			t.Fatalf("insert %s: %v", insert.operationID, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,
			deterministic_key,request_fingerprint
		) VALUES(
			'exact-duplicate','did:plc:effect-action',2,'pds_record','put_record','profile-v2',$1,
			decode(repeat('09',32),'hex')
		)
	`, exactURI); !isUniqueViolation(err) {
		t.Fatalf("exact duplicate error = %v, want unique violation", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,
			deterministic_key,request_fingerprint
		) VALUES(
			'invalid-pair','did:plc:effect-action',2,'pds_record','upload_blob','invalid-pair','blob',
			decode(repeat('04',32),'hex')
		)
	`); !isCheckViolation(err) {
		t.Fatalf("invalid action/kind error = %v, want check violation", err)
	}

	// A down migration can restore the legacy uniqueness only when no data
	// relies on the new version identity. Production rollback must therefore
	// be preceded by a compatibility audit; this pre-production test removes
	// the deliberately incompatible rows first.
	if _, err := pool.Exec(ctx, `DELETE FROM owner_effect_attempts WHERE operation_id<>'legacy-put'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply action down: %v", err)
	}
	var actionExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.columns
			WHERE table_schema=current_schema() AND table_name='owner_effect_attempts'
			  AND column_name='effect_action'
		)
	`).Scan(&actionExists); err != nil {
		t.Fatal(err)
	}
	if actionExists {
		t.Fatal("effect_action remained after down")
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply action second up: %v", err)
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return err != nil && errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return err != nil && errors.As(err, &pgErr) && pgErr.Code == "23514"
}
