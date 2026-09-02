package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type OnboardingStatus struct {
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type OnboardingStatusService interface {
	Status(context.Context, syntax.DID) (OnboardingStatus, error)
	Complete(context.Context, syntax.DID) (OnboardingStatus, error)
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
			status, err = store.Complete(r.Context(), did)
		} else {
			status, err = store.Status(r.Context(), did)
		}
		if err != nil {
			operation := "onboarding.status.read"
			if completing {
				operation = "onboarding.completion.write"
			}
			logger.Error("onboarding completion operation failed",
				slog.String("operation", operation),
				slog.String("error_category", "store"),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error",
				"onboarding status unavailable", runID, nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, status)
	})
}
