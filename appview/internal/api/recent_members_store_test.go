package api_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func recentDID(letter string) syntax.DID {
	return syntax.DID("did:plc:" + strings.Repeat("a", 23) + letter)
}
func recentSeed(t *testing.T, pool *pgxpool.Pool, did syntax.DID, joined time.Time) {
	t.Helper()
	seedMember(t, pool, did.String())
	if _, err := pool.Exec(t.Context(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now()) ON CONFLICT(owner_did) DO NOTHING`, did); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO account_onboarding_completions(account_did,completed_at) VALUES($1,$2)`, did, joined); err != nil {
		t.Fatal(err)
	}
}
func recentRead(t *testing.T, pool *pgxpool.Pool, viewer syntax.DID, want ...syntax.DID) {
	t.Helper()
	rows, err := api.NewProfileStore(pool).ListRecentMembers(t.Context(), viewer)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]syntax.DID, len(rows))
	for i, row := range rows {
		got[i] = row.DID
	}
	if !slices.Equal(got, want) {
		t.Fatalf("members = %v, want %v", got, want)
	}
}

// IT-001 / BR-002, RULE-001 / AC-006: independent cutoff tie oracle.
func TestRecentMembersOrdering(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	for _, letter := range []string{"f", "e", "d", "c", "b", "a"} {
		hours := map[string]int{"a": 5, "b": 4, "c": 3, "d": 2, "e": 1, "f": 1}[letter]
		recentSeed(t, pool, recentDID(letter), base.Add(time.Duration(hours)*time.Hour))
	}
	for range 3 {
		recentRead(t, pool, recentDID("v"), recentDID("a"), recentDID("b"), recentDID("c"), recentDID("d"), recentDID("e"))
	}
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_profiles SET indexed_at=now(),created_at=now(),crafts=ARRAY['sewing'],record_cid='edited' WHERE did=$1`, recentDID("f")); err != nil {
		t.Fatal(err)
	}
	recentRead(t, pool, recentDID("v"), recentDID("a"), recentDID("b"), recentDID("c"), recentDID("d"), recentDID("e"))
}

func recentState(t *testing.T, pool *pgxpool.Pool, did syntax.DID, state string) {
	t.Helper()
	if state == "unknown" {
		if _, err := pool.Exec(t.Context(), `DELETE FROM owner_lifecycles WHERE owner_did=$1`, did); err != nil {
			t.Fatal(err)
		}
		return
	}
	if _, err := pool.Exec(t.Context(), `UPDATE owner_lifecycles SET state=$2,generation=generation+1,auth_epoch=auth_epoch+1,transition_reason='test',transitioned_at=now(),updated_at=now(),terminal_at=CASE WHEN $2='terminal' THEN now() ELSE NULL END WHERE owner_did=$1`, did, state); err != nil {
		t.Fatal(err)
	}
}
func recentModeration(t *testing.T, pool *pgxpool.Pool, did syntax.DID, value, action string, expired bool) {
	t.Helper()
	var expiry any
	if expired {
		expiry = time.Now().Add(-time.Hour)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,value,action,expires_at,indexed_at,internal_reason) VALUES($1,$2,'account',$3,$4,$5,$6,now(),'private-moderation-canary')`, did.String()+value+action, recentDID("m"), did, value, action, expiry); err != nil {
		t.Fatal(err)
	}
}

// IT-002 / FR-005, RULE-002 / AC-005, AC-007.
func TestRecentMembersCardinality(t *testing.T) {
	for _, count := range []int{0, 1, 4, 5, 6} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			pool := testdb.WithMigratedSchema(t)
			want := []syntax.DID{}
			for i := range count {
				did := recentDID(string(rune('a' + i)))
				recentSeed(t, pool, did, time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC).Add(-time.Duration(i)*time.Hour))
				if i < 5 {
					want = append(want, did)
				}
			}
			recentRead(t, pool, recentDID("v"), want...)
		})
	}
}
func TestRecentMembersExclusionsBeforeLimit(t *testing.T) {
	kinds := []string{"viewer", "blocked", "muted", "blocked-by", "hide", "takedown", "departed", "deletion_pending", "deleting", "terminal", "unknown"}
	for _, kind := range append(append([]string{}, kinds...), "combined", "all-excluded") {
		t.Run(kind, func(t *testing.T) {
			pool := testdb.WithMigratedSchema(t)
			base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
			viewer := recentDID("v")
			recentSeed(t, pool, viewer, base.Add(100*time.Hour))
			wanted := []syntax.DID{}
			if kind != "all-excluded" {
				for i := range 6 {
					did := recentDID(string(rune('a' + i)))
					recentSeed(t, pool, did, base.Add(-time.Duration(i)*time.Hour))
					if i < 5 {
						wanted = append(wanted, did)
					}
				}
			}
			selected := []string{kind}
			if kind == "combined" || kind == "all-excluded" {
				selected = kinds
			}
			for i, k := range selected {
				if k == "viewer" {
					continue
				}
				did := syntax.DID("did:plc:" + strings.Repeat("b", 23) + string(rune('a'+i)))
				recentSeed(t, pool, did, base.Add(50*time.Hour))
				switch k {
				case "blocked":
					seedSearchBlock(t, pool, viewer.String(), did.String())
				case "blocked-by":
					seedSearchBlock(t, pool, did.String(), viewer.String())
				case "muted":
					if _, err := pool.Exec(t.Context(), `INSERT INTO actor_mutes(owner_did,subject_did) VALUES($1,$2)`, viewer, did); err != nil {
						t.Fatal(err)
					}
				case "hide", "takedown":
					recentModeration(t, pool, did, k, "apply", false)
				default:
					recentState(t, pool, did, k)
				}
			}
			recentRead(t, pool, viewer, wanted...)
		})
	}
}
func TestRecentMembersVisibleModerationControls(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	for _, letter := range []string{"a", "b", "c"} {
		recentSeed(t, pool, recentDID(letter), base)
	}
	recentModeration(t, pool, recentDID("a"), "hide", "apply", true)
	recentModeration(t, pool, recentDID("b"), "takedown", "apply", false)
	recentModeration(t, pool, recentDID("b"), "takedown", "negate", false)
	recentModeration(t, pool, recentDID("c"), "warn", "apply", false)
	recentRead(t, pool, recentDID("v"), recentDID("a"), recentDID("b"), recentDID("c"))
}

// IT-003 / FR-007, RULE-002 / AC-007, AC-009: no viewer-crossing cache.
func TestRecentMembersViewerSpecificReads(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	a, b := recentDID("v"), recentDID("w")
	for _, viewer := range []syntax.DID{a, b} {
		recentSeed(t, pool, viewer, base)
		if _, err := pool.Exec(t.Context(), `DELETE FROM account_onboarding_completions WHERE account_did=$1`, viewer); err != nil {
			t.Fatal(err)
		}
	}
	for i, letter := range []string{"a", "b", "c", "d"} {
		recentSeed(t, pool, recentDID(letter), base.Add(-time.Duration(i)*time.Hour))
	}
	seedSearchBlock(t, pool, a.String(), recentDID("a").String())
	seedSearchBlock(t, pool, b.String(), recentDID("b").String())
	seedSearchBlock(t, pool, recentDID("d").String(), a.String())
	seedSearchSet(t, pool, "follow", b.String(), recentDID("c").String(), true)
	if _, err := pool.Exec(t.Context(), `INSERT INTO actor_mutes(owner_did,subject_did) VALUES($1,$2)`, a, recentDID("c")); err != nil {
		t.Fatal(err)
	}
	recentRead(t, pool, a, recentDID("b"))
	recentRead(t, pool, b, recentDID("a"), recentDID("c"), recentDID("d"))
	recentRead(t, pool, a, recentDID("b"))
	if _, err := pool.Exec(t.Context(), `DELETE FROM actor_mutes WHERE owner_did=$1`, a); err != nil {
		t.Fatal(err)
	}
	recentRead(t, pool, a, recentDID("b"), recentDID("c"))
	if _, err := pool.Exec(t.Context(), `DELETE FROM pds_set_aggregates WHERE kind='block' AND actor_did=$1`, a); err != nil {
		t.Fatal(err)
	}
	recentRead(t, pool, a, recentDID("a"), recentDID("b"), recentDID("c"))
}

// IT-004 / RULE-001 / AC-006: retained metadata through real existing paths.
func TestRecentMembersRetainedJoinLifecycle(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	old, newer, missing := recentDID("a"), recentDID("b"), recentDID("c")
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	recentSeed(t, pool, old, base)
	recentSeed(t, pool, newer, base.Add(time.Hour))
	recentSeed(t, pool, missing, base.Add(2*time.Hour))
	if _, err := pool.Exec(t.Context(), `DELETE FROM account_onboarding_completions WHERE account_did=$1`, missing); err != nil {
		t.Fatal(err)
	}
	status, err := api.NewOnboardingStatusStore(pool).Complete(ownerlifecycle.WithExpectedGeneration(t.Context(), 1), old, api.OnboardingAcceptance{MeetsMinimumAge: true, PolicyVersion: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if status.CompletedAt == nil || !status.CompletedAt.Equal(base) {
		t.Fatalf("repeated completion = %v, want %v", status.CompletedAt, base)
	}
	idx := index.NewCraftskyProfile(pool, nil, nil)
	for _, cid := range []syntax.CID{"profile-edit", "profile-reindex", "profile-reindex"} {
		if err := idx.Handle(t.Context(), tap.Event{DID: old, URI: syntax.ATURI("at://" + old.String() + "/social.craftsky.actor.profile/self"), Collection: "social.craftsky.actor.profile", Action: "update", CID: cid, Record: []byte(`{"crafts":["sewing"]}`)}); err != nil {
			t.Fatal(err)
		}
		recentRead(t, pool, recentDID("v"), newer, old)
	}
	recentState(t, pool, old, "departed")
	recentRead(t, pool, recentDID("v"), newer)
	recentState(t, pool, old, "active")
	recentRead(t, pool, recentDID("v"), newer, old)
	recentState(t, pool, old, "deletion_pending")
	recentRead(t, pool, recentDID("v"), newer)
	recentState(t, pool, old, "deleting")
	recentRead(t, pool, recentDID("v"), newer)
	if _, err := pool.Exec(t.Context(), `INSERT INTO account_deletion_operations(id,owner_did,owner_generation,state,accepted_at) SELECT '00000000-0000-4000-8000-000000000999',owner_did,generation,'active',now() FROM owner_lifecycles WHERE owner_did=$1`, old); err != nil {
		t.Fatal(err)
	}
	if err := accountdeletion.NewDatabasePrivateCleanup(pool).Purge(t.Context(), old); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM account_onboarding_completions WHERE account_did=$1`, old).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatal("private cleanup retained completion metadata")
	}
	recentState(t, pool, old, "active")
	recentRead(t, pool, recentDID("v"), newer)
	recentState(t, pool, old, "terminal")
	recentRead(t, pool, recentDID("v"), newer)
}
