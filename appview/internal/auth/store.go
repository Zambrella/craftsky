// Package auth contains OAuth storage (oauth.ClientAuthStore impl) and
// Craftsky bearer-token session management.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ErrOAuthSessionNotFound is returned by GetSession / GetAuthRequestInfo
// when the requested row doesn't exist. Callers that need to distinguish
// not-found from other errors use errors.Is.
var ErrOAuthSessionNotFound = errors.New("oauth session/auth-request not found")

// StoreConfig carries TTLs and the logger used for lazy-cleanup errors.
type StoreConfig struct {
	SessionExpiry     time.Duration
	SessionInactivity time.Duration
	AuthRequestExpiry time.Duration
	Logger            *slog.Logger
}

// PostgresAuthStore is a Postgres-backed implementation of
// oauth.ClientAuthStore. The ClientSessionData / AuthRequestData blobs
// are round-tripped as opaque JSONB; this code never inspects them
// beyond what indigo's serializer provides.
//
// Cleanup is lazy, inside the Get methods, matching the indigo cookbook
// example. No separate sweeper in v1.
type PostgresAuthStore struct {
	pool *pgxpool.Pool
	cfg  StoreConfig
}

var _ oauth.ClientAuthStore = (*PostgresAuthStore)(nil)

// NewPostgresAuthStore returns a PostgresAuthStore backed by pool.
// If cfg.Logger is nil, slog.Default() is used.
func NewPostgresAuthStore(pool *pgxpool.Pool, cfg StoreConfig) *PostgresAuthStore {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &PostgresAuthStore{pool: pool, cfg: cfg}
}

// SaveSession upserts an OAuth session row, updating data and updated_at
// on conflict.
func (s *PostgresAuthStore) SaveSession(ctx context.Context, sess oauth.ClientSessionData) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	const q = `
		INSERT INTO oauth_sessions (account_did, session_id, data, created_at, updated_at)
		VALUES ($1, $2, $3, now(), now())
		ON CONFLICT (account_did, session_id) DO UPDATE SET
			data = EXCLUDED.data,
			updated_at = now()
	`
	if _, err := s.pool.Exec(ctx, q, sess.AccountDID.String(), sess.SessionID, data); err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}
	return nil
}

// GetSession returns the session for (did, sessionID), or ErrOAuthSessionNotFound
// if no matching row exists. Lazily cleans up expired rows on each call.
func (s *PostgresAuthStore) GetSession(ctx context.Context, did syntax.DID, sessionID string) (*oauth.ClientSessionData, error) {
	s.cleanupSessions(ctx)
	var data []byte
	err := s.pool.QueryRow(ctx,
		`SELECT data FROM oauth_sessions WHERE account_did = $1 AND session_id = $2`,
		did.String(), sessionID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOAuthSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select session: %w", err)
	}
	var sess oauth.ClientSessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &sess, nil
}

// DeleteSession removes the session for (did, sessionID). It is not an error
// if the row does not exist.
func (s *PostgresAuthStore) DeleteSession(ctx context.Context, did syntax.DID, sessionID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM oauth_sessions WHERE account_did = $1 AND session_id = $2`,
		did.String(), sessionID)
	return err
}

// SaveAuthRequestInfo inserts a new auth request row. State is the primary key
// so duplicate states will produce a database error (create-only semantics per
// the indigo interface contract).
func (s *PostgresAuthStore) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal auth request: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO oauth_auth_requests (state, data) VALUES ($1, $2)`,
		info.State, data)
	if err != nil {
		return fmt.Errorf("insert auth request: %w", err)
	}
	return nil
}

// GetAuthRequestInfo returns the auth request for state, or ErrOAuthSessionNotFound
// if no matching row exists. Lazily cleans up expired rows on each call.
func (s *PostgresAuthStore) GetAuthRequestInfo(ctx context.Context, state string) (*oauth.AuthRequestData, error) {
	s.cleanupAuthRequests(ctx)
	var data []byte
	err := s.pool.QueryRow(ctx,
		`SELECT data FROM oauth_auth_requests WHERE state = $1`, state).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOAuthSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select auth request: %w", err)
	}
	var info oauth.AuthRequestData
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshal auth request: %w", err)
	}
	return &info, nil
}

// DeleteAuthRequestInfo removes the auth request for state. It is not an error
// if the row does not exist.
func (s *PostgresAuthStore) DeleteAuthRequestInfo(ctx context.Context, state string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM oauth_auth_requests WHERE state = $1`, state)
	return err
}

// cleanupSessions deletes rows older than SessionExpiry by created_at
// or untouched for SessionInactivity by updated_at. Best-effort; errors
// are logged at WARN and otherwise ignored so cleanup doesn't mask the
// caller's real query.
func (s *PostgresAuthStore) cleanupSessions(ctx context.Context) {
	expiry := time.Now().Add(-s.cfg.SessionExpiry)
	inactivity := time.Now().Add(-s.cfg.SessionInactivity)
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM oauth_sessions WHERE created_at < $1 OR updated_at < $2`,
		expiry, inactivity); err != nil {
		s.cfg.Logger.Warn("oauth_sessions cleanup failed", slog.String("err", err.Error()))
	}
}

func (s *PostgresAuthStore) cleanupAuthRequests(ctx context.Context) {
	cutoff := time.Now().Add(-s.cfg.AuthRequestExpiry)
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM oauth_auth_requests WHERE created_at < $1`, cutoff); err != nil {
		s.cfg.Logger.Warn("oauth_auth_requests cleanup failed", slog.String("err", err.Error()))
	}
}
