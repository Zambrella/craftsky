package postpublication

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/postfence"
)

// SeedAcceptedCreate records proven original facts in the creation command's
// acceptance transaction. Source ingestion may not have received the URI yet.
func SeedAcceptedCreate(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, raw json.RawMessage) error {
	facts, err := factsForRecord(raw, true)
	if err != nil || !facts.Known {
		return err
	}
	if err := postfence.LockTx(ctx, tx, uri); err != nil {
		return err
	}
	original, err := Read(ctx, tx, uri)
	if err != nil || original.Known || original.Deleted {
		return err
	}
	if err := Persist(ctx, tx, uri, uri.Authority().DID(), facts); err != nil {
		return err
	}
	// The unknown projection may already have completed. Wake only current,
	// valid serving content; the worker still applies its normal owner, source
	// version, membership, safety and relationship gates.
	_, err = tx.Exec(ctx, `INSERT INTO tap_projection_jobs(source_uri,projection_kind,source_event_id,state)
		SELECT source.uri,'post_notifications',source.source_event_id,'pending'
		FROM tap_source_records source JOIN craftsky_posts post ON post.uri=source.uri AND post.cid=source.cid
		WHERE source.uri=$1 AND source.action<>'delete' AND source.ordering_status='authoritative'
		AND source.structural_validation_status='valid' AND source.semantic_validation_status='valid'
		ON CONFLICT(source_uri,projection_kind) DO UPDATE SET state='pending',completed_at=NULL,next_attempt_at=$2,
		dependency_kind=NULL,dependency_key=NULL,updated_at=$2
		WHERE tap_projection_jobs.state='complete' AND tap_projection_jobs.source_event_id=EXCLUDED.source_event_id`, uri, time.Now().UTC())
	return err
}
