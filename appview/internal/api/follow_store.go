// appview/internal/api/follow_store.go
package api

import (
	"context"
	"encoding/json"
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
		SELECT uri, did, rkey, cid, subject_did, created_at
		FROM atproto_follows
		WHERE did = $1 AND subject_did = $2
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

// UpsertActive stores one active follow row for (did, subject_did).
// It collapses any existing alternate active row for the pair.
func (s *FollowStore) UpsertActive(ctx context.Context, row FollowRow, record json.RawMessage) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert follow %s: %w", row.URI, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM atproto_follows
		WHERE did = $1 AND subject_did = $2 AND uri <> $3
	`, row.DID, row.SubjectDID, row.URI); err != nil {
		return fmt.Errorf("collapse duplicate pair %s: %w", row.URI, err)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM atproto_follows
		WHERE did = $1 AND rkey = $2 AND uri <> $3
	`, row.DID, row.Rkey, row.URI); err != nil {
		return fmt.Errorf("collapse duplicate rkey %s: %w", row.URI, err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO atproto_follows
			(uri, did, rkey, cid, subject_did, record, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (uri) DO UPDATE SET
			did = EXCLUDED.did,
			rkey = EXCLUDED.rkey,
			cid = EXCLUDED.cid,
			subject_did = EXCLUDED.subject_did,
			record = EXCLUDED.record,
			created_at = EXCLUDED.created_at,
			indexed_at = now()
	`, row.URI, row.DID, row.Rkey, row.CID, row.SubjectDID, record, row.CreatedAt); err != nil {
		return fmt.Errorf("upsert follow %s: %w", row.URI, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert follow %s: %w", row.URI, err)
	}
	return nil
}

// DeleteActiveByURI removes the active follow row by AT-URI.
func (s *FollowStore) DeleteActiveByURI(ctx context.Context, uri string) error {
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM atproto_follows WHERE uri = $1`, uri); err != nil {
		return fmt.Errorf("delete follow %s: %w", uri, err)
	}
	return nil
}

// ListActiveFollowedDIDs returns active followed subject DIDs for a follower.
func (s *FollowStore) ListActiveFollowedDIDs(ctx context.Context, did string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT subject_did
		FROM atproto_follows
		WHERE did = $1
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
