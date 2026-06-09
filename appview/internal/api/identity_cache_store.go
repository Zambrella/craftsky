package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const identityCacheFreshness = 24 * time.Hour

type IdentityCacheRow struct {
	DID        syntax.DID
	Handle     syntax.Handle
	ResolvedAt time.Time
}

type IdentityCacheStore struct {
	pool *pgxpool.Pool
}

type IdentityCacheService struct {
	store    *IdentityCacheStore
	resolver HandleResolver
	now      func() time.Time
}

func NewIdentityCacheStore(pool *pgxpool.Pool) *IdentityCacheStore {
	return &IdentityCacheStore{pool: pool}
}

func NewIdentityCacheService(pool *pgxpool.Pool, resolver HandleResolver, now func() time.Time) *IdentityCacheService {
	if now == nil {
		now = time.Now
	}
	return &IdentityCacheService{store: NewIdentityCacheStore(pool), resolver: resolver, now: now}
}

func (s *IdentityCacheService) UpsertCurrentHandle(ctx context.Context, did syntax.DID) error {
	if s == nil || s.resolver == nil || s.store == nil {
		return fmt.Errorf("identity cache service unavailable")
	}
	handle, err := s.resolver.ResolveHandle(ctx, did)
	if err != nil || handle.String() == "" {
		if err == nil {
			err = fmt.Errorf("empty handle")
		}
		return fmt.Errorf("resolve current handle %s: %w", did.String(), err)
	}
	return s.store.Upsert(ctx, did, handle, s.now().UTC())
}

func (s *IdentityCacheStore) FreshByHandle(ctx context.Context, handle syntax.Handle, now time.Time) (*IdentityCacheRow, error) {
	var row IdentityCacheRow
	err := s.pool.QueryRow(ctx, `
		SELECT ic.did, ic.handle, ic.resolved_at
		FROM atproto_identity_cache ic
		JOIN craftsky_profiles cp ON cp.did = ic.did
		WHERE ic.handle_lower = $1 AND ic.resolved_at >= $2
	`, strings.ToLower(handle.String()), now.Add(-identityCacheFreshness)).Scan(&row.DID, &row.Handle, &row.ResolvedAt)
	if err == nil {
		return &row, nil
	}
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return nil, fmt.Errorf("identity cache fresh by handle: %w", err)
}

func (s *IdentityCacheStore) Upsert(ctx context.Context, did syntax.DID, handle syntax.Handle, resolvedAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("identity cache begin upsert %s: %w", did.String(), err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	handleLower := strings.ToLower(handle.String())
	if _, err := tx.Exec(ctx, `
		DELETE FROM atproto_identity_cache
		WHERE handle_lower = $2 AND did <> $1
	`, did.String(), handleLower); err != nil {
		return fmt.Errorf("identity cache delete stale handle owner %s: %w", did.String(), err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO atproto_identity_cache (did, handle, handle_lower, resolved_at, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (did) DO UPDATE SET
			handle = EXCLUDED.handle,
			handle_lower = EXCLUDED.handle_lower,
			resolved_at = EXCLUDED.resolved_at,
			updated_at = now()
	`, did.String(), handle.String(), handleLower, resolvedAt)
	if err != nil {
		return fmt.Errorf("identity cache upsert %s: %w", did.String(), err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("identity cache commit upsert %s: %w", did.String(), err)
	}
	return nil
}

func (s *IdentityCacheStore) IsCraftskyProfile(ctx context.Context, did syntax.DID) (bool, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`, did.String()).Scan(&exists); err != nil {
		return false, fmt.Errorf("craftsky profile exists %s: %w", did.String(), err)
	}
	return exists, nil
}

func (s *IdentityCacheStore) BackfillCandidateDIDs(ctx context.Context, limit int, now time.Time) ([]syntax.DID, error) {
	if limit <= 0 {
		return []syntax.DID{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT cp.did
		FROM craftsky_profiles cp
		LEFT JOIN atproto_identity_cache ic ON ic.did = cp.did
		WHERE ic.did IS NULL OR ic.resolved_at < $1
		ORDER BY cp.did ASC
		LIMIT $2
	`, now.Add(-identityCacheFreshness), limit)
	if err != nil {
		return nil, fmt.Errorf("identity cache backfill candidates: %w", err)
	}
	defer rows.Close()
	out := make([]syntax.DID, 0, limit)
	for rows.Next() {
		var did syntax.DID
		if err := rows.Scan(&did); err != nil {
			return nil, fmt.Errorf("identity cache backfill candidate scan: %w", err)
		}
		out = append(out, did)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("identity cache backfill candidate rows: %w", err)
	}
	return out, nil
}
