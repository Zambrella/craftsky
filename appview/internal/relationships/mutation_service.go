package relationships

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

var ErrBlockMutationUnavailable = errors.New("block mutation unavailable")

// MutationService combines private mute persistence with public block
// orchestration. Block orchestration is filled by the dedicated PDS/Tap TDD
// phase; keeping it behind this interface lets route policy land first.
type MutationService struct {
	store       *Store
	newEffects  pdseffects.ExecutorFactory
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

func NewMutationService(store *Store, newEffects pdseffects.ExecutorFactory, now func() time.Time, observers ...interface {
	ObserveRelationship(operation, result string, duration time.Duration)
}) *MutationService {
	return NewMutationServiceWithRestoration(
		store,
		newEffects,
		now,
		nil,
		observers...,
	)
}

func NewMutationServiceWithRestoration(
	store *Store,
	newEffects pdseffects.ExecutorFactory,
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
		store: store, newEffects: newEffects, now: now,
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

func (s *MutationService) Block(ctx context.Context, owner, subject syntax.DID, sid string) (result BlockMutationResult, err error) {
	started := time.Now()
	stage := "store"
	defer func() {
		metricResult := "success"
		errorClass := "none"
		if err != nil {
			metricResult = "error"
			errorClass = stage
		}
		s.observeOutcome("block", stage, metricResult, errorClass, time.Since(started))
	}()
	indexed, err := s.store.OwnedBlockRecords(ctx, owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	if len(indexed) > 0 {
		stage = "store"
		state, err := s.store.State(ctx, owner, subject)
		if err != nil {
			return BlockMutationResult{}, err
		}
		state.Blocking = true
		return BlockMutationResult{
			State: state,
			URI:   indexed[0].URI,
			CID:   indexed[0].CID,
			Rkey:  indexed[0].Rkey,
		}, nil
	}
	stage = "pds"
	executor, generation, expected, err := s.effectExecutor(ctx, owner, subject, sid)
	if err != nil {
		return BlockMutationResult{}, fmt.Errorf("create block effect executor: %w", err)
	}
	rkey, err := deterministicBlockRecordKey(owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	var current bsky.GraphBlock
	currentCID, readErr := executor.ReadRecord(ctx, pdseffects.ReadRecordRequest{
		Owner: owner, OwnerGeneration: generation, ExpectedOwners: expected,
		Collection: syntax.NSID(blueskyBlockCollection), Rkey: rkey,
	}, &current)
	if readErr == nil {
		if current.Subject != subject.String() {
			return BlockMutationResult{}, &pdseffects.ConflictError{ExactKey: rkey.String()}
		}
		stage = "store"
		state, err := s.store.State(ctx, owner, subject)
		if err != nil {
			return BlockMutationResult{}, err
		}
		state.Blocking = true
		return BlockMutationResult{
			State: state,
			URI: syntax.ATURI(fmt.Sprintf(
				"at://%s/%s/%s", owner, blueskyBlockCollection, rkey,
			)),
			CID: currentCID, Rkey: rkey,
		}, nil
	}
	if !errors.Is(readErr, auth.ErrRecordNotFound) {
		return BlockMutationResult{}, fmt.Errorf("read deterministic block record: %w", readErr)
	}
	record := &bsky.GraphBlock{
		LexiconTypeID: blueskyBlockCollection,
		Subject:       subject.String(),
		CreatedAt:     s.now().UTC().Format(time.RFC3339),
	}
	operationID := relationshipEffectIdentity("block", "put", rkey)
	created, err := executor.PutRecord(ctx, pdseffects.PutRecordRequest{
		OperationID: operationID,
		MutationKey: operationID,
		Owner:       owner, OwnerGeneration: generation, ExpectedOwners: expected,
		Collection: syntax.NSID(blueskyBlockCollection), Rkey: rkey, Record: record,
	})
	if err != nil {
		return BlockMutationResult{}, fmt.Errorf("put block record: %w", err)
	}
	stage = "store"
	state, err := s.store.State(ctx, owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	state.Blocking = true
	return BlockMutationResult{State: state, URI: created.URI, CID: created.CID, Rkey: rkey}, nil
}

func (s *MutationService) Unblock(ctx context.Context, owner, subject syntax.DID, sid string) (result BlockMutationResult, err error) {
	defer s.observe("unblock", time.Now(), &err)
	indexed, err := s.store.OwnedBlockRecords(ctx, owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	executor, generation, expected, err := s.effectExecutor(ctx, owner, subject, sid)
	if err != nil {
		return BlockMutationResult{}, fmt.Errorf("create unblock effect executor: %w", err)
	}

	rkeys := make(map[syntax.RecordKey]syntax.CID)
	stableRkey, err := deterministicBlockRecordKey(owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	rkeys[stableRkey] = ""
	for _, record := range indexed {
		rkeys[record.Rkey] = record.CID
	}

	ordered := make([]syntax.RecordKey, 0, len(rkeys))
	for rkey := range rkeys {
		ordered = append(ordered, rkey)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].String() < ordered[j].String() })
	for _, rkey := range ordered {
		operationID := relationshipEffectIdentity("block", "delete", rkey)
		if _, err := executor.DeleteRecord(ctx, pdseffects.DeleteRecordRequest{
			OperationID: operationID,
			MutationKey: operationID,
			Owner:       owner, OwnerGeneration: generation, ExpectedOwners: expected,
			Collection: syntax.NSID(blueskyBlockCollection), Rkey: rkey,
			ExpectedCID: rkeys[rkey],
		}); err != nil {
			return BlockMutationResult{}, fmt.Errorf("delete block record: %w", err)
		}
	}

	state, err := s.store.State(ctx, owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	// Tap owns projection. Hide a still-indexed outbound row in the mutation
	// response while preserving a separately-owned inbound block direction.
	state.Blocking = false
	if s.restoration != nil {
		if err := s.restoration.EnqueueRelationshipSafetyRestoration(
			ctx,
			owner,
			subject,
		); err != nil {
			return BlockMutationResult{}, err
		}
	}
	return BlockMutationResult{State: state}, nil
}

func (s *MutationService) effectExecutor(
	ctx context.Context,
	owner, subject syntax.DID,
	sessionID string,
) (pdseffects.EffectExecutor, int64, []ownerlifecycle.ExpectedOwner, error) {
	if s == nil || s.newEffects == nil || owner == "" || subject == "" ||
		strings.TrimSpace(sessionID) == "" {
		return nil, 0, nil, ErrBlockMutationUnavailable
	}
	generation, ok := ownerlifecycle.ExpectedGeneration(ctx)
	if !ok || generation <= 0 {
		return nil, 0, nil, ownerlifecycle.ErrGenerationChanged
	}
	executor, err := s.newEffects(ctx, owner, sessionID)
	if err != nil {
		return nil, 0, nil, err
	}
	if executor == nil {
		return nil, 0, nil, ErrBlockMutationUnavailable
	}
	expected, err := executor.ResolveExpectedOwners(ctx, generation, []syntax.DID{subject})
	if err != nil {
		return nil, 0, nil, err
	}
	return executor, generation, expected, nil
}

func deterministicBlockRecordKey(owner, subject syntax.DID) (syntax.RecordKey, error) {
	digest := sha256.Sum256([]byte(owner.String() + "\x00" + subject.String()))
	return syntax.ParseRecordKey(fmt.Sprintf("craftsky-%x", digest[:16]))
}

func relationshipEffectIdentity(kind, action string, rkey syntax.RecordKey) string {
	return kind + ":" + action + ":" + uuid.NewString() + ":" + rkey.String()
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

const blueskyBlockCollection = "app.bsky.graph.block"
