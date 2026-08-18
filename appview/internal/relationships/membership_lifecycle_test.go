package relationships

import (
	"context"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/testdb"
)

func TestStoreCurrentMemberRequiresActiveLifecycle(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles(did TEXT PRIMARY KEY);
		CREATE TABLE owner_lifecycles(
			owner_did TEXT PRIMARY KEY,
			state TEXT NOT NULL
		);
		CREATE FUNCTION appview_owner_is_active(candidate_did TEXT)
		RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
			SELECT COALESCE((
				SELECT state='active' FROM owner_lifecycles WHERE owner_did=candidate_did
			), false)
		$$;
	`)
	ctx := context.Background()
	owner := syntax.DID("did:plc:membership-lifecycle")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did) VALUES($1)`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(owner_did,state) VALUES($1,'departed')
	`, owner); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	for _, state := range []string{"departed", "deletion_pending", "deleting", "terminal"} {
		if _, err := pool.Exec(ctx, `UPDATE owner_lifecycles SET state=$2 WHERE owner_did=$1`, owner, state); err != nil {
			t.Fatal(err)
		}
		current, err := store.IsCurrentMember(ctx, owner)
		if err != nil {
			t.Fatal(err)
		}
		if current {
			t.Fatalf("lifecycle state %q remained a current member", state)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE owner_lifecycles SET state='active' WHERE owner_did=$1`, owner); err != nil {
		t.Fatal(err)
	}
	current, err := store.IsCurrentMember(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if !current {
		t.Fatal("active lifecycle with a profile was not a current member")
	}
}
