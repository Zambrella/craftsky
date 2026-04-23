package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HTTPHandlers bundles the OAuth-related HTTP handlers. Construct via
// NewHTTPHandlers; wire the resulting methods into routes.AddRoutes.
type HTTPHandlers struct {
	OAuth            *oauth.ClientApp
	CraftskySessions *CraftskySessionStore
	Pool             *pgxpool.Pool // for handoff read/write
	Logger           *slog.Logger
	DevMode          bool // emits the session token in the callback HTML when true
	// NewPDSClient builds a PDSClient scoped to the given OAuth session.
	// Injected so tests can supply a mock without standing up indigo.
	NewPDSClient PDSClientFactory
}

func NewHTTPHandlers(
	oauthApp *oauth.ClientApp,
	craftskyStore *CraftskySessionStore,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	devMode bool,
	newPDSClient PDSClientFactory,
) *HTTPHandlers {
	return &HTTPHandlers{
		OAuth:            oauthApp,
		CraftskySessions: craftskyStore,
		Pool:             pool,
		Logger:           logger,
		DevMode:          devMode,
		NewPDSClient:     newPDSClient,
	}
}

// ClientMetadataHandler serves /oauth/client-metadata.json — the
// discovery document Authorization Servers fetch to learn about our
// client.
func (h *HTTPHandlers) ClientMetadataHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := h.OAuth.Config
		meta := cfg.ClientMetadata()
		if cfg.IsConfidential() {
			jwksURL := fmt.Sprintf("https://%s/oauth/jwks.json", r.Host)
			meta.JWKSURI = &jwksURL
		}
		if err := meta.Validate(cfg.ClientID); err != nil {
			h.Logger.Error("client metadata validation failed", slog.String("err", err.Error()))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	})
}

// JWKSHandler serves /oauth/jwks.json — the public keys for confidential
// client auth. In dev (public client) this is an empty keys array.
func (h *HTTPHandlers) JWKSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(h.OAuth.Config.PublicJWKS())
	})
}

// CallbackHandler receives the user browser after the PDS authentication
// step. It completes the OAuth dance via indigo, issues a Craftsky
// bearer token, and renders an HTML page that hands the token to the
// client (deep link for mobile/desktop; loopback POST for CLI/dev).
func (h *HTTPHandlers) CallbackHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		sessData, err := h.OAuth.ProcessCallback(r.Context(), r.URL.Query())
		if err != nil {
			h.Logger.Warn("ProcessCallback failed",
				slog.String("state", state),
				slog.String("err", err.Error()))
			renderErrorHTML(w, http.StatusBadRequest, "Sign-in could not be completed. Please try again.")
			return
		}

		mode, loopbackURI, herr := h.loadHandoff(r.Context(), state)
		if herr != nil {
			// Best-effort: ProcessCallback may already have deleted the
			// auth-request row (it's single-use). Fall back to deep_link.
			h.Logger.Warn("loadHandoff failed; defaulting to deep_link",
				slog.String("state", state),
				slog.String("err", herr.Error()))
			mode = "deep_link"
		}
		if mode == "" {
			mode = "deep_link"
		}

		pdsClient, err := h.NewPDSClient(r.Context(), sessData.AccountDID, sessData.SessionID)
		if err != nil {
			h.Logger.Error("NewPDSClient failed",
				slog.String("did", sessData.AccountDID.String()),
				slog.String("err", err.Error()))
			renderErrorHTML(w, http.StatusBadGateway,
				"Sign-in succeeded but we couldn't initialise your profile. Please try again.")
			return
		}
		if err := InitializeProfile(r.Context(), pdsClient, sessData.AccountDID); err != nil {
			h.Logger.Warn("InitializeProfile failed",
				slog.String("did", sessData.AccountDID.String()),
				slog.String("err", err.Error()))
			switch {
			case errors.Is(err, ErrProfileDataInvalid):
				renderErrorHTML(w, http.StatusBadGateway,
					"Your Craftsky profile record is in an unexpected format. Contact support.")
			default:
				renderErrorHTML(w, http.StatusBadGateway,
					"Sign-in succeeded but we couldn't initialise your profile. Please try again.")
			}
			return
		}

		token, err := h.CraftskySessions.Create(r.Context(), sessData.AccountDID.String(), sessData.SessionID, "")
		if err != nil {
			h.Logger.Error("CraftskySessions.Create failed", slog.String("err", err.Error()))
			renderErrorHTML(w, http.StatusInternalServerError, "Internal error. Please try again.")
			return
		}

		data := callbackPageData{Token: token, DevMode: h.DevMode}
		switch mode {
		case "loopback":
			if loopbackURI == "" {
				renderErrorHTML(w, http.StatusInternalServerError, "Missing loopback redirect URI.")
				return
			}
			// Re-validate at egress (defence in depth; ingress check is primary).
			if !loopbackRedirectPattern.MatchString(loopbackURI) {
				h.Logger.Error("loopback_redirect_uri failed egress validation",
					slog.String("uri", loopbackURI))
				renderErrorHTML(w, http.StatusInternalServerError, "Invalid loopback redirect URI.")
				return
			}
			data.LoopbackURI = loopbackURI
		default: // deep_link
			// Triple slash — empty host, path "/auth/complete". Matches
			// the Flutter route at RouteLocations.authComplete ('/auth/complete').
			data.DeepLinkURL = "craftsky:///auth/complete?token=" + urlEscape(token)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := callbackTmpl.Execute(w, data); err != nil {
			h.Logger.Error("callback template", slog.String("err", err.Error()))
		}
	})
}

// urlEscape avoids importing net/url just for the tiny QueryEscape call;
// we re-export it here for the callback's deep-link rendering.
var urlEscape = url.QueryEscape
