package api_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestProfilePinPermanentDeleteAndMembershipCascades(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, postStoreDDL+string(migration))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	seedMember(t, pool, owner.String())
	standardURI := seedPost(t, pool, owner.String(), "standard", "Standard", time.Now())
	projectURI := seedPost(t, pool, owner.String(), "project", "Project", time.Now())
	seedProjectMaterialization(t, pool, projectURI, "social.craftsky.feed.defs#knitting", "Project")
	store := api.NewProfilePinStore(pool)
	if _, err := store.Pin(ctx, owner, owner, syntax.RecordKey("standard")); err != nil {
		t.Fatalf("pin standard: %v", err)
	}
	if _, err := store.Pin(ctx, owner, owner, syntax.RecordKey("project")); err != nil {
		t.Fatalf("pin project: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_posts WHERE uri = $1`, standardURI); err != nil {
		t.Fatalf("delete standard target: %v", err)
	}
	afterTargetDelete, err := store.Read(ctx, owner)
	if err != nil {
		t.Fatalf("read after target delete: %v", err)
	}
	if afterTargetDelete.StandardPostURI != nil || afterTargetDelete.ProjectPostURI == nil || afterTargetDelete.ProjectPostURI.String() != projectURI {
		t.Fatalf("state after target delete = %+v", afterTargetDelete)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = $1`, owner); err != nil {
		t.Fatalf("delete owner membership: %v", err)
	}
	afterMembershipDelete, err := store.Read(ctx, owner)
	if err != nil {
		t.Fatalf("read after membership delete: %v", err)
	}
	if afterMembershipDelete.StandardPostURI != nil || afterMembershipDelete.ProjectPostURI != nil {
		t.Fatalf("state after membership delete = %+v", afterMembershipDelete)
	}
}
