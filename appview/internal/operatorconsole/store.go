// Package operatorconsole owns DID-allowlisted operator access, independently
// of consumer membership, onboarding, PDS writes and consumer OAuth sessions.
package operatorconsole

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/dbx"
	"social.craftsky/appview/internal/safetyincident"
)

var ErrDenied = errors.New("operator access denied")
var ErrInvalid = errors.New("invalid operator configuration")
var ErrLastAdmin = errors.New("another active administrator is required")

type Store struct {
	pool  dbx.DB
	now   func() time.Time
	actor string
}

// InTransaction keeps registry mutations and console replay receipts atomic.
func (s *Store) InTransaction(tx pgx.Tx, actor string) *Store {
	return &Store{pool: tx, now: s.now, actor: actor}
}

func NewStore(pool *pgxpool.Pool, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{pool: dbx.Normalize(pool), now: now}
}

type Operator struct {
	ID            uuid.UUID           `json:"operatorId"`
	DID           syntax.DID          `json:"did"`
	ActorID       string              `json:"actorId"`
	Role          safetyincident.Role `json:"role"`
	PermissionSet string              `json:"permissionSet"`
	Permissions   map[string]bool     `json:"permissions"`
}

func roleFor(set string) (safetyincident.Role, error) {
	switch set {
	case "admin":
		return safetyincident.RoleSafetyAdministrator, nil
	case "mod":
		return safetyincident.RoleModerator, nil
	case "helper":
		return safetyincident.RoleHelper, nil
	}
	return "", ErrInvalid
}
func completeOperator(op Operator) (Operator, error) {
	switch op.Role {
	case safetyincident.RoleSafetyAdministrator:
		op.PermissionSet = "admin"
	case safetyincident.RoleModerator:
		op.PermissionSet = "mod"
	case safetyincident.RoleHelper:
		op.PermissionSet = "helper"
	default:
		return Operator{}, ErrDenied
	}
	op.Permissions = map[string]bool{}
	for _, p := range safetyincident.OperatorPermissions() {
		op.Permissions[string(p)] = op.Role.AllowsPermission(p)
	}
	return op, nil
}
func (s *Store) Lookup(ctx context.Context, did syntax.DID) (Operator, error) {
	var op Operator
	err := s.pool.QueryRow(ctx, `SELECT id,did,actor_id,role FROM safety_operators WHERE did=$1 AND active AND deactivated_at IS NULL AND NOT EXISTS(SELECT 1 FROM owner_lifecycles WHERE owner_did=$1 AND state='terminal')`, did).Scan(&op.ID, &op.DID, &op.ActorID, &op.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return op, ErrDenied
	}
	if err != nil {
		return op, fmt.Errorf("read operator registry: %w", err)
	}
	return completeOperator(op)
}
func (s *Store) maintenance(ctx context.Context, run func(pgx.Tx, time.Time) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('craftsky.operator-management'))`); err != nil {
			return err
		}
		return run(tx, s.now().UTC())
	})
}
func (s *Store) audit(ctx context.Context, tx pgx.Tx, id uuid.UUID, action, set string, now time.Time) error {
	actor := s.actor
	if actor == "" {
		actor = "maintenance:operator-cli"
	}
	_, err := tx.Exec(ctx, `INSERT INTO safety_operator_events(id,operator_id,actor_id,action,permission,created_at) VALUES($1,$2,$3,$4,NULLIF($5,''),$6)`, uuid.New(), id, actor, action, set, now)
	return err
}
func (s *Store) Allow(ctx context.Context, did syntax.DID, set string) (Operator, error) {
	if _, err := syntax.ParseDID(did.String()); err != nil {
		return Operator{}, ErrInvalid
	}
	role, err := roleFor(set)
	if err != nil {
		return Operator{}, err
	}
	id := uuid.New()
	err = s.maintenance(ctx, func(tx pgx.Tx, now time.Time) error {
		_, err := tx.Exec(ctx, `INSERT INTO safety_operators(id,did,actor_id,role,active,created_at,updated_at) VALUES($1,$2,$3,$4,true,$5,$5)`, id, did, "operator:"+id.String(), role, now)
		if err != nil {
			return fmt.Errorf("allowlist operator: %w", err)
		}
		return s.audit(ctx, tx, id, "allowlisted", set, now)
	})
	if err != nil {
		return Operator{}, err
	}
	return s.Lookup(ctx, did)
}
func (s *Store) SetPermissionSet(ctx context.Context, id uuid.UUID, set string) error {
	role, err := roleFor(set)
	if err != nil {
		return err
	}
	return s.maintenance(ctx, func(tx pgx.Tx, now time.Time) error {
		if set != "admin" {
			if err := s.protectLastAdmin(ctx, tx, id); err != nil {
				return err
			}
		}
		tag, err := tx.Exec(ctx, `UPDATE safety_operators SET role=$2,updated_at=$3 WHERE id=$1 AND did IS NOT NULL AND active`, id, role, now)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrDenied
		}
		return s.audit(ctx, tx, id, "permissionSetChanged", set, now)
	})
}
func (s *Store) Disable(ctx context.Context, id uuid.UUID) error {
	return s.maintenance(ctx, func(tx pgx.Tx, now time.Time) error {
		if err := s.protectLastAdmin(ctx, tx, id); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE safety_operators SET active=false,deactivated_at=$2,updated_at=$2 WHERE id=$1 AND did IS NOT NULL AND active`, id, now)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrDenied
		}
		if _, err := tx.Exec(ctx, `DELETE FROM operator_browser_sessions WHERE operator_id=$1`, id); err != nil {
			return err
		}
		return s.audit(ctx, tx, id, "deactivated", "", now)
	})
}

// Browser administration cannot remove its last usable administrator. The
// privileged maintenance CLI remains the separately controlled recovery path.
func (s *Store) protectLastAdmin(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	if s.actor == "" {
		return nil
	}
	var last bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM safety_operators WHERE id=$1 AND active AND role='safetyAdministrator') AND NOT EXISTS(SELECT 1 FROM safety_operators o WHERE o.id<>$1 AND o.did IS NOT NULL AND o.active AND o.deactivated_at IS NULL AND o.role='safetyAdministrator' AND NOT EXISTS(SELECT 1 FROM owner_lifecycles WHERE owner_did=o.did AND state='terminal'))`, id).Scan(&last)
	if err != nil {
		return err
	}
	if last {
		return ErrLastAdmin
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]Operator, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,did,actor_id,role FROM safety_operators WHERE did IS NOT NULL AND active ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Operator{}
	for rows.Next() {
		var op Operator
		if err := rows.Scan(&op.ID, &op.DID, &op.ActorID, &op.Role); err != nil {
			return nil, err
		}
		op, err = completeOperator(op)
		if err != nil {
			return nil, err
		}
		out = append(out, op)
	}
	return out, rows.Err()
}

type Event struct {
	Sequence      int64     `json:"sequence"`
	ActorID       string    `json:"actorId"`
	Action        string    `json:"action"`
	PermissionSet *string   `json:"permissionSet,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (s *Store) Audit(ctx context.Context, id uuid.UUID) ([]Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT sequence,actor_id,action,permission,created_at FROM safety_operator_events WHERE operator_id=$1 ORDER BY sequence DESC LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.Sequence, &e.ActorID, &e.Action, &e.PermissionSet, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
