package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/safetyintake"
)

const maxSafetyIntakeBodyBytes = 16 << 10

type ExternalSafetyIntakeStore interface {
	AcceptEmail(context.Context, safetyintake.EmailMetadata) (safetyintake.Intake, error)
	AppendCorrespondence(context.Context, safetyintake.CorrespondenceMetadata) (uuid.UUID, bool, error)
	BindAppeal(context.Context, safetyintake.AppealLink) (uuid.UUID, error)
}

type safetyIntakeRequest struct {
	ProviderMessageReference string                        `json:"providerMessageReference"`
	ReceivedAt               time.Time                     `json:"receivedAt"`
	CanonicalSubject         string                        `json:"canonicalSubject"`
	Kind                     safetyintake.Kind             `json:"kind"`
	Urgency                  safetyintake.Urgency          `json:"urgency"`
	SenderContact            string                        `json:"senderContact"`
	HasMediaAttachment       bool                          `json:"hasMediaAttachment"`
	AttachmentStatus         safetyintake.AttachmentStatus `json:"attachmentStatus"`
}

type safetyCorrespondenceRequest struct {
	ProviderMessageReference string                        `json:"providerMessageReference"`
	ReceivedAt               time.Time                     `json:"receivedAt"`
	SenderContact            string                        `json:"senderContact"`
	HasMediaAttachment       bool                          `json:"hasMediaAttachment"`
	AttachmentStatus         safetyintake.AttachmentStatus `json:"attachmentStatus"`
}

type safetyAppealLinkRequest struct {
	CaseID           uuid.UUID `json:"caseId"`
	VerifiedOwnerDID string    `json:"verifiedOwnerDid"`
	ReplayID         string    `json:"replayId"`
	ReceivedAt       time.Time `json:"receivedAt"`
}

func SafetyExternalIntakeHandler(store ExternalSafetyIntakeStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		moderator, ok := ctxkeys.GetModerator(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusUnauthorized, "moderator_authentication_failed", "moderator authentication failed", middleware.GetRunID(r.Context()), nil)
			return
		}
		if store == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "external_intake_unavailable", "external intake unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		var body safetyIntakeRequest
		if !decodeSafetyIntakeJSON(w, r, &body) {
			return
		}
		accepted, err := store.AcceptEmail(r.Context(), safetyintake.EmailMetadata{
			ProviderMessageReference: body.ProviderMessageReference,
			ReceivedAt:               body.ReceivedAt,
			CanonicalSubject:         body.CanonicalSubject,
			Kind:                     body.Kind,
			Urgency:                  body.Urgency,
			OwnerActorID:             moderator.ActorID,
			SenderContact:            body.SenderContact,
			HasMediaAttachment:       body.HasMediaAttachment,
			AttachmentStatus:         body.AttachmentStatus,
		})
		if !writeSafetyIntakeError(w, r, err) {
			return
		}
		envelope.WriteJSON(w, http.StatusCreated, struct {
			ID        uuid.UUID `json:"id"`
			Reference string    `json:"reference"`
			Replayed  bool      `json:"replayed"`
		}{ID: accepted.ID, Reference: accepted.Reference, Replayed: accepted.Replayed})
	})
}

func SafetyExternalCorrespondenceHandler(store ExternalSafetyIntakeStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ctxkeys.GetModerator(r.Context()); !ok {
			envelope.WriteError(w, http.StatusUnauthorized, "moderator_authentication_failed", "moderator authentication failed", middleware.GetRunID(r.Context()), nil)
			return
		}
		if store == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "external_intake_unavailable", "external intake unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		intakeID, err := uuid.Parse(r.PathValue("intakeId"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid intake identifier", middleware.GetRunID(r.Context()), nil)
			return
		}
		var body safetyCorrespondenceRequest
		if !decodeSafetyIntakeJSON(w, r, &body) {
			return
		}
		id, replayed, err := store.AppendCorrespondence(r.Context(), safetyintake.CorrespondenceMetadata{
			IntakeID:                 intakeID,
			ProviderMessageReference: body.ProviderMessageReference,
			ReceivedAt:               body.ReceivedAt,
			SenderContact:            body.SenderContact,
			HasMediaAttachment:       body.HasMediaAttachment,
			AttachmentStatus:         body.AttachmentStatus,
		})
		if !writeSafetyIntakeError(w, r, err) {
			return
		}
		envelope.WriteJSON(w, http.StatusCreated, struct {
			ID       uuid.UUID `json:"id"`
			Replayed bool      `json:"replayed"`
		}{ID: id, Replayed: replayed})
	})
}

func SafetyExternalAppealLinkHandler(store ExternalSafetyIntakeStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		moderator, ok := ctxkeys.GetModerator(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusUnauthorized, "moderator_authentication_failed", "moderator authentication failed", middleware.GetRunID(r.Context()), nil)
			return
		}
		if store == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "external_intake_unavailable", "external intake unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		intakeID, err := uuid.Parse(r.PathValue("intakeId"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid intake identifier", middleware.GetRunID(r.Context()), nil)
			return
		}
		var body safetyAppealLinkRequest
		if !decodeSafetyIntakeJSON(w, r, &body) {
			return
		}
		ownerDID, err := syntax.ParseDID(body.VerifiedOwnerDID)
		if err != nil {
			envelope.WriteError(w, http.StatusUnprocessableEntity, "invalid_external_intake", "external intake is invalid", middleware.GetRunID(r.Context()), nil)
			return
		}
		id, err := store.BindAppeal(r.Context(), safetyintake.AppealLink{
			IntakeID: intakeID, CaseID: body.CaseID, VerifiedOwnerDID: ownerDID,
			ActorID: moderator.ActorID, SourceSystem: "externalEmail",
			ReplayID: body.ReplayID, ReceivedAt: body.ReceivedAt,
		})
		if !writeSafetyIntakeError(w, r, err) {
			return
		}
		envelope.WriteJSON(w, http.StatusCreated, struct {
			CorrespondenceID uuid.UUID `json:"correspondenceId"`
		}{CorrespondenceID: id})
	})
}

func decodeSafetyIntakeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxSafetyIntakeBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		envelope.WriteError(w, http.StatusBadRequest, "malformed_body", "could not parse body", middleware.GetRunID(r.Context()), nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		envelope.WriteError(w, http.StatusBadRequest, "malformed_body", "could not parse body", middleware.GetRunID(r.Context()), nil)
		return false
	}
	return true
}

func writeSafetyIntakeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return true
	}
	runID := middleware.GetRunID(r.Context())
	switch {
	case errors.Is(err, safetyintake.ErrInvalidEmailMetadata):
		envelope.WriteError(w, http.StatusUnprocessableEntity, "invalid_external_intake", "external intake is invalid", runID, nil)
	case errors.Is(err, safetyintake.ErrUnsafeAttachment):
		envelope.WriteError(w, http.StatusUnprocessableEntity, "unsafe_attachment_status", "attachment handling status is unsafe", runID, nil)
	case errors.Is(err, safetyintake.ErrAppealOwnerMismatch):
		envelope.WriteError(w, http.StatusUnprocessableEntity, "appeal_owner_mismatch", "appeal owner could not be verified", runID, nil)
	default:
		envelope.WriteError(w, http.StatusServiceUnavailable, "external_intake_unavailable", "external intake unavailable", runID, nil)
	}
	return false
}
