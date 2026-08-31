package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

type loginRequest struct {
	Handle              string `json:"handle"`
	HandoffMode         string `json:"handoffMode"` // "verified_link" | "loopback" | dev-only "dev_scheme"
	LoopbackRedirectURI string `json:"loopbackRedirectUri,omitempty"`
}

type loginResponse struct {
	AuthURL string `json:"authUrl"`
}

type registrationRequest struct {
	HandoffMode         string `json:"handoffMode"`
	LoopbackRedirectURI string `json:"loopbackRedirectUri,omitempty"`
}

// LoginHandler starts the OAuth flow and returns the authorization URL.
// The client (Flutter/CLI) opens this URL in the user's system browser.
func (h *HTTPHandlers) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := ctxkeys.GetRunID(r.Context())
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_body",
				"request body could not be decoded",
				runID, nil)
			return
		}
		req.Handle = strings.TrimPrefix(strings.TrimSpace(req.Handle), "@")
		h.Logger.Debug("login: request decoded",
			append(authLogAttrs(runID, "login.start"),
				slog.String("handoff_mode", req.HandoffMode),
				slog.Bool("has_loopback_redirect_uri", req.LoopbackRedirectURI != ""))...)
		if req.Handle == "" {
			envelope.WriteError(w, http.StatusBadRequest, "handle_required",
				"handle is required",
				runID, nil)
			return
		}
		handle, err := syntax.ParseHandle(req.Handle)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handle",
				"handle is malformed",
				runID, nil)
			return
		}
		mode := HandoffMode(req.HandoffMode)
		if mode != HandoffVerifiedLink && mode != HandoffLoopback && !(mode == HandoffDevScheme && h.AllowDevScheme) {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff_mode",
				"handoffMode is not available",
				runID, nil)
			return
		}
		if mode == HandoffLoopback {
			if req.LoopbackRedirectURI == "" {
				envelope.WriteError(w, http.StatusBadRequest, "loopback_redirect_uri_required",
					"loopbackRedirectUri is required when handoffMode is loopback",
					runID, nil)
				return
			}
			if !loopbackRedirectPattern.MatchString(req.LoopbackRedirectURI) {
				envelope.WriteError(w, http.StatusBadRequest, "loopback_redirect_uri_invalid",
					"loopbackRedirectUri must match http://127.0.0.1:<port>[/path]",
					runID, nil)
				return
			}
		}
		deviceID, ok := ctxkeys.GetDeviceID(r.Context())
		if !ok || deviceID == "" {
			envelope.WriteError(w, http.StatusBadRequest, "device_id_required",
				"X-Craftsky-Device-Id is required", runID, nil)
			return
		}
		if h.OAuthFlow == nil {
			w.Header().Set("Retry-After", "5")
			envelope.WriteError(w, http.StatusServiceUnavailable, "authentication_unavailable",
				"authentication is temporarily unavailable", runID, nil)
			return
		}
		h.Logger.Debug("login: starting OAuth flow",
			authLogAttrs(runID, "login.start")...)

		authURL, err := h.OAuthFlow.StartLogin(
			r.Context(), handle, mode, req.LoopbackRedirectURI, deviceID,
		)
		if err != nil {
			if errors.Is(err, ErrOAuthOwnerIneligible) {
				h.Logger.Warn("login rejected for ineligible account",
					authLogErrorAttrs(runID, "login.start", "account_state")...)
				envelope.WriteError(w, http.StatusConflict, "account_unavailable",
					"this account cannot start an ordinary sign-in", runID, nil)
				return
			}
			if errors.Is(err, ErrAuthRequestCapacity) {
				h.Logger.Warn("login rejected by pending authentication capacity",
					authLogErrorAttrs(runID, "login.start", "capacity")...)
				w.Header().Set("Retry-After", "5")
				envelope.WriteError(w, http.StatusServiceUnavailable, "authentication_capacity_exhausted",
					"authentication is temporarily unavailable", runID, nil)
				return
			}
			h.Logger.Warn("StartAuthFlow failed",
				authLogErrorAttrs(runID, "login.start", "authorization_server")...)
			envelope.WriteError(w, http.StatusBadGateway, "authorization_server_unavailable",
				"could not reach the authorization server",
				runID, nil)
			return
		}
		h.Logger.Debug("login: OAuth flow started",
			authLogSuccessAttrs(runID, "login.start")...)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(loginResponse{AuthURL: authURL})
	})
}

// RegistrationHandler starts provider-first OAuth using only the provider
// configured on the AppView. The client can select only its handoff mode.
func (h *HTTPHandlers) RegistrationHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := ctxkeys.GetRunID(r.Context())
		var req registrationRequest
		if err := decodeSingleJSON(r, &req); err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_body",
				"request body could not be decoded", runID, nil)
			return
		}
		mode := HandoffMode(req.HandoffMode)
		if mode != HandoffVerifiedLink && mode != HandoffLoopback && !(mode == HandoffDevScheme && h.AllowDevScheme) {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff_mode",
				"handoffMode is not available", runID, nil)
			return
		}
		if mode != HandoffLoopback && req.LoopbackRedirectURI != "" {
			envelope.WriteError(w, http.StatusBadRequest, "loopback_redirect_uri_invalid",
				"loopbackRedirectUri is available only when handoffMode is loopback", runID, nil)
			return
		}
		if mode == HandoffLoopback {
			if req.LoopbackRedirectURI == "" {
				envelope.WriteError(w, http.StatusBadRequest, "loopback_redirect_uri_required",
					"loopbackRedirectUri is required when handoffMode is loopback", runID, nil)
				return
			}
			if !loopbackRedirectPattern.MatchString(req.LoopbackRedirectURI) {
				envelope.WriteError(w, http.StatusBadRequest, "loopback_redirect_uri_invalid",
					"loopbackRedirectUri must match http://127.0.0.1:<port>[/path]", runID, nil)
				return
			}
		}
		deviceID, ok := ctxkeys.GetDeviceID(r.Context())
		if !ok || deviceID == "" {
			envelope.WriteError(w, http.StatusBadRequest, "device_id_required",
				"X-Craftsky-Device-Id is required", runID, nil)
			return
		}
		if h.OAuthFlow == nil {
			w.Header().Set("Retry-After", "5")
			envelope.WriteError(w, http.StatusServiceUnavailable, "authentication_unavailable",
				"authentication is temporarily unavailable", runID, nil)
			return
		}
		authURL, err := h.OAuthFlow.StartRegistration(r.Context(), mode, req.LoopbackRedirectURI, deviceID)
		if err != nil {
			if errors.Is(err, ErrAuthRequestCapacity) {
				if h.Logger != nil {
					h.Logger.Warn("registration start rejected",
						append(authLogErrorAttrs(runID, "registration.start", "capacity"),
							slog.String("stage", "admission"))...)
				}
				w.Header().Set("Retry-After", "5")
				envelope.WriteError(w, http.StatusServiceUnavailable, "authentication_capacity_exhausted",
					"authentication is temporarily unavailable", runID, nil)
				return
			}
			var registrationFailure *RegistrationOAuthError
			if h.Logger != nil {
				stage := "start"
				category := "providerUnavailable"
				if errors.As(err, &registrationFailure) {
					stage = string(registrationFailure.Stage)
					category = string(registrationFailure.Code)
				}
				h.Logger.Warn("registration start failed",
					append(authLogErrorAttrs(runID, "registration.start", category),
						slog.String("stage", stage))...)
			}
			if errors.As(err, &registrationFailure) && registrationFailure.Code == RegistrationOAuthIncomplete {
				envelope.WriteError(w, http.StatusBadGateway, "registration_incomplete",
					"registration could not be completed", runID, nil)
				return
			}
			envelope.WriteError(w, http.StatusBadGateway, "registration_provider_unavailable",
				"could not reach the registration provider", runID, nil)
			return
		}
		if h.Logger != nil {
			h.Logger.Debug("registration flow started",
				append(authLogSuccessAttrs(runID, "registration.start"),
					slog.String("stage", "par"))...)
		}
		w.Header().Set("Cache-Control", "no-store")
		envelope.WriteJSON(w, http.StatusOK, loginResponse{AuthURL: authURL})
	})
}

// bearerToken extracts the Bearer token from the Authorization header.
// Returns "" if missing or malformed.
func bearerToken(r *http.Request) string {
	hdr := r.Header.Get("Authorization")
	const p = "Bearer "
	if !strings.HasPrefix(hdr, p) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(hdr, p))
}

// authInfoFromCtx pulls DID and OAuth session ID off the context.
// Assumes the request has passed through middleware.Authenticated.
func authInfoFromCtx(ctx context.Context) (did syntax.DID, sid string, ok bool) {
	did, ok = ctxkeys.GetDID(ctx)
	if !ok {
		return "", "", false
	}
	sid, _ = ctxkeys.GetOAuthSessionID(ctx)
	return did, sid, true
}

// LogoutHandler commits local invalidation before returning. Provider and push
// cleanup are durable background work and cannot retain a usable bearer or
// turn a committed logout into an HTTP failure.
func (h *HTTPHandlers) LogoutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := ctxkeys.GetRunID(r.Context())
		did, _, ok := authInfoFromCtx(r.Context())
		if !ok {
			// Authenticated middleware should have rejected already;
			// a 401 here means routing bug.
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		all := r.URL.Query().Get("all") == "true"
		token := bearerToken(r)
		if h.SessionLifecycle == nil {
			w.Header().Set("Retry-After", "5")
			envelope.WriteError(w, http.StatusServiceUnavailable, "logout_unavailable", "logout is temporarily unavailable", runID, nil)
			return
		}
		h.Logger.Debug("logout: request started",
			append(authLogAttrs(runID, "logout"),
				slog.Bool("all", all),
				slog.Bool("has_bearer_token", token != ""))...)
		var err error
		if all {
			err = h.SessionLifecycle.RevokeAllForDID(r.Context(), did)
		} else {
			deviceID, _ := ctxkeys.GetDeviceID(r.Context())
			err = h.SessionLifecycle.RevokeOne(r.Context(), did, token, deviceID)
		}
		if err != nil && !errors.Is(err, ErrCraftskySessionNotFound) {
			h.Logger.Error("logout local invalidation failed",
				append(authLogErrorAttrs(runID, "logout", "store"), slog.Bool("all", all))...)
			w.Header().Set("Retry-After", "5")
			envelope.WriteError(w, http.StatusServiceUnavailable, "logout_unavailable", "logout is temporarily unavailable", runID, nil)
			return
		}
		h.Logger.Debug("logout: revoked session",
			append(authLogSuccessAttrs(runID, "logout"),
				slog.Bool("all", all))...)
		w.WriteHeader(http.StatusNoContent)
	})
}
