package safetyintake

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidEmailMetadata         = errors.New("invalid external safety email metadata")
	ErrUnsafeAttachment             = errors.New("media attachment was not safely rejected or quarantined")
	ErrAppealOwnerMismatch          = errors.New("external appeal does not match the case owner")
	ErrCorrespondenceIntakeMismatch = errors.New("external correspondence belongs to another intake")
)

type Kind string

const (
	KindAllegation Kind = "allegation"
	KindComplaint  Kind = "complaint"
	KindIncident   Kind = "incident"
	KindAppeal     Kind = "appeal"
)

type Urgency string

const (
	UrgencyRoutine   Urgency = "routine"
	UrgencyPriority  Urgency = "priority"
	UrgencyUrgent    Urgency = "urgent"
	UrgencyImmediate Urgency = "immediate"
)

type AttachmentStatus string

const (
	AttachmentNone        AttachmentStatus = "none"
	AttachmentRejected    AttachmentStatus = "rejected"
	AttachmentQuarantined AttachmentStatus = "quarantined"
)

type EmailMetadata struct {
	ProviderMessageReference string
	ReceivedAt               time.Time
	CanonicalSubject         string
	Kind                     Kind
	Urgency                  Urgency
	OwnerActorID             string
	SenderContact            string
	HasMediaAttachment       bool
	AttachmentStatus         AttachmentStatus
}

type Intake struct {
	ID               uuid.UUID
	Reference        string
	ContactReference string
	Replayed         bool
}

type Store struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewStore(pool *pgxpool.Pool, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{pool: pool, now: now}
}

func (store *Store) AcceptEmail(ctx context.Context, metadata EmailMetadata) (Intake, error) {
	if store == nil || store.pool == nil || strings.TrimSpace(metadata.ProviderMessageReference) == "" ||
		metadata.ReceivedAt.IsZero() || !validCanonicalSubject(metadata.CanonicalSubject, metadata.Kind) ||
		strings.TrimSpace(metadata.OwnerActorID) == "" || !validKind(metadata.Kind) || !validUrgency(metadata.Urgency) {
		return Intake{}, ErrInvalidEmailMetadata
	}
	if metadata.HasMediaAttachment && metadata.AttachmentStatus != AttachmentRejected && metadata.AttachmentStatus != AttachmentQuarantined {
		return Intake{}, ErrUnsafeAttachment
	}
	if !metadata.HasMediaAttachment && metadata.AttachmentStatus != AttachmentNone || !validAttachmentStatus(metadata.AttachmentStatus) {
		return Intake{}, ErrInvalidEmailMetadata
	}
	contactReference := minimizeContact(metadata.SenderContact)
	id := uuid.New()
	result := Intake{ID: id, Reference: "CS-SAF-" + strings.ToUpper(strings.ReplaceAll(id.String()[:13], "-", "")), ContactReference: contactReference}
	now := store.now().UTC()
	command, err := store.pool.Exec(ctx, `INSERT INTO external_safety_intakes(
		id,reference,provider_message_ref,intake_kind,urgency,canonical_subject,owner_actor_id,
		contact_reference,attachment_status,received_at,created_at
	) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	ON CONFLICT(provider_message_ref) DO NOTHING`, id, result.Reference, metadata.ProviderMessageReference,
		metadata.Kind, metadata.Urgency, metadata.CanonicalSubject, metadata.OwnerActorID,
		nullableString(contactReference), metadata.AttachmentStatus, metadata.ReceivedAt.UTC(), now)
	if err != nil {
		return Intake{}, fmt.Errorf("accept external safety email: %w", err)
	}
	if command.RowsAffected() == 1 {
		return result, nil
	}
	if err := store.pool.QueryRow(ctx, `SELECT id,reference,COALESCE(contact_reference,'') FROM external_safety_intakes WHERE provider_message_ref=$1`, metadata.ProviderMessageReference).Scan(&result.ID, &result.Reference, &result.ContactReference); err != nil {
		return Intake{}, err
	}
	result.Replayed = true
	return result, nil
}

type CorrespondenceMetadata struct {
	IntakeID                 uuid.UUID
	ProviderMessageReference string
	ReceivedAt               time.Time
	SenderContact            string
	HasMediaAttachment       bool
	AttachmentStatus         AttachmentStatus
}

func (store *Store) AppendCorrespondence(ctx context.Context, metadata CorrespondenceMetadata) (uuid.UUID, bool, error) {
	if store == nil || store.pool == nil || metadata.IntakeID == uuid.Nil || strings.TrimSpace(metadata.ProviderMessageReference) == "" || metadata.ReceivedAt.IsZero() || !validAttachmentStatus(metadata.AttachmentStatus) {
		return uuid.Nil, false, ErrInvalidEmailMetadata
	}
	if metadata.HasMediaAttachment && metadata.AttachmentStatus != AttachmentRejected && metadata.AttachmentStatus != AttachmentQuarantined {
		return uuid.Nil, false, ErrUnsafeAttachment
	}
	if !metadata.HasMediaAttachment && metadata.AttachmentStatus != AttachmentNone {
		return uuid.Nil, false, ErrInvalidEmailMetadata
	}
	id := uuid.New()
	command, err := store.pool.Exec(ctx, `INSERT INTO external_safety_correspondence(
		id,intake_id,provider_message_ref,contact_reference,attachment_status,received_at,created_at
	) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(provider_message_ref) DO NOTHING`, id, metadata.IntakeID,
		metadata.ProviderMessageReference, nullableString(minimizeContact(metadata.SenderContact)), metadata.AttachmentStatus,
		metadata.ReceivedAt.UTC(), store.now().UTC())
	if err != nil {
		return uuid.Nil, false, err
	}
	if command.RowsAffected() == 1 {
		return id, false, nil
	}
	var intakeID uuid.UUID
	if err := store.pool.QueryRow(ctx, `SELECT id,intake_id FROM external_safety_correspondence WHERE provider_message_ref=$1`, metadata.ProviderMessageReference).Scan(&id, &intakeID); err != nil {
		return uuid.Nil, false, err
	}
	if intakeID != metadata.IntakeID {
		return uuid.Nil, false, ErrCorrespondenceIntakeMismatch
	}
	return id, true, nil
}

type AppealLink struct {
	IntakeID         uuid.UUID
	CaseID           uuid.UUID
	VerifiedOwnerDID syntax.DID
	ActorID          string
	SourceSystem     string
	ReplayID         string
	ReceivedAt       time.Time
}

func (store *Store) BindAppeal(ctx context.Context, link AppealLink) (uuid.UUID, error) {
	if store == nil || store.pool == nil || link.IntakeID == uuid.Nil || link.CaseID == uuid.Nil || link.VerifiedOwnerDID == "" ||
		strings.TrimSpace(link.ActorID) == "" || strings.TrimSpace(link.SourceSystem) == "" || strings.TrimSpace(link.ReplayID) == "" || link.ReceivedAt.IsZero() {
		return uuid.Nil, ErrInvalidEmailMetadata
	}
	correspondenceID := uuid.New()
	contactHash := sha256.Sum256([]byte(link.VerifiedOwnerDID))
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var ownerDID, kind string
		if err := tx.QueryRow(ctx, `SELECT c.owner_did,i.intake_kind FROM moderation_cases c CROSS JOIN external_safety_intakes i WHERE c.id=$1 AND i.id=$2 FOR UPDATE OF c,i`, link.CaseID, link.IntakeID).Scan(&ownerDID, &kind); err != nil {
			return err
		}
		if kind != string(KindAppeal) || ownerDID != link.VerifiedOwnerDID.String() {
			return ErrAppealOwnerMismatch
		}
		var existingCaseID uuid.UUID
		var existingOwnerDID string
		err := tx.QueryRow(ctx, `SELECT moderation_case_id,appeal_correspondence_id,verified_owner_did
			FROM external_safety_appeal_links WHERE intake_id=$1`, link.IntakeID).
			Scan(&existingCaseID, &correspondenceID, &existingOwnerDID)
		if err == nil {
			if existingCaseID != link.CaseID || existingOwnerDID != link.VerifiedOwnerDID.String() {
				return ErrAppealOwnerMismatch
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO moderation_appeal_correspondence(
			id,case_id,source_system,replay_id,sender_reference_hash,received_at,created_at
		) VALUES($1,$2,$3,$4,$5,$6,$6)`, correspondenceID, link.CaseID, link.SourceSystem, link.ReplayID, contactHash[:], link.ReceivedAt.UTC()); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO external_safety_appeal_links(
			intake_id,moderation_case_id,appeal_correspondence_id,verified_owner_did,linked_by_actor_id,linked_at
		) VALUES($1,$2,$3,$4,$5,$6)`, link.IntakeID, link.CaseID, correspondenceID, link.VerifiedOwnerDID, link.ActorID, store.now().UTC())
		return err
	})
	return correspondenceID, err
}

func minimizeContact(contact string) string {
	contact = strings.TrimSpace(strings.ToLower(contact))
	if contact == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(contact))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func validKind(kind Kind) bool {
	return kind == KindAllegation || kind == KindComplaint || kind == KindIncident || kind == KindAppeal
}

func validUrgency(urgency Urgency) bool {
	return urgency == UrgencyRoutine || urgency == UrgencyPriority || urgency == UrgencyUrgent || urgency == UrgencyImmediate
}

func validAttachmentStatus(status AttachmentStatus) bool {
	return status == AttachmentNone || status == AttachmentRejected || status == AttachmentQuarantined
}

func validCanonicalSubject(subject string, kind Kind) bool {
	subject = strings.TrimSpace(subject)
	if _, err := syntax.ParseATURI(subject); err == nil {
		return true
	}
	if kind == KindAppeal && strings.HasPrefix(subject, "case:") {
		_, err := uuid.Parse(strings.TrimPrefix(subject, "case:"))
		return err == nil
	}
	parsed, err := url.Parse(subject)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawFragment != "" || parsed.Path == "" || parsed.Path == "/" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "craftsky.social" || host == "www.craftsky.social"
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
