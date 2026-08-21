package scheduledposts

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

func TestPublicationHoldsOneGuardedEffectScopeThroughFinalization(t *testing.T) {
	base := newScheduledPostStoreTestPool(t)
	config := base.Config().Copy()
	// One connection holds the canonical owner/session boundary and one holds
	// the later schedule-effect lock. Attempt persistence and finalization must
	// reuse those connections rather than starving on a third acquisition.
	config.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store := NewStore(pool)
	fencer := newScheduledTestOwnerFencer(t, pool)
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:alice")
	created, err := store.Create(context.Background(), capacityCreateParams(owner, 96, now))
	if err != nil {
		t.Fatal(err)
	}
	claims, err := store.ClaimDue(context.Background(), 1, now, time.Minute)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim scheduled publication: claims=%d err=%v", len(claims), err)
	}
	claim := claims[0]

	remoteStarted := make(chan struct{})
	continueRemote := make(chan struct{})
	var blockFirstRead sync.Once
	pds := &recordingScheduledPDS{onGet: func() {
		blockFirstRead.Do(func() {
			close(remoteStarted)
			<-continueRemote
		})
	}}
	boundary := &countingPublicationEffectBoundary{
		lifecycles: lifecycles,
		client:     pds,
	}
	executor, err := pdseffects.NewExecutor(lifecycles, boundary, owner, time.Minute, func() time.Time {
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			wantOwner: owner, sessionID: "owner-session",
		},
		NewEffects: func(
			context.Context,
			syntax.DID,
			string,
		) (pdseffects.GuardedEffectCoordinator, error) {
			return executor, nil
		},
		Objects: newMemoryPrivateObjectStore(),
		Now:     func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}

	publicationDone := make(chan error, 1)
	go func() {
		publicationDone <- processor.Process(context.Background(), WorkItem{
			ID: claim.ID, OwnerDID: claim.OwnerDID,
			OwnerGeneration: claim.OwnerGeneration, LeaseToken: claim.LeaseToken,
			PayloadVersion: claim.PayloadVersion, Rkey: claim.Rkey,
			CreatedAt: claim.CreatedAt,
		})
	}()
	<-remoteStarted

	departure := NewAccountDeletion(pool, func() time.Time { return now }, fencer)
	departureDone := make(chan error, 1)
	go func() {
		_, transitionErr := lifecycles.TransitionWith(
			context.Background(),
			ownerlifecycle.TransitionRequest{
				Owner: owner, ExpectedGeneration: created.OwnerGeneration,
				To: ownerlifecycle.StateDeparted, Reason: "publication_race_test",
			},
			departure.DepartureParticipant(),
		)
		departureDone <- transitionErr
	}()
	select {
	case err := <-departureDone:
		t.Fatalf("departure crossed the live publication fence: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(continueRemote)
	if err := <-publicationDone; err != nil {
		t.Fatalf("complete publication: %v", err)
	}
	if err := <-departureDone; err != nil {
		t.Fatalf("complete departure: %v", err)
	}
	if got := boundary.calls.Load(); got != 1 {
		t.Fatalf("outer active-effect scopes=%d, want exactly one", got)
	}
	if pds.putCalls != 1 {
		t.Fatalf("PDS record writes=%d, want one", pds.putCalls)
	}
	if _, err := store.Get(context.Background(), owner, created.ID); !errors.Is(err, ErrScheduleNotFound) {
		t.Fatalf("finalized schedule remained after departure: %v", err)
	}
	var outcome string
	if err := pool.QueryRow(context.Background(), `
		SELECT remote_outcome
		FROM owner_effect_attempts
		WHERE operation_id=$1 AND owner_did=$2 AND owner_generation=$3
	`, scheduledRecordEffectIdentity(claim), owner, claim.OwnerGeneration).Scan(&outcome); err != nil {
		t.Fatalf("read durable scheduled effect: %v", err)
	}
	if outcome != string(ownerlifecycle.OutcomeAccepted) {
		t.Fatalf("durable scheduled effect outcome=%q, want accepted", outcome)
	}
	lifecycle, err := lifecycles.Get(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if lifecycle.State != ownerlifecycle.StateDeparted || lifecycle.Generation != claim.OwnerGeneration+1 {
		t.Fatalf("post-publication lifecycle=%+v, want departed next generation", lifecycle)
	}
}

type countingPublicationEffectBoundary struct {
	lifecycles *ownerlifecycle.Store
	client     auth.PDSClient
	calls      atomic.Int32
}

func (boundary *countingPublicationEffectBoundary) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	boundary.calls.Add(1)
	return boundary.lifecycles.WithActiveEffects(ctx, expected, func(effectCtx context.Context) error {
		return operation(effectCtx, boundary.client)
	})
}

var _ auth.ActiveEffectPDSBoundary = (*countingPublicationEffectBoundary)(nil)
