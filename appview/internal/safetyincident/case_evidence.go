package safetyincident

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
)

// PreserveFromCase links private evidence to a case without recording a scan
// result. The console transaction owns both the upload cleanup and receipt.
func (service *EvidenceService) PreserveFromCase(ctx context.Context, tx pgx.Tx, actor Actor, caseID uuid.UUID, command PreserveCommand) (EvidenceRef, uuid.UUID, error) {
	if !actor.HasPermission(PermissionEvidencePreserve) {
		return EvidenceRef{}, uuid.Nil, ErrUnauthorized
	}
	var owner syntax.DID
	var uri syntax.ATURI
	var cid syntax.CID
	var kind string
	if err := tx.QueryRow(ctx, `
		SELECT owner_did,subject_uri,subject_cid_snapshot,subject_type FROM moderation_cases
		WHERE id=$1 AND origin<>'systemDetected'
		  AND NULLIF(subject_uri,'') IS NOT NULL AND NULLIF(subject_cid_snapshot,'') IS NOT NULL
		  AND NOT appview_owner_is_terminal(owner_did)
	`, caseID).Scan(&owner, &uri, &cid, &kind); err != nil {
		return EvidenceRef{}, uuid.Nil, fmt.Errorf("read case for restricted preservation: %w", err)
	}

	now := service.now().UTC()
	var incidentID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO safety_incidents(id,incident_kind,source_case_id,state,detected_at,updated_at)
		VALUES($1,'manualEvidence',$2,'caseCreated',$3,$3)
		ON CONFLICT(source_case_id) DO UPDATE SET updated_at=GREATEST(safety_incidents.updated_at,EXCLUDED.updated_at)
		RETURNING id
	`, uuid.New(), caseID, now).Scan(&incidentID); err != nil {
		return EvidenceRef{}, uuid.Nil, fmt.Errorf("link manual evidence incident: %w", err)
	}
	command.IncidentID = incidentID
	ref, err := service.Preserve(ctx, actor, command)
	if err != nil {
		return EvidenceRef{}, uuid.Nil, err
	}

	// Finish the owner check after the remote write. The fence covers the
	// subject link and commit; a terminal owner rolls back the pending upload.
	states, err := ownerlifecycle.LockOwnerStatesTx(ctx, tx, []syntax.DID{owner})
	if err != nil {
		return EvidenceRef{}, uuid.Nil, err
	}
	switch states[owner].State {
	case ownerlifecycle.StateDeletionPending, ownerlifecycle.StateDeleting, ownerlifecycle.StateTerminal:
		return EvidenceRef{}, uuid.Nil, ErrInvalidEvidence
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO safety_incident_subjects(incident_id,owner_did,subject_uri,source_cid,subject_kind,image_slot,linked_at)
		VALUES($1,$2,$3,$4,$5,'caseEvidence',$6)
		ON CONFLICT(incident_id,subject_uri,image_slot) DO UPDATE SET owner_did=EXCLUDED.owner_did
	`, incidentID, owner, uri, cid, kind, now); err != nil {
		return EvidenceRef{}, uuid.Nil, fmt.Errorf("link restricted evidence subject: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO safety_incident_events(id,incident_id,event_type,actor_id,reference_id,created_at)
		VALUES($1,$2,'manualEvidencePreserved',$3,$4,$5)
	`, uuid.New(), incidentID, actor.ID, ref.ID.String(), now); err != nil {
		return EvidenceRef{}, uuid.Nil, fmt.Errorf("record manual evidence preservation: %w", err)
	}
	return ref, incidentID, nil
}
