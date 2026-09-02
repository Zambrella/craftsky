package ownerlifecycle

import (
	"reflect"
	"testing"
)

func TestOnboardingCompletionHasTerminalDeletionInventory(t *testing.T) {
	t.Parallel()
	want := TerminalDIDEntry{
		Table: "account_onboarding_completions", Column: "account_did",
		Component: "account_onboarding_completions", Role: "owner",
		Action: TerminalDeleteRow, KeyColumns: []string{"account_did"},
	}
	for _, entry := range TerminalDIDInventory() {
		if entry.Table == want.Table && entry.Role == want.Role {
			if !reflect.DeepEqual(entry, want) {
				t.Fatalf("onboarding terminal inventory = %+v, want %+v", entry, want)
			}
			return
		}
	}
	t.Fatalf("terminal inventory omitted %s owner DID", want.Table)
}
