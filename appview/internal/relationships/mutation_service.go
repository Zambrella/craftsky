package relationships

import (
	"context"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// MutationService owns only AppView-private relationship mutations and their
// restoration side effects. Public follows and blocks use PDS commands.
type MutationService struct {
	store       *Store
	now         func() time.Time
	restoration RelationshipSafetyRestorationEnqueuer
	observer    interface {
		ObserveRelationship(operation, result string, duration time.Duration)
	}
}

type RelationshipSafetyRestorationEnqueuer interface {
	EnqueueRelationshipSafetyRestoration(
		context.Context,
		syntax.DID,
		syntax.DID,
	) error
}

type relationshipOutcomeObserver interface {
	ObserveRelationshipOutcome(operation, stage, result, errorClass string, duration time.Duration)
}

func NewMutationService(store *Store, now func() time.Time, observers ...interface {
	ObserveRelationship(operation, result string, duration time.Duration)
}) *MutationService {
	return NewMutationServiceWithRestoration(
		store,
		now,
		nil,
		observers...,
	)
}

func NewMutationServiceWithRestoration(
	store *Store,
	now func() time.Time,
	restoration RelationshipSafetyRestorationEnqueuer,
	observers ...interface {
		ObserveRelationship(operation, result string, duration time.Duration)
	},
) *MutationService {
	if now == nil {
		now = time.Now
	}
	var observer interface {
		ObserveRelationship(operation, result string, duration time.Duration)
	}
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &MutationService{
		store: store, now: now,
		restoration: restoration, observer: observer,
	}
}

func (s *MutationService) Mute(ctx context.Context, owner, subject syntax.DID) (state State, err error) {
	started := time.Now()
	canceled, err := s.store.MuteAndCancelPendingDeliveries(ctx, owner, subject)
	if err != nil {
		s.observeOutcome("mute", "store", "error", "store", time.Since(started))
		return State{}, err
	}
	result := "none"
	if canceled > 0 {
		result = "some"
	}
	s.observeOutcome("push_cancellation", "delivery", result, "none", 0)
	state, err = s.store.State(ctx, owner, subject)
	if err != nil {
		s.observeOutcome("mute", "store", "error", "store", time.Since(started))
		return State{}, err
	}
	s.observeOutcome("mute", "complete", "success", "none", time.Since(started))
	return state, nil
}

func (s *MutationService) Unmute(ctx context.Context, owner, subject syntax.DID) (state State, err error) {
	defer s.observe("unmute", time.Now(), &err)
	if err := s.store.Unmute(ctx, owner, subject); err != nil {
		return State{}, err
	}
	state, err = s.store.State(ctx, owner, subject)
	if err != nil {
		return State{}, err
	}
	if s.restoration != nil {
		if err := s.restoration.EnqueueRelationshipSafetyRestoration(
			ctx,
			owner,
			subject,
		); err != nil {
			return State{}, err
		}
	}
	return state, nil
}

func (s *MutationService) EnqueueRelationshipSafetyRestoration(ctx context.Context, owner, subject syntax.DID) error {
	if s == nil || s.restoration == nil {
		return nil
	}
	return s.restoration.EnqueueRelationshipSafetyRestoration(ctx, owner, subject)
}

func (s *MutationService) observe(operation string, started time.Time, err *error) {
	if s == nil || s.observer == nil {
		return
	}
	result := "success"
	if err != nil && *err != nil {
		result = "error"
	}
	s.observer.ObserveRelationship(operation, result, time.Since(started))
}

func (s *MutationService) observeOutcome(operation, stage, result, errorClass string, duration time.Duration) {
	if s == nil || s.observer == nil {
		return
	}
	if detailed, ok := s.observer.(relationshipOutcomeObserver); ok {
		detailed.ObserveRelationshipOutcome(operation, stage, result, errorClass, duration)
		return
	}
	s.observer.ObserveRelationship(operation, result, duration)
}
