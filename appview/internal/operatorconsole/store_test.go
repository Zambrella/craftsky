package operatorconsole_test

import (
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/operatorconsole"
	"social.craftsky/appview/internal/testdb"
	"testing"
	"time"
)

func TestApprovedDIDReceivesOnlySelectedPermissionSet(t *testing.T) {
	store := operatorconsole.NewStore(testdb.WithMigratedSchema(t), time.Now)
	did := syntax.DID("did:plc:consolemoderator")
	if _, err := store.Lookup(t.Context(), did); err == nil {
		t.Fatal("unlisted account admitted")
	}
	op, err := store.Allow(t.Context(), did, "mod")
	if err != nil {
		t.Fatal(err)
	}
	current, err := store.Lookup(t.Context(), did)
	if err != nil || current.ID != op.ID || !current.Permissions["moderation.manage"] || current.Permissions["evidence.access"] || current.Permissions["operator.manage"] {
		t.Fatalf("incorrect moderator capabilities: %+v %v", current, err)
	}
	if err := store.SetPermissionSet(t.Context(), op.ID, "admin"); err != nil {
		t.Fatal(err)
	}
	current, err = store.Lookup(t.Context(), did)
	if err != nil || !current.Permissions["operator.manage"] || !current.Permissions["evidence.access"] {
		t.Fatal("permission set change not effective")
	}
	if err := store.Disable(t.Context(), op.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Lookup(t.Context(), did); err == nil {
		t.Fatal("disabled account admitted")
	}
}
