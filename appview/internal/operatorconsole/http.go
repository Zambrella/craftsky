package operatorconsole

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/observability"
)

const browserCookie = "craftsky_operator_browser"
const sessionCookie = "craftsky_operator_session"

type Handlers struct {
	store  *Store
	app    *oauth.ClientApp
	origin string
	secure bool
	logger *slog.Logger
}

func (h *Handlers) Metadata() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		w.Header().Set("Content-Type", "application/json")
		m := h.app.Config.ClientMetadata()
		if h.app.Config.IsConfidential() {
			jwksURL := h.origin + "/oauth/admin/jwks.json"
			m.JWKSURI = &jwksURL
		}
		name := "CraftSky moderation"
		m.ClientName = &name
		json.NewEncoder(w).Encode(m)
	})
}
func (h *Handlers) JWKS() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(h.app.Config.PublicJWKS())
	})
}
func NewHandlers(store *Store, app *oauth.ClientApp, origin string, logger *slog.Logger) (*Handlers, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && u.Hostname() == "127.0.0.1")) || store == nil || app == nil || app.Config == nil || app.Dir == nil || app.Resolver == nil {
		return nil, ErrInvalid
	}
	if app.Config.CallbackURL != origin+"/oauth/admin/callback" || len(app.Config.Scopes) != 1 || app.Config.Scopes[0] != "atproto" {
		return nil, ErrInvalid
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{store: store, app: app, origin: origin, secure: u.Scheme == "https", logger: logger}, nil
}
func (h *Handlers) cookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
}
func (h *Handlers) failure(w http.ResponseWriter, r *http.Request, status int, code string, err error) {
	h.failureAt(w, r, status, code, "execution", err)
}

func (h *Handlers) failureAt(w http.ResponseWriter, r *http.Request, status int, code, stage string, err error) {
	noStore(w)
	if err != nil && !errors.Is(err, ErrDenied) && !errors.Is(err, ErrInvalid) {
		observability.ReportRequestFailure(r.Context(), err, "operator.auth", stage)
	} else if err != nil {
		if stage == "execution" {
			stage = "validation"
		}
		observability.LogDiagnostic(r.Context(), h.logger, observability.DiagnosticInput{Error: err, Context: observability.EventContext{"component": "operator-console", "operation": "operator.auth", "failure_stage": stage, "result": "expected"}})
	}
	if r.URL.Path == "/oauth/admin/callback" && strings.Contains(r.Header.Get("Accept"), "text/html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>CraftSky sign-in</title><main><h1>Sign-in could not be completed</h1><p>Please check that your account has operator access, then try again.</p><a href="/admin/">Return to CraftSky moderation</a></main></html>`))
		return
	}
	envelope.WriteError(w, status, code, "Operator sign-in could not be completed.", ctxkeys.GetRunID(r.Context()), nil)
}
func (h *Handlers) sameOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == h.origin && r.Header.Get("Sec-Fetch-Site") != "cross-site"
}
func (h *Handlers) csrf(r *http.Request) bool {
	c, err := r.Cookie(browserCookie)
	v := r.Header.Get("X-CSRF-Token")
	return err == nil && len(c.Value) == 43 && len(v) == 43 && h.sameOrigin(r) && subtle.ConstantTimeCompare([]byte(c.Value), []byte(v)) == 1
}
func (h *Handlers) Bootstrap() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != h.origin) {
			h.failure(w, r, 403, "operator_request_forbidden", ErrDenied)
			return
		}
		value := ""
		if c, err := r.Cookie(browserCookie); err == nil && len(c.Value) == 43 {
			value = c.Value
		}
		if value == "" {
			var err error
			value, err = randomToken()
			if err != nil {
				h.failure(w, r, 500, "operator_auth_unavailable", err)
				return
			}
			h.cookie(w, browserCookie, value, int(sessionLifetime.Seconds()))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"csrfToken": value})
	})
}
func (h *Handlers) Login() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		if !h.csrf(r) {
			h.failure(w, r, 403, "operator_request_forbidden", ErrDenied)
			return
		}
		var body struct {
			Identifier string `json:"identifier"`
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		d.DisallowUnknownFields()
		if err := d.Decode(&body); err != nil {
			h.failure(w, r, 400, "invalid_request", ErrInvalid)
			return
		}
		var extra any
		if d.Decode(&extra) != io.EOF {
			h.failure(w, r, 400, "invalid_request", ErrInvalid)
			return
		}
		id, err := syntax.ParseAtIdentifier(body.Identifier)
		if err != nil {
			h.failure(w, r, 400, "invalid_request", ErrInvalid)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		ident, err := h.app.Dir.Lookup(ctx, id)
		if err != nil || ident == nil || ident.DID == "" || ident.PDSEndpoint() == "" {
			h.failureAt(w, r, 403, "operator_access_denied", "identity_lookup", errors.Join(ErrDenied, err))
			return
		}
		if _, err := h.store.Lookup(ctx, ident.DID); err != nil {
			h.failureAt(w, r, 403, "operator_access_denied", "operator_lookup", err)
			return
		}
		issuer, err := h.app.Resolver.ResolveAuthServerURL(ctx, ident.PDSEndpoint())
		if err != nil {
			h.failureAt(w, r, 503, "operator_auth_unavailable", "provider_discovery", observability.WrapError("discover operator OAuth provider", observability.OAuthRequestFailure(err)))
			return
		}
		metadata, err := h.app.Resolver.ResolveAuthServerMetadata(ctx, issuer)
		if err != nil {
			h.failureAt(w, r, 503, "operator_auth_unavailable", "provider_metadata", observability.WrapError("fetch and validate operator OAuth provider metadata", observability.OAuthRequestFailure(err)))
			return
		}
		info, err := h.app.SendAuthRequest(ctx, metadata, []string{"atproto"}, body.Identifier)
		if err != nil {
			h.failureAt(w, r, 503, "operator_auth_unavailable", "authorization_request", observability.WrapError("start operator OAuth authorization request", observability.OAuthRequestFailure(err)))
			return
		}
		info.AccountDID = &ident.DID
		browser, _ := r.Cookie(browserCookie)
		if err := h.store.saveAttempt(ctx, attempt{Info: *info, DID: ident.DID, PDS: ident.PDSEndpoint()}, browser.Value); err != nil {
			h.failureAt(w, r, 503, "operator_auth_unavailable", "login_attempt_storage", observability.WrapError("save operator OAuth login attempt", err))
			return
		}
		redirect := metadata.AuthorizationEndpoint + "?" + url.Values{"client_id": {h.app.Config.ClientID}, "request_uri": {info.RequestURI}}.Encode()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"authorizationUrl": redirect})
	})
}
func (h *Handlers) Callback() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		q, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			h.failure(w, r, 400, "operator_login_failed", ErrDenied)
			return
		}
		for _, v := range q {
			if len(v) != 1 {
				h.failure(w, r, 400, "operator_login_failed", ErrDenied)
				return
			}
		}
		browser, err := r.Cookie(browserCookie)
		if err != nil || len(browser.Value) != 43 || q.Get("state") == "" {
			h.failure(w, r, 400, "operator_login_failed", ErrDenied)
			return
		}
		a, err := h.store.consumeAttempt(ctx, q.Get("state"), browser.Value)
		if err != nil {
			h.failure(w, r, 400, "operator_login_failed", err)
			return
		}
		if q.Get("error") != "" || q.Get("iss") != a.Info.AuthServerURL || q.Get("code") == "" || a.Info.AccountDID == nil || *a.Info.AccountDID != a.DID {
			h.failure(w, r, 400, "operator_login_failed", ErrDenied)
			return
		}
		token, err := h.app.SendInitialTokenRequest(ctx, q.Get("code"), a.Info)
		if err != nil {
			h.failure(w, r, 400, "operator_login_failed", err)
			return
		}
		scopes := strings.Fields(token.Scope)
		if token.Subject != a.DID.String() || len(scopes) != 1 || scopes[0] != "atproto" || token.AccessToken == "" {
			h.failure(w, r, 403, "operator_access_denied", ErrDenied)
			return
		}
		// Re-resolve authority after token exchange; never trust an issuer to assert
		// an arbitrary allowlisted account. OAuth tokens are discarded, not stored:
		// this client performs identity authentication only and never calls a PDS.
		ident, err := h.app.Dir.LookupDID(ctx, a.DID)
		if err != nil || ident == nil || ident.DID != a.DID || ident.PDSEndpoint() != a.PDS {
			h.failure(w, r, 403, "operator_access_denied", errors.Join(ErrDenied, err))
			return
		}
		issuer, err := h.app.Resolver.ResolveAuthServerURL(ctx, ident.PDSEndpoint())
		if err != nil || issuer != a.Info.AuthServerURL {
			h.failure(w, r, 403, "operator_access_denied", errors.Join(ErrDenied, err))
			return
		}
		session, err := h.store.issue(ctx, a.DID)
		if err != nil {
			h.failure(w, r, 403, "operator_access_denied", err)
			return
		}
		if old, err := r.Cookie(sessionCookie); err == nil {
			if err := h.store.logout(ctx, old.Value); err != nil {
				_ = h.store.logout(ctx, session)
				h.failure(w, r, 500, "operator_auth_unavailable", err)
				return
			}
		}
		h.cookie(w, sessionCookie, session, int(sessionLifetime.Seconds()))
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
	})
}
func (h *Handlers) Authentication() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			noStore(w)
			c, err := r.Cookie(sessionCookie)
			if err != nil {
				h.failure(w, r, 401, "operator_authentication_required", ErrDenied)
				return
			}
			m, err := h.store.authenticate(r.Context(), c.Value)
			if err != nil {
				h.failure(w, r, 401, "operator_authentication_required", err)
				return
			}
			if r.Method != "GET" && r.Method != "HEAD" && !h.csrf(r) {
				h.failure(w, r, 403, "operator_request_forbidden", ErrDenied)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctxkeys.WithModerator(r.Context(), m)))
		})
	}
}
func (h *Handlers) Logout() http.Handler {
	return h.Authentication()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Cookie(sessionCookie)
		if err := h.store.logout(r.Context(), c.Value); err != nil {
			h.failure(w, r, 500, "operator_auth_unavailable", err)
			return
		}
		h.cookie(w, sessionCookie, "", -1)
		h.cookie(w, browserCookie, "", -1)
		w.WriteHeader(204)
	}))
}
