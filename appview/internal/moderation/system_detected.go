package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/safetyincident"
)

type ConfirmSystemDetectionCommand struct {
	IncidentID   uuid.UUID
	SubjectURI   syntax.ATURI
	Actor        safetyincident.Actor
	SourceSystem string
	ReplayID     string
	ConfirmedAt  time.Time
}

func (store *Store) ConfirmSystemDetection(ctx context.Context, command ConfirmSystemDetectionCommand) (Case, error) {
	if store == nil || store.pool == nil || command.IncidentID == uuid.Nil || command.SubjectURI == "" ||
		strings.TrimSpace(command.SourceSystem) == "" || strings.TrimSpace(command.ReplayID) == "" || command.ConfirmedAt.IsZero() {
		return Case{}, ErrInvalidCommand
	}
	if !command.Actor.Allowed(safetyincident.PermissionIncidentConfirm, command.IncidentID) {
		return Case{}, safetyincident.ErrUnauthorized
	}
	var result Case
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var owner, sourceCID, kind string
		if err := tx.QueryRow(ctx, `SELECT owner_did,source_cid,subject_kind FROM safety_incident_subjects
			WHERE incident_id=$1 AND subject_uri=$2 ORDER BY image_slot LIMIT 1`, command.IncidentID, command.SubjectURI).Scan(&owner, &sourceCID, &kind); err != nil {
			return err
		}
		subjectType := SubjectType(kind)
		if subjectType != SubjectPost && subjectType != SubjectEvent && subjectType != SubjectAccount {
			return ErrInvalidCommand
		}
		parts := strings.Split(command.SubjectURI.String(), "/")
		if len(parts) < 5 {
			return ErrInvalidCommand
		}
		collection, rkey := parts[3], parts[4]
		snapshot, err := json.Marshal(safeSubjectSnapshot{Type: subjectType, DID: owner, Collection: collection, Rkey: rkey, URI: command.SubjectURI.String(), CID: sourceCID})
		if err != nil {
			return err
		}
		subjectKey := string(subjectType) + ":" + command.SubjectURI.String()
		err = tx.QueryRow(ctx, `INSERT INTO moderation_cases(
			id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,
			subject_cid_snapshot,owner_did,safe_snapshot,origin,incident_id,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$4,$9,'systemDetected',$10,$11,$11)
		ON CONFLICT (subject_key,origin) WHERE state='open' DO NOTHING
		RETURNING id,subject_key,state,revision,created_at`, uuid.New(), subjectKey, subjectType,
			owner, collection, rkey, command.SubjectURI, sourceCID, snapshot, command.IncidentID,
			command.ConfirmedAt.UTC()).Scan(&result.ID, &result.SubjectKey, &result.State, &result.Revision, &result.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `SELECT id,subject_key,state,revision,created_at FROM moderation_cases
				WHERE subject_key=$1 AND origin='systemDetected' AND incident_id=$2 AND state='open'`, subjectKey, command.IncidentID).Scan(&result.ID, &result.SubjectKey, &result.State, &result.Revision, &result.CreatedAt)
		}
		if err != nil {
			return fmt.Errorf("create system detected case: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO safety_incident_events(id,incident_id,event_type,actor_id,source_system,replay_id,created_at)
			VALUES($1,$2,'confirmed',$3,$4,$5,$6) ON CONFLICT(source_system,replay_id) DO NOTHING`, uuid.New(), command.IncidentID,
			command.Actor.ID, command.SourceSystem, command.ReplayID, command.ConfirmedAt.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO safety_incident_events(id,incident_id,event_type,actor_id,source_system,replay_id,created_at)
			VALUES($1,$2,'caseCreated',$3,$4,$5,$6) ON CONFLICT(source_system,replay_id) DO NOTHING`, uuid.New(), command.IncidentID,
			command.Actor.ID, command.SourceSystem, command.ReplayID+":case", command.ConfirmedAt.UTC()); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE safety_incidents SET state='caseCreated',updated_at=$2 WHERE id=$1 AND state IN ('detected','confirmed','caseCreated')`, command.IncidentID, command.ConfirmedAt.UTC())
		return err
	})
	return result, err
}
