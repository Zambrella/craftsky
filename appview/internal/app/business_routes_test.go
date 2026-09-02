package app

import (
	"testing"
	"time"

	"social.craftsky/appview/internal/business"
)

func TestRouteDependenciesPropagateBusinessStoreAndClockNilSafely(t *testing.T) {
	if got := RouteDependencies(nil); got != nil {
		t.Fatalf("RouteDependencies(nil) = %#v, want nil", got)
	}

	store := business.NewStore(nil)
	fixed := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return fixed }
	adapted := RouteDependencies(&Deps{BusinessStore: store, Now: now})
	if adapted.BusinessStore != store {
		t.Fatal("business store was not preserved across route dependency adaptation")
	}
	if adapted.Now == nil || !adapted.Now().Equal(fixed) {
		t.Fatalf("adapted clock returned wrong value, want %v", fixed)
	}

	empty := RouteDependencies(&Deps{})
	if empty.BusinessStore != nil || empty.Now != nil {
		t.Fatal("empty business dependencies must preserve nil store and clock")
	}
}
