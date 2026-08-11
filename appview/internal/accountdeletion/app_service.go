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
	Signer    *StatusCapabilitySigner
	OAuth     OAuthFlowStarter
	Now       func() time.Time
	Random    io.Reader
	IntentTTL time.Duration
	StatusTTL time.Duration
}

// AppService owns the narrow account-deletion request/status contract. It
// starts fresh OAuth but never exposes the resulting OAuth session to a
// device; only the lifecycle worker can resume the bound session.
type AppService struct {
	pool      *pgxpool.Pool
	store     *Store
	signer    *StatusCapabilitySigner
	oauth     OAuthFlowStarter
	now       func() time.Time
	random    io.Reader
	intentTTL time.Duration
	statusTTL time.Duration
}

func NewAppService(options AppServiceOptions) (*AppService, error) {
	if options.Pool == nil || options.Store == nil || options.Signer == nil || options.OAuth == nil {
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
	if options.StatusTTL <= 0 {
		options.StatusTTL = 30 * 24 * time.Hour
	}
	return &AppService{
		pool: options.Pool, store: options.Store, signer: options.Signer,
		oauth: options.OAuth, now: options.Now, random: options.Random,
		intentTTL: options.IntentTTL, statusTTL: options.StatusTTL,
	}, nil
}

func (service *AppService) CreateIntent(ctx context.Context, params CreateIntentParams) (IntentResult, error) {
	if params.Owner == "" || params.DeviceID == "" {
		return IntentResult{}, errors.New("invalid account deletion intent scope")
	}
	var handle syntax.Handle
	if err := service.pool.QueryRow(ctx, `SELECT handle FROM atproto_identity_cache WHERE did=$1`, params.Owner).Scan(&handle); err != nil {
		return IntentResult{}, fmt.Errorf("resolve account deletion handle: %w", err)
	}
	now := service.now().UTC()
	jobID := uuid.New()
	intentExpiresAt := now.Add(service.intentTTL)
	statusExpiresAt := now.Add(service.statusTTL)
	capability, err := service.signer.Generate(jobID, params.Owner, statusExpiresAt)
	if err != nil {
		return IntentResult{}, err
	}
	if err := service.store.CreateIntent(ctx, IntentRecord{
		JobID: jobID, Owner: params.Owner, DeviceID: params.DeviceID,
		StatusCapabilityHash:   capability.Hash,
		ConfirmationHandleHash: HashSecret("@" + handle.String()),
		ExpiresAt:              intentExpiresAt,
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
	return IntentResult{
		JobID: jobID.String(), StatusToken: capability.Token,
		AuthURL: authURL, ExpiresAt: intentExpiresAt,
	}, nil
}

func (service *AppService) Accept(ctx context.Context, params AcceptParams) (AcceptResult, error) {
	jobID, err := uuid.Parse(params.JobID)
	if err != nil {
		return AcceptResult{}, ErrOperationNotFound
	}
	operation, err := service.store.Accept(ctx, AcceptanceRequest{
		JobID: jobID, Owner: params.Owner,
		StatusCapability:   params.StatusCapability,
		ReauthProof:        params.ReauthProof,
		ConfirmationHandle: params.ConfirmationHandle,
	})
	if err != nil {
		return AcceptResult{}, err
	}
	return AcceptResult{JobID: operation.JobID.String(), Status: operation.Status, Phase: operation.Phase}, nil
}

func (service *AppService) CancelIntent(ctx context.Context, rawJobID string, owner syntax.DID, statusCapability string) error {
	jobID, err := uuid.Parse(rawJobID)
	if err != nil {
		return ErrOperationNotFound
	}
	return service.store.CancelIntent(ctx, jobID, owner, statusCapability)
}

func (service *AppService) Recover(ctx context.Context, formerBearer, deviceID string) (RecoveryResult, error) {
	if formerBearer == "" || deviceID == "" {
		return RecoveryResult{}, ErrRecoveryUnauthorized
	}
	now := service.now().UTC()
	var result RecoveryResult
	err := pgx.BeginFunc(ctx, service.pool, func(tx pgx.Tx) error {
		var owner syntax.DID
		err := tx.QueryRow(ctx, `
			SELECT recovery.job_id,recovery.owner_did,operation.state,COALESCE(operation.phase,'')
			FROM account_deletion_recovery_credentials recovery
			JOIN account_deletion_operations operation ON operation.id=recovery.job_id
			WHERE recovery.token_hash=$1 AND recovery.used_at IS NULL
			  AND (recovery.device_id IS NULL OR recovery.device_id=$2)
			FOR UPDATE OF recovery
		`, HashSecret(formerBearer), deviceID).Scan(&result.JobID, &owner, &result.Status, &result.Phase)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRecoveryUnauthorized
		}
		if err != nil {
			return err
		}
		jobID, err := uuid.Parse(result.JobID)
		if err != nil {
			return ErrRecoveryUnauthorized
		}
		capability, err := service.signer.Generate(jobID, owner, now.Add(service.statusTTL))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_status_credentials(token_hash,job_id,owner_did,device_id,expires_at)
			VALUES($1,$2,$3,$4,$5)
		`, capability.Hash, jobID, owner, deviceID, now.Add(service.statusTTL)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE account_deletion_recovery_credentials SET used_at=$2 WHERE token_hash=$1
		`, HashSecret(formerBearer), now); err != nil {
			return err
		}
		result.StatusToken = capability.Token
		return nil
	})
	if err != nil {
		return RecoveryResult{}, err
	}
	return result, nil
}

func (service *AppService) PendingLogin(ctx context.Context, owner syntax.DID, deviceID string) (auth.AccountDeletionPendingLogin, bool, error) {
	if deviceID == "" {
		return auth.AccountDeletionPendingLogin{}, false, ErrStatusUnauthorized
	}
	var (
		jobID  uuid.UUID
		handle syntax.Handle
	)
	err := service.pool.QueryRow(ctx, `
		SELECT operation.id,identity.handle
		FROM account_deletion_operations operation
		JOIN atproto_identity_cache identity ON identity.did=operation.owner_did
		WHERE operation.owner_did=$1 AND operation.state<>'intent'
	`, owner).Scan(&jobID, &handle)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.AccountDeletionPendingLogin{}, false, nil
	}
	if err != nil {
		return auth.AccountDeletionPendingLogin{}, false, err
	}
	expiresAt := service.now().UTC().Add(service.statusTTL)
	capability, err := service.signer.Generate(jobID, owner, expiresAt)
	if err != nil {
		return auth.AccountDeletionPendingLogin{}, false, err
	}
	if _, err := service.pool.Exec(ctx, `
		INSERT INTO account_deletion_status_credentials(token_hash,job_id,owner_did,device_id,expires_at)
		VALUES($1,$2,$3,$4,$5)
	`, capability.Hash, jobID, owner, deviceID, expiresAt); err != nil {
		return auth.AccountDeletionPendingLogin{}, false, err
	}
	return auth.AccountDeletionPendingLogin{
		JobID: jobID.String(), Owner: owner, Handle: handle, StatusToken: capability.Token,
	}, true, nil
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

func (service *AppService) AuthorizeStatusRoute(ctx context.Context, token string, jobID uuid.UUID, deviceID string, action StatusAction) (StatusGrant, error) {
	grant, err := service.signer.Verify(token)
	if err != nil || grant.JobID != jobID.String() {
		return StatusGrant{}, ErrStatusUnauthorized
	}
	return service.store.AuthorizeStatusCapability(ctx, service.signer, token, jobID, grant.Owner, deviceID, action)
}

func (service *AppService) GetStatus(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (DeletionStatusView, error) {
	operation, err := service.store.GetOperation(ctx, jobID, owner)
	if err == nil {
		return ProjectDeletionStatus(
			jobID.String(), operation.Status, operation.Phase,
			operation.Status == StatusNeedsAttention,
			operation.Status == StatusNeedsAttention && service.operationNeedsReauthentication(ctx, jobID, owner),
		), nil
	}
	if !errors.Is(err, ErrOperationNotFound) {
		return DeletionStatusView{}, err
	}
	var exists bool
	if err := service.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM account_deletion_audits WHERE job_id=$1 AND did=$2 AND expires_at>$3)
	`, jobID, owner, service.now().UTC()).Scan(&exists); err != nil || !exists {
		if err == nil {
			err = ErrOperationNotFound
		}
		return DeletionStatusView{}, err
	}
	return ProjectDeletionStatus(jobID.String(), StatusDeleted, PhaseNone, false, false), nil
}

func (service *AppService) Retry(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (DeletionStatusView, error) {
	if err := service.store.ManualRetry(ctx, jobID, owner); err != nil {
		return DeletionStatusView{}, err
	}
	return service.GetStatus(ctx, jobID, owner)
}

func (service *AppService) StartReauthentication(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (ReauthenticationStart, error) {
	if !service.operationNeedsReauthentication(ctx, jobID, owner) {
		return ReauthenticationStart{}, ErrReauthenticationRequired
	}
	var handle syntax.Handle
	if err := service.pool.QueryRow(ctx, `SELECT handle FROM atproto_identity_cache WHERE did=$1`, owner).Scan(&handle); err != nil {
		return ReauthenticationStart{}, ErrReauthenticationRequired
	}
	authContext := auth.WithAccountDeletionAuthRequest(ctx, owner, jobID)
	authURL, err := service.oauth.StartAuthFlow(authContext, handle.String())
	if err != nil {
		return ReauthenticationStart{}, fmt.Errorf("start replacement account deletion OAuth: %w", err)
	}
	requestURI, err := deletionRequestURI(authURL)
	if err != nil {
		return ReauthenticationStart{}, err
	}
	persisted, err := service.deletionOAuthRequestPersisted(ctx, requestURI, jobID, owner)
	if err != nil || !persisted {
		service.discardOAuthRequest(ctx, requestURI)
		if err == nil {
			err = errors.New("OAuth request metadata was not persisted atomically")
		}
		return ReauthenticationStart{}, fmt.Errorf("verify replacement account deletion OAuth request: %w", err)
	}
	return ReauthenticationStart{AuthURL: authURL, ExpiresAt: service.now().UTC().Add(service.intentTTL)}, nil
}

func (service *AppService) operationNeedsReauthentication(ctx context.Context, jobID uuid.UUID, owner syntax.DID) bool {
	var needed bool
	_ = service.pool.QueryRow(ctx, `
		SELECT COALESCE(error_category='reauthentication',false)
		FROM account_deletion_operations WHERE id=$1 AND owner_did=$2
	`, jobID, owner).Scan(&needed)
	return needed
}

func (service *AppService) discardIntent(ctx context.Context, jobID uuid.UUID, owner syntax.DID) {
	_, _ = service.pool.Exec(ctx, `
		DELETE FROM oauth_auth_requests
		WHERE purpose='accountDeletion'
		  AND account_deletion_owner_did=$1
		  AND account_deletion_job_id=$2
	`, owner, jobID)
	_, _ = service.pool.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1 AND owner_did=$2 AND state='intent'`, jobID, owner)
}

func (service *AppService) discardOAuthRequest(ctx context.Context, requestURI string) {
	if requestURI == "" {
		return
	}
	_, _ = service.pool.Exec(ctx, `DELETE FROM oauth_auth_requests WHERE data->>'request_uri'=$1`, requestURI)
}

func (service *AppService) deletionOAuthRequestPersisted(
	ctx context.Context,
	requestURI string,
	jobID uuid.UUID,
	owner syntax.DID,
) (bool, error) {
	var persisted bool
	err := service.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM oauth_auth_requests
			WHERE data->>'request_uri'=$1
			  AND purpose='accountDeletion'
			  AND account_deletion_job_id=$2
			  AND account_deletion_owner_did=$3
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
	_ Service                            = (*AppService)(nil)
	_ StatusRouteService                 = (*AppService)(nil)
	_ auth.AccountDeletionOAuthCallbacks = (*AppService)(nil)
)
