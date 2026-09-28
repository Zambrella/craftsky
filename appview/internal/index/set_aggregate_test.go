package index

import (
	"reflect"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/tap"
)

func TestAffectedSetScopesAreCanonicalSortedAndDeduplicated(t *testing.T) {
	actor := syntax.DID("did:plc:set-actor")
	alpha := SetScope{Kind: "like", Actor: actor, Key: "post:alpha"}
	zulu := SetScope{Kind: "like", Actor: actor, Key: "post:zulu"}
	tests := []struct {
		name     string
		previous *SetSource
		next     *SetSource
		want     []SetScope
	}{
		{name: "create", next: &SetSource{Scope: zulu}, want: []SetScope{zulu}},
		{name: "invalidate or delete", previous: &SetSource{Scope: alpha}, want: []SetScope{alpha}},
		{name: "retarget sorts old and new", previous: &SetSource{Scope: zulu}, next: &SetSource{Scope: alpha}, want: []SetScope{alpha, zulu}},
		{name: "same scope is deduplicated", previous: &SetSource{Scope: alpha}, next: &SetSource{Scope: alpha}, want: []SetScope{alpha}},
		{name: "eligibility change keeps one scope", previous: &SetSource{Scope: alpha, Eligible: false}, next: &SetSource{Scope: alpha, Eligible: true}, want: []SetScope{alpha}},
		{name: "no prior or next fact", want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := affectedSetScopes(test.previous, test.next)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("affected scopes = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestClassifySetSourceKeepsValidBlockedFactsAndAllowsWakeup(t *testing.T) {
	base := SetSource{
		URI:      "at://did:plc:actor/social.craftsky.feed.like/3aaaaaaaaaaa2",
		Scope:    SetScope{Kind: "like", Actor: "did:plc:actor", Key: "post:subject"},
		Eligible: true,
	}
	memberDependency := tap.Dependency{Kind: "member_did", Key: "did:plc:actor"}
	subjectDependency := tap.Dependency{Kind: "subject_uri", Key: "at://did:plc:subject/social.craftsky.feed.post/3aaaaaaaaaaa3"}

	departed := classifySetSource(base, false, memberDependency, false, "owner_departed")
	if departed.Eligible || departed.IneligibilityReason != "owner_departed" || departed.Dependency != memberDependency {
		t.Fatalf("departed fact = %+v", departed)
	}

	missing := classifySetSource(base, true, subjectDependency, false, "missing_subject")
	if missing.Eligible || missing.IneligibilityReason != "missing_subject" || missing.Dependency != subjectDependency {
		t.Fatalf("missing dependency fact = %+v", missing)
	}

	woken := classifySetSource(missing, true, subjectDependency, true, "missing_subject")
	if !woken.Eligible || woken.IneligibilityReason != "" || woken.Dependency != (tap.Dependency{}) {
		t.Fatalf("woken fact = %+v", woken)
	}
}

func TestReduceSetAggregatePreservesLogicalEdgesAndRepresentativeOrder(t *testing.T) {
	activatedAt := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	now := activatedAt.Add(24 * time.Hour)
	older := SetSource{URI: "at://did:plc:a/social.craftsky.feed.like/3aaaaaaaaaaa2", ActivityAt: activatedAt.Add(time.Hour), Eligible: true}
	sameTimeLaterURI := SetSource{URI: "at://did:plc:z/social.craftsky.feed.like/3aaaaaaaaaaa2", ActivityAt: older.ActivityAt, Eligible: true}
	newer := SetSource{URI: "at://did:plc:b/social.craftsky.feed.like/3aaaaaaaaaaa3", ActivityAt: activatedAt.Add(2 * time.Hour), Eligible: true}
	ineligible := SetSource{URI: "at://did:plc:c/social.craftsky.feed.like/3aaaaaaaaaaa4", ActivityAt: activatedAt, Eligible: false}

	created, transition := reduceSetAggregate(nil, []SetSource{newer, sameTimeLaterURI, older, ineligible}, now)
	if transition != SetActivated || created == nil || created.EligibleSourceCount != 3 || created.RepresentativeURI != older.URI || !created.ActivatedAt.Equal(now) {
		t.Fatalf("created aggregate=%+v transition=%s", created, transition)
	}

	remaining, transition := reduceSetAggregate(created, []SetSource{newer, sameTimeLaterURI}, now.Add(time.Hour))
	if transition != SetUnchanged || remaining == nil || remaining.EligibleSourceCount != 2 || remaining.RepresentativeURI != sameTimeLaterURI.URI || !remaining.ActivatedAt.Equal(now) {
		t.Fatalf("remaining aggregate=%+v transition=%s", remaining, transition)
	}

	removed, transition := reduceSetAggregate(remaining, []SetSource{ineligible}, now.Add(2*time.Hour))
	if transition != SetDeactivated || removed != nil {
		t.Fatalf("removed aggregate=%+v transition=%s", removed, transition)
	}
}
