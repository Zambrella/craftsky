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

func (s *PostStore) findActiveInteraction(ctx context.Context, table, label, did, subjectURI string) (*InteractionRow, error) {
	q := `
		SELECT uri, did, rkey, cid, subject_uri, subject_cid, created_at, indexed_at
		FROM ` + table + `
		WHERE did = $1 AND subject_uri = $2 AND deleted_at IS NULL
		  AND NOT appview_owner_is_terminal(did)
	`
	row, err := scanInteractionRow(s.pool.QueryRow(ctx, q, did, subjectURI))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInteractionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s find active %s/%s: %w", label, did, subjectURI, err)
	}
	return row, nil
}

// FindActiveLike returns the active like by did for subjectURI.
func (s *PostStore) FindActiveLike(ctx context.Context, did, subjectURI string) (*InteractionRow, error) {
	return s.findActiveInteraction(ctx, "craftsky_likes", "like", did, subjectURI)
}

// FindActiveRepost returns the active repost by did for subjectURI.
func (s *PostStore) FindActiveRepost(ctx context.Context, did, subjectURI string) (*InteractionRow, error) {
	return s.findActiveInteraction(ctx, "craftsky_reposts", "repost", did, subjectURI)
}
