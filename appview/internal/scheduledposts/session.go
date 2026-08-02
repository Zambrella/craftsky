package scheduledposts

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

type PublicationSessionSelector interface {
	Select(context.Context, syntax.DID) (string, error)
}

func SelectPublicationSession(
	ctx context.Context,
	selector PublicationSessionSelector,
	owner syntax.DID,
) (string, error) {
	if selector == nil || owner == "" {
		return "", ErrAuthUnavailable
	}
	sessionID, err := selector.Select(ctx, owner)
	if errors.Is(err, auth.ErrNoUsableBackgroundSession) || (err == nil && sessionID == "") {
		return "", ErrAuthUnavailable
	}
	if err != nil {
		return "", fmt.Errorf("select scheduled publication session: %w", err)
	}
	return sessionID, nil
}
