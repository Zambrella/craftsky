package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

type InstagramPostImportRequest struct {
	SourceKey string `json:"sourceKey"`
	CreatedAt string `json:"createdAt"`
}

type InstagramPostImportAccounts interface {
	GetAccount(context.Context, syntax.DID) (*instagram.AccountView, error)
}

type InstagramPostImportLookup interface {
	Lookup(context.Context, syntax.DID, []string) ([]instagram.PostImport, error)
}

func validateInstagramPostImport(req PostCreateRequest, fields map[string]string) {
	if req.InstagramImport == nil {
		return
	}
	if !instagram.ValidPostImportSourceKey(req.InstagramImport.SourceKey) {
		fields["instagramImport.sourceKey"] = "must be a lowercase 128-bit source fingerprint"
	}
	created, err := time.Parse(time.RFC3339Nano, req.InstagramImport.CreatedAt)
	if err != nil || created.Before(time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)) || created.After(time.Now().UTC()) {
		fields["instagramImport.createdAt"] = "must be an original Instagram date in the past"
	}
	if req.Reply != nil || (req.Embed != nil && req.Embed.Quote != nil) {
		fields["instagramImport"] = "historical imports cannot be replies or quotes"
	}
}

func requireInstagramPostImportAccount(w http.ResponseWriter, r *http.Request, accounts InstagramPostImportAccounts, owner syntax.DID) bool {
	if accounts == nil {
		envelope.WriteError(w, http.StatusServiceUnavailable, "instagram_unavailable", "Instagram import unavailable", middleware.GetRunID(r.Context()), nil)
		return false
	}
	account, err := accounts.GetAccount(r.Context(), owner)
	if err != nil {
		observability.ReportRequestFailure(r.Context(), err, "instagram.post_import", "account")
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "Instagram account operation failed", middleware.GetRunID(r.Context()), nil)
		return false
	}
	if account == nil || account.State != instagram.LinkActive || account.ConflictPending || account.ReactivationRequired || account.VerifiedAt.IsZero() {
		envelope.WriteError(w, http.StatusForbidden, "instagram_verification_required", "An active verified Instagram account is required", middleware.GetRunID(r.Context()), nil)
		return false
	}
	return true
}

func LookupInstagramPostImportsHandler(accounts InstagramPostImportAccounts, store InstagramPostImportLookup) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramAccountDID(w, r)
			return
		}
		var request struct {
			SourceKeys []string `json:"sourceKeys"`
		}
		if err := decodeStrictJSONObject(r, &request); err != nil || len(request.SourceKeys) < 1 || len(request.SourceKeys) > 100 {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "Choose between one and 100 source keys", middleware.GetRunID(r.Context()), nil)
			return
		}
		for _, key := range request.SourceKeys {
			if !instagram.ValidPostImportSourceKey(key) {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid source key", middleware.GetRunID(r.Context()), nil)
				return
			}
		}
		if !requireInstagramPostImportAccount(w, r, accounts, owner) {
			return
		}
		if store == nil {
			observability.ReportRequestFailure(r.Context(), errors.New("instagram post import store unavailable"), "instagram.post_import", "lookup")
			envelope.WriteError(w, http.StatusServiceUnavailable, "instagram_unavailable", "Instagram import unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		items, err := store.Lookup(r.Context(), owner, request.SourceKeys)
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "instagram.post_import", "lookup")
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "Could not read Instagram import history", middleware.GetRunID(r.Context()), nil)
			return
		}
		if items == nil {
			items = []instagram.PostImport{}
		}
		writeJSONStatus(w, http.StatusOK, struct {
			Items []instagram.PostImport `json:"items"`
		}{items})
	})
}
