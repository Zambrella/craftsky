package moderation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/testdb"
)

func TestEquivalentTrustedAdaptersUseOneAdjudicationBoundaryAndReplaySafely(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	store := NewStore(pool)
	service := NewService(store, testVisibilityWriter{}, time.Now)
	adapter := NewTrustedInputAdapter(service)
	ctx := context.Background()
	decision := Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectStrike}}
	sources := []AuthenticatedSource{
		{SourceSystem: "admin-api", ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler")},
		{SourceSystem: "future-adapter", ActorID: "trusted-ingest", SourceDID: syntax.DID("did:plc:labeler")},
	}
	for index, source := range sources {
		caseRow := seedAdjudicationCase(t, pool, store, "adapter-report-"+string(rune('0'+index)), syntax.DID("did:plc:adapter-owner"+string(rune('0'+index))))
		request := ResolveRequest{CaseID: caseRow.ID, ExpectedRevision: 0, Decision: decision}
		command, err := adapter.Normalize(source, "source-replay-0001", request)
		if err != nil {
			t.Fatalf("normalize source %s: %v", source.SourceSystem, err)
		}
		first, err := adapter.Invoke(ctx, command)
		if err != nil {
			t.Fatalf("source %s: %v", source.SourceSystem, err)
		}
		replay, err := adapter.Invoke(ctx, command)
		if err != nil || !replay.Replayed || replay.EventID != first.EventID {
			t.Fatalf("source %s replay = %+v, err=%v", source.SourceSystem, replay, err)
		}
		invalid := command
		invalid.ReplayID += "-invalid"
		invalid.Decision = Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence"}
		if _, err := adapter.Invoke(ctx, invalid); !errors.Is(err, ErrInvalidDecision) {
			t.Fatalf("source %s invalid decision error = %v, want %v", source.SourceSystem, err, ErrInvalidDecision)
		}
	}
	var events, strikes int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM moderation_case_events),(SELECT count(*) FROM moderation_case_strikes)`).Scan(&events, &strikes); err != nil {
		t.Fatal(err)
	}
	if events != 2 || strikes != 2 {
		t.Fatalf("events/strikes = %d/%d, want 2/2", events, strikes)
	}
}
