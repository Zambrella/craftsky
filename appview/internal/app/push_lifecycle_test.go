package app

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/push"
	"social.craftsky/appview/internal/testdb"
)

type scriptedPushLifecycleStore struct {
	lifecycle ownerlifecycle.Lifecycle
	effectErr error
	expected  []ownerlifecycle.ExpectedOwner
}

type generationBumpingPushLifecycleStore struct {
	store  *ownerlifecycle.Store
	bumped bool
}

func (store *scriptedPushLifecycleStore) Get(
	context.Context,
	syntax.DID,
) (ownerlifecycle.Lifecycle, error) {
	return store.lifecycle, nil
}

func (store *scriptedPushLifecycleStore) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	callback func(context.Context) error,
) error {
	store.expected = append([]ownerlifecycle.ExpectedOwner(nil), expected...)
	if store.effectErr != nil {
		return store.effectErr
	}
	return callback(ctx)
}

func (store *generationBumpingPushLifecycleStore) Get(
	ctx context.Context,
	owner syntax.DID,
) (ownerlifecycle.Lifecycle, error) {
	lifecycle, err := store.store.Get(ctx, owner)
	if err != nil || store.bumped {
		return lifecycle, err
	}
	store.bumped = true
	departed, err := store.store.Transition(ctx, ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: lifecycle.Generation,
		To: ownerlifecycle.StateDeparted, Reason: "push stale generation test",
	})
	if err != nil {
		return ownerlifecycle.Lifecycle{}, err
	}
	if _, err := store.store.Transition(ctx, ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: departed.Generation,
		To: ownerlifecycle.StateActive, Reason: "push stale generation test",
	}); err != nil {
		return ownerlifecycle.Lifecycle{}, err
	}
	return lifecycle, nil
}

func (store *generationBumpingPushLifecycleStore) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	callback func(context.Context) error,
) error {
	return store.store.WithActiveEffects(ctx, expected, callback)
}

func TestPushOwnerLifecycleFenceDeniesStaleGenerationBeforeCallback(t *testing.T) {
	owner := syntax.DID("did:plc:recipient")
	store := &scriptedPushLifecycleStore{
		lifecycle: ownerlifecycle.Lifecycle{
			Owner: owner, State: ownerlifecycle.StateActive, Generation: 7,
		},
		effectErr: ownerlifecycle.ErrGenerationChanged,
	}
	fence := pushOwnerLifecycleFence{store: store}
	called := false

	err := fence.WithActiveOwners(
		context.Background(),
		[]syntax.DID{owner},
		func(context.Context) error {
			called = true
			return nil
		},
	)
	if !errors.Is(err, push.ErrDeliveryLifecycleInactive) || called {
		t.Fatalf("error=%v callback=%t, want inactive without callback", err, called)
	}
	if len(store.expected) != 1 || store.expected[0].Owner != owner ||
		store.expected[0].Generation != 7 {
		t.Fatalf("expected owners = %+v", store.expected)
	}
}

func TestPushOwnerLifecycleFenceDeniesRealStaleGenerationBeforeCallback(t *testing.T) {
	fixture := newPushLifecycleFixture(t)
	recipient := syntax.DID("did:plc:stale-recipient")
	original := activatePushLifecycleOwner(t, fixture.storeA, recipient)
	store := &generationBumpingPushLifecycleStore{store: fixture.storeA}
	called := false

	err := (pushOwnerLifecycleFence{store: store}).WithActiveOwners(
		context.Background(),
		[]syntax.DID{recipient},
		func(context.Context) error {
			called = true
			return nil
		},
	)
	if !errors.Is(err, push.ErrDeliveryLifecycleInactive) || called {
		t.Fatalf("error=%v callback=%t, want real stale generation denial", err, called)
	}
	current, err := fixture.storeA.Get(context.Background(), recipient)
	if err != nil {
		t.Fatal(err)
	}
	if current.State != ownerlifecycle.StateActive || current.Generation <= original.Generation {
		t.Fatalf("current lifecycle = %+v, original = %+v", current, original)
	}
}

func TestPushOwnerLifecycleFenceRecipientOnlyAndTerminalDenial(t *testing.T) {
	fixture := newPushLifecycleFixture(t)
	recipient := syntax.DID("did:plc:recipient")
	activatePushLifecycleOwner(t, fixture.storeA, recipient)
	fence := pushOwnerLifecycleFence{store: fixture.storeA}
	called := false
	if err := fence.WithActiveOwners(
		context.Background(),
		[]syntax.DID{recipient},
		func(context.Context) error {
			called = true
			return nil
		},
	); err != nil || !called {
		t.Fatalf("recipient-only fence callback=%t err=%v", called, err)
	}

	if _, err := fixture.storeB.Terminalize(context.Background(), ownerlifecycle.TerminalizeRequest{
		Owner: recipient, Reason: "push lifecycle terminal test",
	}); err != nil {
		t.Fatalf("terminalize recipient: %v", err)
	}
	called = false
	err := fence.WithActiveOwners(
		context.Background(),
		[]syntax.DID{recipient},
		func(context.Context) error {
			called = true
			return nil
		},
	)
	if !errors.Is(err, push.ErrDeliveryLifecycleInactive) || called {
		t.Fatalf("terminal error=%v callback=%t", err, called)
	}
}

func TestPushOwnerLifecycleFenceCanonicalInversePairsDoNotDeadlock(t *testing.T) {
	fixture := newPushLifecycleFixture(t)
	a := syntax.DID("did:plc:aaa")
	b := syntax.DID("did:plc:bbb")
	activatePushLifecycleOwner(t, fixture.storeA, a)
	activatePushLifecycleOwner(t, fixture.storeA, b)

	start := make(chan struct{})
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	done := make(chan error, 2)
	for index, owners := range [][]syntax.DID{{a, b}, {b, a}} {
		owners := owners
		store := fixture.storeA
		if index == 1 {
			store = fixture.storeB
		}
		go func() {
			<-start
			done <- (pushOwnerLifecycleFence{store: store}).WithActiveOwners(
				context.Background(),
				owners,
				func(context.Context) error {
					entered <- struct{}{}
					<-release
					return nil
				},
			)
		}()
	}
	close(start)
	for range 2 {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("inverse actor/recipient pair did not enter canonical shared fence")
		}
	}
	close(release)
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("inverse actor/recipient fence: %v", err)
		}
	}
}

func TestPushLifecycleTransitionWaitsThroughSendFinalization(t *testing.T) {
	for _, test := range []struct {
		name       string
		transition syntax.DID
	}{
		{name: "recipient", transition: "did:plc:recipient"},
		{name: "actor", transition: "did:plc:actor"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPushLifecycleFixture(t)
			recipient := syntax.DID("did:plc:recipient")
			actor := syntax.DID("did:plc:actor")
			active := map[syntax.DID]ownerlifecycle.Lifecycle{
				recipient: activatePushLifecycleOwner(t, fixture.storeA, recipient),
				actor:     activatePushLifecycleOwner(t, fixture.storeA, actor),
			}
			seedPushLifecycleDelivery(t, fixture.poolA, recipient, actor)
			sender := &pushLifecycleBarrierSender{
				started: make(chan struct{}), release: make(chan struct{}),
				returned: make(chan struct{}),
			}
			dispatcher, err := push.NewDispatcherValidated(
				fixture.poolA,
				sender,
				push.DispatcherOptions{
					BatchSize: 1, Concurrency: 1, LeaseDuration: time.Minute,
					LifecycleFence: pushOwnerLifecycleFence{store: fixture.storeA},
				},
			)
			if err != nil {
				t.Fatalf("new dispatcher: %v", err)
			}

			processDone := make(chan error, 1)
			go func() {
				_, err := dispatcher.ProcessBatch(context.Background(), "lifecycle-test")
				processDone <- err
			}()
			select {
			case <-sender.started:
			case <-time.After(2 * time.Second):
				t.Fatal("provider send did not start")
			}

			finalizationBlock, err := fixture.poolB.Begin(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer finalizationBlock.Rollback(context.Background())
			if _, err := finalizationBlock.Exec(context.Background(), `
				SELECT id FROM push_deliveries FOR UPDATE
			`); err != nil {
				t.Fatalf("lock delivery finalization row: %v", err)
			}
			close(sender.release)
			select {
			case <-sender.returned:
			case <-time.After(time.Second):
				t.Fatal("provider did not return accepted result")
			}

			transitionDone := make(chan error, 1)
			go func() {
				_, err := fixture.storeB.Transition(context.Background(), ownerlifecycle.TransitionRequest{
					Owner: test.transition, ExpectedGeneration: active[test.transition].Generation,
					To: ownerlifecycle.StateDeparted, Reason: "push lifecycle barrier test",
				})
				transitionDone <- err
			}()
			select {
			case err := <-transitionDone:
				t.Fatalf("transition completed before local finalization: %v", err)
			case <-time.After(150 * time.Millisecond):
			}

			if err := finalizationBlock.Commit(context.Background()); err != nil {
				t.Fatalf("release finalization row: %v", err)
			}
			select {
			case err := <-processDone:
				if err != nil {
					t.Fatalf("process delivery: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("delivery did not finalize")
			}
			select {
			case err := <-transitionDone:
				if err != nil {
					t.Fatalf("transition after finalization: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("transition did not complete after finalization")
			}

			if _, err := fixture.poolA.Exec(context.Background(), `
				UPDATE push_deliveries
				SET status='pending',attempts=0,next_attempt_at=now(),
				    lease_owner=NULL,lease_expires_at=NULL,
				    provider_result_class=NULL,sent_at=NULL
			`); err != nil {
				t.Fatalf("reset delivery after lifecycle transition: %v", err)
			}
			lateSender := &pushLifecycleCountingSender{}
			lateDispatcher, err := push.NewDispatcherValidated(
				fixture.poolA,
				lateSender,
				push.DispatcherOptions{
					BatchSize: 1, Concurrency: 1, LeaseDuration: time.Minute,
					LifecycleFence: pushOwnerLifecycleFence{store: fixture.storeA},
				},
			)
			if err != nil {
				t.Fatalf("new post-transition dispatcher: %v", err)
			}
			if processed, err := lateDispatcher.ProcessBatch(
				context.Background(), "post-transition-test",
			); err != nil || processed != 1 {
				t.Fatalf("post-transition processed=%d err=%v", processed, err)
			}
			if lateSender.calls != 0 {
				t.Fatalf("provider calls after completed transition = %d", lateSender.calls)
			}
			var status string
			if err := fixture.poolA.QueryRow(
				context.Background(), `SELECT status FROM push_deliveries`,
			).Scan(&status); err != nil {
				t.Fatal(err)
			}
			if status != "cancelled" {
				t.Fatalf("post-transition delivery status=%q, want cancelled", status)
			}

			called := false
			err = (pushOwnerLifecycleFence{store: fixture.storeA}).WithActiveOwners(
				context.Background(),
				[]syntax.DID{actor, recipient},
				func(context.Context) error {
					called = true
					return nil
				},
			)
			if !errors.Is(err, push.ErrDeliveryLifecycleInactive) || called {
				t.Fatalf("post-transition error=%v callback=%t", err, called)
			}
		})
	}
}

type pushLifecycleBarrierSender struct {
	started  chan struct{}
	release  chan struct{}
	returned chan struct{}
	once     sync.Once
}

type pushLifecycleCountingSender struct{ calls int }

func (sender *pushLifecycleCountingSender) Send(
	context.Context,
	push.SendRequest,
) (push.ProviderResult, error) {
	sender.calls++
	return push.ProviderResult{Class: push.ResultSuccess}, nil
}

func (sender *pushLifecycleBarrierSender) Send(
	context.Context,
	push.SendRequest,
) (push.ProviderResult, error) {
	sender.once.Do(func() { close(sender.started) })
	<-sender.release
	close(sender.returned)
	return push.ProviderResult{Class: push.ResultSuccess}, nil
}

type pushLifecycleFixture struct {
	poolA, poolB   *pgxpool.Pool
	storeA, storeB *ownerlifecycle.Store
}

func newPushLifecycleFixture(t *testing.T) pushLifecycleFixture {
	t.Helper()
	poolA := testdb.WithSchema(t, `
		CREATE TABLE bluesky_profiles(did TEXT PRIMARY KEY,display_name TEXT,avatar_cid TEXT);
		CREATE TABLE craftsky_posts(uri TEXT PRIMARY KEY,reply_root_uri TEXT,reply_parent_uri TEXT);
		CREATE TABLE actor_mutes(owner_did TEXT NOT NULL,subject_did TEXT NOT NULL,PRIMARY KEY(owner_did,subject_did));
		CREATE TABLE atproto_blocks(uri TEXT PRIMARY KEY,blocker_did TEXT NOT NULL,subject_did TEXT NOT NULL);
		CREATE TABLE atproto_follows(uri TEXT PRIMARY KEY,did TEXT NOT NULL,subject_did TEXT NOT NULL,UNIQUE(did,subject_did));
		CREATE TABLE owner_lifecycles(
			owner_did TEXT PRIMARY KEY,state TEXT NOT NULL,generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL,transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL,terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ,created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE owner_effect_attempts(
			owner_did TEXT NOT NULL,owner_generation BIGINT NOT NULL,
			remote_outcome TEXT NOT NULL,projection_disposition TEXT NOT NULL,
			repeat_forbidden BOOLEAN NOT NULL DEFAULT false,completed_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE owner_purge_components(
			owner_did TEXT NOT NULL,owner_generation BIGINT NOT NULL,
			component TEXT NOT NULL,did_role TEXT NOT NULL,state TEXT NOT NULL,
			next_attempt_at TIMESTAMPTZ NOT NULL,created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY(owner_did,owner_generation,component,did_role)
		);
	`)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatalf("read push migration: %v", err)
	}
	if _, err := poolA.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply push migration: %v", err)
	}
	if _, err := poolA.Exec(context.Background(), `
		CREATE OR REPLACE FUNCTION appview_owner_is_terminal(candidate_did TEXT)
		RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
			SELECT COALESCE((SELECT state='terminal' FROM owner_lifecycles WHERE owner_did=candidate_did),false)
		$$;
		CREATE OR REPLACE FUNCTION appview_owner_is_active(candidate_did TEXT)
		RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
			SELECT COALESCE((SELECT state='active' FROM owner_lifecycles WHERE owner_did=candidate_did),false)
		$$;
	`); err != nil {
		t.Fatalf("install lifecycle predicates: %v", err)
	}
	poolB, err := pgxpool.NewWithConfig(context.Background(), poolA.Config().Copy())
	if err != nil {
		t.Fatalf("second pool: %v", err)
	}
	t.Cleanup(poolB.Close)
	newStore := func(pool *pgxpool.Pool) *ownerlifecycle.Store {
		fencer, err := ownerlifecycle.NewFencer(pool, 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		store, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		return store
	}
	return pushLifecycleFixture{
		poolA: poolA, poolB: poolB,
		storeA: newStore(poolA), storeB: newStore(poolB),
	}
}

func activatePushLifecycleOwner(
	t *testing.T,
	store *ownerlifecycle.Store,
	owner syntax.DID,
) ownerlifecycle.Lifecycle {
	t.Helper()
	row, err := store.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatalf("ensure owner %s: %v", owner, err)
	}
	row, err = store.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: row.Generation,
		To: ownerlifecycle.StateActive, Reason: "push lifecycle active test",
	})
	if err != nil {
		t.Fatalf("activate owner %s: %v", owner, err)
	}
	return row
}

func seedPushLifecycleDelivery(
	t *testing.T,
	pool *pgxpool.Pool,
	recipient syntax.DID,
	actor syntax.DID,
) {
	t.Helper()
	statements := []struct {
		statement string
		arguments []any
	}{
		{
			statement: `INSERT INTO bluesky_profiles(did,display_name) VALUES($1,'Alice')`,
			arguments: []any{actor},
		},
		{
			statement: `INSERT INTO notification_events(
			id,recipient_did,actor_did,category,subject_key,source_uri,
			source_cid,source_rkey,eligibility_scope,recipient_followed_actor,
			push_enabled_snapshot,state,first_activity_at,activity_at,indexed_at,
			initial_push_evaluated_at
		) VALUES(
			'00000000-0000-0000-0000-000000000001',$1,$2,'like','subject','source',
			'cid','rkey','everyone',false,true,'active',now(),now(),now(),now()
		)`,
			arguments: []any{recipient, actor},
		},
		{
			statement: `INSERT INTO push_installations(id,device_id,platform,fcm_token)
		VALUES('10000000-0000-0000-0000-000000000001','device','android','token')`,
		},
		{
			statement: `INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id)
		VALUES(
			'20000000-0000-0000-0000-000000000001',
			'10000000-0000-0000-0000-000000000001',$1,
			'30000000-0000-0000-0000-000000000001'
		)`,
			arguments: []any{recipient},
		},
		{
			statement: `INSERT INTO push_deliveries(
			id,notification_id,account_subscription_id,status,next_attempt_at,deadline_at
		) VALUES(
			'40000000-0000-0000-0000-000000000001',
			'00000000-0000-0000-0000-000000000001',
			'20000000-0000-0000-0000-000000000001','pending',now(),now()+interval '1 hour'
		)`,
		},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(
			context.Background(), statement.statement, statement.arguments...,
		); err != nil {
			t.Fatalf("seed push delivery: %v", err)
		}
	}
}
