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

func TestProfilePinMutationsOnlyChangePrivateAppViewState(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, postStoreDDL+string(migration)+`
		CREATE TABLE pin_privacy_audit (
			surface TEXT NOT NULL,
			operation TEXT NOT NULL
		);
		CREATE FUNCTION audit_pin_external_change() RETURNS trigger AS $$
		BEGIN
			INSERT INTO pin_privacy_audit (surface, operation)
			VALUES (TG_TABLE_NAME, TG_OP);
			RETURN COALESCE(NEW, OLD);
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_pin_posts
		AFTER INSERT OR UPDATE OR DELETE ON craftsky_posts
		FOR EACH ROW EXECUTE FUNCTION audit_pin_external_change();
		CREATE TRIGGER audit_pin_likes
		AFTER INSERT OR UPDATE OR DELETE ON craftsky_likes
		FOR EACH ROW EXECUTE FUNCTION audit_pin_external_change();
		CREATE TRIGGER audit_pin_reposts
		AFTER INSERT OR UPDATE OR DELETE ON craftsky_reposts
		FOR EACH ROW EXECUTE FUNCTION audit_pin_external_change();
		CREATE TRIGGER audit_pin_saves
		AFTER INSERT OR UPDATE OR DELETE ON saved_posts
		FOR EACH ROW EXECUTE FUNCTION audit_pin_external_change();
	`)
	ctx := context.Background()
	for _, did := range []string{"did:plc:alice", "did:plc:bob"} {
		seedMember(t, pool, did)
	}
	seedPost(t, pool, "did:plc:alice", "a", "A", time.Now())
	seedPost(t, pool, "did:plc:alice", "b", "B", time.Now())
	seedPost(t, pool, "did:plc:bob", "private-bob", "Bob", time.Now())
	if _, err := pool.Exec(ctx, `TRUNCATE pin_privacy_audit`); err != nil {
		t.Fatalf("clear setup audit: %v", err)
	}
	store := api.NewProfilePinStore(pool)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	if _, err := store.Pin(ctx, bob, bob, syntax.RecordKey("private-bob")); err != nil {
		t.Fatalf("pin Bob: %v", err)
	}
	first, err := store.Pin(ctx, alice, alice, syntax.RecordKey("a"))
	if err != nil {
		t.Fatalf("pin Alice A: %v", err)
	}
	replacement, err := store.Pin(ctx, alice, alice, syntax.RecordKey("b"))
	if err != nil {
		t.Fatalf("replace Alice with B: %v", err)
	}
	cleared, err := store.Unpin(ctx, alice, syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/b"))
	if err != nil {
		t.Fatalf("unpin Alice B: %v", err)
	}

	for name, result := range map[string]api.ProfilePinMutationResult{
		"pin":     first,
		"replace": replacement,
		"unpin":   cleared,
	} {
		if result.State.ProjectPostURI != nil ||
			(result.State.StandardPostURI != nil && result.State.StandardPostURI.String() == "at://did:plc:bob/social.craftsky.feed.post/private-bob") {
			t.Fatalf("%s response exposed another owner's state: %+v", name, result.State)
		}
	}
	var externalChanges int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pin_privacy_audit`).Scan(&externalChanges); err != nil {
		t.Fatalf("count external changes: %v", err)
	}
	if externalChanges != 0 {
		t.Fatalf("pin operations changed post/interaction/saved surfaces: %d", externalChanges)
	}
	bobState, err := store.Read(ctx, bob)
	if err != nil {
		t.Fatalf("read Bob: %v", err)
	}
	if bobState.StandardPostURI == nil || bobState.StandardPostURI.String() != "at://did:plc:bob/social.craftsky.feed.post/private-bob" {
		t.Fatalf("Bob private state changed: %+v", bobState)
	}
}
