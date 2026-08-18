package followwrite

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestCoordinatorPersistsBeforeOneDeterministicFollow(t *testing.T) {
	pool, lifecycles, now := newFollowCoordinatorTestStore(t)
	owner := syntax.DID("did:plc:coordinator-owner")
	target := syntax.DID("did:plc:coordinator-target")
	seedFollowCoordinatorOwner(t, pool, owner, 2, now)
	seedFollowCoordinatorOwner(t, pool, target, 4, now)
	pds := &coordinatorPDS{}
	coordinator, err := NewCoordinator(
		lifecycles,
		NewService(func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil }),
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := DeterministicRequest{
		OperationID: "suggestion:one", Owner: owner, Target: target,
		OwnerGeneration: 2, SessionID: "owner-session",
		Rkey: "3lcoordinatorone", CreatedAt: now,
	}

	var first, replay DeterministicResult
	err = lifecycles.WithActiveEffects(context.Background(), []ownerlifecycle.ExpectedOwner{
		{Owner: owner, Generation: 2}, {Owner: target, Generation: 4},
	}, func(effectCtx context.Context) error {
		var executeErr error
		first, executeErr = coordinator.ExecuteDeterministic(effectCtx, request)
		if executeErr != nil {
			return executeErr
		}
		replay, executeErr = coordinator.ExecuteDeterministic(effectCtx, request)
		return executeErr
	})
	if err != nil {
		t.Fatal(err)
	}
	if pds.puts != 1 || pds.gets != 0 {
		t.Fatalf("PDS puts/gets = %d/%d, want 1/0", pds.puts, pds.gets)
	}
	if first.RecordURI == "" || replay.RecordURI != first.RecordURI {
		t.Fatalf("results = %+v / %+v", first, replay)
	}
	var outcome ownerlifecycle.EffectOutcome
	if err := pool.QueryRow(context.Background(), `
		SELECT remote_outcome FROM owner_effect_attempts WHERE operation_id='suggestion:one'
	`).Scan(&outcome); err != nil {
		t.Fatal(err)
	}
	if outcome != ownerlifecycle.OutcomeAccepted {
		t.Fatalf("effect outcome = %q", outcome)
	}
}

func TestCoordinatorReconcilesLostPutResponseWithoutRepeatingWrite(t *testing.T) {
	pool, lifecycles, now := newFollowCoordinatorTestStore(t)
	owner := syntax.DID("did:plc:lost-owner")
	target := syntax.DID("did:plc:lost-target")
	seedFollowCoordinatorOwner(t, pool, owner, 3, now)
	seedFollowCoordinatorOwner(t, pool, target, 6, now)
	pds := &coordinatorPDS{putErr: errors.New("response lost")}
	coordinator, err := NewCoordinator(
		lifecycles,
		NewService(func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil }),
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := DeterministicRequest{
		OperationID: "suggestion:lost", Owner: owner, Target: target,
		OwnerGeneration: 3, SessionID: "owner-session",
		Rkey: "3llostresponse", CreatedAt: now,
	}
	expected := []ownerlifecycle.ExpectedOwner{
		{Owner: owner, Generation: 3}, {Owner: target, Generation: 6},
	}
	err = lifecycles.WithActiveEffects(context.Background(), expected, func(effectCtx context.Context) error {
		_, executeErr := coordinator.ExecuteDeterministic(effectCtx, request)
		return executeErr
	})
	if !errors.Is(err, ErrOutcomeUncertain) {
		t.Fatalf("first execute error = %v, want ErrOutcomeUncertain", err)
	}
	if pds.puts != 1 {
		t.Fatalf("first puts = %d", pds.puts)
	}
	pds.putErr = nil
	err = lifecycles.WithActiveEffects(context.Background(), expected, func(effectCtx context.Context) error {
		_, executeErr := coordinator.ExecuteDeterministic(effectCtx, request)
		return executeErr
	})
	if err != nil {
		t.Fatalf("reconcile execute: %v", err)
	}
	if pds.puts != 1 || pds.gets != 1 {
		t.Fatalf("reconciled puts/gets = %d/%d, want 1/1", pds.puts, pds.gets)
	}
}

func newFollowCoordinatorTestStore(
	t *testing.T,
) (*pgxpool.Pool, *ownerlifecycle.Store, time.Time) {
	t.Helper()
	pool := testdb.WithSchema(t, "")
	for _, path := range []string{
		"../../migrations/000038_owner_auth_lifecycle.up.sql",
		"../../migrations/000039_owner_effects_terminal_purge.up.sql",
		"../../migrations/000049_pds_effect_action.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	now := time.Date(2026, 8, 14, 17, 0, 0, 0, time.UTC)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return pool, lifecycles, now
}

func seedFollowCoordinatorOwner(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	generation int64,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',$2,1,'testActive',$3,$3,$3)
	`, owner, generation, now); err != nil {
		t.Fatal(err)
	}
}

type coordinatorPDS struct {
	puts      int
	gets      int
	putErr    error
	stored    *followRecord
	storedCID string
}

func (pds *coordinatorPDS) PutRecord(
	_ context.Context,
	_ syntax.DID,
	_ string,
	_ string,
	record any,
) error {
	pds.puts++
	value := record.(map[string]any)
	pds.stored = &followRecord{
		Type:    Collection,
		Subject: syntax.DID(value["subject"].(string)),
	}
	pds.storedCID = "bafy-follow"
	return pds.putErr
}

func (pds *coordinatorPDS) GetRecord(
	_ context.Context,
	_ syntax.DID,
	_ string,
	_ string,
	out any,
) (string, error) {
	pds.gets++
	if pds.stored == nil {
		return "", auth.ErrRecordNotFound
	}
	*out.(*followRecord) = *pds.stored
	return pds.storedCID, nil
}

func (*coordinatorPDS) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", errors.New("unexpected create record")
}

func (*coordinatorPDS) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return errors.New("unexpected delete record")
}

func (*coordinatorPDS) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, errors.New("unexpected upload blob")
}

var _ auth.PDSClient = (*coordinatorPDS)(nil)
