package index_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// IT-006 / FR-009, RULE-004 / AC-010, AC-017.
func TestBlueskyProfileGuestVisibilitySameCIDAndOrdering(t *testing.T) {
	for _, safe := range []bool{false, true} {
		name := "ordinary"
		if safe {
			name = "imageSafety"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			pool := testdb.WithMigratedSchema(t)
			did := syntax.DID("did:plc:guest-visibility")
			if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profiles(did,display_name,record_cid) VALUES ($1,'Legacy','same-cid')`, did); err != nil {
				t.Fatal(err)
			}
			idx := index.NewBlueskyProfile(pool)
			if safe {
				idx = index.NewImageSafetyBlueskyProfile(pool, imagesafety.ScanKey{ScannerID: "fixture", PolicyVersion: "policy", CorpusVersion: "corpus"})
			}
			ev := tap.Event{URI: syntax.ATURI("at://" + did.String() + "/app.bsky.actor.profile/self"), DID: did, Collection: "app.bsky.actor.profile", Rkey: "self", CID: "same-cid", Rev: "3m234567abc22", Action: "update", Record: json.RawMessage(`{"$type":"app.bsky.actor.profile","displayName":"Legacy"}`)}
			check := func(want *bool, revision string) {
				t.Helper()
				var hidden *bool
				var rev *string
				if err := pool.QueryRow(ctx, `SELECT guest_visibility_hidden,guest_visibility_revision FROM bluesky_profiles WHERE did=$1`, did).Scan(&hidden, &rev); err != nil {
					t.Fatal(err)
				}
				if (hidden == nil) != (want == nil) || (hidden != nil && *hidden != *want) || rev == nil || *rev != revision {
					t.Fatalf("hidden=%v revision=%v, want %v/%s", hidden, rev, want, revision)
				}
			}
			run := func() {
				t.Helper()
				if err := idx.Handle(ctx, ev); err != nil {
					t.Fatal(err)
				}
			}
			public, hidden := false, true
			run()
			check(&public, ev.Rev.String())
			ev.Rev = "3m234567abc23"
			ev.CID = "hidden-cid"
			ev.Record = json.RawMessage(`{"$type":"app.bsky.actor.profile","labels":{"$type":"com.atproto.label.defs#selfLabels","values":[{"val":"!no-unauthenticated"}]}}`)
			run()
			check(&hidden, ev.Rev.String())
			run()
			check(&hidden, ev.Rev.String())
			ev.Rev = "3m234567abc22"
			ev.CID = "same-cid"
			ev.Record = json.RawMessage(`{"$type":"app.bsky.actor.profile"}`)
			run()
			check(&hidden, "3m234567abc23")
			ev.Rev = "3m234567abc24"
			ev.Action = "delete"
			ev.CID = ""
			ev.Record = nil
			run()
			check(nil, ev.Rev.String())
			ev.Rev = "3m234567abc23"
			ev.Action = "update"
			ev.CID = "old-public-cid"
			ev.Record = json.RawMessage(`{"$type":"app.bsky.actor.profile"}`)
			run()
			check(nil, "3m234567abc24")
		})
	}
}

// IT-006 / RULE-004 / AC-010, AC-017: an owner observation fences older
// revisions without preventing the matching observed record's normal projection.
func TestBlueskyProfileObservedCIDCanProjectWithoutLoweringVisibilityFloor(t *testing.T) {
	for _, safe := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "imageSafety"}[safe], func(t *testing.T) {
			ctx := context.Background()
			pool := testdb.WithMigratedSchema(t)
			did := syntax.DID("did:plc:observed-profile")
			if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profiles(did,display_name,record_cid,guest_visibility_hidden,guest_visibility_revision,guest_visibility_record_cid,guest_visibility_evidence) VALUES ($1,'Old presentation','old-cid',false,'3m234567abc29','observed-cid','observation')`, did); err != nil {
				t.Fatal(err)
			}
			idx := index.NewBlueskyProfile(pool)
			if safe {
				idx = index.NewImageSafetyBlueskyProfile(pool, imagesafety.ScanKey{ScannerID: "fixture", PolicyVersion: "policy", CorpusVersion: "corpus"})
			}
			event := tap.Event{URI: syntax.ATURI("at://" + did.String() + "/app.bsky.actor.profile/self"), DID: did, Collection: "app.bsky.actor.profile", Rkey: "self", CID: "observed-cid", Rev: "3m234567abc22", Action: "update", Record: json.RawMessage(`{"$type":"app.bsky.actor.profile","displayName":"Current presentation"}`)}
			if err := idx.Handle(ctx, event); err != nil {
				t.Fatal(err)
			}
			var name, revision, cid string
			var hidden bool
			if err := pool.QueryRow(ctx, `SELECT display_name,guest_visibility_revision,guest_visibility_record_cid,guest_visibility_hidden FROM bluesky_profiles WHERE did=$1`, did).Scan(&name, &revision, &cid, &hidden); err != nil || name != "Current presentation" || revision != "3m234567abc29" || cid != "observed-cid" || hidden {
				t.Fatalf("matching source did not project behind observation floor: %s %s %s hidden=%v err=%v", name, revision, cid, hidden, err)
			}
		})
	}
}
