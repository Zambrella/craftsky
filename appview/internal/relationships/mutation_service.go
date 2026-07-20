package relationships

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

var ErrBlockMutationUnavailable = errors.New("block mutation unavailable")

// MutationService combines private mute persistence with public block
// orchestration. Block orchestration is filled by the dedicated PDS/Tap TDD
// phase; keeping it behind this interface lets route policy land first.
type MutationService struct {
	store    *Store
	newPDS   auth.PDSClientFactory
	now      func() time.Time
	observer interface {
		ObserveRelationship(operation, result string, duration time.Duration)
	}
}

type relationshipOutcomeObserver interface {
	ObserveRelationshipOutcome(operation, stage, result, errorClass string, duration time.Duration)
}

func NewMutationService(store *Store, newPDS auth.PDSClientFactory, now func() time.Time, observers ...interface {
	ObserveRelationship(operation, result string, duration time.Duration)
}) *MutationService {
	if now == nil {
		now = time.Now
	}
	var observer interface {
		ObserveRelationship(operation, result string, duration time.Duration)
	}
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &MutationService{store: store, newPDS: newPDS, now: now, observer: observer}
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
	return s.store.State(ctx, owner, subject)
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
	if s.newPDS == nil {
		return BlockMutationResult{}, ErrBlockMutationUnavailable
	}
	pds, err := s.newPDS(ctx, owner, sid)
	if err != nil {
		return BlockMutationResult{}, fmt.Errorf("create block PDS client: %w", err)
	}
	lister, ok := pds.(auth.PDSRecordLister)
	if !ok {
		return BlockMutationResult{}, fmt.Errorf("create block: PDS client cannot list records")
	}
	existing, err := listMatchingPDSBlocks(ctx, lister, owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	if len(existing) > 0 {
		stage = "store"
		state, err := s.store.State(ctx, owner, subject)
		if err != nil {
			return BlockMutationResult{}, err
		}
		state.Blocking = true
		return BlockMutationResult{
			State: state,
			URI:   existing[0].URI,
			CID:   existing[0].CID,
			Rkey:  existing[0].URI.RecordKey(),
		}, nil
	}

	record := &bsky.GraphBlock{
		LexiconTypeID: blueskyBlockCollection,
		Subject:       subject.String(),
		CreatedAt:     s.now().UTC().Format(time.RFC3339),
	}
	uri, cid, err := pds.CreateRecord(ctx, owner, blueskyBlockCollection, record)
	if err != nil {
		return BlockMutationResult{}, fmt.Errorf("create block record: %w", err)
	}
	rkey := uri.RecordKey()
	if rkey == "" {
		return BlockMutationResult{}, fmt.Errorf("create block record: PDS returned invalid uri")
	}
	stage = "store"
	state, err := s.store.State(ctx, owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	state.Blocking = true
	return BlockMutationResult{State: state, URI: uri, CID: cid, Rkey: rkey}, nil
}

func (s *MutationService) Unblock(ctx context.Context, owner, subject syntax.DID, sid string) (result BlockMutationResult, err error) {
	defer s.observe("unblock", time.Now(), &err)
	indexed, err := s.store.OwnedBlockRecords(ctx, owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	if s.newPDS == nil {
		return BlockMutationResult{}, ErrBlockMutationUnavailable
	}
	pds, err := s.newPDS(ctx, owner, sid)
	if err != nil {
		return BlockMutationResult{}, fmt.Errorf("create unblock PDS client: %w", err)
	}

	rkeys := make(map[syntax.RecordKey]struct{})
	for _, record := range indexed {
		rkeys[record.Rkey] = struct{}{}
	}
	lister, ok := pds.(auth.PDSRecordLister)
	if !ok {
		return BlockMutationResult{}, fmt.Errorf("delete block: PDS client cannot list records")
	}
	matches, err := listMatchingPDSBlocks(ctx, lister, owner, subject)
	if err != nil {
		return BlockMutationResult{}, err
	}
	for _, match := range matches {
		rkey := match.URI.RecordKey()
		if rkey == "" {
			return BlockMutationResult{}, fmt.Errorf("delete block: PDS returned invalid uri")
		}
		rkeys[rkey] = struct{}{}
	}

	ordered := make([]syntax.RecordKey, 0, len(rkeys))
	for rkey := range rkeys {
		ordered = append(ordered, rkey)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].String() < ordered[j].String() })
	for _, rkey := range ordered {
		if err := pds.DeleteRecord(ctx, owner, blueskyBlockCollection, rkey.String()); err != nil && !errors.Is(err, auth.ErrRecordNotFound) {
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
	return BlockMutationResult{State: state}, nil
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

func listMatchingPDSBlocks(
	ctx context.Context,
	lister auth.PDSRecordLister,
	owner, subject syntax.DID,
) ([]auth.PDSRecord, error) {
	var matches []auth.PDSRecord
	cursor := ""
	seenCursors := map[string]bool{}
	for {
		records, next, err := lister.ListRecords(ctx, owner, blueskyBlockCollection, cursor, 100)
		if err != nil {
			return nil, fmt.Errorf("list block records: %w", err)
		}
		for _, record := range records {
			if record.URI.Authority().String() != owner.String() ||
				record.URI.Collection().String() != blueskyBlockCollection ||
				record.URI.RecordKey() == "" {
				continue
			}
			block, ok := record.Value.(*bsky.GraphBlock)
			if !ok || block == nil {
				continue
			}
			recordSubject, err := syntax.ParseDID(block.Subject)
			if err != nil {
				continue
			}
			if recordSubject == subject {
				matches = append(matches, record)
			}
		}
		if next == "" {
			break
		}
		if next == cursor || seenCursors[next] {
			return nil, fmt.Errorf("list block records: repeated cursor")
		}
		seenCursors[next] = true
		cursor = next
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].URI.String() < matches[j].URI.String() })
	return matches, nil
}
