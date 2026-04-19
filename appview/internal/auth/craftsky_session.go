package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrCraftskySessionNotFound is returned by Lookup when no unrevoked
// row matches the presented token. Callers use errors.Is.
var ErrCraftskySessionNotFound = errors.New("craftsky session not found")

// CraftskySessionStore manages opaque bearer tokens issued to clients.
// Each token is generated as 32 random bytes (base64url-encoded) and the
// SHA-256 hash is stored in the craftsky_sessions table — the plaintext
// token is shown to the client exactly once at Create.
type CraftskySessionStore struct {
	pool             *pgxpool.Pool
	lastSeenThrottle time.Duration

	mu             sync.Mutex
	lastSeenMemory map[string]time.Time
}

func NewCraftskySessionStore(pool *pgxpool.Pool, lastSeenThrottle time.Duration) *CraftskySessionStore {
	return &CraftskySessionStore{
		pool:             pool,
		lastSeenThrottle: lastSeenThrottle,
		lastSeenMemory:   make(map[string]time.Time),
	}
}

// Create generates a fresh opaque bearer token, stores its SHA-256 hash
// in the DB, and returns the plaintext token. deviceLabel is optional;
// pass "" for none.
func (s *CraftskySessionStore) Create(ctx context.Context, did, oauthSessionID, deviceLabel string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	_, err := s.pool.Exec(ctx,
		`INSERT INTO craftsky_sessions (token_hash, account_did, oauth_session_id, device_label) VALUES ($1, $2, $3, $4)`,
		hash[:], did, oauthSessionID, nullableString(deviceLabel))
	if err != nil {
		return "", fmt.Errorf("insert craftsky session: %w", err)
	}
	return token, nil
}

func (s *CraftskySessionStore) Lookup(ctx context.Context, token string) (AuthInfo, error) {
	hash := sha256.Sum256([]byte(token))
	var did, sessID string
	err := s.pool.QueryRow(ctx,
		`SELECT account_did, oauth_session_id FROM craftsky_sessions WHERE token_hash = $1 AND revoked_at IS NULL`,
		hash[:]).Scan(&did, &sessID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthInfo{}, ErrCraftskySessionNotFound
	}
	if err != nil {
		return AuthInfo{}, fmt.Errorf("lookup craftsky session: %w", err)
	}
	s.maybeTouchLastSeen(ctx, hash[:])
	return AuthInfo{DID: did, SessionID: sessID}, nil
}

func (s *CraftskySessionStore) Revoke(ctx context.Context, token string) error {
	hash := sha256.Sum256([]byte(token))
	_, err := s.pool.Exec(ctx,
		`UPDATE craftsky_sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`,
		hash[:])
	return err
}

func (s *CraftskySessionStore) RevokeAll(ctx context.Context, did string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE craftsky_sessions SET revoked_at = now() WHERE account_did = $1 AND revoked_at IS NULL`,
		did)
	return err
}

// maybeTouchLastSeen updates last_seen_at at most once per
// lastSeenThrottle interval per token, keeping last-write times in an
// in-process map. The map only grows during normal operation; it is
// fine for v1 scale and is reset on process restart.
func (s *CraftskySessionStore) maybeTouchLastSeen(ctx context.Context, hash []byte) {
	key := fmt.Sprintf("%x", hash)
	s.mu.Lock()
	last, ok := s.lastSeenMemory[key]
	now := time.Now()
	if ok && now.Sub(last) < s.lastSeenThrottle {
		s.mu.Unlock()
		return
	}
	s.lastSeenMemory[key] = now
	s.mu.Unlock()
	_, _ = s.pool.Exec(ctx,
		`UPDATE craftsky_sessions SET last_seen_at = now() WHERE token_hash = $1`, hash)
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
