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
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/testdb"
)

func TestPublicationHoldsOneGuardedEffectScopeThroughFinalization(t *testing.T) {
	base := testdb.WithMigratedSchema(t)
	config := base.Config().Copy()
	// One connection holds the canonical owner/session boundary and one holds
	// the later schedule-effect lock. Attempt persistence and finalization must
	// reuse those connections rather than starving on a third acquisition. The
	// third slot stays reserved until the competing lifecycle transition starts.
	config.MaxConns = 3
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	reserved, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	reservedReleased := false
	t.Cleanup(func() {
		if !reservedReleased {
			reserved.Release()
		}
	})
	store := NewStore(pool)
	fencer := newScheduledTestOwnerFencer(t, pool)
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_profiles(did,record_cid,created_at)
		VALUES($1,'bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
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
	commandStore, err := pdscommands.NewStore(pdscommands.StoreConfig{
		Pool: pool, Lifecycles: lifecycles, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	appendService, err := pdscommands.NewAppendCommandService(pdscommands.AppendCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return boundary, nil
		},
		NewRecordKey: func() (syntax.RecordKey, error) { return "unused", nil },
		Now:          func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewPublicationProcessor(PublicationProcessorOptions{
		Store: store,
		Sessions: stubPublicationSessionSelector{
			wantOwner: owner, sessionID: "owner-session",
		},
		NewCommands: func(
			context.Context,
			syntax.DID,
			string,
		) (GuardedCommandCoordinator, error) {
			return &journaledRecordingGuardedCoordinator{
				executor: executor, append: appendService, client: pds,
			}, nil
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
	select {
	case <-remoteStarted:
	case publicationErr := <-publicationDone:
		t.Fatalf("publication stopped before remote read: %v", publicationErr)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for publication remote read")
	}

	departure := NewAccountDeletion(pool, func() time.Time { return now }, fencer)
	departureDone := make(chan error, 1)
	departureStarted := make(chan struct{})
	go func() {
		close(departureStarted)
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
	waitForScheduledSignal(t, departureStarted, "departure transition start")
	reserved.Release()
	reservedReleased = true
	waitForScheduledAdvisoryWaiter(t, base, departureDone)

	close(continueRemote)
	if err := waitForScheduledResult(t, publicationDone, "publication completion"); err != nil {
		t.Fatalf("complete publication: %v", err)
	}
	if err := waitForScheduledResult(t, departureDone, "departure completion"); err != nil {
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
	var commandState string
	if err := pool.QueryRow(context.Background(), `
		SELECT state
		FROM pds_commands
		WHERE operation_kind='scheduled_post_publish'
		  AND owner_did=$1 AND owner_generation=$2
	`, owner, claim.OwnerGeneration).Scan(&commandState); err != nil {
		t.Fatalf("read durable scheduled command: %v", err)
	}
	if commandState != string(pdscommands.CommandAccepted) {
		t.Fatalf("durable scheduled command state=%q, want accepted", commandState)
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
