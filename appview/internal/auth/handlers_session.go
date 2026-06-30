package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

type loginRequest struct {
	Handle              string `json:"handle"`
	HandoffMode         string `json:"handoffMode"` // "deep_link" | "loopback"
	LoopbackRedirectURI string `json:"loopbackRedirectUri,omitempty"`
}

type loginResponse struct {
	AuthURL string `json:"authUrl"`
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
		if _, err := syntax.ParseHandle(req.Handle); err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handle",
				"handle is malformed",
				runID, nil)
			return
		}
		if req.HandoffMode != "deep_link" && req.HandoffMode != "loopback" {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_handoff_mode",
				"handoffMode must be deep_link or loopback",
				runID, nil)
			return
		}
		if req.HandoffMode == "loopback" {
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
		h.Logger.Debug("login: starting OAuth flow",
			authLogAttrs(runID, "login.start")...)

		authURL, err := h.OAuth.StartAuthFlow(r.Context(), req.Handle)
		if err != nil {
			h.Logger.Warn("StartAuthFlow failed",
				authLogErrorAttrs(runID, "login.start", "authorization_server")...)
			envelope.WriteError(w, http.StatusBadGateway, "authorization_server_unavailable",
				"could not reach the authorization server",
				runID, nil)
			return
		}
		h.Logger.Debug("login: OAuth flow started",
			authLogSuccessAttrs(runID, "login.start")...)

		requestURI, err := extractRequestURI(authURL)
		if err != nil {
			h.Logger.Error("extractRequestURI from StartAuthFlow URL",
				authLogErrorAttrs(runID, "login.start", "internal")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal",
				"internal error",
				runID, nil)
			return
		}
		// Race note: StartAuthFlow already inserted the auth-request row.
		// We update the handoff columns in a follow-up UPDATE keyed by the
		// request_uri stored in the JSONB blob (indigo doesn't expose state
		// in the returned URL, only in its persisted AuthRequestData). A
		// parallel callback arriving between INSERT and UPDATE would see
		// the default handoff_mode='deep_link'. Acceptable for v1.
		if err := h.recordHandoff(r.Context(), requestURI, req.HandoffMode, req.LoopbackRedirectURI); err != nil {
			h.Logger.Error("recordHandoff failed",
				authLogErrorAttrs(runID, "login.record_handoff", "store")...)
			// Continue: callback's loadHandoff falls back to deep_link.
		}
		h.Logger.Debug("login: handoff recorded",
			append(authLogSuccessAttrs(runID, "login.record_handoff"),
				slog.String("handoff_mode", req.HandoffMode),
				slog.Bool("has_loopback_redirect_uri", req.LoopbackRedirectURI != ""))...)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(loginResponse{AuthURL: authURL})
	})
}

var errRequestURIMissing = errors.New("authorization URL missing request_uri parameter")

// extractRequestURI pulls request_uri out of the redirect URL returned
// by indigo's StartAuthFlow. indigo does NOT include state in that URL —
// it's only in the persisted AuthRequestData JSONB blob. We use
// request_uri as our row-lookup key for the handoff UPDATE.
func extractRequestURI(authURL string) (string, error) {
	u, err := url.Parse(authURL)
	if err != nil {
		return "", err
	}
	r := u.Query().Get("request_uri")
	if r == "" {
		return "", errRequestURIMissing
	}
	return r, nil
}

// recordHandoff persists the handoff mode + loopback URI on the
// oauth_auth_requests row identified by request_uri (extracted from the
// redirect URL indigo returns). Sibling-column variant per Appendix A.
//
// We match on data->>'request_uri' since indigo persists request_uri
// inside the opaque JSONB blob, not as a top-level column. The UPDATE is
// idempotent and affects at most one row (request_uri is a per-flow
// random string).
func (h *HTTPHandlers) recordHandoff(ctx context.Context, requestURI, mode, loopbackURI string) error {
	_, err := h.Pool.Exec(ctx,
		`UPDATE oauth_auth_requests SET handoff_mode = $1, loopback_redirect_uri = $2 WHERE data->>'request_uri' = $3`,
		mode, nullableString(loopbackURI), requestURI)
	return err
}

// loadHandoff is the counterpart used by CallbackHandler. Sibling-column variant.
func (h *HTTPHandlers) loadHandoff(ctx context.Context, state string) (mode string, loopbackURI string, err error) {
	var uri *string
	err = h.Pool.QueryRow(ctx,
		`SELECT handoff_mode, loopback_redirect_uri FROM oauth_auth_requests WHERE state = $1`,
		state).Scan(&mode, &uri)
	if uri != nil {
		loopbackURI = *uri
	}
	return
}

// oauthLogout is a thin wrapper around indigo's Logout. The DID has
// already been parsed at the auth boundary, so no extra validation is
// needed here.
func (h *HTTPHandlers) oauthLogout(ctx context.Context, did syntax.DID, sessionID string) error {
	return h.OAuth.Logout(ctx, did, sessionID)
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

// LogoutHandler revokes the presented Craftsky session. With ?all=true,
// revokes every session for the caller's DID and deletes the underlying
// OAuth session (subject to AS-side revocation success).
//
// Invariant for ?all=true: oauth.Logout is called FIRST so the FK
// cascade can remove craftsky_sessions rows; RevokeAll runs as a
// defensive backstop in case AS-side revocation failed.
func (h *HTTPHandlers) LogoutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := ctxkeys.GetRunID(r.Context())
		did, sid, ok := authInfoFromCtx(r.Context())
		if !ok {
			// Authenticated middleware should have rejected already;
			// a 401 here means routing bug.
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		all := r.URL.Query().Get("all") == "true"
		token := bearerToken(r)
		h.Logger.Debug("logout: request started",
			append(authLogAttrs(runID, "logout"),
				slog.Bool("all", all),
				slog.Bool("has_oauth_session", sid != ""),
				slog.Bool("has_bearer_token", token != ""))...)
		if all {
			// Step 1: delete the OAuth session. Cascade removes
			// craftsky_sessions rows on success.
			if sid != "" {
				if err := h.oauthLogout(r.Context(), did, sid); err != nil {
					h.Logger.Warn("oauth.Logout failed; revoke-all will cover",
						append(authLogErrorAttrs(runID, "logout", "oauth"),
							slog.Bool("all", all))...)
				}
			}
			// Step 2: belt-and-braces. If Logout succeeded, the cascade
			// already deleted these rows and RevokeAll is a no-op. If
			// Logout failed, this at least invalidates local tokens.
			if err := h.CraftskySessions.RevokeAll(r.Context(), did.String()); err != nil {
				h.Logger.Error("RevokeAll failed",
					append(authLogErrorAttrs(runID, "logout", "store"),
						slog.Bool("all", all))...)
				envelope.WriteError(w, http.StatusInternalServerError, "internal",
					"internal error",
					ctxkeys.GetRunID(r.Context()), nil)
				return
			}
		} else {
			if err := h.CraftskySessions.Revoke(r.Context(), token); err != nil {
				envelope.WriteError(w, http.StatusInternalServerError, "internal",
					"internal error",
					runID, nil)
				return
			}
		}
		h.Logger.Debug("logout: revoked session",
			append(authLogSuccessAttrs(runID, "logout"),
				slog.Bool("all", all))...)
		w.WriteHeader(http.StatusNoContent)
	})
}
