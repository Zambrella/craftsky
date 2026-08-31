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
	App                        *oauth.ClientApp
	Store                      *PostgresAuthStore
	Owners                     *ownerlifecycle.Store
	StartOperationTimeout      time.Duration
	CallbackOperationTimeout   time.Duration
	DeletionRequests           DeletionOAuthRequestVerifier
	RegistrationProviderOrigin string
	RegistrationOAuth          *RegistrationOAuthAdapter
}

type OAuthFlowService struct {
	app                        *oauth.ClientApp
	store                      *PostgresAuthStore
	owners                     *ownerlifecycle.Store
	startOperationTimeout      time.Duration
	callbackOperationTimeout   time.Duration
	deletionRequests           DeletionOAuthRequestVerifier
	registrationProviderOrigin string
	registrationOAuth          *RegistrationOAuthAdapter
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
		deletionRequests:           options.DeletionRequests,
		registrationProviderOrigin: options.RegistrationProviderOrigin,
		registrationOAuth:          options.RegistrationOAuth,
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
	StartRegistration(context.Context, HandoffMode, string, string) (string, error)
	CompleteCallback(context.Context, url.Values, OAuthCallbackFinalizer) error
}

var _ OAuthFlowCoordinator = (*OAuthFlowService)(nil)

func (service *OAuthFlowService) StartRegistration(
	ctx context.Context,
	mode HandoffMode,
	loopbackURI string,
	deviceID string,
) (string, error) {
	operationCtx, cancel := oauthOperationContext(ctx, service.startOperationTimeout)
	defer cancel()
	ctx = operationCtx
	if service.registrationOAuth == nil || service.registrationProviderOrigin == "" || deviceID == "" ||
		(mode != HandoffVerifiedLink && mode != HandoffLoopback && mode != HandoffDevScheme) ||
		(mode == HandoffLoopback) != (loopbackURI != "") {
		return "", ErrOAuthFlowInvalid
	}
	reservation, err := service.store.ReserveAuthRequestCapacity(ctx)
	if err != nil {
		return "", err
	}
	persisted := false
	defer func() {
		if !persisted {
			releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer releaseCancel()
			_ = service.store.ReleaseAuthRequestCapacity(releaseCtx, reservation.ID)
		}
	}()
	metadata, err := service.registrationOAuth.ResolveAuthorizationServer(ctx, service.registrationProviderOrigin)
	if err != nil {
		return "", err
	}
	requestInfo, err := service.registrationOAuth.SendAuthorizationRequest(ctx, metadata, service.app.Config.Scopes)
	if err != nil {
		return "", err
	}
	requestCtx := WithRegistrationAuthRequest(
		ctx, service.registrationProviderOrigin, metadata.Issuer, mode, deviceID, loopbackURI,
	)
	if err := service.store.SaveRegistrationAuthRequest(requestCtx, reservation.ID, *requestInfo); err != nil {
		return "", err
	}
	persisted = true
	params := url.Values{"client_id": {service.app.Config.ClientID}, "request_uri": {requestInfo.RequestURI}}
	return strings.TrimSuffix(metadata.AuthorizationEndpoint, "?") + "?" + params.Encode(), nil
}

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
		(mode != HandoffVerifiedLink && mode != HandoffLoopback && mode != HandoffDevScheme) ||
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
	state, ok := exactOAuthCallbackParam(params, "state")
	if !ok || state == "" || finalize == nil {
		return ErrOAuthFlowInvalid
	}
	metadata, err := service.store.LoadAuthRequestMetadata(ctx, state)
	if err != nil || !metadata.valid() || metadata.RequestState != AuthRequestReady {
		return ErrOAuthFlowInvalid
	}
	if metadata.Purpose == RegistrationOAuthPurpose {
		return service.completeRegistrationCallback(ctx, params, metadata, finalize)
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

func (service *OAuthFlowService) completeRegistrationCallback(
	ctx context.Context,
	params url.Values,
	metadata AuthRequestMetadata,
	finalize OAuthCallbackFinalizer,
) error {
	issuerValues, codeValues := params["iss"], params["code"]
	callbackErrors := params["error"]
	if len(issuerValues) > 1 || len(codeValues) > 1 || len(callbackErrors) > 1 {
		return ErrOAuthFlowInvalid
	}
	issuer, authorizationCode := params.Get("iss"), params.Get("code")
	attemptID, err := service.store.BeginRegistrationExchange(ctx, params.Get("state"))
	if err != nil {
		return err
	}
	trustedFailure := func(code RegistrationFailureCode, cause error) error {
		failure, buildErr := newTrustedRegistrationFailure(metadata, code, cause)
		if buildErr != nil {
			return cause
		}
		return failure
	}
	state := params.Get("state")
	info, err := service.store.GetRegistrationAuthRequestInfo(ctx, state)
	if err != nil || info.State != state || info.AuthServerURL != metadata.RegistrationIssuer {
		service.finishRegistrationExchange(ctx, state, attemptID, AuthRequestExchangeFailed)
		return trustedFailure(RegistrationFailureIncomplete, ErrOAuthFlowInvalid)
	}
	if len(callbackErrors) == 1 {
		service.finishRegistrationExchange(ctx, state, attemptID, AuthRequestExchangeFailed)
		code := RegistrationFailureIncomplete
		if callbackErrors[0] == "access_denied" {
			code = RegistrationFailureCanceled
		}
		return trustedFailure(code, ErrOAuthFlowInvalid)
	}
	if issuer == "" || authorizationCode == "" || issuer != metadata.RegistrationIssuer {
		service.finishRegistrationExchange(ctx, state, attemptID, AuthRequestExchangeFailed)
		return trustedFailure(RegistrationFailureIncomplete, ErrOAuthFlowInvalid)
	}
	tokenResult, err := service.registrationOAuth.SendInitialTokenRequest(ctx, authorizationCode, *info)
	if err != nil {
		if tokenResult.cleanupTokens != nil {
			session := oauth.ClientSessionData{
				SessionID: state, HostURL: metadata.RegistrationProviderOrigin,
				AuthServerURL:                info.AuthServerURL,
				AuthServerTokenEndpoint:      info.AuthServerTokenEndpoint,
				AuthServerRevocationEndpoint: info.AuthServerRevocationEndpoint,
				AccessToken:                  tokenResult.cleanupTokens.accessToken,
				RefreshToken:                 tokenResult.cleanupTokens.refreshToken,
				DPoPAuthServerNonce:          info.DPoPAuthServerNonce,
				DPoPHostNonce:                info.DPoPAuthServerNonce,
				DPoPPrivateKeyMultibase:      info.DPoPPrivateKeyMultibase,
			}
			if cleanupErr := service.quarantineRegistrationForCleanup(ctx, state, attemptID, session); cleanupErr != nil {
				return trustedFailure(RegistrationFailureIncomplete, cleanupErr)
			}
			return trustedFailure(RegistrationFailureIncomplete, err)
		}
		finalState := AuthRequestExchangeFailed
		var oauthFailure *RegistrationOAuthError
		if ctx.Err() != nil || (errors.As(err, &oauthFailure) && oauthFailure.IssuanceUncertain) {
			finalState = AuthRequestExchangeAmbiguous
		}
		service.finishRegistrationExchange(ctx, state, attemptID, finalState)
		cause := fmt.Errorf("initial registration OAuth token request: %w", err)
		return trustedFailure(registrationCallbackFailureCode(err), cause)
	}
	tokenResponse := tokenResult.response
	candidate, candidateErr := syntax.ParseDID(tokenResponse.Subject)
	session := oauth.ClientSessionData{
		AccountDID: syntax.DID(tokenResponse.Subject), SessionID: state,
		HostURL: metadata.RegistrationProviderOrigin, AuthServerURL: info.AuthServerURL,
		AuthServerTokenEndpoint:      info.AuthServerTokenEndpoint,
		AuthServerRevocationEndpoint: info.AuthServerRevocationEndpoint,
		Scopes:                       strings.Fields(tokenResponse.Scope),
		AccessToken:                  tokenResponse.AccessToken,
		RefreshToken:                 tokenResponse.RefreshToken,
		DPoPAuthServerNonce:          info.DPoPAuthServerNonce,
		DPoPHostNonce:                info.DPoPAuthServerNonce,
		DPoPPrivateKeyMultibase:      info.DPoPPrivateKeyMultibase,
	}
	tokenValid := candidateErr == nil && tokenResponse.AccessToken != "" && tokenResponse.RefreshToken != "" &&
		validRegistrationTokenScopes(tokenResponse.Scope, service.app.Config.Scopes)
	if !tokenValid {
		if tokenResponse.AccessToken == "" && tokenResponse.RefreshToken == "" {
			service.finishRegistrationExchange(ctx, state, attemptID, AuthRequestExchangeAmbiguous)
			return trustedFailure(RegistrationFailureIncomplete, ErrOAuthFlowInvalid)
		}
		if err := service.quarantineRegistrationForCleanup(ctx, state, attemptID, session); err != nil {
			return trustedFailure(RegistrationFailureIncomplete, err)
		}
		return trustedFailure(RegistrationFailureIncomplete, ErrOAuthFlowInvalid)
	}
	session.AccountDID = candidate
	if err := service.store.QuarantineRegistrationCredential(
		ctx, state, attemptID, session, time.Now().Add(service.callbackOperationTimeout),
	); err != nil {
		service.disposeUnquarantinedRegistrationCredential(ctx, state, attemptID, session)
		return trustedFailure(RegistrationFailureIncomplete, err)
	}
	resolved, err := service.app.Dir.LookupDID(ctx, candidate)
	if err != nil || resolved == nil || resolved.DID != candidate || resolved.Handle == "" || resolved.PDSEndpoint() == "" {
		service.markRegistrationCredentialForCleanup(ctx, state, attemptID)
		return trustedFailure(RegistrationFailureIncomplete, ErrOAuthFlowInvalid)
	}
	authoritativeIssuer, err := service.app.Resolver.ResolveAuthServerURL(ctx, resolved.PDSEndpoint())
	if err != nil || authoritativeIssuer != metadata.RegistrationIssuer {
		service.markRegistrationCredentialForCleanup(ctx, state, attemptID)
		return trustedFailure(RegistrationFailureIncomplete, ErrOAuthFlowInvalid)
	}
	session.HostURL = resolved.PDSEndpoint()
	err = service.owners.WithOnboardingAuth(ctx, candidate, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		if authority.State != ownerlifecycle.StateDeparted && authority.State != ownerlifecycle.StateActive {
			return ErrOAuthOwnerIneligible
		}
		attempt := CallbackAttempt{
			State: state, AttemptID: attemptID, Owner: candidate,
			OwnerGeneration: authority.Generation, AuthEpoch: authority.AuthEpoch,
			Purpose: RegistrationOAuthPurpose,
		}
		if err := service.store.BindRegistrationAuthority(authCtx, RegistrationAuthorityBinding{
			State: state, AttemptID: attemptID, Owner: candidate,
			OwnerGeneration: authority.Generation, AuthEpoch: authority.AuthEpoch,
			Session: session,
		}); err != nil {
			return err
		}
		metadata.Owner = candidate
		metadata.OwnerGeneration = authority.Generation
		metadata.AuthEpoch = authority.AuthEpoch
		metadata.RequestState = AuthRequestExchangeStarted
		metadata.ExchangeAttemptID = attemptID
		callbackCtx := WithCallbackAttempt(authCtx, attempt)
		if err := finalize(callbackCtx, OAuthCallbackResult{
			Session: session, Metadata: metadata, Attempt: attempt, Handle: resolved.Handle,
		}); err != nil {
			cleanupErr := service.store.AbandonPendingSession(callbackCtx, attempt)
			return errors.Join(err, cleanupErr)
		}
		return nil
	})
	if err != nil {
		service.markRegistrationCredentialForCleanup(ctx, state, attemptID)
		return trustedFailure(RegistrationFailureIncomplete, err)
	}
	return nil
}

func registrationCallbackFailureCode(err error) RegistrationFailureCode {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return RegistrationFailureProviderUnavailable
	}
	var oauthFailure *RegistrationOAuthError
	if errors.As(err, &oauthFailure) && oauthFailure.Code == RegistrationOAuthProviderUnavailable {
		return RegistrationFailureProviderUnavailable
	}
	return RegistrationFailureIncomplete
}

func exactOAuthCallbackParam(params url.Values, key string) (string, bool) {
	values, ok := params[key]
	if !ok || len(values) != 1 {
		return "", false
	}
	return values[0], true
}

func (service *OAuthFlowService) quarantineRegistrationForCleanup(
	ctx context.Context,
	state string,
	attemptID uuid.UUID,
	session oauth.ClientSessionData,
) error {
	if err := service.store.QuarantineRegistrationCredential(
		ctx, state, attemptID, session, time.Now().Add(service.callbackOperationTimeout),
	); err != nil {
		service.disposeUnquarantinedRegistrationCredential(ctx, state, attemptID, session)
		return err
	}
	cleanupCtx, cancel := service.registrationFinalizationContext(ctx)
	defer cancel()
	return service.store.MarkRegistrationCredentialForCleanup(cleanupCtx, state, attemptID)
}

func (service *OAuthFlowService) finishRegistrationExchange(
	ctx context.Context,
	state string,
	attemptID uuid.UUID,
	finalState AuthRequestState,
) {
	finalCtx, cancel := service.registrationFinalizationContext(ctx)
	defer cancel()
	_ = service.store.FinishRegistrationExchange(finalCtx, state, attemptID, finalState)
}

func (service *OAuthFlowService) markRegistrationCredentialForCleanup(
	ctx context.Context,
	state string,
	attemptID uuid.UUID,
) {
	finalCtx, cancel := service.registrationFinalizationContext(ctx)
	defer cancel()
	_ = service.store.MarkRegistrationCredentialForCleanup(finalCtx, state, attemptID)
}

func (service *OAuthFlowService) registrationFinalizationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), service.callbackOperationTimeout)
}

func (service *OAuthFlowService) disposeUnquarantinedRegistrationCredential(
	ctx context.Context,
	state string,
	attemptID uuid.UUID,
	session oauth.ClientSessionData,
) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), service.callbackOperationTimeout)
	defer cancel()
	finalState := AuthRequestExchangeAmbiguous
	if service.revokeInMemorySession(cleanupCtx, session) == nil {
		finalState = AuthRequestExchangeFailed
	}
	_ = service.store.FinishRegistrationExchange(cleanupCtx, state, attemptID, finalState)
}

func validRegistrationTokenScopes(raw string, required []string) bool {
	fields := strings.Split(raw, " ")
	if len(fields) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(fields))
	for _, scope := range fields {
		if scope == "" {
			return false
		}
		for index := 0; index < len(scope); index++ {
			character := scope[index]
			if character != 0x21 && (character < 0x23 || character > 0x5b) && (character < 0x5d || character > 0x7e) {
				return false
			}
		}
		if _, duplicate := seen[scope]; duplicate {
			return false
		}
		seen[scope] = struct{}{}
	}
	for _, scope := range required {
		if _, ok := seen[scope]; !ok {
			return false
		}
	}
	_, hasATProto := seen["atproto"]
	return hasATProto
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
