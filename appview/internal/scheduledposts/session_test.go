package scheduledposts

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

type stubPublicationSessionSelector struct {
	wantOwner syntax.DID
	sessionID string
	err       error
}

func (selector stubPublicationSessionSelector) Select(_ context.Context, owner syntax.DID) (string, error) {
	if owner != selector.wantOwner {
		return "", errors.New("unexpected owner")
	}
	return selector.sessionID, selector.err
}

func TestSelectPublicationSession(t *testing.T) {
	t.Parallel()

	owner, err := syntax.ParseDID("did:plc:ewvi7nxzyoun6zhxrhs64oiz")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("another active owner device remains usable", func(t *testing.T) {
		sessionID, err := SelectPublicationSession(context.Background(), stubPublicationSessionSelector{
			wantOwner: owner,
			sessionID: "active-owner-session",
		}, owner)
		if err != nil || sessionID != "active-owner-session" {
			t.Fatalf("SelectPublicationSession() = %q, %v", sessionID, err)
		}
	})

	for _, name := range []string{"last session signed out", "session expired or revoked"} {
		t.Run(name, func(t *testing.T) {
			_, err := SelectPublicationSession(context.Background(), stubPublicationSessionSelector{
				wantOwner: owner,
				err:       auth.ErrNoUsableBackgroundSession,
			}, owner)
			if !errors.Is(err, ErrAuthUnavailable) {
				t.Fatalf("SelectPublicationSession() error = %v, want ErrAuthUnavailable", err)
			}
		})
	}
}
