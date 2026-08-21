package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestComposeTransitionParticipantsPreservesOrderAndStopsOnFailure(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("second participant failed")
	called := make([]string, 0, 3)
	participant := func(name string, result error) ownerlifecycle.TransitionParticipant {
		return func(
			context.Context,
			pgx.Tx,
			ownerlifecycle.Lifecycle,
			ownerlifecycle.Lifecycle,
		) error {
			called = append(called, name)
			return result
		}
	}
	composed := composeTransitionParticipants(
		participant("first", nil),
		participant("second", wantErr),
		participant("third", nil),
	)
	err := composed(
		context.Background(),
		nil,
		ownerlifecycle.Lifecycle{Owner: syntax.DID("did:plc:alice")},
		ownerlifecycle.Lifecycle{Owner: syntax.DID("did:plc:alice")},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("composed error = %v, want %v", err, wantErr)
	}
	if want := []string{"first", "second"}; !reflect.DeepEqual(called, want) {
		t.Fatalf("participant order = %v, want %v", called, want)
	}
}

func TestComposeTransitionParticipantsRejectsMissingDependency(t *testing.T) {
	t.Parallel()
	composed := composeTransitionParticipants(nil)
	err := composed(
		context.Background(),
		nil,
		ownerlifecycle.Lifecycle{},
		ownerlifecycle.Lifecycle{},
	)
	if err == nil {
		t.Fatal("missing lifecycle participant was accepted")
	}
}
