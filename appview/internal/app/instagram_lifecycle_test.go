package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/tap"
)

type lifecycleActorDeletionFake struct {
	calls *[]string
	err   error
}

func (f lifecycleActorDeletionFake) HardDeleteByActor(context.Context, pgx.Tx, syntax.DID) error {
	*f.calls = append(*f.calls, "notifications")
	return f.err
}

type lifecycleInstagramInactivatorFake struct {
	calls *[]string
	now   time.Time
	err   error
}

func (f *lifecycleInstagramInactivatorFake) InactivateMembershipTx(_ context.Context, _ pgx.Tx, _ syntax.DID, now time.Time) error {
	*f.calls = append(*f.calls, "instagram")
	f.now = now
	return f.err
}

func TestProfileMembershipDeletionComposesCleanupInOrder(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.FixedZone("test", 3600))
	calls := make([]string, 0, 2)
	instagram := &lifecycleInstagramInactivatorFake{calls: &calls}
	lifecycle := &profileMembershipDeletion{
		notifications: lifecycleActorDeletionFake{calls: &calls},
		instagram:     instagram,
		now:           func() time.Time { return now },
	}
	if err := lifecycle.HardDeleteByActor(context.Background(), nil, "did:plc:member"); err != nil {
		t.Fatalf("HardDeleteByActor: %v", err)
	}
	if want := []string{"notifications", "instagram"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	if want := now.UTC(); !instagram.now.Equal(want) || instagram.now.Location() != time.UTC {
		t.Fatalf("inactivation time = %v, want UTC %v", instagram.now, want)
	}
}

func TestProfileMembershipDeletionStopsBeforeInactivationOnNotificationFailure(t *testing.T) {
	calls := make([]string, 0, 1)
	wantErr := errors.New("synthetic notification failure")
	lifecycle := &profileMembershipDeletion{
		notifications: lifecycleActorDeletionFake{calls: &calls, err: wantErr},
		instagram:     &lifecycleInstagramInactivatorFake{calls: &calls},
		now:           time.Now,
	}
	err := lifecycle.HardDeleteByActor(context.Background(), nil, "did:plc:member")
	if !errors.Is(err, wantErr) {
		t.Fatalf("HardDeleteByActor error = %v, want %v", err, wantErr)
	}
	if want := []string{"notifications"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

type lifecycleIdentityDeletionFake struct {
	name  string
	calls *[]string
	err   error
}

func (f lifecycleIdentityDeletionFake) HandleIdentityDeleted(context.Context, syntax.DID) error {
	*f.calls = append(*f.calls, f.name)
	return f.err
}

func TestTerminalIdentityDeletionRetriesThroughOrderedIdempotentHandlers(t *testing.T) {
	calls := make([]string, 0, 2)
	wantErr := errors.New("synthetic private purge failure")
	lifecycle := &terminalIdentityDeletion{handlers: []tap.IdentityDeletionHandler{
		lifecycleIdentityDeletionFake{name: "notifications", calls: &calls},
		lifecycleIdentityDeletionFake{name: "instagram", calls: &calls, err: wantErr},
	}}
	err := lifecycle.HandleIdentityDeleted(context.Background(), "did:plc:deleted")
	if !errors.Is(err, wantErr) {
		t.Fatalf("HandleIdentityDeleted error = %v, want %v", err, wantErr)
	}
	if want := []string{"notifications", "instagram"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}
