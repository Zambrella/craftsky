package api_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestAccountTypeSelectionWithoutDeclaration(t *testing.T) {
	accountMigration, err := os.ReadFile("../../migrations/000061_business_account_types.up.sql")
	if err != nil {
		t.Fatalf("read account migration: %v", err)
	}
	recordMigration, err := os.ReadFile("../../migrations/000062_business_records.up.sql")
	if err != nil {
		t.Fatalf("read record migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY, state TEXT NOT NULL, generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL, transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL, terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
	`+string(accountMigration)+string(recordMigration))
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
		VALUES ($1,'active',1,1,'test',now(),now(),now())
	`, alice); err != nil {
		t.Fatalf("seed lifecycle: %v", err)
	}
	store := business.NewStore(pool)
	lifecycles := businessLifecycleReader{alice: {Owner: alice, State: ownerlifecycle.StateActive, Generation: 1}}
	handler := businessAccountTypeHandler(store, alice, lifecycles)
	hydrator := api.NewIdentityAccountTypeHydrator(store)
	raw := []byte(`{"profile":{"did":"did:plc:alice","handle":"alice.test"},"items":[{"author":{"did":"did:plc:alice","handle":"alice.test"}}]}`)

	assertHydratedAccountTypes(t, hydrator, raw, "regular")
	response := serveBusinessAccountType(handler, `{"accountType":"business"}`, true)
	if response.Code != 200 {
		t.Fatalf("set business status=%d body=%s", response.Code, response.Body.String())
	}
	assertHydratedAccountTypes(t, hydrator, raw, "business")
	response = serveBusinessAccountType(handler, `{"accountType":"regular"}`, true)
	if response.Code != 200 {
		t.Fatalf("set regular status=%d body=%s", response.Code, response.Body.String())
	}
	assertHydratedAccountTypes(t, hydrator, raw, "regular")

	var declarations int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_business_profiles WHERE owner_did=$1`, alice).Scan(&declarations); err != nil || declarations != 0 {
		t.Fatalf("declarations=%d error=%v, want none", declarations, err)
	}
}

func assertHydratedAccountTypes(t *testing.T, hydrator *api.IdentityAccountTypeHydrator, raw []byte, want string) {
	t.Helper()
	hydrated, err := hydrator.HydrateJSON(context.Background(), raw)
	if err != nil {
		t.Fatalf("hydrate account types: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(hydrated, &body); err != nil {
		t.Fatalf("decode hydration: %v", err)
	}
	profile := body["profile"].(map[string]any)
	author := body["items"].([]any)[0].(map[string]any)["author"].(map[string]any)
	if profile["accountType"] != want || author["accountType"] != want {
		t.Fatalf("account types profile/author=%v/%v, want %q", profile["accountType"], author["accountType"], want)
	}
}
