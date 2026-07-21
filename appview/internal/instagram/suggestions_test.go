package instagram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestSuggestionServiceFiltersEveryListItemThroughPolicy(t *testing.T) {
	store, pool := newSuggestionTestStore(t)
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	now := time.Date(2026, 7, 19, 18, 0, 0, 0, time.UTC)
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000251")
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000252")
	seedSuggestionImport(t, pool, importID, alice, "synthetic.bob", now)
	seedSuggestionLink(t, pool, bob, "synthetic.bob", now)
	if _, err := store.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{ID: suggestionID, ImporterDID: alice, TargetDID: bob, ImportID: importID, Username: "synthetic.bob", Now: now}); err != nil {
		t.Fatal(err)
	}
	policy := &recordingSuggestionPolicy{decision: EligibilityDecision{Reason: EligibilityMembership}}
	service, err := NewSuggestionService(SuggestionServiceOptions{Repository: store, Policy: policy, Now: func() time.Time { return now }, DefaultPageSize: 20, MaxPageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	items, _, err := service.ListSuggestions(ctx, alice, 20, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 || len(policy.calls) != 1 || policy.calls[0].stage != EligibilityAtList {
		t.Fatalf("items=%+v calls=%+v", items, policy.calls)
	}
	var state InstagramSuggestionState
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_follow_suggestions WHERE id = $1`, suggestionID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != SuggestionInvalidated {
		t.Fatalf("denied list state=%s", state)
	}
}

func TestSuggestionServiceAcceptanceUsesStableIdempotentPutRecord(t *testing.T) {
	store, pool := newSuggestionTestStore(t)
	ctx := context.Background()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	now := time.Date(2026, 7, 19, 18, 30, 0, 0, time.UTC)
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000253")
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000254")
	seedSuggestionImport(t, pool, importID, alice, "synthetic.bob", now)
	seedSuggestionLink(t, pool, bob, "synthetic.bob", now)
	if _, err := store.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{ID: suggestionID, ImporterDID: alice, TargetDID: bob, ImportID: importID, Username: "synthetic.bob", Now: now}); err != nil {
		t.Fatal(err)
	}
	policy := &recordingSuggestionPolicy{decision: EligibilityDecision{Eligible: true, Reason: EligibilityAllowed}}
	service, err := NewSuggestionService(SuggestionServiceOptions{Repository: store, Policy: policy, Now: func() time.Time { return now }, DefaultPageSize: 20, MaxPageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	writer := &recordingFollowWriter{err: errors.New("synthetic provider unavailable")}
	if _, err := service.AcceptSuggestion(ctx, alice, suggestionID, writer); !errors.Is(err, ErrInstagramFollowWriteUnavailable) {
		t.Fatalf("first accept error=%v", err)
	}
	if len(writer.calls) != 1 {
		t.Fatalf("first writer calls=%+v", writer.calls)
	}
	firstRkey := writer.calls[0].rkey
	writer.err = nil
	accepted, err := service.AcceptSuggestion(ctx, alice, suggestionID, writer)
	if err != nil {
		t.Fatalf("retry accept: %v", err)
	}
	if accepted.State != SuggestionAccepted || len(writer.calls) != 2 || writer.calls[1].rkey != firstRkey {
		t.Fatalf("accepted=%+v writer calls=%+v", accepted, writer.calls)
	}
	replay, err := service.AcceptSuggestion(ctx, alice, suggestionID, writer)
	if err != nil || replay.State != SuggestionAccepted || len(writer.calls) != 2 {
		t.Fatalf("replay=%+v err=%v calls=%+v", replay, err, writer.calls)
	}
	if len(policy.calls) != 2 || policy.calls[0].stage != EligibilityAtAccept || policy.calls[1].stage != EligibilityAtAccept {
		t.Fatalf("policy calls=%+v", policy.calls)
	}
}

func TestSuggestionServiceAlreadyFollowingAndIneligibleNeverWrite(t *testing.T) {
	for _, test := range []struct {
		name      string
		decision  EligibilityDecision
		wantState InstagramSuggestionState
		wantErr   error
	}{
		{name: "already following", decision: EligibilityDecision{Reason: EligibilityAlreadyFollowing}, wantState: SuggestionAlreadyFollowing},
		{name: "ineligible", decision: EligibilityDecision{Reason: EligibilityRelationshipSafety}, wantState: SuggestionInvalidated, wantErr: ErrInstagramSuggestionIneligible},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, pool := newSuggestionTestStore(t)
			ctx := context.Background()
			alice := syntax.DID("did:plc:synthetic-alice")
			bob := syntax.DID("did:plc:synthetic-bob")
			now := time.Date(2026, 7, 19, 19, 0, 0, 0, time.UTC)
			importID := uuid.New()
			suggestionID := uuid.New()
			seedSuggestionImport(t, pool, importID, alice, "synthetic.bob", now)
			seedSuggestionLink(t, pool, bob, "synthetic.bob", now)
			if _, err := store.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{ID: suggestionID, ImporterDID: alice, TargetDID: bob, ImportID: importID, Username: "synthetic.bob", Now: now}); err != nil {
				t.Fatal(err)
			}
			policy := &recordingSuggestionPolicy{decision: test.decision}
			service, err := NewSuggestionService(SuggestionServiceOptions{Repository: store, Policy: policy, Now: func() time.Time { return now }})
			if err != nil {
				t.Fatal(err)
			}
			writer := &recordingFollowWriter{}
			result, err := service.AcceptSuggestion(ctx, alice, suggestionID, writer)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error=%v want=%v", err, test.wantErr)
			}
			if len(writer.calls) != 0 {
				t.Fatalf("writer calls=%+v", writer.calls)
			}
			if test.wantErr == nil && result.State != test.wantState {
				t.Fatalf("result=%+v", result)
			}
			var state InstagramSuggestionState
			if err := pool.QueryRow(ctx, `SELECT state FROM instagram_follow_suggestions WHERE id = $1`, suggestionID).Scan(&state); err != nil {
				t.Fatal(err)
			}
			if state != test.wantState {
				t.Fatalf("stored state=%s want=%s", state, test.wantState)
			}
		})
	}
}

type recordingSuggestionPolicy struct {
	decision EligibilityDecision
	err      error
	calls    []struct {
		stage   EligibilityStage
		request SuggestionEligibilityRequest
	}
}

func (p *recordingSuggestionPolicy) Evaluate(_ context.Context, stage EligibilityStage, request SuggestionEligibilityRequest) (EligibilityDecision, error) {
	p.calls = append(p.calls, struct {
		stage   EligibilityStage
		request SuggestionEligibilityRequest
	}{stage: stage, request: request})
	return p.decision, p.err
}

type recordingFollowWriter struct {
	err   error
	calls []struct {
		owner     syntax.DID
		target    syntax.DID
		rkey      syntax.RecordKey
		createdAt time.Time
	}
}

func (w *recordingFollowWriter) PutFollow(_ context.Context, owner, target syntax.DID, rkey syntax.RecordKey, createdAt time.Time) error {
	w.calls = append(w.calls, struct {
		owner     syntax.DID
		target    syntax.DID
		rkey      syntax.RecordKey
		createdAt time.Time
	}{owner: owner, target: target, rkey: rkey, createdAt: createdAt})
	return w.err
}

var _ InstagramSuggestionEligibilityPolicy = (*recordingSuggestionPolicy)(nil)
var _ InstagramFollowWriter = (*recordingFollowWriter)(nil)
