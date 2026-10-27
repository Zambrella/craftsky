package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/posteditstate"
	"social.craftsky/appview/internal/postrecord"
	"social.craftsky/appview/internal/sourcevalidation"
	"social.craftsky/appview/internal/tap"
)

// preparePostEditStateTx runs while the old source is still present. Replacing
// the source first would erase evidence of an unprojected content transition.
func preparePostEditStateTx(ctx context.Context, tx pgx.Tx, event tap.Event, validation sourcevalidation.Result, orderingStatus, disposition string, now time.Time) (*postrecord.EditState, error) {
	if event.Collection != "social.craftsky.feed.post" || event.Action == "delete" || orderingStatus != "authoritative" || disposition == "denied_terminal" || validation.StructuralStatus != sourcevalidation.Valid || validation.SemanticStatus != sourcevalidation.Valid {
		return nil, nil
	}
	var previous postrecord.EditState
	var digest []byte
	var source *string
	err := tx.QueryRow(ctx, `SELECT source_cid,source_revision,content_digest,edited_at,edit_time_source FROM craftsky_post_edit_state WHERE uri=$1`, event.URI).Scan(&previous.CID, &previous.Revision, &digest, &previous.EditedAt, &source)
	var base *postrecord.EditState
	if err == nil {
		copy(previous.ContentDigest[:], digest)
		if source != nil {
			previous.TimeSource = postrecord.EditTimeSource(*source)
		}
		base = &previous
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("read current post edit state: %w", err)
	}
	if base == nil {
		var oldCID *string
		var oldRevision syntax.TID
		var oldRecord json.RawMessage
		var oldObserved time.Time
		var structural, semantic sourcevalidation.Status
		err := tx.QueryRow(ctx, `SELECT cid,revision,record,observed_at,structural_validation_status,semantic_validation_status FROM tap_source_records WHERE uri=$1 AND action<>'delete'`, event.URI).Scan(&oldCID, &oldRevision, &oldRecord, &oldObserved, &structural, &semantic)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("read prior source for edit transition: %w", err)
		}
		if err == nil && oldCID != nil && structural == sourcevalidation.Valid && semantic == sourcevalidation.Valid {
			previous, err = postrecord.ObserveEdit(nil, oldRecord, syntax.CID(*oldCID), oldRevision, oldObserved)
			if err != nil {
				return nil, err
			}
			base = &previous
		}
	}
	state, err := postrecord.ObserveEdit(base, event.Record, event.CID, event.Rev, now)
	if err == nil && state.TimeSource == postrecord.EditReported {
		confirmedAt, confirmErr := posteditstate.ConfirmedTime(ctx, tx, event.DID, event.URI, event.Rev, event.Record)
		if confirmErr != nil {
			return nil, confirmErr
		}
		if confirmedAt != nil {
			state.EditedAt = confirmedAt
			state.TimeSource = postrecord.EditConfirmed
		}
	}
	return &state, err
}

func persistPostEditStateTx(ctx context.Context, tx pgx.Tx, event tap.Event, state *postrecord.EditState, disposition string) error {
	if event.Collection != "social.craftsky.feed.post" {
		return nil
	}
	if event.Action == "delete" || disposition == "denied_terminal" {
		_, err := tx.Exec(ctx, `DELETE FROM craftsky_post_edit_state WHERE uri=$1 AND source_revision<=$2`, event.URI, event.Rev)
		return err
	}
	if state == nil {
		return nil
	}
	var source any
	if state.EditedAt != nil {
		source = state.TimeSource
	}
	_, err := tx.Exec(ctx, `INSERT INTO craftsky_post_edit_state(uri,source_cid,source_revision,content_digest,edited_at,edit_time_source)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(uri) DO UPDATE SET source_cid=EXCLUDED.source_cid,source_revision=EXCLUDED.source_revision,
		content_digest=EXCLUDED.content_digest,edited_at=EXCLUDED.edited_at,edit_time_source=EXCLUDED.edit_time_source
		WHERE craftsky_post_edit_state.source_revision<EXCLUDED.source_revision`, event.URI, state.CID, state.Revision, state.ContentDigest[:], state.EditedAt, source)
	if err != nil {
		return fmt.Errorf("persist current post edit state: %w", err)
	}
	return nil
}
