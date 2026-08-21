package observability

import (
	"context"

	"social.craftsky/appview/internal/ownerlifecycle"
)

// ObserveTerminalPurge records only bounded catalogue labels and aggregate
// counts. Owner DIDs, row contents, exact object/PDS keys, and raw errors never
// cross the observability boundary.
func (o *Observer) ObserveTerminalPurge(observation ownerlifecycle.TerminalPurgeObservation) {
	if o == nil {
		return
	}
	o.metricRecorder.TerminalPurge(
		context.Background(),
		observation.Operation,
		observation.Result,
		observation.ErrorCategory,
		observation.Component,
		observation.DIDRole,
		observation.Claims,
		observation.RowsAffected,
		observation.Remaining,
		observation.Complete,
	)
}
