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

	"social.craftsky/appview/internal/ctxkeys"
)

type loginRequest struct {
	Handle              string `json:"handle"`
	HandoffMode         string `json:"handoff_mode"` // "deep_link" | "loopback"
	LoopbackRedirectURI string `json:"loopback_redirect_uri,omitempty"`
}

type loginResponse struct {
	AuthURL string `json:"auth_url"`
}

// LoginHandler starts the OAuth flow and returns the authorization URL.
// The client (Flutter/CLI) opens this URL in the user's system browser.
func (h *HTTPHandlers) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		req.Handle = strings.TrimPrefix(strings.TrimSpace(req.Handle), "@")
		if req.Handle == "" {
			writeJSONError(w, http.StatusBadRequest, "handle_required")
			return
		}
		if req.HandoffMode != "deep_link" && req.HandoffMode != "loopback" {
			writeJSONError(w, http.StatusBadRequest, "invalid_handoff_mode")
			return
		}
		if req.HandoffMode == "loopback" {
			if req.LoopbackRedirectURI == "" {
				writeJSONError(w, http.StatusBadRequest, "loopback_redirect_uri_required")
				return
			}
			if !loopbackRedirectPattern.MatchString(req.LoopbackRedirectURI) {
				writeJSONError(w, http.StatusBadRequest, "loopback_redirect_uri_invalid")
				return
			}
		}

		authURL, err := h.OAuth.StartAuthFlow(r.Context(), req.Handle)
		if err != nil {
			h.Logger.Warn("StartAuthFlow failed",
				slog.String("handle", req.Handle),
				slog.String("err", err.Error()))
			writeJSONError(w, http.StatusBadGateway, "authorization_server_unavailable")
			return
		}

		requestURI, err := extractRequestURI(authURL)
		if err != nil {
			h.Logger.Error("extractRequestURI from StartAuthFlow URL", slog.String("err", err.Error()))
			writeJSONError(w, http.StatusInternalServerError, "internal")
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
				slog.String("request_uri", requestURI),
				slog.String("err", err.Error()))
			// Continue: callback's loadHandoff falls back to deep_link.
		}

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

// oauthLogout is a small wrapper that parses+validates the DID before
// calling indigo's Logout. Used by LogoutHandler (?all=true path).
func (h *HTTPHandlers) oauthLogout(ctx context.Context, did, sessionID string) error {
	parsed, err := syntax.ParseDID(did)
	if err != nil {
		return err
	}
	return h.OAuth.Logout(ctx, parsed, sessionID)
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
func authInfoFromCtx(ctx context.Context) (did string, sid string, ok bool) {
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
		did, sid, ok := authInfoFromCtx(r.Context())
		if !ok {
			// Authenticated middleware should have rejected already;
			// a 401 here means routing bug.
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("all") == "true" {
			// Step 1: delete the OAuth session. Cascade removes
			// craftsky_sessions rows on success.
			if sid != "" {
				if err := h.oauthLogout(r.Context(), did, sid); err != nil {
					h.Logger.Warn("oauth.Logout failed; revoke-all will cover",
						slog.String("did", did),
						slog.String("session_id", sid),
						slog.String("err", err.Error()))
				}
			}
			// Step 2: belt-and-braces. If Logout succeeded, the cascade
			// already deleted these rows and RevokeAll is a no-op. If
			// Logout failed, this at least invalidates local tokens.
			if err := h.CraftskySessions.RevokeAll(r.Context(), did); err != nil {
				h.Logger.Error("RevokeAll failed", slog.String("did", did), slog.String("err", err.Error()))
				writeJSONError(w, http.StatusInternalServerError, "internal")
				return
			}
		} else {
			token := bearerToken(r)
			if err := h.CraftskySessions.Revoke(r.Context(), token); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "internal")
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
