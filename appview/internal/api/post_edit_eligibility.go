package api

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type postEditEligibilityStore interface {
	CheckPostEditAllowed(context.Context, syntax.DID, syntax.RecordKey) error
}

// CheckPostEditAllowed checks local policy independently of Tap post presence.
// Pending scans may be previewed only in the private owner editor. Known
// matches and moderation restrictions remain unavailable; public reads still
// require current-version scan clearance.
func (s *PostStore) CheckPostEditAllowed(ctx context.Context, owner syntax.DID, key syntax.RecordKey) error {
	uri := syntax.ATURI("at://" + owner.String() + "/" + craftskyPostNSID + "/" + key.String())
	var allowed bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (
 SELECT 1 FROM (SELECT $1::text AS did,$2::text AS uri) p
 JOIN craftsky_profiles member ON member.did=p.did
 JOIN owner_lifecycles lifecycle ON lifecycle.owner_did=p.did AND lifecycle.state='active'
 WHERE true `+postOwnerModerationPredicate+`
 AND NOT EXISTS (
 SELECT 1 FROM image_subject_requirements requirement
 JOIN image_scan_results scan ON scan.id=requirement.scan_result_id
 WHERE requirement.subject_uri=p.uri AND scan.state='match'
 ))`, owner, uri).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("post edit eligibility: %w", err)
	}
	if !allowed {
		return ErrPostNotFound
	}
	return nil
}
