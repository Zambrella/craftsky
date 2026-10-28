package postpublication

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// CurrentApplies retains original publication eligibility through edits while
// preventing deleted/unknown publications from dispatching catch-up work.
// Independent owner, moderation and block checks remain in the dispatcher.
func CurrentApplies(ctx context.Context, reader Reader, uri syntax.ATURI, _ syntax.DID) (bool, error) {
	original, err := Read(ctx, reader, uri)
	if err != nil {
		return false, err
	}
	// Pre-feature deliveries retain their existing policy when no publication
	// marker exists. An established marker does not certify the current source.
	if !original.Present {
		return true, nil
	}
	if !original.Known {
		return false, nil
	}
	var eligible bool
	err = reader.QueryRow(ctx, `SELECT EXISTS (
  SELECT 1 FROM tap_source_records WHERE uri=$1 AND action<>'delete'
   AND ordering_status='authoritative'
   AND structural_validation_status='valid' AND semantic_validation_status='valid'
 )`, uri).Scan(&eligible)
	return eligible, err
}
