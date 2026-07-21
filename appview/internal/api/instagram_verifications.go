package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
)

type InstagramVerificationService interface {
	CreateVerification(context.Context, syntax.DID) (instagram.CreatedVerification, error)
	GetVerification(context.Context, syntax.DID, uuid.UUID) (*instagram.VerificationAttempt, error)
	CancelVerification(context.Context, syntax.DID, uuid.UUID) error
	ConfirmVerification(context.Context, syntax.DID, uuid.UUID, bool) (instagram.ConfirmationResult, error)
}

type instagramVerificationCreateResponse struct {
	VerificationID string                             `json:"verificationId"`
	State          instagram.VerificationAttemptState `json:"state"`
	Challenge      string                             `json:"challenge"`
	ExpiresAt      string                             `json:"expiresAt"`
	DMURL          string                             `json:"dmUrl"`
}

type instagramVerificationResponse struct {
	VerificationID    string                             `json:"verificationId"`
	State             instagram.VerificationAttemptState `json:"state"`
	ExpiresAt         string                             `json:"expiresAt"`
	CandidateUsername string                             `json:"candidateUsername,omitempty"`
	RetryCode         instagram.AttemptRetryCode         `json:"retryCode,omitempty"`
}

type instagramAccountResponse struct {
	State                instagram.InstagramLinkState `json:"state"`
	Username             string                       `json:"username"`
	Discoverable         bool                         `json:"discoverable"`
	ConflictPending      bool                         `json:"conflictPending"`
	ReactivationRequired bool                         `json:"reactivationRequired"`
	VerifiedAt           string                       `json:"verifiedAt"`
}

type instagramConfirmationResponse struct {
	State   instagram.VerificationAttemptState `json:"state"`
	Account instagramAccountResponse           `json:"account"`
}

func CreateInstagramVerificationHandler(service InstagramVerificationService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		var request struct{}
		if err := decodeStrictJSONObject(r, &request); err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", middleware.GetRunID(r.Context()), nil)
			return
		}
		created, err := service.CreateVerification(r.Context(), owner)
		if err != nil {
			writeInstagramVerificationError(w, r, logger, err)
			return
		}
		writeJSONStatus(w, http.StatusCreated, instagramVerificationCreateResponse{
			VerificationID: created.Attempt.ID.String(),
			State:          created.Attempt.State,
			Challenge:      created.Challenge,
			ExpiresAt:      instagramTime(created.Attempt.ExpiresAt),
			DMURL:          created.DMURL,
		})
	})
}

func GetInstagramVerificationHandler(service InstagramVerificationService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		id, err := uuid.Parse(r.PathValue("verificationId"))
		if err != nil {
			envelope.WriteError(w, http.StatusNotFound, "instagram_verification_not_found", "Instagram verification not found", middleware.GetRunID(r.Context()), nil)
			return
		}
		attempt, err := service.GetVerification(r.Context(), owner, id)
		if err != nil {
			writeInstagramVerificationError(w, r, logger, err)
			return
		}
		response := instagramVerificationResponse{
			VerificationID: attempt.ID.String(),
			State:          attempt.State,
			ExpiresAt:      instagramTime(attempt.ExpiresAt),
		}
		if attempt.State == instagram.AttemptPendingConfirmation {
			response.CandidateUsername = attempt.CandidateUsername
		}
		if attempt.RetryCode.Valid() {
			response.RetryCode = attempt.RetryCode
		}
		writeJSONStatus(w, http.StatusOK, response)
	})
}

func DeleteInstagramVerificationHandler(service InstagramVerificationService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		id, err := uuid.Parse(r.PathValue("verificationId"))
		if err == nil {
			if err := service.CancelVerification(r.Context(), owner, id); err != nil {
				logger.Error("Instagram verification cancellation failed",
					slog.String("run_id", middleware.GetRunID(r.Context())),
					slog.String("error_category", "store"))
				envelope.WriteError(w, http.StatusServiceUnavailable, "instagram_unavailable", "Instagram migration unavailable", middleware.GetRunID(r.Context()), nil)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func ConfirmInstagramVerificationHandler(service InstagramVerificationService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		id, err := uuid.Parse(r.PathValue("verificationId"))
		if err != nil {
			envelope.WriteError(w, http.StatusNotFound, "instagram_verification_not_found", "Instagram verification not found", middleware.GetRunID(r.Context()), nil)
			return
		}
		var request struct {
			Discoverable *bool `json:"discoverable"`
		}
		if err := decodeStrictJSONObject(r, &request); err != nil || request.Discoverable == nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", middleware.GetRunID(r.Context()), nil)
			return
		}
		result, err := service.ConfirmVerification(r.Context(), owner, id, *request.Discoverable)
		if err != nil {
			writeInstagramVerificationError(w, r, logger, err)
			return
		}
		writeJSONStatus(w, http.StatusOK, instagramConfirmationResponse{
			State: result.State,
			Account: instagramAccountResponse{
				State:                result.Account.State,
				Username:             result.Account.Username,
				Discoverable:         result.Account.Discoverable,
				ConflictPending:      result.Account.ConflictPending,
				ReactivationRequired: result.Account.ReactivationRequired,
				VerifiedAt:           instagramTime(result.Account.VerifiedAt),
			},
		})
	})
}

func decodeStrictJSONObject(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func writeInstagramVerificationError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	runID := middleware.GetRunID(r.Context())
	switch {
	case errors.Is(err, instagram.ErrVerificationUnavailable):
		envelope.WriteError(w, http.StatusServiceUnavailable, "instagram_verification_unavailable", "Instagram verification unavailable", runID, nil)
	case errors.Is(err, instagram.ErrInstagramResourceNotFound):
		envelope.WriteError(w, http.StatusNotFound, "instagram_verification_not_found", "Instagram verification not found", runID, nil)
	case errors.Is(err, instagram.ErrInstagramLinkConflict):
		envelope.WriteError(w, http.StatusConflict, "instagram_link_conflict", "Instagram account link conflict", runID, nil)
	case errors.Is(err, instagram.ErrInstagramStateTransition):
		envelope.WriteError(w, http.StatusConflict, "instagram_verification_state_conflict", "Instagram verification state conflict", runID, nil)
	default:
		logger.Error("Instagram verification operation failed",
			slog.String("run_id", runID),
			slog.String("error_category", "internal"))
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "Instagram verification failed", runID, nil)
	}
}

func writeJSONStatus(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func instagramTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func instagramLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}
