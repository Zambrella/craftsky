package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"social.craftsky/appview/internal/observability"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type OnboardingStatus struct {
	Completed             bool       `json:"completed"`
	CompletedAt           *time.Time `json:"completedAt,omitempty"`
	RequiredPolicyVersion string     `json:"requiredPolicyVersion"`
	AcceptedPolicyVersion string     `json:"acceptedPolicyVersion,omitempty"`
	AcceptedAt            *time.Time `json:"acceptedAt,omitempty"`
}

type OnboardingAcceptance struct {
	MeetsMinimumAge bool   `json:"meetsMinimumAge"`
	PolicyVersion   string `json:"policyVersion"`
}

var (
	ErrMinimumAgeDeclarationRequired = errors.New("minimum age declaration required")
	ErrPolicyVersionMismatch         = errors.New("policy version mismatch")
)

type OnboardingStatusService interface {
	Status(context.Context, syntax.DID) (OnboardingStatus, error)
	Complete(context.Context, syntax.DID, OnboardingAcceptance) (OnboardingStatus, error)
}

func GetOnboardingStatusHandler(store OnboardingStatusService, logger *slog.Logger) http.Handler {
	return onboardingStatusHandler(store, logger, false)
}

func CompleteOnboardingHandler(store OnboardingStatusService, logger *slog.Logger) http.Handler {
	return onboardingStatusHandler(store, logger, true)
}

func onboardingStatusHandler(store OnboardingStatusService, logger *slog.Logger, completing bool) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		if len(r.URL.Query()) != 0 {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request",
				"onboarding routes do not accept query parameters", runID, nil)
			return
		}
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did",
				"authenticated DID missing", runID, nil)
			return
		}
		var (
			status OnboardingStatus
			err    error
		)
		if completing {
			var acceptance OnboardingAcceptance
			if decodeErr := decodeStrictJSONObject(r, &acceptance); decodeErr != nil {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_request",
					"invalid onboarding declaration", runID, nil)
				return
			}
			status, err = store.Complete(r.Context(), did, acceptance)
		} else {
			status, err = store.Status(r.Context(), did)
		}
		if err != nil {
			if errors.Is(err, ErrMinimumAgeDeclarationRequired) || errors.Is(err, ErrPolicyVersionMismatch) {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_request",
					"minimum age declaration and current policy version are required", runID, nil)
				return
			}
			operation := "onboarding.status.read"
			if completing {
				operation = "onboarding.completion.write"
			}
			logger.Error("onboarding completion operation failed",
				slog.String("operation", operation),
				slog.String("error_category", "store"),
				slog.String("run_id", runID))
			observability.ReportRequestFailure(r.Context(), err, "api.onboardingStatusHandler", "handler")
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error",
				"onboarding status unavailable", runID, nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, status)
	})
}
