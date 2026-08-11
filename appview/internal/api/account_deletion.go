package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

func CreateAccountDeletionIntentHandler(service accountdeletion.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		owner, deviceID, ok := accountDeletionScope(w, r)
		if !ok {
			return
		}
		if service == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "account_deletion_unavailable", "account deletion is unavailable", runID, nil)
			return
		}
		result, err := service.CreateIntent(r.Context(), accountdeletion.CreateIntentParams{Owner: owner, DeviceID: deviceID})
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
		owner, deviceID, ok := accountDeletionScope(w, r)
		if !ok {
			return
		}
		jobID := r.PathValue("jobId")
		if _, err := uuid.Parse(jobID); err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid account deletion request", runID, nil)
			return
		}
		statusCapability := r.Header.Get("X-Craftsky-Deletion-Status")
		var body requestBody
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if statusCapability == "" || decoder.Decode(&body) != nil || body.ReauthProof == "" || body.ConfirmationHandle == "" {
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
		result, err := service.Accept(r.Context(), accountdeletion.AcceptParams{
			JobID:              jobID,
			Owner:              owner,
			DeviceID:           deviceID,
			StatusCapability:   statusCapability,
			ReauthProof:        body.ReauthProof,
			ConfirmationHandle: body.ConfirmationHandle,
		})
		if err != nil {
			writeAccountDeletionError(w, runID, err)
			return
		}
		envelope.WriteJSON(w, http.StatusAccepted, result)
	})
}

func CancelAccountDeletionIntentHandler(service accountdeletion.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		owner, _, ok := accountDeletionScope(w, r)
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
		if err := service.CancelIntent(r.Context(), jobID, owner, r.Header.Get("X-Craftsky-Deletion-Status")); err != nil {
			writeAccountDeletionError(w, runID, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func RecoverAccountDeletionHandler(service accountdeletion.RecoveryService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		deviceID, ok := middleware.GetDeviceID(r.Context())
		const prefix = "Bearer "
		authorization := r.Header.Get("Authorization")
		if !ok || service == nil || !strings.HasPrefix(authorization, prefix) {
			envelope.WriteError(w, http.StatusUnauthorized, "invalid_deletion_recovery", "invalid account deletion recovery credential", runID, nil)
			return
		}
		result, err := service.Recover(r.Context(), strings.TrimSpace(strings.TrimPrefix(authorization, prefix)), deviceID)
		if err != nil {
			envelope.WriteError(w, http.StatusUnauthorized, "invalid_deletion_recovery", "invalid account deletion recovery credential", runID, nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, result)
	})
}

func GetAccountDeletionStatusHandler(service accountdeletion.StatusRouteService) http.Handler {
	return accountDeletionStatusHandler(service, accountdeletion.StatusRead)
}

func RetryAccountDeletionHandler(service accountdeletion.StatusRouteService) http.Handler {
	return accountDeletionStatusHandler(service, accountdeletion.StatusRetry)
}

func StartAccountDeletionReauthenticationHandler(service accountdeletion.StatusRouteService) http.Handler {
	return accountDeletionStatusHandler(service, accountdeletion.StatusStartReauthentication)
}

func accountDeletionStatusHandler(service accountdeletion.StatusRouteService, action accountdeletion.StatusAction) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		grant, ok := middleware.DeletionStatusGrantFromContext(r.Context())
		jobID, parseErr := uuid.Parse(r.PathValue("jobId"))
		if !ok || parseErr != nil || grant.JobID != jobID.String() || service == nil {
			envelope.WriteError(w, http.StatusUnauthorized, "invalid_deletion_status", "invalid account deletion status credential", runID, nil)
			return
		}
		switch action {
		case accountdeletion.StatusRead:
			result, err := service.GetStatus(r.Context(), jobID, grant.Owner)
			if err != nil {
				writeAccountDeletionStatusError(w, runID, err)
				return
			}
			envelope.WriteJSON(w, http.StatusOK, result)
		case accountdeletion.StatusRetry:
			result, err := service.Retry(r.Context(), jobID, grant.Owner)
			if err != nil {
				writeAccountDeletionStatusError(w, runID, err)
				return
			}
			envelope.WriteJSON(w, http.StatusAccepted, result)
		case accountdeletion.StatusStartReauthentication:
			result, err := service.StartReauthentication(r.Context(), jobID, grant.Owner)
			if err != nil {
				writeAccountDeletionStatusError(w, runID, err)
				return
			}
			envelope.WriteJSON(w, http.StatusOK, result)
		}
	})
}

func writeAccountDeletionStatusError(w http.ResponseWriter, runID string, err error) {
	if errors.Is(err, accountdeletion.ErrStatusUnauthorized) || errors.Is(err, accountdeletion.ErrOperationNotFound) {
		envelope.WriteError(w, http.StatusUnauthorized, "invalid_deletion_status", "invalid account deletion status credential", runID, nil)
		return
	}
	envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "account deletion status operation failed", runID, nil)
}

func accountDeletionScope(w http.ResponseWriter, r *http.Request) (owner syntax.DID, deviceID string, ok bool) {
	did, didOK := middleware.GetDID(r.Context())
	deviceID, deviceOK := middleware.GetDeviceID(r.Context())
	if !didOK || did == "" || !deviceOK || deviceID == "" {
		envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_scope", "authenticated account scope missing", middleware.GetRunID(r.Context()), nil)
		return "", "", false
	}
	return did, deviceID, true
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
	case errors.Is(err, accountdeletion.ErrStatusUnauthorized):
		envelope.WriteError(w, http.StatusUnauthorized, "invalid_deletion_status", "invalid account deletion status credential", runID, nil)
	default:
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "account deletion operation failed", runID, nil)
	}
}
