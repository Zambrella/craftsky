// appview/internal/api/post_interactions.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/relationships"
)

type likePostStore interface {
	DirectedInteractionAuthorizer
	postTargetReader
	FindActiveLike(context.Context, string, string) (*InteractionRow, error)
}

type unlikePostStore interface {
	postTargetReader
	FindActiveLike(context.Context, string, string) (*InteractionRow, error)
}

type repostPostStore interface {
	DirectedInteractionAuthorizer
	shareTargetReader
	FindActiveRepost(context.Context, string, string) (*InteractionRow, error)
}

type unrepostPostStore interface {
	postTargetReader
	FindActiveRepost(context.Context, string, string) (*InteractionRow, error)
}

// DeletePostHandler serves DELETE /v1/posts/{did}/{rkey}. Idempotent —
// returns 204 even if the underlying record was already gone.
func DeletePostHandler(newEffects pdseffects.ExecutorFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		if did != caller {
			envelope.WriteError(w, http.StatusForbidden,
				"forbidden", "cannot delete another user's post", runID, nil)
			return
		}
		ownerGeneration, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		rkey := r.PathValue("rkey")
		parsedRkey, err := syntax.ParseRecordKey(rkey)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "rkey path segment is not a valid record key", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("post delete: request started",
			pdsLogAttrs(runID, pdsOperationPostDelete, pdsStageRequestBuild)...)
		executor, err := newPDSEffectExecutor(r.Context(), newEffects, did, sessionID)
		if err != nil {
			logger.Error("post: durable effect executor unavailable",
				pdsLogErrorAttrs(runID, pdsOperationPostDelete, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		expectedOwners, err := executor.ResolveExpectedOwners(r.Context(), ownerGeneration, nil)
		if err != nil {
			logger.Warn("post: durable effect scope rejected",
				pdsLogErrorAttrs(runID, pdsOperationPostDelete, pdsStageRequestBuild, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, err)
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "post.delete")
		_, err = executor.DeleteRecord(r.Context(), pdseffects.DeleteRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: did, OwnerGeneration: ownerGeneration, ExpectedOwners: expectedOwners,
			Collection: syntax.NSID(craftskyPostNSID), Rkey: parsedRkey,
		})
		if err != nil {
			logger.Warn("post: durable PDS delete failed",
				pdsLogErrorAttrs(runID, pdsOperationPostDelete, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, err)
			return
		}
		logger.Debug("post delete: PDS record deleted",
			pdsLogSuccessAttrs(runID, pdsOperationPostDelete, pdsStagePDSRequest)...)
		w.WriteHeader(http.StatusNoContent)
	})
}

// LikePostHandler serves POST /v1/posts/{did}/{rkey}/likes.
func LikePostHandler(store likePostStore, newEffects pdseffects.ExecutorFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		ownerGeneration, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		if err := rejectNonEmptyBody(r); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"unexpected_field", "request body rejected", runID, nil)
			return
		}
		targetDID, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		logger.Debug("like: resolving target",
			pdsLogAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild)...)
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), rkey)
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("like: resolve target failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		if !authorizeDirectedInteraction(w, r, store, caller, targetDID, relationships.OperationLikeCreate) {
			return
		}
		active, err := store.FindActiveLike(r.Context(), caller.String(), target.URI)
		if err == nil {
			logger.Debug("like: active like already exists",
				pdsLogSuccessAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild)...)
			writeInteractionResponse(w, http.StatusOK, interactionResponseFromRow(active))
			return
		}
		if !errors.Is(err, ErrInteractionNotFound) {
			logger.Error("like: find active failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read like", runID, nil)
			return
		}

		createdAt := time.Now().UTC()
		body := likeRecordBody(target, createdAt)
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("like: creating PDS record",
			pdsLogAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild)...)
		executor, err := newPDSEffectExecutor(r.Context(), newEffects, caller, sessionID)
		if err != nil {
			logger.Error("like: durable effect executor unavailable",
				pdsLogErrorAttrs(runID, pdsOperationLikeCreate, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		expectedOwners, err := executor.ResolveExpectedOwners(r.Context(), ownerGeneration, []syntax.DID{targetDID})
		if err != nil {
			logger.Warn("like: durable effect scope rejected",
				pdsLogErrorAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write like", runID, err)
			return
		}
		effectRkey, err := newImmediateRecordKey()
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not prepare like", runID, nil)
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "like.create")
		result, err := executor.PutRecord(r.Context(), pdseffects.PutRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: caller, OwnerGeneration: ownerGeneration, ExpectedOwners: expectedOwners,
			Collection: syntax.NSID(craftskyLikeNSID), Rkey: effectRkey, Record: body,
		})
		if err != nil {
			logger.Warn("like: durable PDS put failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeCreate, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write like", runID, err)
			return
		}
		logger.Debug("like: PDS record created",
			pdsLogSuccessAttrs(runID, pdsOperationLikeCreate, pdsStagePDSRequest)...)
		writeInteractionResponse(w, http.StatusCreated, &InteractionWriteResponse{
			URI:       string(result.URI),
			CID:       string(result.CID),
			Rkey:      path.Base(string(result.URI)),
			Subject:   ResponseStrongRef{URI: target.URI, CID: target.CID},
			CreatedAt: createdAt,
		})
	})
}

// UnlikePostHandler serves DELETE /v1/posts/{did}/{rkey}/likes.
func UnlikePostHandler(store unlikePostStore, newEffects pdseffects.ExecutorFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		ownerGeneration, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		targetDID, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		logger.Debug("unlike: resolving target",
			pdsLogAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild)...)
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), rkey)
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("unlike: resolve target failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveLike(r.Context(), caller.String(), target.URI)
		if errors.Is(err, ErrInteractionNotFound) {
			logger.Debug("unlike: active like absent",
				pdsLogSuccessAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild)...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			logger.Error("unlike: find active failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read like", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("unlike: deleting PDS record",
			pdsLogAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild)...)
		executor, err := newPDSEffectExecutor(r.Context(), newEffects, caller, sessionID)
		if err != nil {
			logger.Error("unlike: durable effect executor unavailable",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		expectedOwners, err := executor.ResolveExpectedOwners(r.Context(), ownerGeneration, nil)
		if err != nil {
			logger.Warn("unlike: durable effect scope rejected",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, err)
			return
		}
		parsedRkey, err := syntax.ParseRecordKey(active.Rkey)
		if err != nil {
			logger.Error("unlike: indexed record key invalid",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not delete like", runID, nil)
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "like.delete")
		_, err = executor.DeleteRecord(r.Context(), pdseffects.DeleteRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: caller, OwnerGeneration: ownerGeneration, ExpectedOwners: expectedOwners,
			Collection: syntax.NSID(craftskyLikeNSID), Rkey: parsedRkey,
			ExpectedCID: syntax.CID(active.CID),
		})
		if err != nil {
			logger.Warn("unlike: durable PDS delete failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, err)
			return
		}
		logger.Debug("unlike: PDS record deleted",
			pdsLogSuccessAttrs(runID, pdsOperationLikeDelete, pdsStagePDSRequest)...)
		w.WriteHeader(http.StatusNoContent)
	})
}

// RepostPostHandler serves POST /v1/posts/{did}/{rkey}/reposts.
func RepostPostHandler(store repostPostStore, newEffects pdseffects.ExecutorFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		ownerGeneration, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		if err := rejectNonEmptyBody(r); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"unexpected_field", "request body rejected", runID, nil)
			return
		}
		targetDID, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		logger.Debug("repost: resolving target",
			pdsLogAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild)...)
		target, err := store.ResolveShareTarget(r.Context(), targetDID.String(), rkey)
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("repost: resolve target failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		if !authorizeDirectedInteraction(w, r, store, caller, targetDID, relationships.OperationRepostCreate) {
			return
		}
		if target.IsReply {
			envelope.WriteError(w, http.StatusUnprocessableEntity,
				"validation_failed", "validation failed", runID, map[string]string{"target": "reply posts cannot be reposted"})
			return
		}
		active, err := store.FindActiveRepost(r.Context(), caller.String(), target.URI)
		if err == nil {
			logger.Debug("repost: active repost already exists",
				pdsLogSuccessAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild)...)
			writeInteractionResponse(w, http.StatusOK, interactionResponseFromRow(active))
			return
		}
		if !errors.Is(err, ErrInteractionNotFound) {
			logger.Error("repost: find active failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read repost", runID, nil)
			return
		}

		createdAt := time.Now().UTC()
		body := repostRecordBody(&PostTargetRef{URI: target.URI, CID: target.CID}, createdAt)
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("repost: creating PDS record",
			pdsLogAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild)...)
		executor, err := newPDSEffectExecutor(r.Context(), newEffects, caller, sessionID)
		if err != nil {
			logger.Error("repost: durable effect executor unavailable",
				pdsLogErrorAttrs(runID, pdsOperationRepostCreate, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		expectedOwners, err := executor.ResolveExpectedOwners(r.Context(), ownerGeneration, []syntax.DID{targetDID})
		if err != nil {
			logger.Warn("repost: durable effect scope rejected",
				pdsLogErrorAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write repost", runID, err)
			return
		}
		effectRkey, err := newImmediateRecordKey()
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not prepare repost", runID, nil)
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "repost.create")
		result, err := executor.PutRecord(r.Context(), pdseffects.PutRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: caller, OwnerGeneration: ownerGeneration, ExpectedOwners: expectedOwners,
			Collection: syntax.NSID(craftskyRepostNSID), Rkey: effectRkey, Record: body,
		})
		if err != nil {
			logger.Warn("repost: durable PDS put failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostCreate, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write repost", runID, err)
			return
		}
		logger.Debug("repost: PDS record created",
			pdsLogSuccessAttrs(runID, pdsOperationRepostCreate, pdsStagePDSRequest)...)
		writeInteractionResponse(w, http.StatusCreated, &InteractionWriteResponse{
			URI:       string(result.URI),
			CID:       string(result.CID),
			Rkey:      path.Base(string(result.URI)),
			Subject:   ResponseStrongRef{URI: target.URI, CID: target.CID},
			CreatedAt: createdAt,
		})
	})
}

// UnrepostPostHandler serves DELETE /v1/posts/{did}/{rkey}/reposts.
func UnrepostPostHandler(store unrepostPostStore, newEffects pdseffects.ExecutorFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		ownerGeneration, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		targetDID, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		logger.Debug("unrepost: resolving target",
			pdsLogAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild)...)
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), rkey)
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("unrepost: resolve target failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveRepost(r.Context(), caller.String(), target.URI)
		if errors.Is(err, ErrInteractionNotFound) {
			logger.Debug("unrepost: active repost absent",
				pdsLogSuccessAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild)...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			logger.Error("unrepost: find active failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read repost", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("unrepost: deleting PDS record",
			pdsLogAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild)...)
		executor, err := newPDSEffectExecutor(r.Context(), newEffects, caller, sessionID)
		if err != nil {
			logger.Error("unrepost: durable effect executor unavailable",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		expectedOwners, err := executor.ResolveExpectedOwners(r.Context(), ownerGeneration, nil)
		if err != nil {
			logger.Warn("unrepost: durable effect scope rejected",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, err)
			return
		}
		parsedRkey, err := syntax.ParseRecordKey(active.Rkey)
		if err != nil {
			logger.Error("unrepost: indexed record key invalid",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not delete repost", runID, nil)
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "repost.delete")
		_, err = executor.DeleteRecord(r.Context(), pdseffects.DeleteRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: caller, OwnerGeneration: ownerGeneration, ExpectedOwners: expectedOwners,
			Collection: syntax.NSID(craftskyRepostNSID), Rkey: parsedRkey,
			ExpectedCID: syntax.CID(active.CID),
		})
		if err != nil {
			logger.Warn("unrepost: durable PDS delete failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, err)
			return
		}
		logger.Debug("unrepost: PDS record deleted",
			pdsLogSuccessAttrs(runID, pdsOperationRepostDelete, pdsStagePDSRequest)...)
		w.WriteHeader(http.StatusNoContent)
	})
}

func rejectNonEmptyBody(r *http.Request) error {
	if r.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(raw)) != "" {
		return errors.New("body must be empty")
	}
	return nil
}

func likeRecordBody(target *PostTargetRef, createdAt time.Time) map[string]any {
	return interactionRecordBody(craftskyLikeNSID, target, createdAt)
}

func repostRecordBody(target *PostTargetRef, createdAt time.Time) map[string]any {
	return interactionRecordBody(craftskyRepostNSID, target, createdAt)
}

func interactionRecordBody(nsid string, target *PostTargetRef, createdAt time.Time) map[string]any {
	return map[string]any{
		"$type": nsid,
		"subject": map[string]any{
			"uri": target.URI,
			"cid": target.CID,
		},
		"createdAt": createdAt.Format(time.RFC3339),
	}
}

func interactionResponseFromRow(row *InteractionRow) *InteractionWriteResponse {
	return &InteractionWriteResponse{
		URI:       row.URI,
		CID:       row.CID,
		Rkey:      row.Rkey,
		Subject:   ResponseStrongRef{URI: row.SubjectURI, CID: row.SubjectCID},
		CreatedAt: row.CreatedAt.UTC(),
	}
}

func writeInteractionResponse(w http.ResponseWriter, status int, resp *InteractionWriteResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
