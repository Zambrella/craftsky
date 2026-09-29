package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/safetyincident"
)

func PurgeOwnerSafetyEvidence(ctx context.Context, pool *pgxpool.Pool, objects safetyincident.EvidenceStore, owner syntax.DID, now time.Time) error {
	if pool == nil || owner == "" || now.IsZero() {
		return errors.New("owner safety evidence cleanup scope is invalid")
	}
	rows, err := pool.Query(ctx, `SELECT DISTINCT evidence.id,evidence.object_key
		FROM safety_evidence evidence
		JOIN safety_incident_subjects subject ON subject.incident_id=evidence.incident_id
		WHERE subject.owner_did=$1 AND evidence.deleted_at IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM safety_legal_hold_evidence scope
			JOIN safety_legal_holds hold ON hold.id=scope.hold_id
			WHERE scope.evidence_id=evidence.id AND hold.released_at IS NULL AND hold.expires_at>$2
		  ) ORDER BY evidence.id`, owner, now.UTC())
	if err != nil {
		return fmt.Errorf("list owner restricted evidence: %w", err)
	}
	type item struct {
		id  uuid.UUID
		key string
	}
	var items []item
	for rows.Next() {
		var value item
		if err := rows.Scan(&value.id, &value.key); err != nil {
			rows.Close()
			return err
		}
		items = append(items, value)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(items) > 0 && objects == nil {
		return errors.New("restricted evidence object cleanup unavailable")
	}
	for _, item := range items {
		if _, err := safetyincident.DeleteEvidenceIfUnheld(
			ctx, pool, objects, item.id, "account-deletion-worker", "ownerAccountDeletion", now,
		); err != nil {
			return fmt.Errorf("delete owner restricted evidence object: %w", err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE safety_incident_subjects subject
		SET owner_did=NULL
		WHERE subject.owner_did=$1
		  AND NOT EXISTS (
			SELECT 1 FROM safety_evidence evidence
			JOIN safety_legal_hold_evidence scope ON scope.evidence_id=evidence.id
			JOIN safety_legal_holds hold ON hold.id=scope.hold_id
			WHERE evidence.incident_id=subject.incident_id AND evidence.deleted_at IS NULL
			  AND hold.released_at IS NULL AND hold.expires_at>$2
		  )`, owner, now.UTC()); err != nil {
		return fmt.Errorf("anonymize owner restricted incident links: %w", err)
	}
	return nil
}
