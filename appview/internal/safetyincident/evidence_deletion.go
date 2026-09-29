package safetyincident

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DeleteEvidenceIfUnheld serializes object deletion with legal-hold creation.
// The evidence row remains locked until both the object and metadata are deleted.
func DeleteEvidenceIfUnheld(
	ctx context.Context,
	pool *pgxpool.Pool,
	objects EvidenceStore,
	evidenceID uuid.UUID,
	actorID string,
	reason string,
	now time.Time,
) (bool, error) {
	if pool == nil || objects == nil || evidenceID == uuid.Nil || strings.TrimSpace(actorID) == "" ||
		strings.TrimSpace(reason) == "" || now.IsZero() {
		return false, errors.New("restricted evidence deletion scope is invalid")
	}
	deleted := false
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var incidentID uuid.UUID
		var objectKey string
		err := tx.QueryRow(ctx, `SELECT incident_id,object_key FROM safety_evidence
			WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, evidenceID).Scan(&incidentID, &objectKey)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var held bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM safety_legal_hold_evidence scope
			JOIN safety_legal_holds hold ON hold.id=scope.hold_id
			WHERE scope.evidence_id=$1 AND hold.released_at IS NULL AND hold.expires_at>$2
		)`, evidenceID, now.UTC()).Scan(&held); err != nil {
			return err
		}
		if held {
			_, err := tx.Exec(ctx, `INSERT INTO safety_retention_events(
				id,data_class,item_reference,outcome,occurred_at
			) VALUES($1,'restrictedEvidence',$2,'held',$3) ON CONFLICT DO NOTHING`, uuid.New(), evidenceID, now.UTC())
			return err
		}
		if err := objects.Delete(ctx, ObjectRef{Key: objectKey}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE safety_evidence SET deleted_at=$2 WHERE id=$1`, evidenceID, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO safety_evidence_accesses(
			id,evidence_id,actor_id,action,reason,allowed,created_at
		) VALUES($1,$2,$3,'delete',$4,true,$5)`, uuid.New(), evidenceID, actorID, reason, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO safety_retention_events(
			id,data_class,item_reference,outcome,occurred_at
		) VALUES($1,'restrictedEvidence',$2,'deleted',$3) ON CONFLICT DO NOTHING`, uuid.New(), evidenceID, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE safety_incident_subjects subject
			SET owner_did=NULL
			WHERE subject.incident_id=$1
			  AND NOT EXISTS (
				SELECT 1 FROM safety_evidence evidence
				JOIN safety_legal_hold_evidence scope ON scope.evidence_id=evidence.id
				JOIN safety_legal_holds hold ON hold.id=scope.hold_id
				WHERE evidence.incident_id=$1 AND evidence.deleted_at IS NULL
				  AND hold.released_at IS NULL AND hold.expires_at>$2
			  )`, incidentID, now.UTC()); err != nil {
			return err
		}
		deleted = true
		return nil
	})
	return deleted, err
}
