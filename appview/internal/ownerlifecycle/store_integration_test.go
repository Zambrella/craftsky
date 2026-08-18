package ownerlifecycle

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestStoreLifecycleGenerationEpochAndEffectClosure(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:lifecycle-owner")

	onboarding, err := store.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatalf("ensure onboarding owner: %v", err)
	}
	assertLifecycle(t, onboarding, StateDeparted, 1, 1)

	active, err := store.Transition(ctx, TransitionRequest{
		Owner: owner, ExpectedGeneration: 1, To: StateActive, Reason: "profileCreated",
	})
	if err != nil {
		t.Fatalf("activate owner: %v", err)
	}
	assertLifecycle(t, active, StateActive, 2, 1)

	fingerprint := sha256.Sum256([]byte("canonical request"))
	var prepared EffectAttempt
	err = store.WithActiveEffects(ctx, []ExpectedOwner{{Owner: owner, Generation: active.Generation}}, func(effectCtx context.Context) error {
		prepared, err = store.CreateEffectAttempt(effectCtx, NewEffectAttempt{
			OperationID:        "post:one",
			Owner:              owner,
			OwnerGeneration:    active.Generation,
			Kind:               EffectPDSRecord,
			DeterministicKey:   "at://did:plc:lifecycle-owner/social.craftsky.feed.post/one",
			RequestFingerprint: fingerprint,
			RemoteDeadline:     time.Now().Add(time.Minute),
		})
		if err != nil {
			return err
		}
		prepared, err = store.MarkAttemptDispatched(effectCtx, prepared.OperationID, owner, active.Generation)
		if err != nil {
			return err
		}
		if _, retryErr := store.MarkAttemptDispatched(effectCtx, prepared.OperationID, owner, active.Generation); !errors.Is(retryErr, ErrAttemptState) {
			return errors.New("dispatched effect could cross the remote boundary twice")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("prepare and dispatch effect attempt: %v", err)
	}
	if prepared.Outcome != OutcomeDispatched || !prepared.RepeatForbidden {
		t.Fatalf("dispatched attempt = %+v", prepared)
	}

	departed, err := store.Transition(ctx, TransitionRequest{
		Owner: owner, ExpectedGeneration: active.Generation, To: StateDeparted, Reason: "profileDeleted",
	})
	if err != nil {
		t.Fatalf("depart owner: %v", err)
	}
	assertLifecycle(t, departed, StateDeparted, 3, 2)
	closed, err := store.GetEffectAttempt(ctx, prepared.OperationID)
	if err != nil {
		t.Fatalf("read closed effect attempt: %v", err)
	}
	if closed.Outcome != OutcomeUnknownPreTransition ||
		closed.ProjectionDisposition != ProjectionHiddenNonActive || !closed.RepeatForbidden {
		t.Fatalf("closed effect attempt = %+v", closed)
	}

	reconciled, err := store.ReconcileEffectAttempt(ctx, ReconcileEffectRequest{
		OperationID: prepared.OperationID,
		Owner:       owner,
		Outcome:     OutcomeReconciledAccepted,
		ResultCID:   "bafy-reconciled",
	})
	if err != nil {
		t.Fatalf("reconcile effect attempt: %v", err)
	}
	if reconciled.Outcome != OutcomeReconciledAccepted ||
		reconciled.ProjectionDisposition != ProjectionHiddenNonActive {
		t.Fatalf("reconciled attempt = %+v", reconciled)
	}

	rejoined, err := store.Transition(ctx, TransitionRequest{
		Owner: owner, ExpectedGeneration: departed.Generation, To: StateActive, Reason: "profileRecreated",
	})
	if err != nil {
		t.Fatalf("rejoin owner: %v", err)
	}
	assertLifecycle(t, rejoined, StateActive, 4, 2)
	var confirmed EffectAttempt
	err = store.WithActiveEffects(ctx, []ExpectedOwner{{Owner: owner, Generation: rejoined.Generation}}, func(effectCtx context.Context) error {
		confirmed, err = store.ConfirmReconciledEffectCurrent(
			effectCtx, prepared.OperationID, owner, rejoined.Generation, "bafy-reconciled",
		)
		return err
	})
	if err != nil {
		t.Fatalf("confirm reconciled current PDS record: %v", err)
	}
	if confirmed.ProjectionDisposition != ProjectionEligibleCurrent {
		t.Fatalf("confirmed effect disposition = %q, want eligible", confirmed.ProjectionDisposition)
	}

	loggedOut, err := store.AdvanceAuthEpoch(ctx, owner, rejoined.Generation, "logoutAll")
	if err != nil {
		t.Fatalf("advance auth epoch: %v", err)
	}
	assertLifecycle(t, loggedOut, StateActive, 4, 3)
}

func TestStoreActiveEffectSessionUsesOneOwnerFenceScope(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:effect-session-owner")
	participant := syntax.DID("did:plc:effect-session-participant")
	for _, item := range []struct {
		owner      syntax.DID
		generation int64
	}{
		{owner: owner, generation: 2},
		{owner: participant, generation: 5},
	} {
		if _, err := store.pool.Exec(ctx, `
			INSERT INTO owner_lifecycles(
				owner_did,state,generation,auth_epoch,transition_reason,
				transitioned_at,created_at,updated_at
			) VALUES($1,'active',$2,1,'test',now(),now(),now())
		`, item.owner, item.generation); err != nil {
			t.Fatal(err)
		}
	}

	called := false
	err := store.WithActiveEffectSessionAuth(
		ctx,
		[]ExpectedOwner{
			{Owner: participant, Generation: 5},
			{Owner: owner, Generation: 2},
		},
		owner,
		"parent-session",
		func(effectCtx context.Context, authority Lifecycle) error {
			called = true
			if authority.Owner != owner || authority.Generation != 2 || authority.State != StateActive {
				t.Fatalf("session authority = %+v", authority)
			}
			fingerprint := sha256.Sum256([]byte("combined effect session"))
			if _, err := store.CreateEffectAttempt(effectCtx, NewEffectAttempt{
				OperationID: "effect-session:one", Owner: owner, OwnerGeneration: 2,
				Kind:               EffectPDSRecord,
				DeterministicKey:   "at://did:plc:effect-session-owner/social.craftsky.feed.post/one",
				RequestFingerprint: fingerprint,
				RemoteDeadline:     time.Now().Add(time.Minute),
			}); err != nil {
				return err
			}
			return store.WithAuthTransaction(effectCtx, func(tx pgx.Tx) error {
				var state State
				return tx.QueryRow(effectCtx, `
					SELECT state FROM owner_lifecycles WHERE owner_did=$1 FOR SHARE
				`, owner).Scan(&state)
			})
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("combined effect/session callback was not called")
	}
}

func TestStoreMissingTargetExpectationFencesConcurrentTerminalization(t *testing.T) {
	storeA, storeB := ownerLifecycleTestStores(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:missing-target-effect-owner")
	unknownTarget := syntax.DID("did:plc:missing-target-terminal-race")
	if _, err := storeA.pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',2,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	effectDone := make(chan error, 1)
	go func() {
		effectDone <- storeA.WithActiveEffectSessionAuth(
			ctx,
			[]ExpectedOwner{
				{Owner: owner, Generation: 2},
				{Owner: unknownTarget, AllowMissing: true},
			},
			owner,
			"missing-target-parent-session",
			func(context.Context, Lifecycle) error {
				close(entered)
				<-release
				return nil
			},
		)
	}()
	select {
	case <-entered:
	case err := <-effectDone:
		t.Fatalf("missing-target effect failed before callback: %v", err)
	case <-time.After(time.Second):
		t.Fatal("missing-target effect did not enter callback")
	}

	terminalDone := make(chan error, 1)
	go func() {
		_, err := storeB.Terminalize(ctx, TerminalizeRequest{
			Owner: unknownTarget, Reason: "identityDeleted",
		})
		terminalDone <- err
	}()
	select {
	case err := <-terminalDone:
		t.Fatalf("terminalization crossed held missing-target fence: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if err := <-effectDone; err != nil {
		t.Fatal(err)
	}
	if err := <-terminalDone; err != nil {
		t.Fatal(err)
	}
	terminal, err := storeA.Get(ctx, unknownTarget)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.State != StateTerminal {
		t.Fatalf("target lifecycle after release = %+v", terminal)
	}
}

func TestStoreOnboardingEffectUsesExistingAuthFenceForExactDepartedGeneration(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:onboarding-profile-effect")
	authority, err := store.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := sha256.Sum256([]byte("onboarding profile body"))
	called := false
	err = store.WithExistingAuth(ctx, owner, func(authCtx context.Context, fenced Lifecycle) error {
		if fenced != authority {
			t.Fatalf("onboarding authority = %+v, want %+v", fenced, authority)
		}
		return store.WithOnboardingEffect(
			authCtx,
			owner,
			authority.Generation,
			func(effectCtx context.Context) error {
				called = true
				attempt, err := store.CreateEffectAttempt(effectCtx, NewEffectAttempt{
					OperationID: "onboarding-profile:v1",
					Owner:       owner, OwnerGeneration: authority.Generation,
					Kind: EffectPDSRecord, Action: EffectActionPutRecord,
					DeterministicKey:   "at://did:plc:onboarding-profile-effect/social.craftsky.actor.profile/self",
					RequestFingerprint: fingerprint,
					RemoteDeadline:     time.Now().Add(time.Minute),
				})
				if err != nil {
					return err
				}
				_, err = store.MarkAttemptDispatched(
					effectCtx, attempt.OperationID, owner, authority.Generation,
				)
				return err
			},
		)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("onboarding effect callback was not called")
	}
	if err := store.WithOnboardingEffect(
		ctx, owner, authority.Generation, func(context.Context) error { return nil },
	); !errors.Is(err, ErrFenceRequired) {
		t.Fatalf("unfenced onboarding effect = %v, want ErrFenceRequired", err)
	}
	err = store.WithExistingAuth(ctx, owner, func(authCtx context.Context, _ Lifecycle) error {
		return store.WithOnboardingEffect(
			authCtx,
			owner,
			authority.Generation+1,
			func(context.Context) error { return errors.New("stale onboarding effect ran") },
		)
	})
	if !errors.Is(err, ErrGenerationChanged) {
		t.Fatalf("stale onboarding generation = %v, want ErrGenerationChanged", err)
	}
}

func TestStoreOnboardingAuthFenceKeepsOneConnectionAndLinearizesEpochChange(t *testing.T) {
	storeA, storeB := ownerLifecycleTestStores(t)
	owner := syntax.DID("did:plc:auth-fence")
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- storeA.WithOnboardingAuth(context.Background(), owner, func(authCtx context.Context, lifecycle Lifecycle) error {
			if lifecycle.State != StateDeparted || lifecycle.Generation != 1 || lifecycle.AuthEpoch != 1 {
				return errors.New("unexpected onboarding authority")
			}
			if err := storeA.WithAuthTransaction(authCtx, func(tx pgx.Tx) error {
				var observedEpoch int64
				return tx.QueryRow(authCtx, `SELECT auth_epoch FROM owner_lifecycles WHERE owner_did=$1`, owner).Scan(&observedEpoch)
			}); err != nil {
				return err
			}
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	epochDone := make(chan error, 1)
	go func() {
		_, err := storeB.AdvanceAuthEpoch(context.Background(), owner, 1, "logoutAll")
		epochDone <- err
	}()
	select {
	case err := <-epochDone:
		close(release)
		t.Fatalf("epoch change crossed live auth fence: %v", err)
	case <-time.After(75 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("onboarding auth fence: %v", err)
	}
	if err := <-epochDone; err != nil {
		t.Fatalf("epoch change after auth fence: %v", err)
	}
}

func TestStoreActiveSessionAuthSerializesOneParentAcrossPools(t *testing.T) {
	storeA, storeB := ownerLifecycleTestStores(t)
	owner := syntax.DID("did:plc:parent-session-fence")
	row, err := storeA.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	row, err = storeA.Transition(context.Background(), TransitionRequest{
		Owner: owner, ExpectedGeneration: row.Generation, To: StateActive, Reason: "activate",
	})
	if err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- storeA.WithActiveSessionAuth(context.Background(), owner, "parent-a", func(authCtx context.Context, got Lifecycle) error {
			if got.Generation != row.Generation || got.AuthEpoch != row.AuthEpoch {
				return errors.New("unexpected active-session authority")
			}
			if err := storeA.WithAuthTransaction(authCtx, func(tx pgx.Tx) error {
				var state State
				return tx.QueryRow(authCtx, `SELECT state FROM owner_lifecycles WHERE owner_did=$1`, owner).Scan(&state)
			}); err != nil {
				return err
			}
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	secondEntered := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- storeB.WithActiveSessionAuth(context.Background(), owner, "parent-a", func(context.Context, Lifecycle) error {
			close(secondEntered)
			return nil
		})
	}()
	select {
	case <-secondEntered:
		close(release)
		t.Fatal("same parent crossed the live parent-session fence")
	case <-time.After(75 * time.Millisecond):
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first active-session scope: %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second active-session scope: %v", err)
	}
}

func TestStoreOnboardingAuthRejectsTerminalTombstone(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	owner := syntax.DID("did:plc:terminal-auth")
	_, err := store.pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,terminal_at,created_at,updated_at
		) VALUES($1,'terminal',2,2,'identityDeleted',now(),now(),now(),now())
	`, owner)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	err = store.WithOnboardingAuth(context.Background(), owner, func(context.Context, Lifecycle) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrTerminalOwner) || called {
		t.Fatalf("terminal onboarding auth err=%v called=%v", err, called)
	}
}

func TestStoreActiveEffectAndDeterministicAttemptIdentity(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:active-effect")
	row, err := store.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	row, err = store.Transition(ctx, TransitionRequest{
		Owner: owner, ExpectedGeneration: row.Generation, To: StateActive, Reason: "activate",
	})
	if err != nil {
		t.Fatal(err)
	}

	called := false
	err = store.WithActiveEffects(ctx, []ExpectedOwner{{Owner: owner, Generation: row.Generation}}, func(context.Context) error {
		called = true
		return nil
	})
	if err != nil || !called {
		t.Fatalf("active effect = called %t, error %v", called, err)
	}
	if err := store.WithActiveEffects(ctx, []ExpectedOwner{{Owner: owner, Generation: row.Generation + 1}}, func(context.Context) error {
		return errors.New("stale generation callback ran")
	}); !errors.Is(err, ErrGenerationChanged) {
		t.Fatalf("stale generation error = %v, want ErrGenerationChanged", err)
	}

	fingerprint := sha256.Sum256([]byte("same request"))
	request := NewEffectAttempt{
		OperationID:        "deterministic-operation",
		Owner:              owner,
		OwnerGeneration:    row.Generation,
		Kind:               EffectObjectPut,
		Action:             EffectActionPutObject,
		MutationKey:        "deterministic-mutation-v1",
		DeterministicKey:   "scheduled/did-plc-active-effect/2/media",
		RequestFingerprint: fingerprint,
		RemoteDeadline:     time.Now().Add(time.Minute),
	}
	err = store.WithActiveEffects(ctx, []ExpectedOwner{{Owner: owner, Generation: row.Generation}}, func(effectCtx context.Context) error {
		first, err := store.CreateEffectAttempt(effectCtx, request)
		if err != nil {
			return err
		}
		second, err := store.CreateEffectAttempt(effectCtx, request)
		if err != nil {
			return err
		}
		if first.OperationID != second.OperationID {
			return errors.New("idempotent operation IDs differ")
		}
		sameRemoteIdentity := request
		sameRemoteIdentity.OperationID = "alternate-operation-id"
		third, err := store.CreateEffectAttempt(effectCtx, sameRemoteIdentity)
		if err != nil {
			return err
		}
		if third.OperationID != first.OperationID {
			return errors.New("same remote identity did not return canonical attempt")
		}
		versionedPayload := sameRemoteIdentity
		versionedPayload.OperationID = "versioned-operation-id"
		versionedPayload.MutationKey = "deterministic-mutation-v2"
		versionedPayload.RequestFingerprint = sha256.Sum256([]byte("different request"))
		versioned, err := store.CreateEffectAttempt(effectCtx, versionedPayload)
		if err != nil {
			return err
		}
		if versioned.OperationID == first.OperationID {
			return errors.New("different request version reused the earlier attempt")
		}
		conflictingOperation := request
		conflictingOperation.RequestFingerprint = sha256.Sum256([]byte("conflicting operation body"))
		_, err = store.CreateEffectAttempt(effectCtx, conflictingOperation)
		if !errors.Is(err, ErrAttemptConflict) {
			return errors.New("same operation ID with a different version was not rejected")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("deterministic effect identity: %v", err)
	}
}

func TestStoreTerminalTombstoneLedgerAndPurgeCompletion(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	testNow := time.Now().UTC().Truncate(time.Microsecond)
	store.now = func() time.Time { return testNow }
	owner := syntax.DID("did:plc:terminal-owner")
	if _, err := store.EnsureOnboardingOwner(ctx, owner); err != nil {
		t.Fatal(err)
	}
	components := []PurgeComponent{
		{Component: "profiles", DIDRole: "owner"},
		{Component: "notifications", DIDRole: "actor"},
	}
	terminal, err := store.Terminalize(ctx, TerminalizeRequest{
		Owner: owner, Reason: "identityDeleted", Components: components,
	})
	if err != nil {
		t.Fatalf("terminalize owner: %v", err)
	}
	assertLifecycle(t, terminal, StateTerminal, 2, 2)
	if terminal.TerminalAt == nil {
		t.Fatal("terminal lifecycle has no terminal timestamp")
	}

	// Duplicate delivery is monotonic and can fill a newly declared fixed
	// catalogue entry without incrementing the tombstone generation again.
	terminalReplay, err := store.Terminalize(ctx, TerminalizeRequest{
		Owner:      owner,
		Reason:     "identityDeletedReplay",
		Components: append(components, PurgeComponent{Component: "sessions", DIDRole: "owner"}),
	})
	if err != nil {
		t.Fatalf("replay terminalize owner: %v", err)
	}
	assertLifecycle(t, terminalReplay, StateTerminal, 2, 2)
	if _, err := store.Transition(ctx, TransitionRequest{
		Owner: owner, ExpectedGeneration: terminal.Generation, To: StateActive, Reason: "replay",
	}); !errors.Is(err, ErrTerminalOwner) {
		t.Fatalf("terminal transition error = %v, want ErrTerminalOwner", err)
	}
	// Keep this lease/retry test focused on three real catalogue entries. The
	// complete rows represent components already drained by their workers; the
	// store must still refuse finalization until every remaining row completes.
	result, err := store.pool.Exec(ctx, `
		UPDATE owner_purge_components
		SET state='complete',completed_at=$3,updated_at=$3
		WHERE owner_did=$1 AND owner_generation=$2
		  AND (component,did_role) NOT IN (
			('craftsky_profiles','owner'),
			('craftsky_sessions','owner'),
			('notification_events','actor')
		  )
	`, owner, terminal.Generation, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result.RowsAffected(), int64(len(TerminalPurgeCatalogue())-3); got != want {
		t.Fatalf("precompleted components=%d, want %d", got, want)
	}

	leaseToken := uuid.New()
	claims, err := store.ClaimPurgeComponents(ctx, PurgeClaimRequest{
		Worker: "test-worker", LeaseToken: leaseToken, LeaseDuration: time.Minute, Limit: 2,
	})
	if err != nil {
		t.Fatalf("claim purge components: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("purge claims = %d, want 2", len(claims))
	}
	retryAt := testNow.Add(time.Hour)
	if err := store.ReschedulePurgeComponent(ctx, claims[0], leaseToken, retryAt, "dependencyTimeout"); err != nil {
		t.Fatalf("reschedule purge component: %v", err)
	}
	if err := store.CompletePurgeComponent(ctx, claims[1], leaseToken); err != nil {
		t.Fatalf("complete purge component %+v: %v", claims[1], err)
	}
	if _, err := store.FinalizeTerminalPurge(ctx, owner, terminal.Generation); !errors.Is(err, ErrPurgeIncomplete) {
		t.Fatalf("early purge finalization error = %v, want ErrPurgeIncomplete", err)
	}
	claims, err = store.ClaimPurgeComponents(ctx, PurgeClaimRequest{
		Worker: "test-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 2,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("remaining purge claim = %d, error %v", len(claims), err)
	}
	if err := store.CompletePurgeComponent(ctx, claims[0], claims[0].LeaseToken); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FinalizeTerminalPurge(ctx, owner, terminal.Generation); !errors.Is(err, ErrPurgeIncomplete) {
		t.Fatalf("purge with delayed retry finalization error = %v, want ErrPurgeIncomplete", err)
	}
	testNow = testNow.Add(2 * time.Hour)
	claims, err = store.ClaimPurgeComponents(ctx, PurgeClaimRequest{
		Worker: "test-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 2,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("rescheduled purge claim = %d, error %v", len(claims), err)
	}
	if claims[0].Attempts != 2 {
		t.Fatalf("rescheduled purge attempts = %d, want 2", claims[0].Attempts)
	}
	if err := store.CompletePurgeComponent(ctx, claims[0], claims[0].LeaseToken); err != nil {
		t.Fatal(err)
	}
	complete, err := store.FinalizeTerminalPurge(ctx, owner, terminal.Generation)
	if err != nil {
		t.Fatalf("finalize terminal purge: %v", err)
	}
	if complete.PurgeCompletedAt == nil {
		t.Fatal("purge completion timestamp is nil")
	}
}

func TestStoreTerminalizeAlwaysUsesFixedCatalogue(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:fixed-terminal-catalogue")

	terminal, err := store.Terminalize(ctx, TerminalizeRequest{
		Owner: owner, Reason: "identityDeleted",
	})
	if err != nil {
		t.Fatalf("terminalize with package catalogue: %v", err)
	}

	want := TerminalPurgeCatalogue()
	var count int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)::int
		FROM owner_purge_components
		WHERE owner_did=$1 AND owner_generation=$2
	`, owner, terminal.Generation).Scan(&count); err != nil {
		t.Fatalf("count fixed terminal catalogue: %v", err)
	}
	if count != len(want) {
		t.Fatalf("fixed terminal catalogue rows = %d, want %d", count, len(want))
	}

	// A caller cannot weaken or expand the security catalogue. Components is
	// retained temporarily for source compatibility while startup wiring moves
	// to the package-owned catalogue.
	if _, err := store.Terminalize(ctx, TerminalizeRequest{
		Owner: owner, Reason: "identityDeletedReplay",
		Components: []PurgeComponent{{Component: "caller_injected", DIDRole: "owner"}},
	}); err != nil {
		t.Fatalf("replay terminal event: %v", err)
	}
	var injected bool
	if err := store.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM owner_purge_components
			WHERE owner_did=$1 AND component='caller_injected'
		)
	`, owner).Scan(&injected); err != nil {
		t.Fatalf("inspect caller-injected component: %v", err)
	}
	if injected {
		t.Fatal("terminal caller expanded the package-owned purge catalogue")
	}
}

func TestStoreAndSQLTerminalPredicateAreAuthoritative(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	terminalOwner := syntax.DID("did:plc:terminal-predicate")
	activeOwner := syntax.DID("did:plc:active-predicate")

	active, err := store.EnsureOnboardingOwner(ctx, activeOwner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(ctx, TransitionRequest{
		Owner: activeOwner, ExpectedGeneration: active.Generation,
		To: StateActive, Reason: "profileCreated",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Terminalize(ctx, TerminalizeRequest{
		Owner: terminalOwner, Reason: "identityDeleted",
	}); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		owner syntax.DID
		want  bool
	}{
		{owner: terminalOwner, want: true},
		{owner: activeOwner, want: false},
		{owner: syntax.DID("did:plc:unknown-predicate"), want: false},
	} {
		got, err := store.IsTerminal(ctx, test.owner)
		if err != nil {
			t.Fatalf("IsTerminal(%s): %v", test.owner, err)
		}
		if got != test.want {
			t.Fatalf("IsTerminal(%s) = %t, want %t", test.owner, got, test.want)
		}
		var sqlGot bool
		if err := store.pool.QueryRow(ctx,
			`SELECT appview_owner_is_terminal($1)`, test.owner,
		).Scan(&sqlGot); err != nil {
			t.Fatalf("SQL terminal predicate(%s): %v", test.owner, err)
		}
		if sqlGot != test.want {
			t.Fatalf("SQL terminal predicate(%s) = %t, want %t", test.owner, sqlGot, test.want)
		}
	}
}

func TestStoreFencedDatabaseWorkUsesDedicatedConnectionAtPoolCapacity(t *testing.T) {
	base := testdb.WithSchema(t, "")
	applyOwnerLifecycleTestMigrations(t, base)

	newBoundedStore := func(maxConnections int32) (*Store, *pgxpool.Pool) {
		t.Helper()
		config := base.Config().Copy()
		config.MaxConns = maxConnections
		pool, err := pgxpool.NewWithConfig(context.Background(), config)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(pool.Close)
		fencer, err := NewFencer(pool, 250*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		store, err := NewStore(pool, fencer, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		return store, pool
	}

	single, _ := newBoundedStore(1)
	singleCtx, cancelSingle := context.WithTimeout(context.Background(), time.Second)
	defer cancelSingle()
	owner := syntax.DID("did:plc:single-connection")
	row, err := single.EnsureOnboardingOwner(singleCtx, owner)
	if err != nil {
		t.Fatalf("single-connection onboarding: %v", err)
	}
	if _, err := single.Transition(singleCtx, TransitionRequest{
		Owner: owner, ExpectedGeneration: row.Generation, To: StateActive, Reason: "activate",
	}); err != nil {
		t.Fatalf("single-connection transition: %v", err)
	}

	saturated, pool := newBoundedStore(2)
	owners := []syntax.DID{"did:plc:saturation-a", "did:plc:saturation-b"}
	generations := make(map[syntax.DID]int64, len(owners))
	for _, currentOwner := range owners {
		current, err := saturated.EnsureOnboardingOwner(context.Background(), currentOwner)
		if err != nil {
			t.Fatal(err)
		}
		current, err = saturated.Transition(context.Background(), TransitionRequest{
			Owner: currentOwner, ExpectedGeneration: current.Generation, To: StateActive, Reason: "activate",
		})
		if err != nil {
			t.Fatal(err)
		}
		generations[currentOwner] = current.Generation
	}
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	done := make(chan error, 2)
	for _, currentOwner := range owners {
		currentOwner := currentOwner
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			done <- saturated.WithActiveEffects(ctx, []ExpectedOwner{{
				Owner: currentOwner, Generation: generations[currentOwner],
			}}, func(context.Context) error {
				entered <- struct{}{}
				<-release
				return nil
			})
		}()
	}
	for range 2 {
		select {
		case <-entered:
		case <-time.After(750 * time.Millisecond):
			close(release)
			t.Fatalf("fenced store work starved with all %d pool connections holding owner fences", pool.Config().MaxConns)
		}
	}
	close(release)
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("saturated fenced effect: %v", err)
		}
	}
}

func TestStoreConcurrentTransitionsLinearizeAtOneGeneration(t *testing.T) {
	storeA, storeB := ownerLifecycleTestStores(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:transition-race")
	row, err := storeA.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	row, err = storeA.Transition(ctx, TransitionRequest{
		Owner: owner, ExpectedGeneration: row.Generation, To: StateActive, Reason: "activate",
	})
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for _, target := range []State{StateDeparted, StateDeletionPending} {
		target := target
		store := storeA
		if target == StateDeletionPending {
			store = storeB
		}
		go func() {
			<-start
			_, err := store.Transition(context.Background(), TransitionRequest{
				Owner: owner, ExpectedGeneration: row.Generation, To: target, Reason: "race",
			})
			results <- err
		}()
	}
	close(start)
	var success, stale int
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrGenerationChanged):
			stale++
		default:
			t.Fatalf("unexpected transition error: %v", err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("transition outcomes = success %d stale %d, want 1/1", success, stale)
	}
}

func TestStoreTransitionParticipantRollsBackLifecycleAndEffectClosure(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:participant-rollback")
	row, err := store.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	row, err = store.Transition(ctx, TransitionRequest{
		Owner: owner, ExpectedGeneration: row.Generation, To: StateActive, Reason: "activate",
	})
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := sha256.Sum256([]byte("rollback request"))
	var attempt EffectAttempt
	err = store.WithActiveEffects(ctx, []ExpectedOwner{{Owner: owner, Generation: row.Generation}}, func(effectCtx context.Context) error {
		attempt, err = store.CreateEffectAttempt(effectCtx, NewEffectAttempt{
			OperationID: "rollback-operation", Owner: owner, OwnerGeneration: row.Generation,
			Kind: EffectPDSRecord, DeterministicKey: "at://did:plc:participant-rollback/social.craftsky.feed.post/one",
			RequestFingerprint: fingerprint, RemoteDeadline: time.Now().Add(time.Minute),
		})
		if err != nil {
			return err
		}
		attempt, err = store.MarkAttemptDispatched(effectCtx, attempt.OperationID, owner, row.Generation)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("later participant failed")
	_, err = store.TransitionWith(ctx, TransitionRequest{
		Owner: owner, ExpectedGeneration: row.Generation, To: StateDeparted, Reason: "depart",
	}, func(_ context.Context, _ pgx.Tx, before, after Lifecycle) error {
		if before.State != StateActive || after.State != StateDeparted {
			t.Fatalf("participant lifecycle = %q -> %q", before.State, after.State)
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("transition participant error = %v, want sentinel", err)
	}
	current, err := store.Get(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycle(t, current, StateActive, row.Generation, row.AuthEpoch)
	unchanged, err := store.GetEffectAttempt(ctx, attempt.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Outcome != OutcomeDispatched || unchanged.ProjectionDisposition != ProjectionPending {
		t.Fatalf("rolled-back attempt = %+v", unchanged)
	}
}

func TestStoreNonTerminalOwnerTransactionDistinguishesMissingAndTerminal(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	known := syntax.DID("did:plc:known-nonterminal")
	missing := syntax.DID("did:plc:missing-owner")
	terminalOwner := syntax.DID("did:plc:known-terminal")
	if _, err := store.EnsureOnboardingOwner(ctx, known); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Terminalize(ctx, TerminalizeRequest{
		Owner: terminalOwner, Reason: "identityDeleted",
		Components: []PurgeComponent{{Component: "profiles", DIDRole: "owner"}},
	}); err != nil {
		t.Fatal(err)
	}

	called := false
	err := store.WithNonTerminalOwners(ctx, []syntax.DID{missing, known}, func(_ context.Context, _ pgx.Tx, existing map[syntax.DID]Lifecycle) error {
		called = true
		if len(existing) != 1 || existing[known].State != StateDeparted {
			t.Fatalf("existing lifecycle snapshot = %+v", existing)
		}
		if _, exists := existing[missing]; exists {
			t.Fatal("missing owner appeared in lifecycle snapshot")
		}
		return nil
	})
	if err != nil || !called {
		t.Fatalf("non-terminal transaction called=%t err=%v", called, err)
	}

	called = false
	err = store.WithNonTerminalOwners(ctx, []syntax.DID{known, terminalOwner}, func(context.Context, pgx.Tx, map[syntax.DID]Lifecycle) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrTerminalOwner) || called {
		t.Fatalf("terminal transaction called=%t err=%v", called, err)
	}
}

func ownerLifecycleTestStores(t *testing.T) (*Store, *Store) {
	t.Helper()
	poolA := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles(did TEXT PRIMARY KEY, created_at TIMESTAMPTZ NOT NULL DEFAULT now());
		CREATE TABLE oauth_sessions(
			account_did TEXT NOT NULL, session_id TEXT NOT NULL, data JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY(account_did,session_id)
		);
		CREATE TABLE oauth_auth_requests(
			state TEXT PRIMARY KEY, data JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			handoff_mode TEXT NOT NULL DEFAULT 'deep_link', loopback_redirect_uri TEXT,
			purpose TEXT NOT NULL DEFAULT 'login', device_id TEXT,
			account_deletion_owner_did TEXT, account_deletion_job_id UUID
		);
		CREATE TABLE craftsky_sessions(
			token_hash BYTEA PRIMARY KEY, account_did TEXT NOT NULL, oauth_session_id TEXT NOT NULL,
			device_label TEXT, last_device_id TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(), revoked_at TIMESTAMPTZ,
			FOREIGN KEY(account_did,oauth_session_id)
				REFERENCES oauth_sessions(account_did,session_id) ON DELETE CASCADE
		);
		CREATE TABLE account_deletion_operations(
			id UUID PRIMARY KEY, owner_did TEXT NOT NULL UNIQUE, state TEXT NOT NULL,
			reauth_oauth_session_id TEXT, deletion_oauth_session_id TEXT,
			FOREIGN KEY(owner_did,deletion_oauth_session_id)
				REFERENCES oauth_sessions(account_did,session_id),
			FOREIGN KEY(owner_did,reauth_oauth_session_id)
				REFERENCES oauth_sessions(account_did,session_id)
		);
	`)
	applyOwnerLifecycleTestMigrations(t, poolA)
	poolB, err := pgxpool.NewWithConfig(context.Background(), poolA.Config().Copy())
	if err != nil {
		t.Fatalf("second lifecycle pool: %v", err)
	}
	t.Cleanup(poolB.Close)

	newStore := func(pool *pgxpool.Pool) *Store {
		t.Helper()
		fencer, err := NewFencer(pool, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		store, err := NewStore(pool, fencer, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		return store
	}
	return newStore(poolA), newStore(poolB)
}

func applyOwnerLifecycleTestMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS craftsky_profiles(
			did TEXT PRIMARY KEY, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS oauth_sessions(
			account_did TEXT NOT NULL, session_id TEXT NOT NULL, data JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY(account_did,session_id)
		);
		CREATE TABLE IF NOT EXISTS oauth_auth_requests(
			state TEXT PRIMARY KEY, data JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			handoff_mode TEXT NOT NULL DEFAULT 'deep_link', loopback_redirect_uri TEXT,
			purpose TEXT NOT NULL DEFAULT 'login', device_id TEXT,
			account_deletion_owner_did TEXT, account_deletion_job_id UUID
		);
		CREATE TABLE IF NOT EXISTS craftsky_sessions(
			token_hash BYTEA PRIMARY KEY, account_did TEXT NOT NULL, oauth_session_id TEXT NOT NULL,
			device_label TEXT, last_device_id TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(), revoked_at TIMESTAMPTZ,
			FOREIGN KEY(account_did,oauth_session_id)
				REFERENCES oauth_sessions(account_did,session_id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS account_deletion_operations(
			id UUID PRIMARY KEY, owner_did TEXT NOT NULL UNIQUE, state TEXT NOT NULL,
			reauth_oauth_session_id TEXT, deletion_oauth_session_id TEXT,
			FOREIGN KEY(owner_did,deletion_oauth_session_id)
				REFERENCES oauth_sessions(account_did,session_id),
			FOREIGN KEY(owner_did,reauth_oauth_session_id)
				REFERENCES oauth_sessions(account_did,session_id)
		);
		ALTER TABLE account_deletion_operations
			ADD COLUMN IF NOT EXISTS reauth_oauth_session_id TEXT;
	`); err != nil {
		t.Fatalf("create lifecycle test base schema: %v", err)
	}
	for _, path := range []string{
		"../../migrations/000038_owner_auth_lifecycle.up.sql",
		"../../migrations/000039_owner_effects_terminal_purge.up.sql",
		"../../migrations/000049_pds_effect_action.up.sql",
	} {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read lifecycle test migration: %v", err)
		}
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply lifecycle test migration: %v", err)
		}
	}
}

func assertLifecycle(t *testing.T, row Lifecycle, state State, generation, authEpoch int64) {
	t.Helper()
	if row.State != state || row.Generation != generation || row.AuthEpoch != authEpoch {
		t.Fatalf("lifecycle = state %q generation %d auth epoch %d, want %q/%d/%d",
			row.State, row.Generation, row.AuthEpoch, state, generation, authEpoch)
	}
}
