package api

import "testing"

func TestFollowAccountQueryConfig_FollowingRequiresCraftskyProfile(t *testing.T) {
	followers := followAccountQueryConfig("followers")
	if followers.craftskyJoin != "" {
		t.Fatalf("followers craftskyJoin = %q, want empty", followers.craftskyJoin)
	}

	following := followAccountQueryConfig("following")
	if following.accountExpr != "f.subject_did" {
		t.Fatalf("following accountExpr = %q, want f.subject_did", following.accountExpr)
	}
	if following.craftskyJoin == "" {
		t.Fatal("following craftskyJoin is empty, want Craftsky profile filter")
	}
}
