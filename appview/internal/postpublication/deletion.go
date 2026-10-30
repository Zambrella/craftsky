package postpublication

import (
	"context"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/notifications"
)

// Delete withdraws publication work on deletion, never on an ordinary edit.
func Delete(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, now time.Time) error {
	original, err := Read(ctx, tx, uri)
	if err != nil {
		return err
	}
	// Only established original publications could have queued notification work.
	// Unknown/tombstone-only sources need no notification storage dependency.
	if original.Known {
		if err := notifications.RestrictStoredPublication(ctx, tx, uri, []string{}, now); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM post_publication_mentions WHERE post_uri=$1`, uri); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO post_publication_state(uri,owner_did,origin_status)
 VALUES($1,$2,'deleted') ON CONFLICT(uri) DO UPDATE SET origin_status='deleted',original_creation_time=NULL,
 parent_uri=NULL,parent_cid=NULL,root_uri=NULL,root_cid=NULL,quoted_uri=NULL,quoted_cid=NULL`, uri, uri.Authority().DID())
	return err
}
