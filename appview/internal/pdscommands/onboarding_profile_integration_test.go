package pdscommands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

var _ PDSProtocol = (*auth.IndigoPDSClient)(nil)

func TestOnboardingProfileCommandUsesPendingAuthorityAndCommandJournal(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Lifecycles: lifecycles})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewOnboardingProfileService(store, lifecycles)
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:onboarding-command")
	authority, err := lifecycles.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	pds := newSetCommandPDS(owner)
	record := map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []string{}}
	write := func() (syntax.CID, error) {
		var cid syntax.CID
		err := lifecycles.WithOnboardingAuth(ctx, owner, func(authCtx context.Context, _ ownerlifecycle.Lifecycle) error {
			var putErr error
			cid, putErr = service.PutProfile(authCtx, pds, owner, authority.Generation, record)
			return putErr
		})
		return cid, err
	}
	if _, err := service.PutProfile(ctx, pds, owner, authority.Generation, record); !errors.Is(err, ownerlifecycle.ErrFenceRequired) {
		t.Fatalf("unfenced onboarding write = %v, want fence required", err)
	}
	pds.loseNextResponse = true
	if _, err := write(); !errors.Is(err, ErrOnboardingProfileUnresolved) {
		t.Fatalf("lost response = %v, want unresolved", err)
	}
	if pds.applyCalls != 1 {
		t.Fatalf("PDS writes after lost response = %d, want one", pds.applyCalls)
	}
	cid, err := write()
	if err != nil || cid == "" || pds.applyCalls != 1 {
		t.Fatalf("reconciled write CID=%q error=%v writes=%d", cid, err, pds.applyCalls)
	}
	if replayCID, err := write(); err != nil || replayCID != cid || pds.applyCalls != 1 {
		t.Fatalf("accepted replay CID=%q error=%v writes=%d", replayCID, err, pds.applyCalls)
	}
	var commands, dispatches, legacy int
	var state string
	if err := pool.QueryRow(ctx, `
		SELECT count(*),min(state) FROM pds_commands WHERE owner_did=$1 AND operation_kind='oauth_onboarding_profile'
	`, owner).Scan(&commands, &state); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_command_dispatches`).Scan(&dispatches); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM owner_effect_attempts WHERE owner_did=$1`, owner).Scan(&legacy); err != nil {
		t.Fatal(err)
	}
	if commands != 1 || state != "accepted" || dispatches != 1 || legacy != 0 {
		t.Fatalf("journal commands=%d state=%s dispatches=%d legacy=%d", commands, state, dispatches, legacy)
	}
	delete(pds.records, syntax.ATURI("at://"+owner.String()+"/social.craftsky.actor.profile/self"))
	if _, err := write(); !errors.Is(err, ErrOnboardingProfileChanged) || pds.applyCalls != 1 {
		t.Fatalf("profile removed after acceptance: error=%v writes=%d", err, pds.applyCalls)
	}
	if _, err := lifecycles.Terminalize(ctx, ownerlifecycle.TerminalizeRequest{
		Owner: owner, Reason: "identityDeleted",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := write(); !errors.Is(err, ownerlifecycle.ErrTerminalOwner) || pds.applyCalls != 1 {
		t.Fatalf("terminal owner write = %v, PDS writes=%d", err, pds.applyCalls)
	}
}

func TestOnboardingProfileCommandNeverRepeatsUncertainAbsentWrite(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Lifecycles: lifecycles})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewOnboardingProfileService(store, lifecycles)
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:onboarding-uncertain")
	authority, err := lifecycles.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	pds := newSetCommandPDS(owner)
	pds.applyErr = errors.New("unknown remote outcome")
	record := map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []string{}}
	write := func() error {
		return lifecycles.WithOnboardingAuth(ctx, owner, func(authCtx context.Context, _ ownerlifecycle.Lifecycle) error {
			_, err := service.PutProfile(authCtx, pds, owner, authority.Generation, record)
			return err
		})
	}
	if err := write(); !errors.Is(err, ErrOnboardingProfileUnresolved) {
		t.Fatalf("first uncertain write = %v", err)
	}
	pds.applyErr = nil
	if err := write(); !errors.Is(err, ErrOnboardingProfileUnresolved) || pds.applyCalls != 1 {
		t.Fatalf("retry after uncertain absence = %v, PDS writes=%d", err, pds.applyCalls)
	}
}

type profileCommandWriter struct{ service *OnboardingProfileService }

func (writer profileCommandWriter) PutOnboardingProfile(
	ctx context.Context, client auth.PDSClient, request auth.OnboardingProfileWrite,
) (syntax.CID, error) {
	return writer.service.PutProfile(ctx, client, request.Owner, request.OwnerGeneration, request.Record)
}

func TestMissingBlueskyAndCraftskyProfilesInitializeThroughCommandJournal(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Lifecycles: lifecycles})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewOnboardingProfileService(store, lifecycles)
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:onboarding-no-bluesky-profile")
	authority, err := lifecycles.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	pds := newSetCommandPDS(owner)
	attempt := auth.CallbackAttempt{
		State: "pending-onboarding", AttemptID: uuid.New(), Owner: owner,
		OwnerGeneration: authority.Generation, AuthEpoch: authority.AuthEpoch,
		Purpose: auth.RegistrationOAuthPurpose,
	}
	err = lifecycles.WithOnboardingAuth(ctx, owner, func(authCtx context.Context, _ ownerlifecycle.Lifecycle) error {
		return auth.InitializeProfileAndIdentityCache(
			authCtx, pds, attempt, profileCommandWriter{service}, nil, nil, nil, nil,
		)
	})
	if err != nil || pds.applyCalls != 1 {
		t.Fatalf("initialize absent profiles = %v, PDS writes=%d", err, pds.applyCalls)
	}
	var commands, legacy int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_commands WHERE owner_did=$1`, owner).Scan(&commands); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM owner_effect_attempts WHERE owner_did=$1`, owner).Scan(&legacy); err != nil {
		t.Fatal(err)
	}
	if commands != 1 || legacy != 0 {
		t.Fatalf("commands=%d legacy effects=%d, want 1/0", commands, legacy)
	}
}
