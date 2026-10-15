package operatorconsole

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/ctxkeys"
)

const sessionLifetime = 8 * time.Hour
const sessionIdle = 30 * time.Minute

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
func digest(token string) []byte { b := sha256.Sum256([]byte(token)); return b[:] }

func (s *Store) issue(ctx context.Context, did syntax.DID) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var id uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM safety_operators WHERE did=$1 AND active AND deactivated_at IS NULL AND NOT EXISTS(SELECT 1 FROM owner_lifecycles WHERE owner_did=$1 AND state='terminal') FOR UPDATE`, did).Scan(&id); errors.Is(err, pgx.ErrNoRows) {
			return ErrDenied
		} else if err != nil {
			return err
		}
		now := s.now().UTC()
		_, err := tx.Exec(ctx, `INSERT INTO operator_browser_sessions(token_digest,operator_id,created_at,expires_at,last_used_at) VALUES($1,$2,$3,$4,$3)`, digest(token), id, now, now.Add(sessionLifetime))
		return err
	})
	return token, err
}
func (s *Store) authenticate(ctx context.Context, token string) (ctxkeys.Moderator, error) {
	var op Operator
	if len(token) != 43 {
		return ctxkeys.Moderator{}, ErrDenied
	}
	now := s.now().UTC()
	err := s.pool.QueryRow(ctx, `UPDATE operator_browser_sessions sess SET last_used_at=$2 FROM safety_operators op
 WHERE sess.token_digest=$1 AND sess.expires_at>$2 AND sess.last_used_at>$3
 AND op.id=sess.operator_id AND op.did IS NOT NULL AND op.active AND op.deactivated_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM owner_lifecycles WHERE owner_did=op.did AND state='terminal')
 RETURNING op.id,op.did,op.actor_id,op.role`, digest(token), now, now.Add(-sessionIdle)).Scan(&op.ID, &op.DID, &op.ActorID, &op.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ctxkeys.Moderator{}, ErrDenied
	}
	if err != nil {
		return ctxkeys.Moderator{}, err
	}
	op, err = completeOperator(op)
	if err != nil {
		return ctxkeys.Moderator{}, err
	}
	m := ctxkeys.Moderator{OperatorID: op.ID, DID: op.DID, ActorID: op.ActorID, Role: string(op.Role), PermissionSet: op.PermissionSet, Permissions: op.Permissions, SourceSystem: "craftsky-console", AssignedIncidentReferences: map[string]bool{}}
	rows, err := s.pool.Query(ctx, `SELECT incident_id FROM safety_operator_assignments WHERE operator_id=$1 AND released_at IS NULL`, op.ID)
	if err != nil {
		return m, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return m, err
		}
		m.AssignedIncidentReferences[id.String()] = true
	}
	return m, rows.Err()
}
func (s *Store) logout(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM operator_browser_sessions WHERE token_digest=$1`, digest(token))
	return err
}
func (s *Store) RevokeSessions(ctx context.Context, id uuid.UUID) error {
	return s.maintenance(ctx, func(tx pgx.Tx, now time.Time) error {
		var found bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM safety_operators WHERE id=$1 AND did IS NOT NULL)`, id).Scan(&found); err != nil {
			return err
		}
		if !found {
			return ErrDenied
		}
		if _, err := tx.Exec(ctx, `DELETE FROM operator_browser_sessions WHERE operator_id=$1`, id); err != nil {
			return err
		}
		return audit(ctx, tx, id, "sessionsRevoked", "", now)
	})
}

type attempt struct {
	Info oauth.AuthRequestData
	DID  syntax.DID
	PDS  string
}

func (s *Store) saveAttempt(ctx context.Context, a attempt, browser string) error {
	data, err := json.Marshal(a.Info)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('craftsky.operator-oauth-admission'))`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM operator_oauth_attempts WHERE expires_at<=$1`, s.now()); err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM operator_oauth_attempts`).Scan(&count); err != nil {
			return err
		}
		if count >= 1024 {
			return ErrDenied
		}
		_, err := tx.Exec(ctx, `INSERT INTO operator_oauth_attempts(state,browser_digest,did,pds_origin,data,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, a.Info.State, digest(browser), a.DID, a.PDS, data, s.now().Add(10*time.Minute))
		return err
	})
}
func (s *Store) consumeAttempt(ctx context.Context, state, browser string) (attempt, error) {
	var a attempt
	var data []byte
	err := s.pool.QueryRow(ctx, `DELETE FROM operator_oauth_attempts WHERE state=$1 AND browser_digest=$2 AND expires_at>$3 RETURNING did,pds_origin,data`, state, digest(browser), s.now()).Scan(&a.DID, &a.PDS, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrDenied
	}
	if err != nil {
		return a, err
	}
	err = json.Unmarshal(data, &a.Info)
	return a, err
}
