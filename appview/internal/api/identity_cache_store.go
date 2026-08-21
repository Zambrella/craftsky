package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
)

const identityCacheFreshness = 24 * time.Hour

type IdentityCacheRow struct {
	DID        syntax.DID
	Handle     syntax.Handle
	ResolvedAt time.Time
}

type IdentityCacheStore struct {
	pool     *pgxpool.Pool
	observer IdentityCacheObserver
}

type IdentityCacheObserver interface {
	ObserveIdentityCache(result string, age time.Duration)
}

type IdentityCacheService struct {
	store    *IdentityCacheStore
	resolver HandleResolver
	now      func() time.Time
}

func NewIdentityCacheStore(pool *pgxpool.Pool, observers ...IdentityCacheObserver) *IdentityCacheStore {
	var observer IdentityCacheObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &IdentityCacheStore{pool: pool, observer: observer}
}

func NewIdentityCacheService(pool *pgxpool.Pool, resolver HandleResolver, now func() time.Time, observers ...IdentityCacheObserver) *IdentityCacheService {
	if now == nil {
		now = time.Now
	}
	return &IdentityCacheService{store: NewIdentityCacheStore(pool, observers...), resolver: resolver, now: now}
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
		  AND NOT appview_owner_is_terminal(ic.did)
	`, strings.ToLower(handle.String()), now.Add(-identityCacheFreshness)).Scan(&row.DID, &row.Handle, &row.ResolvedAt)
	if err == nil {
		if s.observer != nil {
			s.observer.ObserveIdentityCache("hit", now.Sub(row.ResolvedAt))
		}
		return &row, nil
	}
	if err == pgx.ErrNoRows {
		if s.observer != nil {
			s.observer.ObserveIdentityCache("miss", 0)
		}
		return nil, nil
	}
	return nil, fmt.Errorf("identity cache fresh by handle: %w", err)
}

func (s *IdentityCacheStore) Upsert(ctx context.Context, did syntax.DID, handle syntax.Handle, resolvedAt time.Time) error {
	if used, err := ownerlifecycle.WithPreheldNonTerminalOwnerTx(ctx, did, func(tx pgx.Tx) error {
		return s.upsertAuthorizedTx(ctx, tx, did, handle, resolvedAt)
	}); used {
		if err != nil {
			return fmt.Errorf("identity cache fenced upsert %s: %w", did.String(), err)
		}
		return nil
	}
	return s.upsert(ctx, did, handle, resolvedAt, func(tx pgx.Tx) error {
		return ownerlifecycle.GuardNonTerminalTargetsTx(ctx, tx, []syntax.DID{did})
	})
}

func (s *IdentityCacheStore) upsertForViewer(
	ctx context.Context,
	viewer syntax.DID,
	did syntax.DID,
	handle syntax.Handle,
	resolvedAt time.Time,
) error {
	return s.upsert(ctx, did, handle, resolvedAt, func(tx pgx.Tx) error {
		return ownerlifecycle.GuardPrivateMutationTx(ctx, tx, viewer, []syntax.DID{did})
	})
}

func (s *IdentityCacheStore) upsert(
	ctx context.Context,
	did syntax.DID,
	handle syntax.Handle,
	resolvedAt time.Time,
	authorize func(pgx.Tx) error,
) error {
	if authorize == nil {
		return fmt.Errorf("identity cache authorization missing")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("identity cache begin upsert %s: %w", did.String(), err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorize(tx); err != nil {
		return fmt.Errorf("identity cache authorize upsert: %w", err)
	}
	if err := s.upsertAuthorizedTx(ctx, tx, did, handle, resolvedAt); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("identity cache commit upsert %s: %w", did.String(), err)
	}
	return nil
}

func (s *IdentityCacheStore) upsertAuthorizedTx(
	ctx context.Context,
	tx pgx.Tx,
	did syntax.DID,
	handle syntax.Handle,
	resolvedAt time.Time,
) error {
	handleLower := strings.ToLower(handle.String())
	deleted, err := tx.Exec(ctx, `
		DELETE FROM atproto_identity_cache
		WHERE handle_lower = $2 AND did <> $1
	`, did.String(), handleLower)
	if err != nil {
		return fmt.Errorf("identity cache delete stale handle owner %s: %w", did.String(), err)
	}
	if deleted.RowsAffected() > 0 && s.observer != nil {
		s.observer.ObserveIdentityCache("reassigned", 0)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO atproto_identity_cache (did, handle, handle_lower, resolved_at, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (did) DO UPDATE SET
			handle = EXCLUDED.handle,
			handle_lower = EXCLUDED.handle_lower,
			resolved_at = EXCLUDED.resolved_at,
			updated_at = now()
	`, did.String(), handle.String(), handleLower, resolvedAt); err != nil {
		return fmt.Errorf("identity cache upsert %s: %w", did.String(), err)
	}
	return nil
}

func (s *IdentityCacheStore) IsCraftskyProfile(ctx context.Context, did syntax.DID) (bool, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM craftsky_profiles
			WHERE did = $1 AND NOT appview_owner_is_terminal(did)
		)
	`, did.String()).Scan(&exists); err != nil {
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
		WHERE (ic.did IS NULL OR ic.resolved_at < $1)
		  AND NOT appview_owner_is_terminal(cp.did)
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

type identityRefreshCandidate struct {
	DID        syntax.DID
	Handle     *syntax.Handle
	ResolvedAt *time.Time
	TapEventID *int64
	Version    int64
}

func (s *IdentityCacheStore) refreshCandidates(ctx context.Context, limit int, now time.Time) ([]identityRefreshCandidate, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("identity cache refresh candidates begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT cp.did,ic.handle,ic.resolved_at
		FROM craftsky_profiles cp
		LEFT JOIN atproto_identity_cache ic ON ic.did=cp.did
		LEFT JOIN atproto_identity_refresh_state rs ON rs.did=cp.did
		WHERE (rs.tap_event_id IS NOT NULL OR ic.did IS NULL OR ic.resolved_at < $1)
		  AND (rs.did IS NULL OR rs.next_attempt_at <= $2)
		  AND NOT appview_owner_is_terminal(cp.did)
		ORDER BY
		  COALESCE(rs.next_attempt_at, ic.resolved_at, '-infinity'::timestamptz) ASC,
		  cp.did ASC
		LIMIT $3
	`, now.Add(-identityCacheFreshness), now, limit)
	if err != nil {
		return nil, fmt.Errorf("identity cache refresh candidates: %w", err)
	}
	defer rows.Close()
	result := make([]identityRefreshCandidate, 0, limit)
	for rows.Next() {
		var candidate identityRefreshCandidate
		if err := rows.Scan(&candidate.DID, &candidate.Handle, &candidate.ResolvedAt); err != nil {
			return nil, fmt.Errorf("identity cache refresh candidate scan: %w", err)
		}
		result = append(result, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("identity cache refresh candidate rows: %w", err)
	}
	if len(result) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("identity cache refresh candidates commit: %w", err)
		}
		return result, nil
	}
	dids := make([]string, 0, len(result))
	indices := make(map[string]int, len(result))
	for index, candidate := range result {
		did := candidate.DID.String()
		dids = append(dids, did)
		indices[did] = index
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO atproto_identity_refresh_state(
			did,next_attempt_at,attempt_count,last_result,updated_at
		)
		SELECT did,$2,0,'pending',$2 FROM unnest($1::text[]) AS did
		ON CONFLICT(did) DO NOTHING
	`, dids, now); err != nil {
		return nil, fmt.Errorf("identity cache ensure refresh candidates: %w", err)
	}
	stateRows, err := tx.Query(ctx, `
		SELECT did,tap_event_id,refresh_version
		FROM atproto_identity_refresh_state
		WHERE did=ANY($1::text[])
	`, dids)
	if err != nil {
		return nil, fmt.Errorf("identity cache refresh candidate states: %w", err)
	}
	defer stateRows.Close()
	for stateRows.Next() {
		var did string
		var tapEventID *int64
		var version int64
		if err := stateRows.Scan(&did, &tapEventID, &version); err != nil {
			return nil, fmt.Errorf("identity cache refresh candidate state scan: %w", err)
		}
		index, exists := indices[did]
		if !exists {
			return nil, fmt.Errorf("identity cache refresh state returned unexpected DID %s", did)
		}
		result[index].TapEventID = tapEventID
		result[index].Version = version
		delete(indices, did)
	}
	if err := stateRows.Err(); err != nil {
		return nil, fmt.Errorf("identity cache refresh candidate state rows: %w", err)
	}
	if len(indices) != 0 {
		return nil, fmt.Errorf("identity cache refresh candidate state missing for %d DIDs", len(indices))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("identity cache refresh candidates commit: %w", err)
	}
	return result, nil
}

func (s *IdentityCacheStore) deferRefresh(
	ctx context.Context,
	candidate identityRefreshCandidate,
	nextAttempt time.Time,
	now time.Time,
) (bool, error) {
	result, err := s.pool.Exec(ctx, `
		UPDATE atproto_identity_refresh_state
		SET next_attempt_at=$2,
			attempt_count=attempt_count+1,
			last_result='retry',updated_at=$3
		WHERE did=$1 AND refresh_version=$4
	`, candidate.DID, nextAttempt, now, candidate.Version)
	if err != nil {
		return false, fmt.Errorf("identity cache defer refresh %s: %w", candidate.DID, err)
	}
	return result.RowsAffected() == 1, nil
}

func (s *IdentityCacheStore) completeRefresh(
	ctx context.Context,
	candidate identityRefreshCandidate,
	handle syntax.Handle,
	resolvedAt time.Time,
) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("identity cache complete refresh begin %s: %w", candidate.DID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ownerlifecycle.GuardNonTerminalTargetsTx(ctx, tx, []syntax.DID{candidate.DID}); err != nil {
		return false, fmt.Errorf("identity cache complete refresh authorize %s: %w", candidate.DID, err)
	}
	var currentVersion int64
	err = tx.QueryRow(ctx, `
		SELECT refresh_version
		FROM atproto_identity_refresh_state
		WHERE did=$1
		FOR UPDATE
	`, candidate.DID).Scan(&currentVersion)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("identity cache complete refresh state %s: %w", candidate.DID, err)
	}
	if currentVersion != candidate.Version {
		return false, nil
	}
	if err := s.upsertAuthorizedTx(ctx, tx, candidate.DID, handle, resolvedAt); err != nil {
		return false, err
	}
	result, err := tx.Exec(ctx, `
		DELETE FROM atproto_identity_refresh_state
		WHERE did=$1 AND refresh_version=$2
	`, candidate.DID, candidate.Version)
	if err != nil {
		return false, fmt.Errorf("identity cache complete refresh clear %s: %w", candidate.DID, err)
	}
	if result.RowsAffected() != 1 {
		return false, fmt.Errorf("identity cache complete refresh state changed for %s", candidate.DID)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("identity cache complete refresh commit %s: %w", candidate.DID, err)
	}
	return true, nil
}
