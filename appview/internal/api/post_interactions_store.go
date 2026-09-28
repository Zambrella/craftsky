// appview/internal/api/post_interactions_store.go
package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func scanInteractionRow(scanner pgx.Row) (*InteractionRow, error) {
	out := &InteractionRow{}
	err := scanner.Scan(
		&out.URI, &out.DID, &out.Rkey, &out.CID,
		&out.SubjectURI, &out.SubjectCID, &out.CreatedAt, &out.IndexedAt,
	)
	return out, err
}

func (s *PostStore) findActiveInteraction(ctx context.Context, kind, did, subjectURI string) (*InteractionRow, error) {
	q := `
		SELECT aggregate.representative_source_uri,
		       aggregate.actor_did,
		       source_record.rkey,
		       source_record.cid,
		       aggregate.subject_uri,
		       source.subject_cid,
		       source.activity_at,
		       source_record.updated_at
		FROM pds_set_aggregates aggregate
		JOIN pds_set_sources source
		  ON source.source_uri=aggregate.representative_source_uri
		 AND source.kind=aggregate.kind
		 AND source.actor_did=aggregate.actor_did
		 AND source.scope_key=aggregate.scope_key
		JOIN tap_source_records source_record ON source_record.uri=aggregate.representative_source_uri
		WHERE aggregate.kind=$1
		  AND aggregate.actor_did=$2
		  AND aggregate.subject_uri=$3
		  AND NOT appview_owner_is_terminal(aggregate.actor_did)
	`
	row, err := scanInteractionRow(s.pool.QueryRow(ctx, q, kind, did, subjectURI))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInteractionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s find active %s/%s: %w", kind, did, subjectURI, err)
	}
	return row, nil
}

// FindActiveLike returns the active like by did for subjectURI.
func (s *PostStore) FindActiveLike(ctx context.Context, did, subjectURI string) (*InteractionRow, error) {
	return s.findActiveInteraction(ctx, "like", did, subjectURI)
}

// FindActiveRepost returns the active repost by did for subjectURI.
func (s *PostStore) FindActiveRepost(ctx context.Context, did, subjectURI string) (*InteractionRow, error) {
	return s.findActiveInteraction(ctx, "repost", did, subjectURI)
}
