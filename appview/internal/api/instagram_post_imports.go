package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/pdscommands"
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
	if !account.AllowsImport() {
		writeInstagramPostImportVerificationRequired(w, r)
		return false
	}
	return true
}

func writeInstagramPostImportVerificationRequired(w http.ResponseWriter, r *http.Request) {
	envelope.WriteError(w, http.StatusForbidden, "instagram_verification_required", "An active verified Instagram account is required", middleware.GetRunID(r.Context()), nil)
}

// An exact retry of an earlier dispatch can recover its result after the
// provider's upload proof expires. A new dispatch still needs a valid proof.
func existingInstagramVideoImport(w http.ResponseWriter, r *http.Request, commands AppendCommandExecutor, owner syntax.DID, generation int64, key uuid.UUID, req PostCreateRequest) (allowReadRecovery, handled bool) {
	lookup, ok := commands.(commandReplayLookup)
	if !ok {
		return false, false
	}
	ctx := r.Context()
	runID := middleware.GetRunID(ctx)
	replay, err := lookup.LookupCommand(ctx, owner, "post.create", key)
	if err != nil {
		WriteCommandError(w, runID, err, ctx)
		return false, true
	}
	if replay == nil {
		return false, false
	}
	intent, err := json.Marshal(req)
	if err != nil {
		WriteCommandError(w, runID, err, ctx)
		return false, true
	}
	matches, err := replay.MatchesIntent(intent)
	if err != nil {
		WriteCommandError(w, runID, err, ctx)
		return false, true
	}
	if replay.OwnerGeneration != generation || !matches {
		WriteCommandError(w, runID, pdscommands.ErrIdempotencyConflict, ctx)
		return false, true
	}
	switch replay.Result.State {
	case pdscommands.CommandAccepted, pdscommands.CommandRejected:
		WriteCommandResponse(w, CommandResultFromStored(replay.Result))
		return false, true
	case pdscommands.CommandDispatching, pdscommands.CommandAmbiguous:
		return true, false
	default:
		return false, false
	}
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
