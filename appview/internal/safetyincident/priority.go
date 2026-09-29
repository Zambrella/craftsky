package safetyincident

import "time"

const (
	PriorityImmediate = 1
	PriorityDeadline  = 2
	PriorityDetected  = 3
	PriorityAppeal    = 4
	PriorityRoutine   = 5
)

type PriorityInput struct {
	CredibleImmediateThreat bool
	Deadline                *time.Time
	Detected                bool
	Appealed                bool
	AwaitingInformation     bool
	RetentionExpiry         bool
	ReportCount             int
}

// PriorityFor ranks operational handling, not culpability. ReportCount is
// intentionally ignored: repeated allegations do not become a judgment.
func PriorityFor(input PriorityInput, now time.Time) int {
	if input.CredibleImmediateThreat {
		return PriorityImmediate
	}
	if input.Deadline != nil && !input.Deadline.After(now) {
		return PriorityDeadline
	}
	if input.Detected {
		return PriorityDetected
	}
	if input.Appealed {
		return PriorityAppeal
	}
	if input.RetentionExpiry || input.Deadline != nil {
		return PriorityDeadline
	}
	return PriorityRoutine
}
