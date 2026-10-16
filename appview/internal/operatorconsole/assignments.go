package operatorconsole

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

// AssignIncident is called within the console's incident-revision transaction.
// Owner/cover slots have one current operator; helper assignments are additive.
func (s *Store) AssignIncident(ctx context.Context, incident, operator uuid.UUID, role string) (uuid.UUID, error) {
	if role != "owner" && role != "cover" && role != "helper" {
		return uuid.Nil, ErrInvalid
	}
	id := uuid.New()
	err := s.maintenance(ctx, func(tx pgx.Tx, now time.Time) error {
		var eligible bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM safety_operators o WHERE id=$1 AND active AND deactivated_at IS NULL AND did IS NOT NULL AND ($2='helper' OR role<>'helper') AND NOT EXISTS(SELECT 1 FROM owner_lifecycles WHERE owner_did=o.did AND state='terminal'))`, operator, role).Scan(&eligible); err != nil {
			return err
		}
		if !eligible {
			return ErrInvalid
		}
		if role != "helper" {
			rows, err := tx.Query(ctx, `UPDATE safety_operator_assignments SET released_at=$3 WHERE incident_id=$1 AND assignment_role=$2 AND released_at IS NULL RETURNING id,operator_id`, incident, role, now)
			if err != nil {
				return err
			}
			type previous struct{ id, operator uuid.UUID }
			prior := []previous{}
			for rows.Next() {
				var p previous
				if err := rows.Scan(&p.id, &p.operator); err != nil {
					rows.Close()
					return err
				}
				prior = append(prior, p)
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
			for _, p := range prior {
				if err := s.assignmentAudit(ctx, tx, p.operator, incident, p.id, "assignmentReleased", now); err != nil {
					return err
				}
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO safety_operator_assignments(id,incident_id,operator_id,assignment_role,assigned_by,assigned_at) VALUES($1,$2,$3,$4,$5,$6)`, id, incident, operator, role, s.actor, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE safety_incidents SET assigned_to=CASE WHEN $3='owner' THEN $2 ELSE assigned_to END,updated_at=$4 WHERE id=$1`, incident, "operator:"+operator.String(), role, now); err != nil {
			return err
		}
		return s.assignmentAudit(ctx, tx, operator, incident, id, "assignmentGranted", now)
	})
	return id, err
}

func (s *Store) ReleaseAssignment(ctx context.Context, incident, assignment uuid.UUID) error {
	return s.maintenance(ctx, func(tx pgx.Tx, now time.Time) error {
		var operator uuid.UUID
		var role string
		err := tx.QueryRow(ctx, `UPDATE safety_operator_assignments SET released_at=$3 WHERE id=$1 AND incident_id=$2 AND released_at IS NULL RETURNING operator_id,assignment_role`, assignment, incident, now).Scan(&operator, &role)
		if err == pgx.ErrNoRows {
			return ErrInvalid
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE safety_incidents SET assigned_to=CASE WHEN $3='owner' AND assigned_to=$2 THEN NULL ELSE assigned_to END,updated_at=$4 WHERE id=$1`, incident, "operator:"+operator.String(), role, now); err != nil {
			return err
		}
		return s.assignmentAudit(ctx, tx, operator, incident, assignment, "assignmentReleased", now)
	})
}
func (s *Store) assignmentAudit(ctx context.Context, tx pgx.Tx, operator, incident, assignment uuid.UUID, action string, now time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO safety_operator_events(id,operator_id,actor_id,action,incident_id,assignment_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.New(), operator, s.actor, action, incident, assignment, now)
	return err
}
