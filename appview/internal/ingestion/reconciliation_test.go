package ingestion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
)

func TestReconciliationFailurePreservesSourceOrderUncertainty(t *testing.T) {
	outcome, err := reconciliationFailure(ErrReconciliationSourceChanged)
	if !errors.Is(err, ErrReconciliationSourceChanged) {
		t.Fatalf("error=%v, want source-changed sentinel", err)
	}
	if outcome.Kind != tap.OutcomeRetryable || outcome.Reason != tap.ReasonSourceOrderUncertain {
		t.Fatalf("outcome=%+v, want retryable source-order uncertainty", outcome)
	}
}

func TestReconciliationFailureClassifiesOtherErrorsAsStorageUnavailable(t *testing.T) {
	wantErr := errors.New("database unavailable")
	outcome, err := reconciliationFailure(wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, want %v", err, wantErr)
	}
	if outcome.Kind != tap.OutcomeRetryable || outcome.Reason != tap.ReasonStorageUnavailable {
		t.Fatalf("outcome=%+v, want retryable storage unavailable", outcome)
	}
}

func TestNewServiceRequiresLifecycleParticipants(t *testing.T) {
	base := ServiceConfig{
		Store: &Store{}, Lifecycles: &ownerlifecycle.Store{},
	}
	if _, err := NewService(base); err == nil {
		t.Fatal("service accepted missing profile and terminal lifecycle participants")
	}
	base.ProfileParticipant = func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
		return nil
	}
	if _, err := NewService(base); err == nil {
		t.Fatal("service accepted missing terminal lifecycle participant")
	}
	base.TerminalParticipant = func(context.Context, pgx.Tx, *ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
		return nil
	}
	if _, err := NewService(base); err == nil {
		t.Fatal("service accepted a missing terminal commit timeout")
	}
	base.TerminalCommitTimeout = time.Second
	if _, err := NewService(base); err != nil {
		t.Fatalf("service rejected complete lifecycle participants: %v", err)
	}
}

func TestProfileObservationCannotCancelPendingExplicitDeletion(t *testing.T) {
	target, transition := profileTransition(ownerlifecycle.StateDeletionPending, "update")
	if transition || target != ownerlifecycle.StateDeletionPending {
		t.Fatalf("profile observation selected %s transition=%t; explicit cancel/expiry must own reactivation", target, transition)
	}
}

func TestProfileDeletionDepartsPendingIntentButCannotFinishAcceptedDeletion(t *testing.T) {
	target, transition := profileTransition(ownerlifecycle.StateDeletionPending, "delete")
	if !transition || target != ownerlifecycle.StateDeparted {
		t.Fatalf("pending profile deletion selected %s transition=%t, want departed", target, transition)
	}
	target, transition = profileTransition(ownerlifecycle.StateDeleting, "delete")
	if transition || target != ownerlifecycle.StateDeleting {
		t.Fatalf("accepted deletion profile event selected %s transition=%t; worker must own completion", target, transition)
	}
}
