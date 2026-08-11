package accountdeletion

// TerminalGates is deliberately explicit and fail-closed. Adding a new
// authoritative completion requirement must update TerminalSuccessEligible and
// its test before a deletion can be finalized.
type TerminalGates struct {
	PrivateCleanupComplete           bool
	OrdinarySessionsAbsent           bool
	SubscriptionsAbsent              bool
	ExpectedRecordReceiptsComplete   bool
	IndexedRecordsAbsent             bool
	DerivedEffectsRetracted          bool
	ScheduledObjectCleanupComplete   bool
	FinalPDSRescanEmpty              bool
	BoundOAuthSessionRemoved         bool
	PDSBlobGarbageCollectionComplete bool
}

func TerminalSuccessEligible(gates TerminalGates) bool {
	return gates.PrivateCleanupComplete &&
		gates.OrdinarySessionsAbsent &&
		gates.SubscriptionsAbsent &&
		gates.ExpectedRecordReceiptsComplete &&
		gates.IndexedRecordsAbsent &&
		gates.DerivedEffectsRetracted &&
		gates.ScheduledObjectCleanupComplete &&
		gates.FinalPDSRescanEmpty &&
		gates.BoundOAuthSessionRemoved
}
