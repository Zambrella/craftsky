package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
)

var (
	ErrOAuthFlowInvalid     = errors.New("OAuth flow is invalid or expired")
	ErrOAuthOwnerIneligible = errors.New("owner is not eligible for ordinary login")
)

const (
	DefaultOAuthLoginStartTimeout        = 20 * time.Second
	MaximumOAuthLoginStartTimeout        = 60 * time.Second
	DefaultOAuthCallbackOperationTimeout = 45 * time.Second
	MaximumOAuthCallbackOperationTimeout = 90 * time.Second
)

type OAuthFlowServiceOptions struct {
	App                      *oauth.ClientApp
	Store                    *PostgresAuthStore
	Owners                   *ownerlifecycle.Store
	StartOperationTimeout    time.Duration
	CallbackOperationTimeout time.Duration
	DeletionRequests         DeletionOAuthRequestVerifier
}

type OAuthFlowService struct {
	app                      *oauth.ClientApp
	store                    *PostgresAuthStore
	owners                   *ownerlifecycle.Store
	startOperationTimeout    time.Duration
	callbackOperationTimeout time.Duration
	deletionRequests         DeletionOAuthRequestVerifier
}

func NewOAuthFlowService(options OAuthFlowServiceOptions) (*OAuthFlowService, error) {
	if options.App == nil || options.App.Config == nil || options.App.Resolver == nil ||
		options.App.Dir == nil || options.Store == nil || options.Owners == nil {
		return nil, errors.New("OAuth flow service dependencies are unavailable")
	}
	startTimeout, err := normalizeOAuthOperationTimeout(
		options.StartOperationTimeout,
		DefaultOAuthLoginStartTimeout,
		MaximumOAuthLoginStartTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("OAuth login start timeout: %w", err)
	}
	callbackTimeout, err := normalizeOAuthOperationTimeout(
		options.CallbackOperationTimeout,
		DefaultOAuthCallbackOperationTimeout,
		MaximumOAuthCallbackOperationTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("OAuth callback operation timeout: %w", err)
	}
	return &OAuthFlowService{
		app: options.App, store: options.Store, owners: options.Owners,
		startOperationTimeout: startTimeout, callbackOperationTimeout: callbackTimeout,
		deletionRequests: options.DeletionRequests,
	}, nil
}

type DeletionOAuthRequestVerifier interface {
	VerifyDeletionOAuthRequest(
		context.Context,
		pgx.Tx,
		syntax.DID,
		uuid.UUID,
		ownerlifecycle.Lifecycle,
	) error
}

func normalizeOAuthOperationTimeout(configured, fallback, maximum time.Duration) (time.Duration, error) {
	if configured == 0 {
		configured = fallback
	}
	if configured <= 0 || configured > maximum {
		return 0, fmt.Errorf("must be positive and no greater than %s", maximum)
	}
	return configured, nil
}

func oauthOperationContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

type OAuthCallbackResult struct {
	Session  oauth.ClientSessionData
	Metadata AuthRequestMetadata
	Attempt  CallbackAttempt
	Handle   syntax.Handle
}

type OAuthCallbackFinalizer func(context.Context, OAuthCallbackResult) error

type OAuthFlowCoordinator interface {
	StartLogin(context.Context, syntax.Handle, HandoffMode, string, string) (string, error)
	CompleteCallback(context.Context, url.Values, OAuthCallbackFinalizer) error
}

var _ OAuthFlowCoordinator = (*OAuthFlowService)(nil)

// StartLogin resolves the canonical DID before acquiring authority, then
// inserts Indigo's request data and all AppView metadata atomically. Unlike
// Indigo's StartAuthFlow helper, persistence failures are returned to the
// caller rather than discarded.
func (service *OAuthFlowService) StartLogin(
	ctx context.Context,
	handle syntax.Handle,
	mode HandoffMode,
	loopbackURI string,
	deviceID string,
) (string, error) {
	operationCtx, cancel := oauthOperationContext(ctx, service.startOperationTimeout)
	defer cancel()
	ctx = operationCtx
	if handle == "" || deviceID == "" ||
		(mode != HandoffVerifiedLink && mode != HandoffLoopback) ||
		(mode == HandoffLoopback) != (loopbackURI != "") {
		return "", ErrOAuthFlowInvalid
	}
	identifier, err := syntax.ParseAtIdentifier(handle.String())
	if err != nil {
		return "", ErrOAuthFlowInvalid
	}
	identity, err := service.app.Dir.Lookup(ctx, identifier)
	if err != nil {
		return "", fmt.Errorf("resolve login identity: %w", err)
	}
	if identity == nil || identity.DID == "" || identity.PDSEndpoint() == "" {
		return "", ErrOAuthFlowInvalid
	}
	var redirectURL string
	err = service.owners.WithOnboardingAuth(ctx, identity.DID, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		if authority.State != ownerlifecycle.StateDeparted && authority.State != ownerlifecycle.StateActive {
			return ErrOAuthOwnerIneligible
		}
		authServerURL, err := service.app.Resolver.ResolveAuthServerURL(authCtx, identity.PDSEndpoint())
		if err != nil {
			return fmt.Errorf("resolve authorization server: %w", err)
		}
		serverMetadata, err := service.app.Resolver.ResolveAuthServerMetadata(authCtx, authServerURL)
		if err != nil {
			return fmt.Errorf("resolve authorization server metadata: %w", err)
		}
		requestInfo, err := service.app.SendAuthRequest(
			authCtx, serverMetadata, service.app.Config.Scopes, identity.DID.String(),
		)
		if err != nil {
			return fmt.Errorf("send OAuth authorization request: %w", err)
		}
		requestInfo.AccountDID = &identity.DID
		requestCtx := WithLoginAuthRequest(
			authCtx, identity.DID, authority.Generation, authority.AuthEpoch,
			mode, deviceID, loopbackURI,
		)
		if err := service.store.SaveAuthRequestInfo(requestCtx, *requestInfo); err != nil {
			return err
		}
		params := url.Values{"client_id": {service.app.Config.ClientID}, "request_uri": {requestInfo.RequestURI}}
		redirectURL = strings.TrimSuffix(serverMetadata.AuthorizationEndpoint, "?") + "?" + params.Encode()
		return nil
	})
	if err != nil {
		return "", err
	}
	return redirectURL, nil
}

// StartAccountDeletion creates an authorization request only for the exact
// durable deletion intent selected by the authenticated owner. Purpose is
// server-derived; a client cannot turn an ordinary login into deletion OAuth.
func (service *OAuthFlowService) StartAccountDeletion(
	ctx context.Context,
	owner syntax.DID,
	handle syntax.Handle,
	jobID uuid.UUID,
	deviceID string,
) (string, error) {
	operationCtx, cancel := oauthOperationContext(ctx, service.startOperationTimeout)
	defer cancel()
	ctx = operationCtx
	if owner == "" || handle == "" || jobID == uuid.Nil || deviceID == "" || service.deletionRequests == nil {
		return "", ErrOAuthFlowInvalid
	}
	identifier, err := syntax.ParseAtIdentifier(handle.String())
	if err != nil {
		return "", ErrOAuthFlowInvalid
	}
	identity, err := service.app.Dir.Lookup(ctx, identifier)
	if err != nil {
		return "", fmt.Errorf("resolve deletion identity: %w", err)
	}
	if identity == nil || identity.DID != owner || identity.PDSEndpoint() == "" {
		return "", ErrOAuthFlowInvalid
	}
	var redirectURL string
	err = service.owners.WithExistingAuth(ctx, owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		if authority.State != ownerlifecycle.StateDeletionPending {
			return ErrOAuthOwnerIneligible
		}
		if err := service.owners.WithAuthTransaction(authCtx, func(tx pgx.Tx) error {
			return service.deletionRequests.VerifyDeletionOAuthRequest(
				authCtx, tx, owner, jobID, authority,
			)
		}); err != nil {
			return err
		}
		authServerURL, err := service.app.Resolver.ResolveAuthServerURL(authCtx, identity.PDSEndpoint())
		if err != nil {
			return fmt.Errorf("resolve deletion authorization server: %w", err)
		}
		serverMetadata, err := service.app.Resolver.ResolveAuthServerMetadata(authCtx, authServerURL)
		if err != nil {
			return fmt.Errorf("resolve deletion authorization metadata: %w", err)
		}
		requestInfo, err := service.app.SendAuthRequest(
			authCtx, serverMetadata, service.app.Config.Scopes, owner.String(),
		)
		if err != nil {
			return fmt.Errorf("send deletion OAuth authorization request: %w", err)
		}
		requestInfo.AccountDID = &identity.DID
		requestCtx := WithAccountDeletionAuthRequestAuthority(
			authCtx, owner, authority.Generation, authority.AuthEpoch, jobID, deviceID,
		)
		if err := service.store.SaveAuthRequestInfo(requestCtx, *requestInfo); err != nil {
			return err
		}
		params := url.Values{"client_id": {service.app.Config.ClientID}, "request_uri": {requestInfo.RequestURI}}
		redirectURL = strings.TrimSuffix(serverMetadata.AuthorizationEndpoint, "?") + "?" + params.Encode()
		return nil
	})
	return redirectURL, err
}

// CompleteCallback holds the owner fence from the durable exchange_started
// transition through initial credential persistence and AppView finalization.
// The authorization code is never retried.
func (service *OAuthFlowService) CompleteCallback(
	ctx context.Context,
	params url.Values,
	finalize OAuthCallbackFinalizer,
) error {
	operationCtx, cancel := oauthOperationContext(ctx, service.callbackOperationTimeout)
	defer cancel()
	ctx = operationCtx
	state := params.Get("state")
	if state == "" || finalize == nil {
		return ErrOAuthFlowInvalid
	}
	metadata, err := service.store.LoadAuthRequestMetadata(ctx, state)
	if err != nil || !metadata.valid() || metadata.RequestState != AuthRequestReady {
		return ErrOAuthFlowInvalid
	}
	return service.owners.WithExistingAuth(ctx, metadata.Owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		if authority.Generation != metadata.OwnerGeneration || authority.AuthEpoch != metadata.AuthEpoch {
			return ErrOAuthFlowInvalid
		}
		attemptID, err := service.store.BeginExchange(authCtx, state)
		if err != nil {
			return err
		}
		attempt := CallbackAttempt{
			State: state, AttemptID: attemptID, Owner: metadata.Owner,
			OwnerGeneration: metadata.OwnerGeneration, AuthEpoch: metadata.AuthEpoch,
			Purpose: metadata.Purpose,
		}
		callbackCtx := WithCallbackAttempt(authCtx, attempt)
		result, err := service.processInitialCallback(callbackCtx, params, metadata, attempt)
		if err != nil {
			return err
		}
		if err := finalize(callbackCtx, result); err != nil {
			cleanupErr := service.store.AbandonPendingSession(callbackCtx, attempt)
			return errors.Join(err, cleanupErr)
		}
		return nil
	})
}

func (service *OAuthFlowService) processInitialCallback(
	ctx context.Context,
	params url.Values,
	metadata AuthRequestMetadata,
	attempt CallbackAttempt,
) (OAuthCallbackResult, error) {
	info, err := service.store.GetAuthRequestInfo(ctx, attempt.State)
	if err != nil || info.State != attempt.State || info.AccountDID == nil || *info.AccountDID != metadata.Owner {
		_ = service.store.MarkExchangeFailed(ctx, attempt.State, attempt.AttemptID)
		return OAuthCallbackResult{}, ErrOAuthFlowInvalid
	}
	if callbackError := params.Get("error"); callbackError != "" {
		_ = service.store.MarkExchangeFailed(ctx, attempt.State, attempt.AttemptID)
		return OAuthCallbackResult{}, ErrOAuthFlowInvalid
	}
	issuer, authorizationCode := params.Get("iss"), params.Get("code")
	if issuer == "" || authorizationCode == "" || issuer != info.AuthServerURL {
		_ = service.store.MarkExchangeFailed(ctx, attempt.State, attempt.AttemptID)
		return OAuthCallbackResult{}, ErrOAuthFlowInvalid
	}
	tokenResponse, err := service.app.SendInitialTokenRequest(ctx, authorizationCode, *info)
	if err != nil {
		_ = service.store.MarkExchangeFailed(ctx, attempt.State, attempt.AttemptID)
		return OAuthCallbackResult{}, fmt.Errorf("initial OAuth token request: %w", err)
	}
	if tokenResponse.Subject != metadata.Owner.String() {
		_ = service.store.MarkExchangeAmbiguous(ctx, attempt.State, attempt.AttemptID)
		return OAuthCallbackResult{}, ErrOAuthFlowInvalid
	}
	identity, err := service.app.Dir.LookupDID(ctx, metadata.Owner)
	if err != nil || identity == nil || identity.Handle == "" || identity.PDSEndpoint() == "" {
		_ = service.store.MarkExchangeAmbiguous(ctx, attempt.State, attempt.AttemptID)
		return OAuthCallbackResult{}, ErrOAuthFlowInvalid
	}
	session := oauth.ClientSessionData{
		AccountDID: metadata.Owner, SessionID: attempt.State,
		HostURL: identity.PDSEndpoint(), AuthServerURL: info.AuthServerURL,
		AuthServerTokenEndpoint:      info.AuthServerTokenEndpoint,
		AuthServerRevocationEndpoint: info.AuthServerRevocationEndpoint,
		Scopes:                       strings.Fields(tokenResponse.Scope), AccessToken: tokenResponse.AccessToken,
		RefreshToken:        tokenResponse.RefreshToken,
		DPoPAuthServerNonce: info.DPoPAuthServerNonce, DPoPHostNonce: info.DPoPAuthServerNonce,
		DPoPPrivateKeyMultibase: info.DPoPPrivateKeyMultibase,
	}
	if err := service.store.SaveSession(ctx, session); err != nil {
		revocationErr := service.revokeInMemorySession(ctx, session)
		if revocationErr == nil {
			_ = service.store.MarkExchangeFailed(ctx, attempt.State, attempt.AttemptID)
		} else {
			_ = service.store.MarkExchangeAmbiguous(ctx, attempt.State, attempt.AttemptID)
		}
		return OAuthCallbackResult{}, fmt.Errorf("persist initial OAuth session: %w", err)
	}
	return OAuthCallbackResult{
		Session: session, Metadata: metadata, Attempt: attempt, Handle: identity.Handle,
	}, nil
}

func (service *OAuthFlowService) revokeInMemorySession(ctx context.Context, data oauth.ClientSessionData) error {
	privateKey, err := atcrypto.ParsePrivateMultibase(data.DPoPPrivateKeyMultibase)
	if err != nil {
		return err
	}
	session := oauth.ClientSession{
		Client: service.app.Client, Config: service.app.Config, Data: &data, DPoPPrivateKey: privateKey,
	}
	return session.RevokeSession(ctx)
}
