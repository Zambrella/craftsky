// appview/internal/index/craftsky_profile_test.go
package index_test

import (
	"context"
	"encoding/json"
	"testing"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
)

const craftskyProfilesDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE bluesky_profiles (
    did          TEXT        NOT NULL PRIMARY KEY,
    display_name TEXT,
    description  TEXT,
    avatar_cid   TEXT,
    avatar_mime  TEXT,
    banner_cid   TEXT,
    banner_mime  TEXT,
    record_cid   TEXT        NOT NULL,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func TestCraftskyProfile_Create(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool)

	ev := tap.Event{
		URI:        "at://did:plc:abc/social.craftsky.actor.profile/self",
		CID:        "bafy1",
		DID:        "did:plc:abc",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["knitting","sewing"]}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var crafts []string
	var cid string
	err := pool.QueryRow(context.Background(),
		`SELECT crafts, record_cid FROM craftsky_profiles WHERE did = $1`, ev.DID).
		Scan(&crafts, &cid)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if cid != "bafy1" {
		t.Errorf("record_cid = %q, want bafy1", cid)
	}
	if len(crafts) != 2 || crafts[0] != "knitting" || crafts[1] != "sewing" {
		t.Errorf("crafts = %v, want [knitting sewing]", crafts)
	}
}
