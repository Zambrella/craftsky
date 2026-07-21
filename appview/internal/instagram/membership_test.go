package instagram

import (
	"context"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/testdb"
)

func TestMembershipStoreUsesCurrentCraftskyProfileRows(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:synthetic-alice');
	`)
	store := NewMembershipStore(pool)
	ctx := context.Background()

	for _, tt := range []struct {
		did  syntax.DID
		want bool
	}{
		{did: syntax.DID("did:plc:synthetic-alice"), want: true},
		{did: syntax.DID("did:plc:synthetic-departed"), want: false},
	} {
		got, err := store.IsCurrentMember(ctx, tt.did)
		if err != nil {
			t.Fatalf("IsCurrentMember(%s): %v", tt.did, err)
		}
		if got != tt.want {
			t.Errorf("IsCurrentMember(%s) = %t, want %t", tt.did, got, tt.want)
		}
	}

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:synthetic-alice'`); err != nil {
		t.Fatalf("delete membership: %v", err)
	}
	got, err := store.IsCurrentMember(ctx, syntax.DID("did:plc:synthetic-alice"))
	if err != nil {
		t.Fatalf("IsCurrentMember after loss: %v", err)
	}
	if got {
		t.Fatal("membership store cached a deleted membership row")
	}
}
