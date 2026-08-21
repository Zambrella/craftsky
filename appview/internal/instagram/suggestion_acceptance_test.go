package instagram

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
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
	store, err := NewPrivateSuggestionStore(pool, lifecycles, notifications.NewService(), func() time.Time { return now })
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
		lifecycles: lifecycles,
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
		executor.SessionID() != "session-one" || request.OperationID == "" ||
		request.MutationKey != request.OperationID || request.Rkey == "" {
		t.Fatalf("follow request = %+v", request)
	}

	if _, err := service.Accept(ctx, syntax.DID("did:plc:foreign"), created.Suggestion.ID, "foreign-session"); !errors.Is(err, ErrInstagramResourceNotFound) {
		t.Fatalf("foreign accept error = %v, want hidden not found", err)
	}
	if len(executor.Requests()) != 1 {
		t.Fatal("foreign accept reached follow executor")
	}
}

func TestSuggestionAcceptanceMakesNoCallAfterTargetTerminalization(t *testing.T) {
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
	store, err := NewPrivateSuggestionStore(pool, lifecycles, notifications.NewService(), func() time.Time { return now })
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
	if _, err := lifecycles.Terminalize(ctx, ownerlifecycle.TerminalizeRequest{
		Owner: target, Reason: "didDeactivated",
	}); err != nil {
		t.Fatal(err)
	}
	executor := &recordingSuggestionFollowExecutor{lifecycles: lifecycles}
	service, err := NewSuggestionService(store, lifecycles, privateSuggestionAllowPolicy{}, executor)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Accept(ctx, importer, created.Suggestion.ID, "session")
	if !errors.Is(err, ownerlifecycle.ErrGenerationChanged) &&
		!errors.Is(err, ownerlifecycle.ErrOwnerNotActive) &&
		!errors.Is(err, ownerlifecycle.ErrTerminalOwner) {
		t.Fatalf("stale accept error = %v, want lifecycle denial", err)
	}
	if len(executor.Requests()) != 0 {
		t.Fatal("stale suggestion reached follow executor")
	}
}

func TestSuggestionAcceptanceReplaysTheSameDurableIdentityAfterResponseLoss(t *testing.T) {
	pool := newPrivateSuggestionTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:replay-importer")
	target := syntax.DID("did:plc:replay-target")
	importID := uuid.MustParse("74000000-0000-4000-8000-000000000001")
	suggestionID := uuid.MustParse("75000000-0000-4000-8000-000000000001")
	seedPrivateSuggestionLifecycle(t, pool, importer, 4, now)
	seedPrivateSuggestionLifecycle(t, pool, target, 9, now)
	seedSuggestionImport(t, pool, importID, importer, "replay.target", now)
	seedSuggestionLink(t, pool, target, "replay.target", now)
	lifecycles := newPrivateSuggestionLifecycleStore(t, pool, now)
	store, err := NewPrivateSuggestionStore(pool, lifecycles, notifications.NewService(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.ReconcileCandidate(ctx, ReconcilePrivateSuggestionParams{
		ID: suggestionID, ImporterDID: importer, TargetDID: target,
		ImportID: importID, Username: "replay.target", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantResult := SuggestionFollowResult{
		Outcome: SuggestionFollowed,
		RecordURI: syntax.ATURI(
			"at://did:plc:replay-importer/app.bsky.graph.follow/3l75000000000040008000000000000001",
		),
		RecordCID: "bafy-reconciled-follow",
	}
	executor := &recordingSuggestionFollowExecutor{
		lifecycles: lifecycles,
		errs: []error{&pdseffects.OutcomeAmbiguousError{
			OperationID: "instagram-suggestion:" + suggestionID.String(),
		}},
		results: []SuggestionFollowResult{{}, wantResult},
	}
	service, err := NewSuggestionService(store, lifecycles, privateSuggestionAllowPolicy{}, executor)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.Accept(ctx, importer, created.Suggestion.ID, "session-one"); err == nil {
		t.Fatal("first acceptance unexpectedly resolved an ambiguous response")
	}
	replayed, err := service.Accept(ctx, importer, created.Suggestion.ID, "session-two")
	if err != nil {
		t.Fatalf("replay acceptance: %v", err)
	}
	if replayed.State != SuggestionFollowed || replayed.ResultRecordURI == nil ||
		*replayed.ResultRecordURI != wantResult.RecordURI {
		t.Fatalf("replayed suggestion = %+v", replayed)
	}
	requests := executor.Requests()
	if len(requests) != 2 {
		t.Fatalf("follow executor calls = %d, want 2 durable invocations", len(requests))
	}
	if requests[0].OperationID != requests[1].OperationID ||
		requests[0].MutationKey != requests[1].MutationKey ||
		requests[0].Rkey != requests[1].Rkey {
		t.Fatalf("replay identities differ: %+v / %+v", requests[0], requests[1])
	}
}

type recordingSuggestionFollowExecutor struct {
	mu         sync.Mutex
	lifecycles *ownerlifecycle.Store
	sessionIDs []string
	requests   []SuggestionFollowRequest
	result     SuggestionFollowResult
	err        error
	results    []SuggestionFollowResult
	errs       []error
}

func (executor *recordingSuggestionFollowExecutor) WithSuggestionEffects(
	ctx context.Context,
	_ syntax.DID,
	sessionID string,
	expected []ownerlifecycle.ExpectedOwner,
	operation SuggestionEffectOperation,
) error {
	executor.mu.Lock()
	executor.sessionIDs = append(executor.sessionIDs, sessionID)
	executor.mu.Unlock()
	return executor.lifecycles.WithActiveEffects(ctx, expected, func(effectCtx context.Context) error {
		return operation(effectCtx, executor)
	})
}

func (executor *recordingSuggestionFollowExecutor) FollowSuggestion(
	_ context.Context,
	request SuggestionFollowRequest,
) (SuggestionFollowResult, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.requests = append(executor.requests, request)
	index := len(executor.requests) - 1
	if index < len(executor.errs) && executor.errs[index] != nil {
		return SuggestionFollowResult{}, executor.errs[index]
	}
	if index < len(executor.results) {
		return executor.results[index], nil
	}
	return executor.result, executor.err
}

func (executor *recordingSuggestionFollowExecutor) Requests() []SuggestionFollowRequest {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return append([]SuggestionFollowRequest(nil), executor.requests...)
}

func (executor *recordingSuggestionFollowExecutor) SessionID() string {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if len(executor.sessionIDs) == 0 {
		return ""
	}
	return executor.sessionIDs[0]
}
