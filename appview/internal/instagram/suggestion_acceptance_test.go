package instagram

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestSuggestionAcceptanceIsExplicitGenerationFencedAndIdempotent(t *testing.T) {
	pool := newPrivateSuggestionTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 15, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:accept-importer")
	target := syntax.DID("did:plc:accept-target")
	importID := uuid.MustParse("70000000-0000-4000-8000-000000000001")
	seedPrivateSuggestionLifecycle(t, pool, importer, 2, now)
	seedPrivateSuggestionLifecycle(t, pool, target, 5, now)
	seedSuggestionImport(t, pool, importID, importer, "accept.target", now)
	seedSuggestionLink(t, pool, target, "accept.target", now)

	lifecycles := newPrivateSuggestionLifecycleStore(t, pool, now)
	store, err := NewPrivateSuggestionStore(pool, lifecycles, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.ReconcileCandidate(ctx, ReconcilePrivateSuggestionParams{
		ID:          uuid.MustParse("71000000-0000-4000-8000-000000000001"),
		ImporterDID: importer, TargetDID: target, ImportID: importID,
		Username: "accept.target", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	executor := &recordingSuggestionFollowExecutor{
		result: SuggestionFollowResult{
			Outcome:   SuggestionFollowed,
			RecordURI: syntax.ATURI("at://did:plc:accept-importer/app.bsky.graph.follow/3laccept"),
			RecordCID: "bafy-follow",
		},
	}
	service, err := NewSuggestionService(store, lifecycles, privateSuggestionAllowPolicy{}, executor)
	if err != nil {
		t.Fatal(err)
	}

	first, err := service.Accept(ctx, importer, created.Suggestion.ID, "session-one")
	if err != nil {
		t.Fatalf("first Accept: %v", err)
	}
	second, err := service.Accept(ctx, importer, created.Suggestion.ID, "session-two")
	if err != nil {
		t.Fatalf("second Accept: %v", err)
	}
	if first.State != SuggestionFollowed || second.State != SuggestionFollowed ||
		first.ResultRecordURI == nil || second.ResultRecordURI == nil ||
		*first.ResultRecordURI != *second.ResultRecordURI {
		t.Fatalf("acceptance results = %+v / %+v", first, second)
	}
	requests := executor.Requests()
	if len(requests) != 1 {
		t.Fatalf("follow executor calls = %d, want 1", len(requests))
	}
	request := requests[0]
	if request.Owner != importer || request.Target != target ||
		request.OwnerGeneration != 2 || request.TargetGeneration != 5 ||
		request.SessionID != "session-one" || request.OperationID == "" || request.Rkey == "" {
		t.Fatalf("follow request = %+v", request)
	}

	if _, err := service.Accept(ctx, syntax.DID("did:plc:foreign"), created.Suggestion.ID, "foreign-session"); !errors.Is(err, ErrInstagramResourceNotFound) {
		t.Fatalf("foreign accept error = %v, want hidden not found", err)
	}
	if len(executor.Requests()) != 1 {
		t.Fatal("foreign accept reached follow executor")
	}
}

func TestSuggestionAcceptanceMakesNoCallAfterTargetDeparture(t *testing.T) {
	pool := newPrivateSuggestionTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 15, 30, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:stale-importer")
	target := syntax.DID("did:plc:stale-target")
	importID := uuid.MustParse("72000000-0000-4000-8000-000000000001")
	seedPrivateSuggestionLifecycle(t, pool, importer, 3, now)
	seedPrivateSuggestionLifecycle(t, pool, target, 8, now)
	seedSuggestionImport(t, pool, importID, importer, "stale.target", now)
	seedSuggestionLink(t, pool, target, "stale.target", now)

	lifecycles := newPrivateSuggestionLifecycleStore(t, pool, now)
	store, err := NewPrivateSuggestionStore(pool, lifecycles, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.ReconcileCandidate(ctx, ReconcilePrivateSuggestionParams{
		ID:          uuid.MustParse("73000000-0000-4000-8000-000000000001"),
		ImporterDID: importer, TargetDID: target, ImportID: importID,
		Username: "stale.target", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycles.Transition(ctx, ownerlifecycle.TransitionRequest{
		Owner: target, ExpectedGeneration: 8, To: ownerlifecycle.StateDeparted,
		Reason: "profileDeleted",
	}); err != nil {
		t.Fatal(err)
	}
	executor := &recordingSuggestionFollowExecutor{}
	service, err := NewSuggestionService(store, lifecycles, privateSuggestionAllowPolicy{}, executor)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Accept(ctx, importer, created.Suggestion.ID, "session")
	if !errors.Is(err, ownerlifecycle.ErrGenerationChanged) &&
		!errors.Is(err, ownerlifecycle.ErrOwnerNotActive) {
		t.Fatalf("stale accept error = %v, want lifecycle denial", err)
	}
	if len(executor.Requests()) != 0 {
		t.Fatal("stale suggestion reached follow executor")
	}
}

type recordingSuggestionFollowExecutor struct {
	mu       sync.Mutex
	requests []SuggestionFollowRequest
	result   SuggestionFollowResult
	err      error
}

func (executor *recordingSuggestionFollowExecutor) FollowSuggestion(
	_ context.Context,
	request SuggestionFollowRequest,
) (SuggestionFollowResult, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.requests = append(executor.requests, request)
	return executor.result, executor.err
}

func (executor *recordingSuggestionFollowExecutor) Requests() []SuggestionFollowRequest {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return append([]SuggestionFollowRequest(nil), executor.requests...)
}
