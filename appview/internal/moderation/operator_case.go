package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/safetyincident"
	"strings"
	"time"
)

type OperatorCaseCommand struct {
	SubjectType  SubjectType
	DID          syntax.DID
	URI          syntax.ATURI
	IntakeID     uuid.UUID
	Actor        safetyincident.Actor
	SourceSystem string
	Reason       string
	At           time.Time
}

func (s *Store) OpenOperatorCase(ctx context.Context, c OperatorCaseCommand) (Case, error) {
	if c.DID == "" || strings.TrimSpace(c.Reason) == "" || len(c.Reason) > 512 || c.At.IsZero() || c.SourceSystem == "" {
		return Case{}, ErrInvalidCommand
	}
	if !c.Actor.HasPermission(safetyincident.PermissionModerationManage) {
		return Case{}, safetyincident.ErrUnauthorized
	}
	var result Case
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		origin := "operatorCreated"
		if c.IntakeID != uuid.Nil {
			if !c.Actor.HasPermission(safetyincident.PermissionIntakeManage) {
				return safetyincident.ErrUnauthorized
			}
			var subject string
			if err := tx.QueryRow(ctx, `SELECT canonical_subject FROM external_safety_intakes WHERE id=$1 FOR UPDATE`, c.IntakeID).Scan(&subject); err != nil {
				return err
			}
			if subject != c.URI.String() && !(c.SubjectType == SubjectAccount && (subject == "at://"+c.DID.String()+"/app.bsky.actor.profile/self" || subject == "at://"+c.DID.String()+"/social.craftsky.business.profile/self")) {
				return ErrInvalidCommand
			}
			origin = "externalIntake"
		}
		collection, rkey, cid := "", "", ""
		switch c.SubjectType {
		case SubjectAccount:
			if c.URI != "" {
				return ErrInvalidCommand
			}
		case SubjectPost, SubjectEvent:
			if c.URI == "" || c.URI.Authority().String() != c.DID.String() {
				return ErrInvalidCommand
			}
			collection = c.URI.Collection().String()
			rkey = c.URI.RecordKey().String()
			table := "craftsky_posts"
			if c.SubjectType == SubjectEvent {
				if collection != "social.craftsky.business.event" {
					return ErrInvalidCommand
				}
				table = "craftsky_business_events"
			} else if collection != "social.craftsky.feed.post" {
				return ErrInvalidCommand
			}
			if err := tx.QueryRow(ctx, `SELECT cid FROM `+table+` WHERE uri=$1`, c.URI).Scan(&cid); err != nil {
				return err
			}
		default:
			return ErrInvalidCommand
		}
		key := string(c.SubjectType) + ":" + c.DID.String()
		if c.URI != "" {
			key = string(c.SubjectType) + ":" + c.URI.String()
		}
		snapshot, _ := json.Marshal(safeSubjectSnapshot{Type: c.SubjectType, DID: c.DID.String(), Collection: collection, Rkey: rkey, URI: c.URI.String(), CID: cid})
		err := tx.QueryRow(ctx, `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,subject_cid_snapshot,owner_did,safe_snapshot,origin,created_at,updated_at)
   VALUES($1,$2,$3,$4,$5,$6,$7,$8,$4,$9,$10,$11,$11) ON CONFLICT (subject_key,origin) WHERE state='open' DO NOTHING RETURNING id,subject_key,state,revision,created_at`, uuid.New(), key, c.SubjectType, c.DID, nullIfEmpty(collection), nullIfEmpty(rkey), nullIfEmpty(c.URI.String()), nullIfEmpty(cid), snapshot, origin, c.At).Scan(&result.ID, &result.SubjectKey, &result.State, &result.Revision, &result.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `SELECT id,subject_key,state,revision,created_at FROM moderation_cases WHERE subject_key=$1 AND origin=$2 AND state='open'`, key, origin).Scan(&result.ID, &result.SubjectKey, &result.State, &result.Revision, &result.CreatedAt)
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO moderation_case_provenance(id,case_id,intake_id,actor_id,source_system,reason,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.New(), result.ID, nullUUID(c.IntakeID), c.Actor.ID, c.SourceSystem, c.Reason, c.At)
		return err
	})
	return result, err
}
