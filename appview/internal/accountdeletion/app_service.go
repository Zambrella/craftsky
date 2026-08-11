package accountdeletion

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
)

type OAuthFlowStarter interface {
	StartAuthFlow(context.Context, string) (string, error)
}

type AppServiceOptions struct {
	Pool      *pgxpool.Pool
	Store     *Store
	OAuth     OAuthFlowStarter
	Now       func() time.Time
	Random    io.Reader
	IntentTTL time.Duration
}

type AppService struct {
	pool      *pgxpool.Pool
	store     *Store
	oauth     OAuthFlowStarter
	now       func() time.Time
	random    io.Reader
	intentTTL time.Duration
}

func NewAppService(options AppServiceOptions) (*AppService, error) {
	if options.Pool == nil || options.Store == nil || options.OAuth == nil {
		return nil, errors.New("account deletion service dependencies are unavailable")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Random == nil {
		options.Random = rand.Reader
	}
	if options.IntentTTL <= 0 {
		options.IntentTTL = 10 * time.Minute
	}
	return &AppService{
		pool: options.Pool, store: options.Store, oauth: options.OAuth,
		now: options.Now, random: options.Random, intentTTL: options.IntentTTL,
	}, nil
}

func (service *AppService) CreateIntent(ctx context.Context, params CreateIntentParams) (IntentResult, error) {
	if params.Owner == "" {
		return IntentResult{}, errors.New("invalid account deletion intent scope")
	}
	var handle syntax.Handle
	if err := service.pool.QueryRow(ctx, `SELECT handle FROM atproto_identity_cache WHERE did=$1`, params.Owner).Scan(&handle); err != nil {
		return IntentResult{}, fmt.Errorf("resolve account deletion handle: %w", err)
	}
	now := service.now().UTC()
	jobID := uuid.New()
	expiresAt := now.Add(service.intentTTL)
	if err := service.store.CreateIntent(ctx, IntentRecord{
		JobID: jobID, Owner: params.Owner,
		ConfirmationHandleHash: HashSecret("@" + handle.String()),
		ExpiresAt:              expiresAt,
	}); err != nil {
		return IntentResult{}, err
	}
	authContext := auth.WithAccountDeletionAuthRequest(ctx, params.Owner, jobID)
	authURL, err := service.oauth.StartAuthFlow(authContext, handle.String())
	if err != nil {
		service.discardIntent(ctx, jobID, params.Owner)
		return IntentResult{}, fmt.Errorf("start account deletion OAuth: %w", err)
	}
	requestURI, err := deletionRequestURI(authURL)
	if err != nil {
		service.discardIntent(ctx, jobID, params.Owner)
		return IntentResult{}, err
	}
	persisted, err := service.deletionOAuthRequestPersisted(ctx, requestURI, jobID, params.Owner)
	if err != nil || !persisted {
		service.discardOAuthRequest(ctx, requestURI)
		service.discardIntent(ctx, jobID, params.Owner)
		if err == nil {
			err = errors.New("OAuth request metadata was not persisted atomically")
		}
		return IntentResult{}, fmt.Errorf("verify account deletion OAuth request: %w", err)
	}
	return IntentResult{JobID: jobID.String(), AuthURL: authURL, ExpiresAt: expiresAt}, nil
}

func (service *AppService) Accept(ctx context.Context, params AcceptParams) error {
	jobID, err := uuid.Parse(params.JobID)
	if err != nil {
		return ErrOperationNotFound
	}
	_, err = service.store.Accept(ctx, AcceptanceRequest{
		JobID: jobID, Owner: params.Owner,
		ReauthProof: params.ReauthProof, ConfirmationHandle: params.ConfirmationHandle,
	})
	return err
}

func (service *AppService) CancelIntent(ctx context.Context, rawJobID string, owner syntax.DID) error {
	jobID, err := uuid.Parse(rawJobID)
	if err != nil {
		return ErrOperationNotFound
	}
	return service.store.CancelIntent(ctx, jobID, owner)
}

func (service *AppService) PendingLogin(ctx context.Context, owner syntax.DID, sessionID, _ string) (auth.AccountDeletionPendingLogin, bool, error) {
	refreshed, err := service.store.RefreshBoundOAuthFromLogin(ctx, owner, sessionID)
	if err != nil || !refreshed {
		return auth.AccountDeletionPendingLogin{}, refreshed, err
	}
	return auth.AccountDeletionPendingLogin{}, true, nil
}

func (service *AppService) RequestForState(ctx context.Context, state string) (auth.AccountDeletionAuthRequest, bool, error) {
	var request auth.AccountDeletionAuthRequest
	var purpose auth.OAuthPurpose
	err := service.pool.QueryRow(ctx, `
		SELECT purpose,COALESCE(account_deletion_job_id::text,''),COALESCE(account_deletion_owner_did,'')
		FROM oauth_auth_requests WHERE state=$1
	`, state).Scan(&purpose, &request.JobID, &request.Owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return request, false, nil
	}
	if err != nil {
		return request, false, err
	}
	request.Purpose = purpose
	return request, purpose == auth.AccountDeletionOAuthPurpose, nil
}

func (service *AppService) Complete(ctx context.Context, request auth.AccountDeletionAuthRequest, did syntax.DID, sessionID string) (auth.AccountDeletionOAuthResult, error) {
	jobID, err := uuid.Parse(request.JobID)
	if err != nil || request.Purpose != auth.AccountDeletionOAuthPurpose || request.Owner != did || sessionID == "" {
		return auth.AccountDeletionOAuthResult{}, ErrReauthenticationRequired
	}
	proofBytes := make([]byte, 32)
	if _, err := io.ReadFull(service.random, proofBytes); err != nil {
		return auth.AccountDeletionOAuthResult{}, err
	}
	proof := base64.RawURLEncoding.EncodeToString(proofBytes)
	if err := service.store.CompleteReauthentication(ctx, jobID, did, sessionID, HashSecret(proof)); err != nil {
		return auth.AccountDeletionOAuthResult{}, err
	}
	return auth.AccountDeletionOAuthResult{JobID: jobID.String(), Proof: proof}, nil
}

func (service *AppService) Reject(ctx context.Context, did syntax.DID, sessionID string) error {
	_, err := service.pool.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, did, sessionID)
	return err
}

func (service *AppService) discardIntent(ctx context.Context, jobID uuid.UUID, owner syntax.DID) {
	_, _ = service.pool.Exec(ctx, `
		DELETE FROM oauth_auth_requests
		WHERE purpose='accountDeletion' AND account_deletion_owner_did=$1
		  AND account_deletion_job_id=$2
	`, owner, jobID)
	_, _ = service.pool.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1 AND owner_did=$2 AND state='intent'`, jobID, owner)
}

func (service *AppService) discardOAuthRequest(ctx context.Context, requestURI string) {
	if requestURI != "" {
		_, _ = service.pool.Exec(ctx, `DELETE FROM oauth_auth_requests WHERE data->>'request_uri'=$1`, requestURI)
	}
}

func (service *AppService) deletionOAuthRequestPersisted(ctx context.Context, requestURI string, jobID uuid.UUID, owner syntax.DID) (bool, error) {
	var persisted bool
	err := service.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM oauth_auth_requests
			WHERE data->>'request_uri'=$1 AND purpose='accountDeletion'
			  AND account_deletion_job_id=$2 AND account_deletion_owner_did=$3
		)
	`, requestURI, jobID, owner).Scan(&persisted)
	return persisted, err
}

func deletionRequestURI(authURL string) (string, error) {
	parsed, err := url.Parse(authURL)
	if err != nil {
		return "", err
	}
	requestURI := parsed.Query().Get("request_uri")
	if requestURI == "" {
		return "", errors.New("account deletion OAuth URL missing request_uri")
	}
	return requestURI, nil
}

var (
	_ Service                                = (*AppService)(nil)
	_ auth.AccountDeletionOAuthCallbacks     = (*AppService)(nil)
	_ auth.AccountDeletionPendingLoginPolicy = (*AppService)(nil)
)
