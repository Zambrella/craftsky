package api

import (
	"context"
	"strings"
)

// ReadAudience is explicit at the shared query boundary. Guest reads never
// acquire an account or a viewer overlay; existing member entry points retain
// their current policy unless the anonymous service selects GuestAudience.
type ReadAudience uint8

const (
	MemberAudience ReadAudience = iota
	GuestAudience
)

type readAudienceKey struct{}

func withReadAudience(ctx context.Context, audience ReadAudience) context.Context {
	return context.WithValue(ctx, readAudienceKey{}, audience)
}

func readAudience(ctx context.Context) ReadAudience {
	audience, _ := ctx.Value(readAudienceKey{}).(ReadAudience)
	return audience
}

func postReadPredicate(ctx context.Context) string {
	if readAudience(ctx) == GuestAudience {
		return publicPostPredicate
	}
	return postVisibleModerationPredicate
}

// Correlated eligibility runs inside the candidate query, before ordering and
// LIMIT. It also applies to records hydrated independently as nested subjects.
var guestPostOwnerPredicate = ` AND EXISTS (
 SELECT 1 FROM craftsky_profiles cp LEFT JOIN bluesky_profiles bp ON bp.did=cp.did
 WHERE cp.did=p.did ` + publicOwnerPredicate + `)`

func profileReadPredicate(ctx context.Context) string {
	if readAudience(ctx) == GuestAudience {
		return publicOwnerPredicate
	}
	return profileVisibleModerationPredicate
}

func guestAccountPredicate(didExpression string) string {
	return strings.Replace(guestPostOwnerPredicate, "WHERE cp.did=p.did ", "WHERE cp.did="+didExpression+" ", 1)
}
