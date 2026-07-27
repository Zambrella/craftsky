package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoUsableBackgroundSession = errors.New("no usable background OAuth session")

// BackgroundSessionSelector selects OAuth sessions for server-initiated PDS
// writes. It is deliberately owner-scoped and considers only OAuth sessions
// backed by at least one currently unrevoked Craftsky bearer session.
type BackgroundSessionSelector struct {
	pool *pgxpool.Pool
}

func NewBackgroundSessionSelector(pool *pgxpool.Pool) *BackgroundSessionSelector {
	return &BackgroundSessionSelector{pool: pool}
}

func (s *BackgroundSessionSelector) Select(ctx context.Context, owner syntax.DID) (string, error) {
	if s == nil || s.pool == nil || owner == "" {
		return "", ErrNoUsableBackgroundSession
	}

	var sessionID string
	err := s.pool.QueryRow(ctx, `
		SELECT oauth.session_id
		FROM oauth_sessions oauth
		JOIN LATERAL (
			SELECT max(craftsky.last_seen_at) AS last_seen_at
			FROM craftsky_sessions craftsky
			WHERE craftsky.account_did = oauth.account_did
			  AND craftsky.oauth_session_id = oauth.session_id
			  AND craftsky.revoked_at IS NULL
		) activity ON activity.last_seen_at IS NOT NULL
		WHERE oauth.account_did = $1
		ORDER BY activity.last_seen_at DESC,
		         oauth.updated_at DESC,
		         oauth.session_id ASC
		LIMIT 1
	`, owner).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNoUsableBackgroundSession
	}
	if err != nil {
		return "", fmt.Errorf("select background OAuth session: %w", err)
	}
	return sessionID, nil
}

// Invalidate removes only the exact rejected OAuth session. The composite
// primary key and FK cascade ensure no other account or session is affected.
func (s *BackgroundSessionSelector) Invalidate(ctx context.Context, owner syntax.DID, sessionID string) error {
	if s == nil || s.pool == nil || owner == "" || sessionID == "" {
		return nil
	}
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM oauth_sessions
		WHERE account_did = $1 AND session_id = $2
	`, owner, sessionID); err != nil {
		return fmt.Errorf("invalidate background OAuth session: %w", err)
	}
	return nil
}
