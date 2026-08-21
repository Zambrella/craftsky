// appview/internal/api/post.go
package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

const craftskyPostNSID = "social.craftsky.feed.post"

const craftskyLikeNSID = "social.craftsky.feed.like"

const craftskyRepostNSID = "social.craftsky.feed.repost"

var ErrInteractionBlocked = errors.New("interaction blocked")

var immediateRecordClock = syntax.NewTIDClock(0)

func newImmediateRecordKey() (syntax.RecordKey, error) {
	return syntax.ParseRecordKey(immediateRecordClock.Next().String())
}

type DirectedInteractionAuthorizer interface {
	AuthorizeDirectedInteraction(context.Context, syntax.DID, syntax.DID, relationships.Operation) error
}

type shareTargetReader interface {
	ResolveShareTarget(context.Context, string, string) (*ShareTargetRef, error)
}

type postTargetReader interface {
	ResolvePostTarget(context.Context, string, string) (*PostTargetRef, error)
}

type postByKeyReader interface {
	ReadOne(context.Context, string, string) (*PostRow, error)
}

type relationshipStateReader interface {
	RelationshipState(context.Context, syntax.DID, syntax.DID) (relationships.State, error)
}

type engagementSummaryReader interface {
	EngagementSummaries(context.Context, string, []string) (map[string]EngagementSummary, error)
}

func authorizeDirectedInteraction(
	w http.ResponseWriter,
	r *http.Request,
	authorizer DirectedInteractionAuthorizer,
	actor, subject syntax.DID,
	operation relationships.Operation,
) bool {
	err := authorizer.AuthorizeDirectedInteraction(r.Context(), actor, subject, operation)
	if err == nil {
		return true
	}
	runID := middleware.GetRunID(r.Context())
	if errors.Is(err, relationships.ErrProfileNotFound) {
		envelope.WriteError(w, http.StatusNotFound,
			"profile_not_found", "profile not found", runID, nil)
		return false
	}
	if errors.Is(err, ErrInteractionBlocked) {
		envelope.WriteError(w, http.StatusForbidden,
			"interaction_blocked", "interaction is not allowed across a block", runID, nil)
		return false
	}
	envelope.WriteError(w, http.StatusInternalServerError,
		"internal_error", "could not authorize interaction", runID, nil)
	return false
}

// parseLimit returns the validated limit, defaulting to 50 and capping
// at 100. Per pagination spec §5: caps are silent (we don't 400 on
// overshoot, we cap).
func parseLimit(raw string) int {
	const defaultLimit, maxLimit = 50, 100
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}
