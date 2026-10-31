package chat

import (
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

// IT-005 / FR-016, NFR-001 / AC-014, AC-019.
func TestConnectionDisconnectFencesStaleConsent(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	fencer, err := ownerlifecycle.NewFencer(pool, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	owner, other := syntax.DID("did:plc:connection-owner"), syntax.DID("did:plc:connection-other")
	for _, did := range []syntax.DID{owner, other} {
		if _, err := pool.Exec(t.Context(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, did); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewStore(pool, owners)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := store.Connection(t.Context(), owner)
	if err != nil || initial.Enabled || initial.Generation != 1 || !initial.PushEnabled {
		t.Fatalf("initial state = %+v, %v", initial, err)
	}
	push := false
	connected, err := store.Connect(t.Context(), owner, initial.Generation, &push)
	if err != nil || !connected.Enabled || connected.Generation <= initial.Generation || connected.PushEnabled {
		t.Fatalf("connected state = %+v, %v", connected, err)
	}
	disconnected, err := store.Disconnect(t.Context(), owner)
	if err != nil || disconnected.Enabled || disconnected.Generation <= connected.Generation || disconnected.PushEnabled {
		t.Fatalf("disconnected state = %+v, %v", disconnected, err)
	}
	if _, err := store.Connect(t.Context(), owner, initial.Generation, nil); !errors.Is(err, ErrConnectionChanged) {
		t.Fatalf("stale consent = %v", err)
	}
	stillDisconnected, err := store.Connection(t.Context(), owner)
	if err != nil || stillDisconnected.Enabled || stillDisconnected.Generation != disconnected.Generation {
		t.Fatal("stale consent re-enabled messaging")
	}
	otherState, err := store.Connection(t.Context(), other)
	if err != nil || otherState.Enabled || otherState.Generation != 1 || !otherState.PushEnabled {
		t.Fatal("disconnect affected another owner")
	}
}
