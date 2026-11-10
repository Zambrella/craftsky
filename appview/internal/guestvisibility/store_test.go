package guestvisibility_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/guestvisibility"
	"social.craftsky/appview/internal/testdb"
)

func TestGuestVisibilityConflictsPreserveExplicitOptOut(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	for i, tc := range []struct {
		name                 string
		initial, conflicting guestvisibility.Preference
		wantHidden           bool
	}{
		{"existing opt-out", guestvisibility.Hidden, guestvisibility.Public, true},
		{"incoming opt-out", guestvisibility.Public, guestvisibility.Hidden, true},
		{"unknown conflict", guestvisibility.Public, guestvisibility.Unknown, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := syntax.DID(fmt.Sprintf("did:plc:conflict%d", i))
			apply := func(preference guestvisibility.Preference, cid syntax.CID, revision syntax.TID) bool {
				t.Helper()
				tx, err := pool.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback(ctx)
				kind := guestvisibility.Source
				if cid == "initial-cid" {
					kind = guestvisibility.WriteReceipt
				}
				accepted, err := guestvisibility.ApplyTx(ctx, tx, owner, preference, guestvisibility.Evidence{Revision: revision, CID: cid, Kind: kind})
				if err != nil {
					t.Fatal(err)
				}
				if err := tx.Commit(ctx); err != nil {
					t.Fatal(err)
				}
				return accepted
			}
			if !apply(tc.initial, "initial-cid", "3m234567abc22") {
				t.Fatal("initial evidence rejected")
			}
			if apply(tc.conflicting, "conflicting-cid", "3m234567abc22") {
				t.Fatal("conflicting presentation accepted")
			}
			if apply(guestvisibility.Public, "stale-cid", "3m234567abc21") {
				t.Fatal("stale evidence accepted")
			}
			var hidden *bool
			var cid, kind *string
			if err := pool.QueryRow(ctx, `SELECT guest_visibility_hidden,guest_visibility_record_cid,guest_visibility_evidence FROM bluesky_profiles WHERE did=$1`, owner).Scan(&hidden, &cid, &kind); err != nil {
				t.Fatal(err)
			}
			if tc.wantHidden && (hidden == nil || !*hidden) {
				t.Fatal("conflict lost explicit opt-out")
			}
			if tc.initial == guestvisibility.Hidden && (cid == nil || *cid != "initial-cid" || kind == nil || *kind != string(guestvisibility.WriteReceipt)) {
				t.Fatalf("conflict replaced opt-out provenance: cid=%v kind=%v", cid, kind)
			}
			if !tc.wantHidden && hidden != nil {
				t.Fatalf("non-opt-out conflict must remain unknown: %v", *hidden)
			}
			if !apply(guestvisibility.Unknown, "newer-cid", "3m234567abc23") {
				t.Fatal("newer unknown evidence rejected")
			}
			if err := pool.QueryRow(ctx, `SELECT guest_visibility_hidden FROM bluesky_profiles WHERE did=$1`, owner).Scan(&hidden); err != nil || hidden != nil {
				t.Fatalf("newer unknown state=%v err=%v", hidden, err)
			}
		})
	}
}
