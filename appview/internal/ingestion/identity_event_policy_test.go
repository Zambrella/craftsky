package ingestion

import (
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/tap"
)

func TestIdentityEventPolicyTreatsEveryTapStatusAsRefreshHint(t *testing.T) {
	for _, status := range []string{"active", "deactivated", "suspended", "takendown", "deleted"} {
		for _, handle := range []string{"current.example", "changed.example"} {
			t.Run(status+"/"+handle, func(t *testing.T) {
				policy, err := classifyIdentityEvent(tap.IdentityEvent{
					ID: 1, DID: syntax.DID("did:plc:identity-policy"),
					Handle: handle, IsActive: status == "active", Status: status,
				})
				if err != nil {
					t.Fatalf("classify accepted Tap status: %v", err)
				}
				if !policy.enqueueRefresh {
					t.Fatalf("policy=%+v, want refresh hint", policy)
				}
			})
		}
	}
}

func TestIdentityEventPolicyRejectsStatusOutsidePinnedTapContract(t *testing.T) {
	if _, err := classifyIdentityEvent(tap.IdentityEvent{
		ID: 1, DID: syntax.DID("did:plc:identity-policy"), Status: "inactive",
	}); err == nil {
		t.Fatal("accepted status outside the Tap 0.1.10 wire contract")
	}
}
