package index

import (
	"context"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestProjectionMemberReadyPermanentlyRejectsRetainedTerminalProfile(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles(did TEXT PRIMARY KEY)
	`)
	ctx := context.Background()
	terminal := syntax.DID("did:plc:terminal-projection")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did) VALUES($1)`, terminal); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION appview_owner_is_terminal(candidate_did TEXT)
		RETURNS BOOLEAN
		LANGUAGE SQL
		IMMUTABLE
		AS $$ SELECT candidate_did = 'did:plc:terminal-projection' $$
	`); err != nil {
		t.Fatal(err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	outcome, ready, err := projectionMemberReady(ctx, tx, terminal)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("retained terminal profile was projection-ready")
	}
	if outcome.Kind != tap.OutcomePermanentInvalid || outcome.Reason != tap.ReasonOwnerTerminal {
		t.Fatalf("outcome=%+v, want permanent owner_terminal", outcome)
	}
}
