package accountdeletion

import "testing"

func TestTerminalEligibility(t *testing.T) {
	t.Parallel()

	complete := TerminalGates{
		PrivateCleanupComplete:         true,
		OrdinarySessionsAbsent:         true,
		SubscriptionsAbsent:            true,
		ExpectedRecordReceiptsComplete: true,
		IndexedRecordsAbsent:           true,
		DerivedEffectsRetracted:        true,
		ScheduledObjectCleanupComplete: true,
		FinalPDSRescanEmpty:            true,
		BoundOAuthSessionRemoved:       true,
	}
	if !TerminalSuccessEligible(complete) {
		t.Fatal("complete authoritative gates must permit terminal success")
	}

	cases := map[string]func(*TerminalGates){
		"private cleanup":          func(g *TerminalGates) { g.PrivateCleanupComplete = false },
		"ordinary sessions":        func(g *TerminalGates) { g.OrdinarySessionsAbsent = false },
		"subscriptions":            func(g *TerminalGates) { g.SubscriptionsAbsent = false },
		"expected URI receipts":    func(g *TerminalGates) { g.ExpectedRecordReceiptsComplete = false },
		"indexed records":          func(g *TerminalGates) { g.IndexedRecordsAbsent = false },
		"derived effects":          func(g *TerminalGates) { g.DerivedEffectsRetracted = false },
		"scheduled object cleanup": func(g *TerminalGates) { g.ScheduledObjectCleanupComplete = false },
		"final PDS rescan":         func(g *TerminalGates) { g.FinalPDSRescanEmpty = false },
		"bound OAuth removal":      func(g *TerminalGates) { g.BoundOAuthSessionRemoved = false },
	}
	for name, makeIncomplete := range cases {
		t.Run(name, func(t *testing.T) {
			gates := complete
			makeIncomplete(&gates)
			if TerminalSuccessEligible(gates) {
				t.Fatal("incomplete authoritative gate permitted terminal success")
			}
		})
	}

	withBlobGCUnknown := complete
	withBlobGCUnknown.PDSBlobGarbageCollectionComplete = false
	if !TerminalSuccessEligible(withBlobGCUnknown) {
		t.Fatal("PDS blob garbage collection must not gate terminal success")
	}
}
