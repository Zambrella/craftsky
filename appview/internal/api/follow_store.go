// appview/internal/api/follow_store.go
package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FollowRow is an active indexed atproto follow relationship.
type FollowRow struct {
	URI        string
	DID        string
	Rkey       string
	CID        string
	SubjectDID string
	CreatedAt  time.Time
}

// FollowStore is the Postgres-backed read/write surface for active follows.
type FollowStore struct {
	pool *pgxpool.Pool
}

func NewFollowStore(pool *pgxpool.Pool) *FollowStore {
	return &FollowStore{pool: pool}
}

// FindActiveFollow returns the active row for a follower->subject pair.
func (s *FollowStore) FindActiveFollow(ctx context.Context, did string, subjectDID string) (*FollowRow, error) {
	out := &FollowRow{}
	err := s.pool.QueryRow(ctx, `
		SELECT aggregate.representative_source_uri,
		       aggregate.actor_did,
		       source_record.rkey,
		       source_record.cid,
		       aggregate.subject_did,
		       source.activity_at
		FROM pds_set_aggregates aggregate
		JOIN pds_set_sources source
		  ON source.source_uri = aggregate.representative_source_uri
		 AND source.kind = aggregate.kind
		 AND source.actor_did = aggregate.actor_did
		 AND source.scope_key = aggregate.scope_key
		JOIN tap_source_records source_record
		  ON source_record.uri = aggregate.representative_source_uri
		WHERE aggregate.kind = 'follow'
		  AND aggregate.actor_did = $1 AND aggregate.subject_did = $2
		  AND NOT appview_owner_is_terminal(aggregate.actor_did)
		  AND NOT appview_owner_is_terminal(aggregate.subject_did)
		LIMIT 1
	`, did, subjectDID).Scan(
		&out.URI,
		&out.DID,
		&out.Rkey,
		&out.CID,
		&out.SubjectDID,
		&out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find active follow %s->%s: %w", did, subjectDID, err)
	}
	return out, nil
}

// ListActiveFollowedDIDs returns active followed subject DIDs for a follower.
func (s *FollowStore) ListActiveFollowedDIDs(ctx context.Context, did string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT subject_did
		FROM pds_set_aggregates
		WHERE kind = 'follow'
		  AND actor_did = $1
		  AND NOT appview_owner_is_terminal(actor_did)
		  AND NOT appview_owner_is_terminal(subject_did)
		ORDER BY subject_did ASC
	`, did)
	if err != nil {
		return nil, fmt.Errorf("list active follows for %s: %w", did, err)
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, fmt.Errorf("scan active follow for %s: %w", did, err)
		}
		out = append(out, subject)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active follows for %s: %w", did, err)
	}
	return out, nil
}
