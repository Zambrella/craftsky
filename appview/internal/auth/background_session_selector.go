package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
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
	return s.selectSession(ctx, owner, nil)
}

// SelectForCapability skips eligible base-only sessions instead of borrowing
// another owner's authority or touching a child's inactivity deadline. The
// coordinator revalidates the chosen session under its normal fences.
func (s *BackgroundSessionSelector) SelectForCapability(ctx context.Context, owner syntax.DID, required []string) (string, error) {
	if len(required) == 0 {
		return "", ErrNoUsableBackgroundSession
	}
	return s.selectSession(ctx, owner, required)
}

func (s *BackgroundSessionSelector) selectSession(ctx context.Context, owner syntax.DID, required []string) (string, error) {
	if s == nil || s.pool == nil || owner == "" {
		return "", ErrNoUsableBackgroundSession
	}

	rows, err := s.pool.Query(ctx, `
		SELECT oauth.session_id, COALESCE(oauth.data->'scopes', '[]'::jsonb)
		FROM oauth_sessions oauth
		JOIN owner_lifecycles owner ON owner.owner_did=oauth.account_did
		JOIN LATERAL (
			SELECT max(craftsky.last_seen_at) AS last_seen_at
			FROM craftsky_sessions craftsky
			WHERE craftsky.account_did = oauth.account_did
			  AND craftsky.oauth_session_id = oauth.session_id
			  AND craftsky.lifecycle_state = 'active'
			  AND craftsky.auth_epoch = oauth.auth_epoch
			  AND craftsky.idle_expires_at > now()
		) activity ON activity.last_seen_at IS NOT NULL
		WHERE oauth.account_did = $1
		  AND oauth.lifecycle_state = 'active'
		  AND oauth.absolute_expires_at > now()
		  AND owner.state = 'active'
		  AND owner.generation = oauth.owner_generation
		  AND owner.auth_epoch = oauth.auth_epoch
		ORDER BY activity.last_seen_at DESC,
		         oauth.updated_at DESC,
		         oauth.session_id ASC
	`, owner)
	if err != nil {
		return "", fmt.Errorf("select background OAuth session: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sessionID string
		var rawScopes []byte
		if err := rows.Scan(&sessionID, &rawScopes); err != nil {
			return "", fmt.Errorf("read background OAuth candidate: %w", err)
		}
		if len(required) == 0 {
			return sessionID, nil
		}
		var scopes []string
		if err := json.Unmarshal(rawScopes, &scopes); err != nil {
			// An unusable stored grant cannot authorize work. Continue so that
			// another valid session belonging to this owner remains usable.
			continue
		}
		if validRegistrationTokenScopes(strings.Join(scopes, " "), required) {
			return sessionID, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate background OAuth candidates: %w", err)
	}
	return "", ErrNoUsableBackgroundSession
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
