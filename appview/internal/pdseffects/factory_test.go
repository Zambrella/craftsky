package pdseffects

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestExecutorFactoryBuildsOwnerBoundNarrowExecutor(t *testing.T) {
	owner := syntax.DID("did:plc:factory-owner")
	client := &factoryEffectClient{effectPDS: newEffectPDS()}
	factory, err := NewExecutorFactory(
		&ownerlifecycle.Store{},
		func(_ context.Context, gotOwner syntax.DID, sessionID string) (auth.PDSClient, error) {
			if gotOwner != owner || sessionID != "oauth-parent-session" {
				return nil, errors.New("factory received the wrong OAuth binding")
			}
			return client, nil
		},
		10*time.Second,
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := factory(context.Background(), owner, "oauth-parent-session")
	if err != nil {
		t.Fatal(err)
	}
	var _ EffectExecutor = executor

	other := syntax.DID("did:plc:not-the-session-owner")
	_, err = executor.PutRecord(context.Background(), PutRecordRequest{
		OperationID:     "factory-owner-mismatch-attempt",
		MutationKey:     "factory-owner-mismatch-mutation",
		Owner:           other,
		OwnerGeneration: 1,
		ExpectedOwners:  []ownerlifecycle.ExpectedOwner{{Owner: other, Generation: 1}},
		Collection:      syntax.NSID("social.craftsky.actor.profile"),
		Rkey:            syntax.RecordKey("self"),
		Record:          map[string]any{"$type": "social.craftsky.actor.profile"},
	})
	if !errors.Is(err, ErrExecutorOwnerMismatch) {
		t.Fatalf("owner mismatch = %v, want ErrExecutorOwnerMismatch", err)
	}
	if client.effectCalls != 0 {
		t.Fatalf("owner mismatch entered OAuth boundary %d times", client.effectCalls)
	}

	_, err = executor.PutRecord(context.Background(), PutRecordRequest{
		OperationID:     "factory-missing-mutation-key",
		Owner:           owner,
		OwnerGeneration: 1,
		ExpectedOwners:  []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 1}},
		Collection:      syntax.NSID("social.craftsky.actor.profile"),
		Rkey:            syntax.RecordKey("self"),
		Record:          map[string]any{"$type": "social.craftsky.actor.profile"},
	})
	if err == nil {
		t.Fatal("missing mutation key was accepted")
	}
	if client.effectCalls != 0 {
		t.Fatalf("missing mutation key entered OAuth boundary %d times", client.effectCalls)
	}
}

func TestGuardedExecutorFactoryExposesOnlyCallbackScopedEffects(t *testing.T) {
	owner := syntax.DID("did:plc:guarded-factory-owner")
	client := &factoryEffectClient{effectPDS: newEffectPDS()}
	factory, err := NewGuardedExecutorFactory(
		&ownerlifecycle.Store{},
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return client, nil
		},
		10*time.Second,
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := factory(context.Background(), owner, "oauth-parent-session")
	if err != nil {
		t.Fatal(err)
	}
	var retained EffectExecutor
	err = coordinator.WithGuardedEffects(
		context.Background(),
		[]ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 2}},
		func(_ context.Context, scoped EffectExecutor) error {
			retained = scoped
			if _, exposesCoordinator := scoped.(GuardedEffectCoordinator); exposesCoordinator {
				t.Fatal("guarded callback exposed a recursively enterable coordinator")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if client.effectCalls != 1 || retained == nil {
		t.Fatalf("guarded boundary calls/executor = %d/%T", client.effectCalls, retained)
	}
	if _, err := retained.ResolveExpectedOwners(context.Background(), 2, nil); !errors.Is(
		err, ErrGuardedEffectScopeExpired,
	) {
		t.Fatalf("retained guarded executor error = %v, want expired scope", err)
	}
}

func TestExecutorFactoryTypeReturnsEffectExecutorInterface(t *testing.T) {
	var _ ExecutorFactory = func(
		context.Context, syntax.DID, string,
	) (EffectExecutor, error) {
		return nil, nil
	}
}

func TestCanonicalExpectedOwnersPreservesFenceOnlyMissingTarget(t *testing.T) {
	owner := syntax.DID("did:plc:expected-owner")
	missing := syntax.DID("did:plc:expected-missing-target")
	got, err := canonicalExpectedOwners([]ownerlifecycle.ExpectedOwner{
		{Owner: owner, Generation: 3},
		{Owner: missing, AllowMissing: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []ownerlifecycle.ExpectedOwner{
		{Owner: missing, AllowMissing: true},
		{Owner: owner, Generation: 3},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("canonical expected owners = %+v, want %+v", got, want)
	}
}

func TestExecutorFactoryRejectsAWrapperThatDropsEffectBoundary(t *testing.T) {
	factory, err := NewExecutorFactory(
		&ownerlifecycle.Store{},
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return newEffectPDS(), nil
		},
		10*time.Second,
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := factory(
		context.Background(), syntax.DID("did:plc:dropped-boundary"), "parent",
	); err == nil {
		t.Fatal("factory accepted a PDS wrapper without ActiveEffectPDSBoundary")
	}
}

type factoryEffectClient struct {
	*effectPDS
	effectCalls int
}

func (client *factoryEffectClient) WithActiveEffects(
	ctx context.Context,
	_ []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	client.effectCalls++
	return operation(ctx, client)
}
