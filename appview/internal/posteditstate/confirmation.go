// Package posteditstate correlates current public records with bounded command
// evidence. It stores no previous post bodies or edit history.
package posteditstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

// ConfirmedTime requires the immutable dispatched body and a revision inside
// its evidence window. An old accepted command cannot confirm a later restore
// of identical content. Pending writes can be observed before acknowledgement.
func ConfirmedTime(ctx context.Context, tx pgx.Tx, owner syntax.DID, uri syntax.ATURI, revision syntax.TID, record json.RawMessage) (*time.Time, error) {
	var preparedAt time.Time
	err := tx.QueryRow(ctx, `SELECT c.created_at
		FROM pds_command_steps s
		JOIN pds_commands c ON c.id=s.command_id
		JOIN pds_command_dispatches d ON d.command_id=c.id AND d.plan_version=s.plan_version
		WHERE c.operation_kind='post.edit' AND c.owner_did=$1
		AND s.selected_uri=$2 AND s.action='update' AND s.ordinal=0
		AND s.plan_version=c.active_plan_version AND s.record::jsonb=$4::jsonb
		AND c.created_at=($4::jsonb->>'editedAt')::timestamptz
		AND d.repository_revision COLLATE "C" < $3 COLLATE "C"
		AND ((c.state IN ('dispatching','ambiguous') AND d.outcome IN ('dispatching','ambiguous'))
		 OR (c.state='accepted' AND d.outcome='accepted' AND
		  (d.result_commit_revision=$3 OR (d.result_commit_revision IS NULL AND d.observation_revision COLLATE "C">=$3 COLLATE "C"))))
		ORDER BY c.created_at DESC LIMIT 1`, owner, uri, revision, record).Scan(&preparedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("correlate current post edit confirmation: %w", err)
	}
	return &preparedAt, nil
}
