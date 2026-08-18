package pdseffects

import (
	"errors"
	"fmt"
)

var (
	ErrOutcomeAmbiguous           = errors.New("PDS effect outcome is ambiguous")
	ErrEffectConflict             = errors.New("PDS effect conflicts with durable or remote state")
	ErrEffectRejected             = errors.New("PDS effect was durably rejected")
	ErrExecutorOwnerMismatch      = errors.New("PDS effect owner does not match executor session owner")
	ErrGuardedEffectScopeExpired  = errors.New("guarded PDS effect scope is no longer active")
	ErrGuardedEffectScopeMismatch = errors.New("guarded PDS effect scope does not match expected owners")
)

// OutcomeAmbiguousError means the request crossed the durable dispatch
// boundary and must not be repeated. A later operation may only reconcile the
// deterministic remote identity with a read.
type OutcomeAmbiguousError struct {
	OperationID string
	ExactKey    string
	Cause       error
}

func (err *OutcomeAmbiguousError) Error() string {
	if err == nil {
		return ErrOutcomeAmbiguous.Error()
	}
	if err.Cause == nil {
		return fmt.Sprintf("%s: operation %q", ErrOutcomeAmbiguous, err.OperationID)
	}
	return fmt.Sprintf("%s: operation %q: %v", ErrOutcomeAmbiguous, err.OperationID, err.Cause)
}

func (err *OutcomeAmbiguousError) Unwrap() []error {
	if err == nil || err.Cause == nil {
		return []error{ErrOutcomeAmbiguous}
	}
	return []error{ErrOutcomeAmbiguous, err.Cause}
}

// ConflictError is returned when an idempotency key is reused for a different
// canonical request, or when the deterministic PDS record does not match the
// dispatched request during read-only reconciliation.
type ConflictError struct {
	OperationID string
	ExactKey    string
	Cause       error
}

func (err *ConflictError) Error() string {
	if err == nil {
		return ErrEffectConflict.Error()
	}
	if err.Cause == nil {
		return fmt.Sprintf("%s: operation %q", ErrEffectConflict, err.OperationID)
	}
	return fmt.Sprintf("%s: operation %q: %v", ErrEffectConflict, err.OperationID, err.Cause)
}

func (err *ConflictError) Unwrap() []error {
	if err == nil || err.Cause == nil {
		return []error{ErrEffectConflict}
	}
	return []error{ErrEffectConflict, err.Cause}
}
