package api

import "testing"

func TestFollowAccountQueryConfig_BothDirectionsRequireCraftskyProfile(t *testing.T) {
	followers := followAccountQueryConfig("followers")
	if followers.craftskyJoin == "" {
		t.Fatal("followers craftskyJoin is empty, want Craftsky profile filter")
	}

	following := followAccountQueryConfig("following")
	if following.accountExpr != "f.subject_did" {
		t.Fatalf("following accountExpr = %q, want f.subject_did", following.accountExpr)
	}
	if following.craftskyJoin == "" {
		t.Fatal("following craftskyJoin is empty, want Craftsky profile filter")
	}
}
