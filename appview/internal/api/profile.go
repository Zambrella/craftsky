// appview/internal/api/profile.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"social.craftsky/appview/internal/observability"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/pdseffects"
)

// ProfileReader is the read surface the profile GET handlers use. The
// concrete production implementation is *ProfileStore. Tests inject a
// fake.
type ProfileReader interface {
	Read(ctx context.Context, profileDID string, viewerDID string) (*ProfileRow, error)
}

type BusinessProfileReader interface {
	ReadAccountType(context.Context, syntax.DID) (business.AccountType, error)
	ReadEligibleProfile(context.Context, syntax.DID) (*business.ProfileView, error)
	HasUpcomingEvents(context.Context, syntax.DID, time.Time) (bool, error)
}

type ProfileGraphReader interface {
	ListMutualFollowers(ctx context.Context, viewerDID string, profileDID string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error)
	ListFollowers(ctx context.Context, subjectDID string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error)
	ListFollowing(ctx context.Context, did string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error)
}

// GetProfileHandler serves GET /v1/profiles/@{handleOrDid}.
//
// The "{handleOrDid}" path segment arrives URL-decoded by net/http's
// routing. The mux pattern is registered as /v1/profiles/{handleOrDid}
// (ServeMux does not support literal-prefix wildcards), so clients supply
// the "@" prefix and this handler strips it before resolving the identity.
func GetProfileHandler(store ProfileReader, businessProfiles BusinessProfileReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
		runID := middleware.GetRunID(r.Context())
		logger.Debug("profile get: resolving identity",
			apiLogAttrs(runID, "profile.get")...)
		did, err := resolveToDID(r.Context(), raw, resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				logger.Warn("profile: ResolveDID failed",
					apiLogErrorAttrs(r.Context(), runID, "profile.get", "identity", err)...)
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		logger.Debug("profile get: resolved identity",
			apiLogSuccessAttrs(runID, "profile.get")...)
		writeProfileResponse(w, r, store, businessProfiles, resolver, did, "profile.get", logger)
	})
}

// GetMeProfileHandler serves GET /v1/profiles/me.
func GetMeProfileHandler(store ProfileReader, businessProfiles BusinessProfileReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		logger.Debug("profile me: loading profile",
			apiLogAttrs(runID, "profile.me.get")...)
		writeProfileResponse(w, r, store, businessProfiles, resolver, did, "profile.me.get", logger)
	})
}

// GetMutualFollowersHandler serves GET /v1/profiles/@{handleOrDid}/mutual-followers.
func GetMutualFollowersHandler(store ProfileGraphReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}

		raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
		profileDID, err := resolveToDID(r.Context(), raw, resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				logger.Warn("profile mutual followers: ResolveDID failed",
					apiLogErrorAttrs(r.Context(), runID, "profile.mutual_followers.list", "identity", err)...)
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}

		limit := parseLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		rows, nextCursor, total, err := store.ListMutualFollowers(r.Context(), viewerDID.String(), profileDID.String(), limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("profile mutual followers: list failed",
				apiLogErrorAttrs(r.Context(), runID, "profile.mutual_followers.list", "store", err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "mutual followers list failed", runID, nil)
			return
		}

		items, err := buildProfileAccountSummaries(r.Context(), rows, resolver)
		if err != nil {
			logger.Warn("profile mutual followers: ResolveHandle failed",
				apiLogErrorAttrs(r.Context(), runID, "profile.mutual_followers.list", "identity", err)...)
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		body := ProfileAccountPage{Items: items, TotalCount: total}
		if nextCursor != "" {
			body.Cursor = &nextCursor
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

// GetMeFollowersHandler serves GET /v1/profiles/me/followers.
func GetMeFollowersHandler(store ProfileGraphReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return getMeGraphListHandler("followers", store.ListFollowers, resolver, logger)
}

// GetMeFollowingHandler serves GET /v1/profiles/me/following.
func GetMeFollowingHandler(store ProfileGraphReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return getMeGraphListHandler("following", store.ListFollowing, resolver, logger)
}

func getMeGraphListHandler(
	label string,
	list func(context.Context, string, int, string) ([]*ProfileAccountRow, string, int, error),
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		operation := profileGraphListOperation(label)
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		rows, nextCursor, total, err := list(r.Context(), did.String(), limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("profile "+label+": list failed",
				apiLogErrorAttrs(r.Context(), runID, operation, "store", err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", label+" list failed", runID, nil)
			return
		}
		items, err := buildProfileAccountSummaries(r.Context(), rows, resolver)
		if err != nil {
			logger.Warn("profile "+label+": ResolveHandle failed",
				apiLogErrorAttrs(r.Context(), runID, operation, "identity", err)...)
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		body := ProfileAccountPage{Items: items, TotalCount: total}
		if nextCursor != "" {
			body.Cursor = &nextCursor
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

func buildProfileAccountSummaries(ctx context.Context, rows []*ProfileAccountRow, resolver HandleResolver) ([]ProfileAccountSummary, error) {
	items := make([]ProfileAccountSummary, 0, len(rows))
	handles := make(map[string]syntax.Handle)
	for _, row := range rows {
		handle, ok := handles[row.DID]
		if !ok {
			did, err := syntax.ParseDID(row.DID)
			if err != nil {
				return nil, err
			}
			handle, err = resolver.ResolveHandle(ctx, did)
			if err != nil {
				return nil, err
			}
			handles[row.DID] = handle
		}
		items = append(items, BuildProfileAccountSummary(row, handle))
	}
	return items, nil
}

// writeProfileResponse loads the row, resolves the current handle, and
// emits the JSON response. Used by both GET handlers.
func writeProfileResponse(
	w http.ResponseWriter,
	r *http.Request,
	store ProfileReader,
	businessProfiles BusinessProfileReader,
	resolver HandleResolver,
	did syntax.DID,
	operation string,
	logger *slog.Logger,
) {
	runID := middleware.GetRunID(r.Context())
	viewerDID := ""
	if viewer, ok := middleware.GetDID(r.Context()); ok {
		viewerDID = viewer.String()
	}
	row, err := store.Read(r.Context(), did.String(), viewerDID)
	if err != nil {
		if requestCanceled(r.Context(), err) {
			return
		}
		if errors.Is(err, ErrProfileNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"profile_not_found", "profile not found", runID, nil)
			return
		}
		if errors.Is(err, ErrProfileCountsUnavailable) {
			envelope.WriteError(w, http.StatusInternalServerError,
				"profile_counts_unavailable", "required profile counts unavailable", runID, nil)
			return
		}
		logger.Error("profile: store read failed",
			apiLogErrorAttrs(r.Context(), runID, operation, "store", err)...)
		envelope.WriteError(w, http.StatusInternalServerError,
			"internal_error", "profile read failed", runID, nil)
		return
	}
	logger.Debug("profile: store read succeeded",
		apiLogSuccessAttrs(runID, operation)...)
	handle, err := resolver.ResolveHandle(r.Context(), did)
	if err != nil {
		logger.Warn("profile: ResolveHandle failed",
			apiLogErrorAttrs(r.Context(), runID, operation, "identity", err)...)
		envelope.WriteError(w, http.StatusBadGateway,
			"identity_unavailable", "could not resolve handle", runID, nil)
		return
	}
	resp := BuildProfileResponse(row, handle, row.IsCraftskyProfile)
	if businessProfiles != nil && !row.Blocking && !row.BlockedBy {
		accountType, err := businessProfiles.ReadAccountType(r.Context(), did)
		if err != nil {
			logger.Error("profile: account type read failed",
				apiLogErrorAttrs(r.Context(), runID, operation, "store", err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "profile read failed", runID, nil)
			return
		}
		resp.AccountType = &accountType
		presentation, err := businessProfiles.ReadEligibleProfile(r.Context(), did)
		if err != nil {
			logger.Error("profile: business profile read failed",
				apiLogErrorAttrs(r.Context(), runID, operation, "store", err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "profile read failed", runID, nil)
			return
		}
		resp.Business = BuildBusinessProfileResponse(did, presentation)
		if accountType == business.AccountTypeBusiness {
			resp.HasUpcomingEvents, err = businessProfiles.HasUpcomingEvents(r.Context(), did, time.Now().UTC())
			if err != nil {
				logger.Error("profile: upcoming event availability read failed",
					apiLogErrorAttrs(r.Context(), runID, operation, "store", err)...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "profile read failed", runID, nil)
				return
			}
		}
	}
	logger.Debug("profile: response ready",
		apiLogSuccessAttrs(runID, operation)...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// errInvalidIdentifier is used to signal a path-parsing failure back up
// to the handler. Not surfaced beyond this file.
var errInvalidIdentifier = errors.New("invalid identifier")

// resolveToDID parses raw as either a DID (starts with "did:") or a
// handle, and returns the DID either directly or via handle resolution.
func resolveToDID(ctx context.Context, raw string, resolver HandleResolver) (syntax.DID, error) {
	if strings.HasPrefix(raw, "did:") {
		did, err := syntax.ParseDID(raw)
		if err != nil {
			return "", errInvalidIdentifier
		}
		return did, nil
	}
	handle, err := syntax.ParseHandle(raw)
	if err != nil {
		return "", errInvalidIdentifier
	}
	return resolver.ResolveDID(ctx, handle)
}

// Constants repeated from internal/auth/initialize_profile.go. Keeping
// them local keeps the `api` package's surface self-contained for handler
// logic; the canonical declarations live in the auth package.
const (
	blueskyProfileNSID  = "app.bsky.actor.profile"
	craftskyProfileNSID = "social.craftsky.actor.profile"
	profileRecordKey    = "self"
)

type CompoundPutCommandExecutor interface {
	Put(context.Context, pdscommands.CompoundPutCommandRequest) (pdscommands.CommandResult, error)
}

// PutMeProfileHandler serves PUT /v1/profiles/me.
func PutMeProfileHandler(
	store ProfileReader,
	resolver HandleResolver,
	newEffects pdseffects.ExecutorFactory,
	limits MediaLimits,
	logger *slog.Logger,
	compoundCommands ...CompoundPutCommandExecutor,
) http.Handler {
	limits = normalizeMediaLimits(limits)
	var commands CompoundPutCommandExecutor
	commandMode := len(compoundCommands) > 0
	if len(compoundCommands) > 0 {
		commands = compoundCommands[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())

		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		ownerGeneration, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("profile put: request started",
			pdsLogAttrs(runID, pdsOperationProfilePutBsky, pdsStageRequestBuild)...)

		reqBody, err := DecodeProfilePut(r.Body)
		if err != nil {
			if fe, ok := err.(*FieldError); ok {
				envelope.WriteError(w, http.StatusBadRequest, fe.Code,
					"request body rejected", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusBadRequest,
				"malformed_body", "could not parse body", runID, nil)
			return
		}
		if err := ValidateProfilePutWithLimits(reqBody, limits); err != nil {
			if fe, ok := err.(*FieldError); ok {
				envelope.WriteError(w, http.StatusUnprocessableEntity,
					fe.Code, "validation failed", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusUnprocessableEntity,
				"validation_failed", "validation failed", runID, nil)
			return
		}
		logger.Debug("profile put: validated request",
			pdsLogAttrs(runID, pdsOperationProfilePutBsky, pdsStageRequestBuild)...)
		if commandMode {
			if commands == nil {
				envelope.WriteError(w, http.StatusServiceUnavailable,
					"pds_unavailable", "could not contact PDS", runID, nil)
				return
			}
			operationKey, ok := requireCommandOperationKey(w, r, runID)
			if !ok {
				return
			}
			intent, blobs, err := profileCommandIntent(reqBody)
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.PutMeProfileHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "could not prepare profile update", runID, nil)
				return
			}
			uri := func(collection string) syntax.ATURI {
				return syntax.ATURI("at://" + did.String() + "/" + collection + "/" + profileRecordKey)
			}
			result, err := commands.Put(r.Context(), pdscommands.CompoundPutCommandRequest{
				Owner: did, OwnerGeneration: ownerGeneration, SessionID: sessionID,
				OperationKind: "profile.update", OperationKey: operationKey,
				Intent: intent, Blobs: blobs,
				Records: []pdscommands.CompoundPutRecordRequest{
					{URI: uri(blueskyProfileNSID), BuildRecord: func(current pdscommands.AuthoritativeRecord) (json.RawMessage, error) {
						bsky := map[string]any{}
						if len(current.Record) > 0 {
							if err := json.Unmarshal(current.Record, &bsky); err != nil {
								return nil, err
							}
						}
						return json.Marshal(mergeBlueskyRecord(bsky, reqBody))
					}},
					{URI: uri(craftskyProfileNSID), BuildRecord: func(pdscommands.AuthoritativeRecord) (json.RawMessage, error) {
						return json.Marshal(map[string]any{
							"$type":  craftskyProfileNSID,
							"crafts": nonNilStrings(reqBody.Crafts),
						})
					}},
				},
				Accepted: func(records []pdscommands.AuthoritativeRecord) (pdscommands.TerminalResult, error) {
					if len(records) != 2 {
						return pdscommands.TerminalResult{}, errors.New("compound profile result is incomplete")
					}
					var bsky map[string]any
					if err := json.Unmarshal(records[0].Record, &bsky); err != nil {
						return pdscommands.TerminalResult{}, err
					}
					handle, err := resolver.ResolveHandle(r.Context(), did)
					if err != nil {
						return pdscommands.TerminalResult{}, err
					}
					body, err := json.Marshal(BuildProfileResponse(syntheticRow(did.String(), bsky, reqBody.Crafts), handle, false))
					if err != nil {
						return pdscommands.TerminalResult{}, err
					}
					return pdscommands.TerminalResult{
						State: pdscommands.CommandAccepted, HTTPStatus: http.StatusOK, ResponseBody: body,
						ResponseHeaders: json.RawMessage(`{"Content-Type":"application/json"}`),
					}, nil
				},
				Rejected: func(err error) pdscommands.TerminalResult {
					return RejectedCommandResult(runID, err)
				},
			})
			if err != nil {
				WriteCommandError(w, runID, err, r.Context())
				return
			}
			WriteCommandResponse(w, CommandResultFromStored(result))
			return
		}

		if newEffects == nil {
			logger.Error("profile: durable effect factory unavailable",
				pdsLogErrorAttrs(runID, pdsOperationProfilePutBsky, pdsStageSessionResume, errors.New("effect factory unavailable"))...)
			envelope.WriteError(w, http.StatusServiceUnavailable,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		effects, err := newEffects(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("profile: durable effect executor unavailable",
				pdsLogErrorAttrs(runID, pdsOperationProfilePutBsky, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		if effects == nil {
			logger.Error("profile: durable effect executor unavailable",
				pdsLogErrorAttrs(runID, pdsOperationProfilePutBsky, pdsStageSessionResume, errors.New("nil effect executor"))...)
			envelope.WriteError(w, http.StatusServiceUnavailable,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		expectedOwners := []ownerlifecycle.ExpectedOwner{{
			Owner: did, Generation: ownerGeneration,
		}}
		readRequest := func(collection string) pdseffects.ReadRecordRequest {
			return pdseffects.ReadRecordRequest{
				Owner: did, OwnerGeneration: ownerGeneration, ExpectedOwners: expectedOwners,
				Collection: syntax.NSID(collection), Rkey: syntax.RecordKey(profileRecordKey),
			}
		}

		// Read-before-write on Bluesky so we preserve avatar/banner.
		var bsky map[string]any
		bskyCID, err := effects.ReadRecord(r.Context(), readRequest(blueskyProfileNSID), &bsky)
		if errors.Is(err, auth.ErrRecordNotFound) {
			bsky = map[string]any{}
			bskyCID = ""
		} else if err != nil {
			logger.Warn("profile: bluesky getRecord failed",
				pdsLogErrorAttrs(runID, pdsOperationProfilePutBsky, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_read_failed", "could not read current bluesky profile", runID, err)
			return
		}
		var craftsky map[string]any
		craftskyCID, err := effects.ReadRecord(r.Context(), readRequest(craftskyProfileNSID), &craftsky)
		if errors.Is(err, auth.ErrRecordNotFound) {
			craftskyCID = ""
		} else if err != nil {
			logger.Warn("profile: craftsky getRecord failed",
				pdsLogErrorAttrs(runID, pdsOperationProfilePutCraftsky, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_read_failed", "could not read current CraftSky profile", runID, err)
			return
		}
		mergedBsky := mergeBlueskyRecord(bsky, reqBody)
		cskyBody := map[string]any{
			"$type":  craftskyProfileNSID,
			"crafts": nonNilStrings(reqBody.Crafts),
		}
		logger.Debug("profile put: prepared PDS records",
			pdsLogAttrs(runID, pdsOperationProfilePutBsky, pdsStageRequestBuild)...)

		bskyOperationID, bskyMutationKey := immediateEffectIdentity(runID, "profile.put_bsky")
		_, bskyErr := effects.PutRecord(r.Context(), pdseffects.PutRecordRequest{
			OperationID: bskyOperationID, MutationKey: bskyMutationKey,
			Owner: did, OwnerGeneration: ownerGeneration, ExpectedOwners: expectedOwners,
			Collection: syntax.NSID(blueskyProfileNSID), Rkey: syntax.RecordKey(profileRecordKey),
			Record: mergedBsky, ExpectedCID: bskyCID,
		})
		craftskyOperationID, craftskyMutationKey := immediateEffectIdentity(runID, "profile.put_craftsky")
		_, cskyErr := effects.PutRecord(r.Context(), pdseffects.PutRecordRequest{
			OperationID: craftskyOperationID, MutationKey: craftskyMutationKey,
			Owner: did, OwnerGeneration: ownerGeneration, ExpectedOwners: expectedOwners,
			Collection: syntax.NSID(craftskyProfileNSID), Rkey: syntax.RecordKey(profileRecordKey),
			Record: cskyBody, ExpectedCID: craftskyCID,
		})

		switch {
		case bskyErr == nil && cskyErr == nil:
			handle, herr := resolver.ResolveHandle(r.Context(), did)
			if herr != nil {
				observability.ReportRequestFailure(r.Context(), herr, "api.PutMeProfileHandler", "handler")
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			row := syntheticRow(did.String(), mergedBsky, reqBody.Crafts)
			if store != nil {
				previous, readErr := store.Read(r.Context(), did.String(), did.String())
				if readErr != nil && !errors.Is(readErr, ErrProfileNotFound) {
					logger.Warn("profile: last-cleared image lookup failed",
						pdsLogErrorAttrs(runID, pdsOperationProfilePutBsky, pdsStagePDSRequest, readErr)...)
				}
				if previous != nil {
					if !reqBody.Avatar.Present || reqBody.Avatar.Blob != nil {
						row.AvatarCID, row.AvatarMime = previous.AvatarCID, previous.AvatarMime
					}
					if !reqBody.Banner.Present || reqBody.Banner.Blob != nil {
						row.BannerCID, row.BannerMime = previous.BannerCID, previous.BannerMime
					}
				}
			}
			resp := BuildProfileResponse(row, handle, false)
			logger.Debug("profile put: writes succeeded",
				pdsLogSuccessAttrs(runID, pdsOperationProfilePutBsky, pdsStagePDSRequest)...)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		case bskyErr != nil && cskyErr != nil:
			logger.Error("profile: both PDS writes failed",
				append(pdsLogErrorAttrs(runID, pdsOperationProfilePutBsky, pdsStagePDSRequest, bskyErr),
					slog.String("bsky_result", okOrFailed(bskyErr)),
					slog.String("craftsky_result", okOrFailed(cskyErr)))...)
			if errors.Is(bskyErr, auth.ErrPDSSessionExpired) {
				writePDSError(w, http.StatusBadGateway,
					"pds_write_failed", "both profile writes failed", runID, bskyErr)
				return
			}
			if errors.Is(cskyErr, auth.ErrPDSSessionExpired) {
				writePDSError(w, http.StatusBadGateway,
					"pds_write_failed", "both profile writes failed", runID, cskyErr)
				return
			}
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_failed", "both profile writes failed", runID, nil)
		default:
			logger.Warn("profile: partial PDS write",
				append(pdsLogErrorAttrs(runID, pdsOperationProfilePutBsky, pdsStagePDSRequest, firstErr(bskyErr, cskyErr)),
					slog.String("bsky_result", okOrFailed(bskyErr)),
					slog.String("craftsky_result", okOrFailed(cskyErr)))...)
			if errors.Is(bskyErr, auth.ErrPDSSessionExpired) {
				writePDSError(w, http.StatusBadGateway,
					"pds_write_partial", "partial profile write", runID, bskyErr)
				return
			}
			if errors.Is(cskyErr, auth.ErrPDSSessionExpired) {
				writePDSError(w, http.StatusBadGateway,
					"pds_write_partial", "partial profile write", runID, cskyErr)
				return
			}
			fields := map[string]string{
				"bsky":     okOrFailed(bskyErr),
				"craftsky": okOrFailed(cskyErr),
			}
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_partial", "partial profile write", runID, fields)
		}
	})
}

func profileCommandIntent(request ProfilePutRequest) (json.RawMessage, []pdscommands.BlobReference, error) {
	intent := map[string]any{"crafts": nonNilStrings(request.Crafts)}
	if request.DisplayName != nil {
		intent["displayName"] = *request.DisplayName
	}
	if request.Description != nil {
		intent["description"] = *request.Description
	}
	if request.Pronouns != nil {
		intent["pronouns"] = *request.Pronouns
	}
	blobs := make([]pdscommands.BlobReference, 0, 2)
	for _, image := range []struct {
		name   string
		update ProfileImageUpdate
	}{{"avatar", request.Avatar}, {"banner", request.Banner}} {
		if !image.update.Present {
			continue
		}
		intent[image.name] = image.update.Blob
		if image.update.Blob == nil {
			continue
		}
		size, _ := positiveIntegerAsInt64(image.update.Blob["size"])
		ref, _ := image.update.Blob["ref"].(map[string]any)
		cid, _ := ref["$link"].(string)
		mimeType, _ := image.update.Blob["mimeType"].(string)
		blobs = append(blobs, pdscommands.BlobReference{
			CID: syntax.CID(cid), MIMEType: mimeType, Size: size,
		})
	}
	raw, err := json.Marshal(intent)
	return raw, blobs, err
}

// mergeBlueskyRecord returns a fresh record body formed from `existing`
// with displayName, description, and pronouns replaced by the request.
// avatar/banner are tri-state: omitted preserves, null clears, blob replaces.
func mergeBlueskyRecord(existing map[string]any, req ProfilePutRequest) map[string]any {
	out := map[string]any{"$type": blueskyProfileNSID}
	for k, v := range existing {
		switch k {
		case "$type", "displayName", "description", "pronouns":
			continue
		default:
			out[k] = v
		}
	}
	if req.DisplayName != nil {
		out["displayName"] = *req.DisplayName
	}
	if req.Description != nil {
		out["description"] = *req.Description
	}
	if req.Pronouns != nil {
		out["pronouns"] = *req.Pronouns
	}
	applyProfileImageUpdate(out, "avatar", req.Avatar)
	applyProfileImageUpdate(out, "banner", req.Banner)
	return out
}

func applyProfileImageUpdate(out map[string]any, field string, update ProfileImageUpdate) {
	if !update.Present {
		return
	}
	if update.Blob == nil {
		delete(out, field)
		return
	}
	out[field] = update.Blob
}

// syntheticRow constructs the safe text portion of a ProfileRow from the bodies
// just written. Serving images come only from the AppView's last-cleared row.
func syntheticRow(did string, bsky map[string]any, crafts []string) *ProfileRow {
	row := &ProfileRow{DID: did, Crafts: nonNilStrings(crafts), IsCraftskyProfile: true}
	if dn, ok := bsky["displayName"].(string); ok {
		row.DisplayName = &dn
	}
	if desc, ok := bsky["description"].(string); ok {
		row.Description = &desc
	}
	if pronouns, ok := bsky["pronouns"].(string); ok {
		row.Pronouns = &pronouns
	}
	return row
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func okOrFailed(err error) string {
	if err == nil {
		return "ok"
	}
	return "failed"
}
