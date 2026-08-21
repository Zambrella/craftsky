package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
)

// ErrCraftskySessionNotFound deliberately collapses unknown, revoked,
// expired, wrong-epoch, and otherwise intentionally unusable credentials.
// Infrastructure failures remain distinct so middleware can return 503.
var ErrCraftskySessionNotFound = errors.New("craftsky session not found")

// ErrRecoverySessionNotAuthorized is intentionally collapsed into the same
// invalid-credential result as an unknown session. Database failures remain
// distinct so HTTP middleware can return 503 rather than logging the user out.
var ErrRecoverySessionNotAuthorized = errors.New("craftsky recovery session is not authorized")

type CraftskySessionConfig struct {
	Inactivity            time.Duration
	ActivityWriteInterval time.Duration
	RecoveryAuthorization RecoverySessionAuthorization
}

// RecoverySessionAuthorization is the narrow product-policy check used when
// an otherwise valid child session belongs to an owner whose lifecycle is
// deletion_pending. It runs after the owner row is locked and before any
// OAuth parent or CraftSky child row is locked.
type RecoverySessionAuthorization interface {
	AuthorizeRecoverySession(
		context.Context,
		pgx.Tx,
		syntax.DID,
		ownerlifecycle.Lifecycle,
	) error
}

// CraftskySessionStore keeps the database as the sole authentication and
// activity authority. It intentionally has no token- or session-keyed maps.
type CraftskySessionStore struct {
	pool                  *pgxpool.Pool
	cfg                   CraftskySessionConfig
	recoveryAuthorization RecoverySessionAuthorization
}

func NewCraftskySessionStoreWithConfig(
	pool *pgxpool.Pool,
	cfg CraftskySessionConfig,
) (*CraftskySessionStore, error) {
	if pool == nil {
		return nil, errors.New("craftsky session store requires a database pool")
	}
	if cfg.Inactivity <= 0 {
		return nil, errors.New("craftsky session inactivity must be positive")
	}
	if cfg.ActivityWriteInterval <= 0 || cfg.ActivityWriteInterval >= cfg.Inactivity {
		return nil, errors.New("craftsky activity write interval must be positive and shorter than inactivity")
	}
	return &CraftskySessionStore{
		pool: pool, cfg: cfg,
		recoveryAuthorization: cfg.RecoveryAuthorization,
	}, nil
}

// NewCraftskySessionStore is retained as a source-compatible constructor for
// existing wiring. New startup wiring should use the validated constructor and
// pass both immutable inactivity and activity-write intervals explicitly.
func NewCraftskySessionStore(pool *pgxpool.Pool, activityWriteInterval time.Duration) *CraftskySessionStore {
	if activityWriteInterval <= 0 {
		activityWriteInterval = time.Nanosecond
	}
	store, err := NewCraftskySessionStoreWithConfig(pool, CraftskySessionConfig{
		Inactivity:            30 * 24 * time.Hour,
		ActivityWriteInterval: activityWriteInterval,
	})
	if err != nil {
		panic(err)
	}
	return store
}

// Create issues an already-active child only for an already-active parent.
// OAuth browser handoff must use createPendingTx instead.
func (s *CraftskySessionStore) Create(ctx context.Context, did, oauthSessionID, deviceLabel string) (string, error) {
	var token string
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		token, _, err = s.createTx(ctx, tx, syntax.DID(did), oauthSessionID, deviceLabel, "active")
		return err
	})
	return token, err
}

func (s *CraftskySessionStore) createPendingTx(
	ctx context.Context,
	tx pgx.Tx,
	did syntax.DID,
	oauthSessionID string,
	deviceID string,
) (string, []byte, error) {
	return s.createTx(ctx, tx, did, oauthSessionID, deviceID, "pending_confirmation")
}

func (s *CraftskySessionStore) createTx(
	ctx context.Context,
	tx pgx.Tx,
	did syntax.DID,
	oauthSessionID string,
	deviceLabel string,
	childState string,
) (string, []byte, error) {
	if did == "" || oauthSessionID == "" {
		return "", nil, ErrCraftskySessionNotFound
	}
	var parentState string
	var authEpoch int64
	var parentExpiry time.Time
	if err := tx.QueryRow(ctx, `
		SELECT lifecycle_state,auth_epoch,absolute_expires_at
		FROM oauth_sessions
		WHERE account_did=$1 AND session_id=$2
		FOR UPDATE
	`, did, oauthSessionID).Scan(&parentState, &authEpoch, &parentExpiry); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, ErrCraftskySessionNotFound
		}
		return "", nil, fmt.Errorf("lock OAuth parent for child issuance: %w", err)
	}
	allowedParent := parentState == "active"
	if childState == "pending_confirmation" {
		allowedParent = parentState == "pending_handoff"
	}
	if !allowedParent {
		return "", nil, ErrCraftskySessionNotFound
	}
	var now time.Time
	if err := tx.QueryRow(ctx, `SELECT now()`).Scan(&now); err != nil {
		return "", nil, fmt.Errorf("load session issuance time: %w", err)
	}
	if !now.Before(parentExpiry) {
		return "", nil, ErrCraftskySessionNotFound
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate craftsky bearer: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	idleExpiry := now.Add(s.cfg.Inactivity)
	if idleExpiry.After(parentExpiry) {
		idleExpiry = parentExpiry
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,device_label,last_device_id,
			lifecycle_state,auth_epoch,created_at,last_seen_at,idle_expires_at,last_device_seen_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8,$9,$8)
	`, hash[:], did, oauthSessionID, nullableString(deviceLabel), nullableString(deviceLabel),
		childState, authEpoch, now, idleExpiry)
	if err != nil {
		return "", nil, fmt.Errorf("insert craftsky session: %w", err)
	}
	return token, hash[:], nil
}

func (s *CraftskySessionStore) Lookup(ctx context.Context, token string) (AuthInfo, error) {
	return s.lookup(ctx, token, false)
}

// LookupRecovery accepts the same active credentials as Lookup, plus an
// active child owned by a departed onboarding/rejoin owner or by a
// deletion_pending owner with a current, unexpired deletion intent. It is
// deliberately exposed only through the recovery route access class.
func (s *CraftskySessionStore) LookupRecovery(ctx context.Context, token string) (AuthInfo, error) {
	return s.lookup(ctx, token, true)
}

func (s *CraftskySessionStore) lookup(ctx context.Context, token string, recovery bool) (AuthInfo, error) {
	if token == "" {
		return AuthInfo{}, ErrCraftskySessionNotFound
	}
	hash := sha256.Sum256([]byte(token))
	var info AuthInfo
	var outcomeErr error
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var didString, sessionID string
		if err := tx.QueryRow(ctx, `
			SELECT account_did,oauth_session_id
			FROM craftsky_sessions WHERE token_hash=$1
		`, hash[:]).Scan(&didString, &sessionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCraftskySessionNotFound
			}
			return fmt.Errorf("discover craftsky session: %w", err)
		}

		var ownerState ownerlifecycle.State
		var ownerGeneration, ownerEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT state,generation,auth_epoch FROM owner_lifecycles WHERE owner_did=$1 FOR UPDATE
		`, didString).Scan(&ownerState, &ownerGeneration, &ownerEpoch); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCraftskySessionNotFound
			}
			return fmt.Errorf("lock session owner authority: %w", err)
		}

		owner := syntax.DID(didString)
		ownerAuthorized := ownerState == ownerlifecycle.StateActive
		if recovery {
			switch ownerState {
			case ownerlifecycle.StateActive, ownerlifecycle.StateDeparted:
				ownerAuthorized = true
			case ownerlifecycle.StateDeletionPending:
				if s.recoveryAuthorization != nil {
					authority := ownerlifecycle.Lifecycle{
						Owner: owner, State: ownerState,
						Generation: ownerGeneration, AuthEpoch: ownerEpoch,
					}
					authorizeErr := s.recoveryAuthorization.AuthorizeRecoverySession(
						ctx, tx, owner, authority,
					)
					switch {
					case authorizeErr == nil:
						ownerAuthorized = true
					case errors.Is(authorizeErr, ErrRecoverySessionNotAuthorized):
						ownerAuthorized = false
					default:
						return fmt.Errorf("authorize recovery session: %w", authorizeErr)
					}
				}
			}
		}

		var parentState string
		var parentEpoch int64
		var parentExpiry time.Time
		if err := tx.QueryRow(ctx, `
			SELECT lifecycle_state,auth_epoch,absolute_expires_at
			FROM oauth_sessions
			WHERE account_did=$1 AND session_id=$2
			FOR UPDATE
		`, didString, sessionID).Scan(&parentState, &parentEpoch, &parentExpiry); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCraftskySessionNotFound
			}
			return fmt.Errorf("lock OAuth parent: %w", err)
		}

		var childState string
		var childEpoch int64
		var idleExpiry time.Time
		if err := tx.QueryRow(ctx, `
			SELECT lifecycle_state,auth_epoch,idle_expires_at
			FROM craftsky_sessions WHERE token_hash=$1 FOR UPDATE
		`, hash[:]).Scan(&childState, &childEpoch, &idleExpiry); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCraftskySessionNotFound
			}
			return fmt.Errorf("lock craftsky child: %w", err)
		}

		var now time.Time
		if err := tx.QueryRow(ctx, `SELECT now()`).Scan(&now); err != nil {
			return fmt.Errorf("load authentication time: %w", err)
		}
		if !ownerAuthorized || parentState != "active" || childState != "active" ||
			ownerEpoch != parentEpoch || ownerEpoch != childEpoch {
			outcomeErr = ErrCraftskySessionNotFound
			return nil
		}
		if !now.Before(parentExpiry) {
			if _, err := tx.Exec(ctx, `
				UPDATE oauth_sessions
				SET lifecycle_state='revocation_pending',revocation_requested_at=$3,
				    cleanup_next_attempt_at=$3,row_version=row_version+1,updated_at=$3
				WHERE account_did=$1 AND session_id=$2 AND lifecycle_state='active'
			`, didString, sessionID, now); err != nil {
				return fmt.Errorf("expire OAuth parent: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				UPDATE craftsky_sessions
				SET lifecycle_state='revoked',revoked_at=$3
				WHERE account_did=$1 AND oauth_session_id=$2 AND lifecycle_state<>'revoked'
			`, didString, sessionID, now); err != nil {
				return fmt.Errorf("revoke expired parent children: %w", err)
			}
			outcomeErr = ErrCraftskySessionNotFound
			return nil
		}
		if !now.Before(idleExpiry) {
			if _, err := tx.Exec(ctx, `
				UPDATE craftsky_sessions
				SET lifecycle_state='revoked',revoked_at=$2
				WHERE token_hash=$1 AND lifecycle_state='active'
			`, hash[:], now); err != nil {
				return fmt.Errorf("expire craftsky child: %w", err)
			}
			outcomeErr = ErrCraftskySessionNotFound
			return nil
		}

		if _, err := tx.Exec(ctx, `
			UPDATE craftsky_sessions
			SET last_seen_at=$2,
			    idle_expires_at=LEAST($3,(SELECT absolute_expires_at FROM oauth_sessions
			        WHERE account_did=$4 AND session_id=$5))
			WHERE token_hash=$1 AND lifecycle_state='active'
			  AND last_seen_at <= ($2::timestamptz)-make_interval(secs => $6::double precision)
		`, hash[:], now, now.Add(s.cfg.Inactivity), didString, sessionID, int64(s.cfg.ActivityWriteInterval/time.Second)); err != nil {
			return fmt.Errorf("touch craftsky session activity: %w", err)
		}
		info = AuthInfo{DID: syntax.DID(didString), SessionID: sessionID}
		return nil
	})
	if err != nil {
		return AuthInfo{}, err
	}
	if outcomeErr != nil {
		return AuthInfo{}, outcomeErr
	}
	return info, nil
}

func (s *CraftskySessionStore) Revoke(ctx context.Context, token string) error {
	hash := sha256.Sum256([]byte(token))
	_, err := s.pool.Exec(ctx, `
		UPDATE craftsky_sessions
		SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,now())
		WHERE token_hash=$1 AND lifecycle_state<>'revoked'
	`, hash[:])
	return err
}

func (s *CraftskySessionStore) RevokeAll(ctx context.Context, did string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE craftsky_sessions
		SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,now())
		WHERE account_did=$1 AND lifecycle_state<>'revoked'
	`, did)
	return err
}

func (s *CraftskySessionStore) RevokeOAuthSession(ctx context.Context, did, oauthSessionID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE craftsky_sessions
		SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,now())
		WHERE account_did=$1 AND oauth_session_id=$2 AND lifecycle_state<>'revoked'
	`, did, oauthSessionID)
	return err
}

// TouchDeviceID persists a changed device immediately and otherwise applies
// the same database-backed write interval across replicas and restarts.
func (s *CraftskySessionStore) TouchDeviceID(ctx context.Context, did, oauthSessionID, deviceID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE craftsky_sessions
		SET last_device_id=$1,last_device_seen_at=now()
		WHERE account_did=$2 AND oauth_session_id=$3 AND lifecycle_state='active'
		  AND (
			last_device_id IS DISTINCT FROM $1
			OR last_device_seen_at IS NULL
			OR last_device_seen_at <= now()-make_interval(secs => $4::double precision)
		  )
	`, deviceID, did, oauthSessionID, int64(s.cfg.ActivityWriteInterval/time.Second))
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
