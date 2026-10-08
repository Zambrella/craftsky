package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/eligibility"
	"social.craftsky/appview/internal/middleware"
)

type AgeEligibilityStatusReader interface {
	Status(context.Context, syntax.DID) (eligibility.Status, error)
}

type AgeEligibilityReviewer interface {
	Review(context.Context, eligibility.Review) (eligibility.Status, error)
}

type ageEligibilityReviewRequest struct {
	AccountDID        string                   `json:"accountDid"`
	State             eligibility.State        `json:"state"`
	EvidenceKind      eligibility.EvidenceKind `json:"evidenceKind"`
	EvidenceReference string                   `json:"evidenceReference"`
	Reason            string                   `json:"reason"`
	AppealGuidance    string                   `json:"appealGuidance"`
}

func AgeEligibilityStatusHandler(reader AgeEligibilityStatusReader, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "authenticated account unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		status, err := reader.Status(r.Context(), did)
		if err != nil {
			logger.Error("age eligibility status failed", slog.String("operation", "eligibility.status"), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusServiceUnavailable, "eligibility_unavailable", "account eligibility unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, status)
	})
}

func AgeEligibilityReviewHandler(reviewer AgeEligibilityReviewer, now func() time.Time) http.Handler {
	if now == nil {
		now = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		moderator, ok := ctxkeys.GetModerator(r.Context())
		if !ok || moderator.Role != "safetyAdministrator" {
			envelope.WriteError(w, http.StatusForbidden, "eligibility_review_forbidden", "eligibility review is forbidden", middleware.GetRunID(r.Context()), nil)
			return
		}
		if reviewer == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "eligibility_review_unavailable", "eligibility review unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		var body ageEligibilityReviewRequest
		if !decodeSafetyIntakeJSON(w, r, &body) {
			return
		}
		did, err := syntax.ParseDID(strings.TrimSpace(body.AccountDID))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "request is invalid", middleware.GetRunID(r.Context()), nil)
			return
		}
		status, err := reviewer.Review(r.Context(), eligibility.Review{
			AccountDID: did, State: body.State, EvidenceKind: body.EvidenceKind,
			EvidenceReference: body.EvidenceReference, ReviewerID: moderator.ActorID,
			Reason: body.Reason, AppealGuidance: body.AppealGuidance, ReviewedAt: now().UTC(),
		})
		if errors.Is(err, eligibility.ErrInvalidReview) {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "request is invalid", middleware.GetRunID(r.Context()), nil)
			return
		}
		if err != nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "eligibility_review_unavailable", "eligibility review unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, status)
	})
}
