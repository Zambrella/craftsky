package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

func CreateAccountDeletionIntentHandler(service accountdeletion.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		owner, ok := accountDeletionScope(w, r)
		if !ok {
			return
		}
		if service == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "account_deletion_unavailable", "account deletion is unavailable", runID, nil)
			return
		}
		deviceID, ok := middleware.GetDeviceID(r.Context())
		if !ok || deviceID == "" {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "device identity is required", runID, nil)
			return
		}
		result, err := service.CreateIntent(r.Context(), accountdeletion.CreateIntentParams{
			Owner:    owner,
			DeviceID: deviceID,
		})
		if err != nil {
			writeAccountDeletionError(w, runID, err)
			return
		}
		envelope.WriteJSON(w, http.StatusCreated, result)
	})
}

func AcceptAccountDeletionHandler(service accountdeletion.Service) http.Handler {
	type requestBody struct {
		ReauthProof        string `json:"reauthProof"`
		ConfirmationHandle string `json:"confirmationHandle"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		owner, ok := accountDeletionScope(w, r)
		if !ok {
			return
		}
		jobID := r.PathValue("jobId")
		if _, err := uuid.Parse(jobID); err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid account deletion request", runID, nil)
			return
		}
		var body requestBody
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if decoder.Decode(&body) != nil || body.ReauthProof == "" || body.ConfirmationHandle == "" {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid account deletion request", runID, nil)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid account deletion request", runID, nil)
			return
		}
		if service == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "account_deletion_unavailable", "account deletion is unavailable", runID, nil)
			return
		}
		err := service.Accept(r.Context(), accountdeletion.AcceptParams{
			JobID:              jobID,
			Owner:              owner,
			ReauthProof:        body.ReauthProof,
			ConfirmationHandle: body.ConfirmationHandle,
		})
		if err != nil {
			writeAccountDeletionError(w, runID, err)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
}

func CancelAccountDeletionIntentHandler(service accountdeletion.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		owner, ok := accountDeletionScope(w, r)
		if !ok {
			return
		}
		jobID := r.PathValue("jobId")
		if _, err := uuid.Parse(jobID); err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid account deletion request", runID, nil)
			return
		}
		if service == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "account_deletion_unavailable", "account deletion is unavailable", runID, nil)
			return
		}
		if err := service.CancelIntent(r.Context(), jobID, owner); err != nil {
			writeAccountDeletionError(w, runID, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func accountDeletionScope(w http.ResponseWriter, r *http.Request) (owner syntax.DID, ok bool) {
	did, didOK := middleware.GetDID(r.Context())
	if !didOK || did == "" {
		envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_scope", "authenticated account scope missing", middleware.GetRunID(r.Context()), nil)
		return "", false
	}
	return did, true
}

func writeAccountDeletionError(w http.ResponseWriter, runID string, err error) {
	switch {
	case errors.Is(err, accountdeletion.ErrReauthenticationRequired):
		envelope.WriteError(w, http.StatusUnauthorized, "reauthentication_required", "fresh account reauthentication is required", runID, nil)
	case errors.Is(err, accountdeletion.ErrConfirmationHandleMismatch):
		envelope.WriteError(w, http.StatusBadRequest, "confirmation_handle_mismatch", "confirmation handle does not match", runID, nil)
	case errors.Is(err, accountdeletion.ErrDeletionAlreadyPending):
		envelope.WriteError(w, http.StatusConflict, "deletion_already_pending", "account deletion is already pending", runID, nil)
	case errors.Is(err, accountdeletion.ErrPointOfNoReturn):
		envelope.WriteError(w, http.StatusConflict, "deletion_already_accepted", "account deletion has already been accepted", runID, nil)
	default:
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "account deletion operation failed", runID, nil)
	}
}
