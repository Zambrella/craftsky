package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ctxkeys"
)

// HTTPHandlers bundles the OAuth-related HTTP handlers. Construct via
// NewHTTPHandlers; wire the resulting methods into routes.AddRoutes.
type HTTPHandlers struct {
	OAuth                     *oauth.ClientApp
	OAuthFlow                 OAuthFlowCoordinator
	ClientMetadata            oauth.ClientMetadata
	PublicJWKS                oauth.JWKS
	CraftskySessions          *CraftskySessionStore
	Handoffs                  HandoffCoordinator
	SessionLifecycle          *SessionLifecycleService
	NewPendingPDSClient       PendingOnboardingPDSClientFactory
	OnboardingProfile         OnboardingProfileWriter
	LoginCompleteURL          string
	DeletionCompleteURL       string
	AllowDevScheme            bool
	Pool                      *pgxpool.Pool // for handoff read/write
	Logger                    *slog.Logger
	IdentityCacheUpdater      IdentityCacheUpdater
	RepositoryTracker         RepositoryTracker
	NotificationSubscriptions NotificationSubscriptionCleaner
	DeletionOAuthCallbacks    AccountDeletionOAuthCallbacks
	DeletionPendingLogin      AccountDeletionPendingLoginPolicy
}

type NotificationSubscriptionCleaner interface {
	DeactivateForInstallation(context.Context, string, string) error
	DeactivateForAccount(context.Context, string) error
}

func NewHTTPHandlers(
	oauthApp *oauth.ClientApp,
	artifacts ClientArtifacts,
	craftskyStore *CraftskySessionStore,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	identityCacheUpdater ...IdentityCacheUpdater,
) *HTTPHandlers {
	var updater IdentityCacheUpdater
	if len(identityCacheUpdater) > 0 {
		updater = identityCacheUpdater[0]
	}
	handlers := &HTTPHandlers{
		OAuth:                oauthApp,
		ClientMetadata:       artifacts.Metadata,
		PublicJWKS:           artifacts.JWKS,
		CraftskySessions:     craftskyStore,
		Pool:                 pool,
		Logger:               logger,
		IdentityCacheUpdater: updater,
	}
	return handlers
}

// ClientMetadataHandler serves /oauth/client-metadata.json — the
// discovery document Authorization Servers fetch to learn about our
// client.
func (h *HTTPHandlers) ClientMetadataHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.OAuth == nil || h.OAuth.Config == nil || h.ClientMetadata.ClientID == "" ||
			h.ClientMetadata.Validate(h.OAuth.Config.ClientID) != nil {
			h.Logger.Error("client metadata validation failed",
				authLogErrorAttrs(ctxkeys.GetRunID(r.Context()), "oauth.client_metadata", "validation")...)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		setOAuthDiscoveryHeaders(w)
		_ = json.NewEncoder(w).Encode(h.ClientMetadata)
	})
}

// JWKSHandler serves /oauth/jwks.json — the public keys for confidential
// client auth. In dev (public client) this is an empty keys array.
func (h *HTTPHandlers) JWKSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		setOAuthDiscoveryHeaders(w)
		_ = json.NewEncoder(w).Encode(h.PublicJWKS)
	})
}

func setOAuthDiscoveryHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

type PendingOnboardingPDSClientFactory func(context.Context, CallbackAttempt) (PDSClient, error)

// CallbackHandler receives the browser after PDS authorization. The callback
// owns the fenced attempt and emits only a short-lived handoff code; no bearer
// or PDS credential crosses the browser boundary.
func (h *HTTPHandlers) CallbackHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := ctxkeys.GetRunID(r.Context())
		var (
			result         OAuthCallbackResult
			handoffCode    string
			deletionResult AccountDeletionOAuthResult
			deletionFlow   bool
		)
		if h.OAuthFlow == nil {
			renderErrorHTML(w, http.StatusInternalServerError, "Internal error. Please try again.")
			return
		}
		err := h.OAuthFlow.CompleteCallback(r.Context(), r.URL.Query(), func(callbackCtx context.Context, callbackResult OAuthCallbackResult) error {
			result = callbackResult
			switch callbackResult.Metadata.Purpose {
			case AccountDeletionOAuthPurpose:
				callbacks, ok := h.DeletionOAuthCallbacks.(AccountDeletionOAuthAttemptCallbacks)
				if !ok {
					return errors.New("account deletion callback lacks attempt-aware finalization")
				}
				request := AccountDeletionAuthRequest{
					Purpose: AccountDeletionOAuthPurpose,
					JobID:   callbackResult.Metadata.JobID.String(),
					Owner:   callbackResult.Metadata.Owner,
				}
				var err error
				deletionResult, err = callbacks.CompleteAttempt(callbackCtx, request, callbackResult.Attempt)
				deletionFlow = err == nil
				return err
			case LoginOAuthPurpose, RegistrationOAuthPurpose:
				if h.NewPendingPDSClient == nil || h.OnboardingProfile == nil || h.Handoffs == nil {
					return errors.New("onboarding callback finalization unavailable")
				}
				pdsClient, err := h.NewPendingPDSClient(callbackCtx, callbackResult.Attempt)
				if err != nil {
					return fmt.Errorf("build pending onboarding PDS client: %w", err)
				}
				if err := InitializeProfileAndIdentityCache(
					callbackCtx, pdsClient, callbackResult.Attempt, h.OnboardingProfile,
					h.IdentityCacheUpdater, h.Logger, h.RepositoryTracker,
				); err != nil {
					return err
				}
				handoffCode, err = h.Handoffs.CreateExchange(
					callbackCtx, callbackResult.Attempt, callbackResult.Handle,
					callbackResult.Metadata.DeviceID,
				)
				return err
			default:
				return ErrOAuthFlowInvalid
			}
		})
		if err != nil {
			var trustedFailure *TrustedRegistrationFailure
			if errors.As(err, &trustedFailure) {
				data, renderErr := trustedRegistrationFailurePageData(
					trustedFailure.Metadata,
					trustedFailure.Code,
					h.LoginCompleteURL,
					h.AllowDevScheme,
					r.URL.Query(),
					time.Now(),
				)
				if renderErr == nil {
					renderErr = renderCallbackHTML(w, data)
				}
				if renderErr == nil {
					if h.Logger != nil {
						h.Logger.Warn("registration callback failed",
							authLogErrorAttrs(runID, "registration.callback", string(trustedFailure.Code))...)
					}
					return
				}
			}
			if h.Logger != nil {
				h.Logger.Warn("OAuth callback finalization failed",
					authLogErrorAttrs(runID, "oauth.callback", "finalization")...)
			}
			renderErrorHTML(w, http.StatusBadRequest, "Sign-in could not be completed. Please try again.")
			return
		}
		data := callbackPageData{Code: handoffCode}
		if deletionFlow {
			query := url.Values{"job-id": {deletionResult.JobID}, "proof": {deletionResult.Proof}}
			if h.AllowDevScheme {
				data.DeepLinkURL, err = devSchemeCompletionURL("/account-deletion/reauth-complete", query)
			} else {
				data.DeepLinkURL, err = verifiedCompletionURL(h.DeletionCompleteURL, query)
			}
		} else {
			switch result.Metadata.HandoffMode {
			case HandoffLoopback:
				if !loopbackRedirectPattern.MatchString(result.Metadata.LoopbackURI) {
					err = ErrOAuthFlowInvalid
				} else {
					data.LoopbackURI = result.Metadata.LoopbackURI
				}
			case HandoffVerifiedLink:
				data.DeepLinkURL, err = verifiedCompletionURL(h.LoginCompleteURL, url.Values{"code": {handoffCode}})
			case HandoffDevScheme:
				if !h.AllowDevScheme {
					err = ErrOAuthFlowInvalid
				} else {
					data.DeepLinkURL, err = devSchemeCompletionURL("/auth/complete", url.Values{"code": {handoffCode}})
				}
			default:
				err = ErrOAuthFlowInvalid
			}
		}
		if err != nil {
			renderErrorHTML(w, http.StatusInternalServerError, "Sign-in could not be completed. Please try again.")
			return
		}
		if err := renderCallbackHTML(w, data); err != nil && h.Logger != nil {
			h.Logger.Error("callback template", authLogErrorAttrs(runID, "oauth.callback", "template")...)
		}
	})
}

func verifiedCompletionURL(raw string, query url.Values) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path == "" {
		return "", errors.New("verified completion URL is not configured")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func devSchemeCompletionURL(path string, query url.Values) (string, error) {
	if path != "/auth/complete" && path != "/account-deletion/reauth-complete" {
		return "", errors.New("development completion path is not allowed")
	}
	parsed, err := url.Parse("craftsky-dev://" + path)
	if err != nil {
		return "", errors.New("development completion URL is invalid")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
