package ingestion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/tap"
)

const maxOperatorPageSize = 1000

// ProjectionBacklogItem is bounded operational state for one unfinished
// projection. Callers may print the DID/URI only to an authenticated operator;
// neither value is suitable for metric labels.
type ProjectionBacklogItem struct {
	SourceURI    syntax.ATURI
	Collection   syntax.NSID
	State        string
	Dependency   tap.Dependency
	Attempts     int
	NextAttempt  time.Time
	LastReason   tap.ReasonCode
	UpdatedAt    time.Time
	LeaseExpires *time.Time
}

func (store *Store) ListProjectionBacklog(ctx context.Context, limit int) ([]ProjectionBacklogItem, error) {
	if limit <= 0 || limit > maxOperatorPageSize {
		return nil, errors.New("projection backlog limit must be between 1 and 1000")
	}
	rows, err := store.pool.Query(ctx, `
		SELECT job.source_uri,source.collection,job.state,
		       job.dependency_kind,job.dependency_key,job.attempts,
		       job.next_attempt_at,job.last_reason_code,job.updated_at,
		       job.lease_expires_at
		FROM tap_projection_jobs job
		JOIN tap_source_records source ON source.uri=job.source_uri
		WHERE job.state IN ('pending','blocked','processing')
		ORDER BY job.next_attempt_at,job.id
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list Tap projection backlog: %w", err)
	}
	defer rows.Close()
	items := make([]ProjectionBacklogItem, 0, limit)
	for rows.Next() {
		var item ProjectionBacklogItem
		var dependencyKind, dependencyKey *string
		var reason *tap.ReasonCode
		if err := rows.Scan(
			&item.SourceURI,
			&item.Collection,
			&item.State,
			&dependencyKind,
			&dependencyKey,
			&item.Attempts,
			&item.NextAttempt,
			&reason,
			&item.UpdatedAt,
			&item.LeaseExpires,
		); err != nil {
			return nil, fmt.Errorf("scan Tap projection backlog: %w", err)
		}
		if dependencyKind != nil {
			item.Dependency.Kind = *dependencyKind
		}
		if dependencyKey != nil {
			item.Dependency.Key = *dependencyKey
		}
		if reason != nil {
			item.LastReason = *reason
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Tap projection backlog: %w", err)
	}
	return items, nil
}

func (store *Store) ListRepositoryBacklog(ctx context.Context, limit int) ([]RepositoryJob, error) {
	if limit <= 0 || limit > maxOperatorPageSize {
		return nil, errors.New("repository backlog limit must be between 1 and 1000")
	}
	rows, err := store.pool.Query(ctx, `
		SELECT id,did,job_kind,state,attempts,next_attempt_at,
		       lease_owner,lease_token,lease_expires_at,last_reason_code,
		       authoritative_revision,last_successful_at
		FROM tap_repository_jobs
		WHERE state IN ('pending','processing')
		ORDER BY next_attempt_at,id
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list Tap repository backlog: %w", err)
	}
	defer rows.Close()
	items := make([]RepositoryJob, 0, limit)
	for rows.Next() {
		item, err := scanRepositoryJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Tap repository backlog: %w", err)
	}
	return items, nil
}
