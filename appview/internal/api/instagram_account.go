package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
)

type InstagramAccountStore interface {
	GetAccount(context.Context, syntax.DID) (*instagram.AccountView, error)
	UpdateSettings(context.Context, syntax.DID, instagram.AccountSettingsPatch) (*instagram.AccountView, error)
	RevokeAccount(context.Context, syntax.DID) error
}

// InstagramIntegrationAvailability reports whether new Meta-dependent work is
// currently enabled. It does not gate local account reads or privacy controls.
type InstagramIntegrationAvailability func() bool

type instagramAccountStatusResponse struct {
	IntegrationAvailable bool                      `json:"integrationAvailable"`
	Account              *instagramAccountResponse `json:"account"`
}

func GetInstagramAccountHandler(store InstagramAccountStore, available InstagramIntegrationAvailability, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramAccountDID(w, r)
			return
		}
		if store == nil {
			writeInstagramAccountError(w, r, logger, errors.New("Instagram account store unavailable"))
			return
		}

		account, err := store.GetAccount(r.Context(), owner)
		if err != nil {
			writeInstagramAccountError(w, r, logger, err)
			return
		}
		writeJSONStatus(w, http.StatusOK, newInstagramAccountStatusResponse(integrationAvailable(available), account))
	})
}

func PatchInstagramSettingsHandler(store InstagramAccountStore, available InstagramIntegrationAvailability, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramAccountDID(w, r)
			return
		}

		var request struct {
			Discoverable *bool `json:"discoverable"`
			Reactivate   *bool `json:"reactivate"`
		}
		if err := decodeStrictJSONObject(r, &request); err != nil || !validInstagramSettingsRequest(request.Discoverable, request.Reactivate) {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", middleware.GetRunID(r.Context()), nil)
			return
		}
		if store == nil {
			writeInstagramAccountError(w, r, logger, errors.New("Instagram account store unavailable"))
			return
		}

		account, err := store.UpdateSettings(r.Context(), owner, instagram.AccountSettingsPatch{
			Discoverable: request.Discoverable,
			Reactivate:   request.Reactivate,
		})
		if err != nil {
			writeInstagramAccountError(w, r, logger, err)
			return
		}
		writeJSONStatus(w, http.StatusOK, newInstagramAccountStatusResponse(integrationAvailable(available), account))
	})
}

func DeleteInstagramAccountHandler(store InstagramAccountStore, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramAccountDID(w, r)
			return
		}
		if store == nil {
			writeInstagramAccountError(w, r, logger, errors.New("Instagram account store unavailable"))
			return
		}

		if err := store.RevokeAccount(r.Context(), owner); err != nil {
			writeInstagramAccountError(w, r, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func newInstagramAccountStatusResponse(available bool, account *instagram.AccountView) instagramAccountStatusResponse {
	response := instagramAccountStatusResponse{IntegrationAvailable: available}
	if account == nil {
		return response
	}
	response.Account = &instagramAccountResponse{
		State:                account.State,
		Username:             account.Username,
		Discoverable:         account.Discoverable,
		ConflictPending:      account.ConflictPending,
		ReactivationRequired: account.ReactivationRequired,
		VerifiedAt:           instagramTime(account.VerifiedAt),
	}
	return response
}

func validInstagramSettingsRequest(discoverable, reactivate *bool) bool {
	if discoverable == nil && reactivate == nil {
		return false
	}
	if reactivate != nil && (!*reactivate || discoverable == nil) {
		return false
	}
	return true
}

func integrationAvailable(available InstagramIntegrationAvailability) bool {
	return available != nil && available()
}

func writeMissingInstagramAccountDID(w http.ResponseWriter, r *http.Request) {
	envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
}

func writeInstagramAccountError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	runID := middleware.GetRunID(r.Context())
	switch {
	case errors.Is(err, instagram.ErrInvalidInstagramSettings):
		envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", runID, nil)
	case errors.Is(err, instagram.ErrInstagramLinkNotFound):
		envelope.WriteError(w, http.StatusNotFound, "instagram_link_not_found", "Instagram account link not found", runID, nil)
	case errors.Is(err, instagram.ErrInstagramReactivationRequired):
		envelope.WriteError(w, http.StatusConflict, "instagram_reactivation_required", "Instagram account reactivation required", runID, nil)
	case errors.Is(err, instagram.ErrInstagramLinkConflict):
		envelope.WriteError(w, http.StatusConflict, "instagram_link_conflict", "Instagram account link conflict", runID, nil)
	default:
		logger.Error("Instagram account operation failed",
			slog.String("run_id", runID),
			slog.String("error_category", "internal"))
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "Instagram account operation failed", runID, nil)
	}
}
