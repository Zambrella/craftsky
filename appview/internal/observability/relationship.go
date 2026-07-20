package observability

import (
	"context"
	"time"
)

// ObserveRelationship records a relationship-policy operation using only
// bounded operation and result dimensions. Actor pairs, DIDs, record keys, and
// content URIs are deliberately absent from this boundary.
func (o *Observer) ObserveRelationship(operation, result string, duration time.Duration) {
	errorClass := "none"
	if result == "error" {
		errorClass = "internal"
	}
	o.ObserveRelationshipOutcome(operation, "complete", result, errorClass, duration)
}

// ObserveRelationshipOutcome records the bounded stage and error class needed
// to distinguish policy denials, indexing failures, and delivery cancellation
// without accepting any actor or record identifiers.
func (o *Observer) ObserveRelationshipOutcome(operation, stage, result, errorClass string, duration time.Duration) {
	if o == nil {
		return
	}
	if duration < 0 {
		duration = 0
	}
	o.metricRecorder.RelationshipOperation(context.Background(), operation, stage, result, errorClass, duration)
}
