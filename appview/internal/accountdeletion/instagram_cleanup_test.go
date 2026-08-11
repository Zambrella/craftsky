package accountdeletion

import (
	"context"
	"reflect"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestInstagramExplicitDeletionPlanAlwaysHardDeletes(t *testing.T) {
	t.Parallel()

	want := []InstagramDataCategory{
		InstagramAccountLinks,
		InstagramGraphImports,
		InstagramAutomaticFollowSuggestions,
		InstagramVerification,
		InstagramPrivateImportedData,
		InstagramUsernameClaims,
	}
	plan := InstagramExplicitDeletionPlan()
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("Instagram explicit deletion plan = %#v, want %#v", plan, want)
	}
	for _, category := range plan {
		if category.Policy != HardDelete {
			t.Fatalf("Instagram category %q policy = %q, want hard delete", category.Name, category.Policy)
		}
	}

	owner := syntax.DID("did:plc:alice")
	purger := &recordingInstagramPurger{}
	if err := PurgeInstagramForAccountDeletion(context.Background(), purger, owner); err != nil {
		t.Fatal(err)
	}
	if purger.owner != owner || purger.calls != 1 {
		t.Fatalf("explicit purge calls = %d owner = %q", purger.calls, purger.owner)
	}

	capability := reflect.TypeOf((*InstagramExplicitPurger)(nil)).Elem()
	if capability.NumMethod() != 1 {
		t.Fatalf("Instagram deletion capability has %d methods, want only PurgeOwner", capability.NumMethod())
	}
	if _, ok := capability.MethodByName("InactivateMembership"); ok {
		t.Fatal("explicit deletion capability must not expose ordinary inactivation")
	}
}

type recordingInstagramPurger struct {
	owner syntax.DID
	calls int
}

func (purger *recordingInstagramPurger) PurgeOwner(_ context.Context, owner syntax.DID) error {
	purger.owner = owner
	purger.calls++
	return nil
}

var _ InstagramExplicitPurger = (*recordingInstagramPurger)(nil)
