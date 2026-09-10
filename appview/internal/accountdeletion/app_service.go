package accountdeletion

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/subscriptions"
)

type DeletionOAuthFlowStarter interface {
	StartAccountDeletion(
		context.Context,
		syntax.DID,
		uuid.UUID,
		string,
	) (string, error)
}

type AppServiceOptions struct {
	Pool                 *pgxpool.Pool
	Store                *Store
	OAuth                DeletionOAuthFlowStarter
	Owners               *ownerlifecycle.Store
	Sessions             *auth.SessionLifecycleService
	OAuthStore           *auth.PostgresAuthStore
	DepartureParticipant ownerlifecycle.TransitionParticipant
	BillingDeletion      BillingDeletionParticipant
	BillingObserver      subscriptions.BillingObserver
	Now                  func() time.Time
	Random               io.Reader
	IntentTTL            time.Duration
}

type AppService struct {
	pool       *pgxpool.Pool
	store      *Store
	oauth      DeletionOAuthFlowStarter
	owners     *ownerlifecycle.Store
	sessions   *auth.SessionLifecycleService
	oauthStore *auth.PostgresAuthStore
	departure  ownerlifecycle.TransitionParticipant
	billing    BillingDeletionParticipant
	observer   subscriptions.BillingObserver
	now        func() time.Time
	random     io.Reader
	intentTTL  time.Duration
}

func NewAppService(options AppServiceOptions) (*AppService, error) {
	if options.Pool == nil || options.Store == nil || options.OAuth == nil ||
		options.Owners == nil || options.Sessions == nil || options.OAuthStore == nil ||
		options.DepartureParticipant == nil {
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
		owners: options.Owners, sessions: options.Sessions, oauthStore: options.OAuthStore,
		departure: options.DepartureParticipant,
		billing:   options.BillingDeletion,
		observer:  options.BillingObserver,
		now:       options.Now, random: options.Random, intentTTL: options.IntentTTL,
	}, nil
}

func (service *AppService) CreateIntent(ctx context.Context, params CreateIntentParams) (IntentResult, error) {
	if params.Owner == "" || params.DeviceID == "" {
		return IntentResult{}, errors.New("invalid account deletion intent scope")
	}
	now := service.now().UTC()
	jobID := uuid.New()
	expiresAt := now.Add(service.intentTTL)
	intent := IntentRecord{
		JobID: jobID, Owner: params.Owner,
		ConfirmationDIDHash: HashSecret(params.Owner.String()),
		ExpiresAt:           expiresAt,
	}
	current, err := service.owners.Get(ctx, params.Owner)
	if err != nil {
		return IntentResult{}, err
	}
	if current.State != ownerlifecycle.StateActive {
		return IntentResult{}, ErrDeletionAlreadyPending
	}
	intentParticipant := service.store.CreateIntentParticipant(intent)
	var billingOwner bool
	if _, err := service.owners.TransitionWith(ctx, ownerlifecycle.TransitionRequest{
		Owner: params.Owner, ExpectedGeneration: current.Generation,
		To: ownerlifecycle.StateDeletionPending, Reason: "accountDeletionIntent",
	}, func(
		participantCtx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if err := intentParticipant(participantCtx, tx, before, after); err != nil {
			return err
		}
		if service.billing != nil {
			billingOwner, err = service.billing.BeginDeletion(participantCtx, tx, params.Owner, now)
			if err != nil {
				return err
			}
		}
		return service.departure(participantCtx, tx, before, after)
	}); err != nil {
		return IntentResult{}, err
	}
	authURL, err := service.oauth.StartAccountDeletion(ctx, params.Owner, jobID, params.DeviceID)
	if err != nil {
		rollbackErr := service.cancelIntent(ctx, jobID, params.Owner, "accountDeletionStartFailed")
		if rollbackErr != nil {
			return IntentResult{}, errors.Join(fmt.Errorf("start account deletion OAuth: %w", err), rollbackErr)
		}
		return IntentResult{}, fmt.Errorf("start account deletion OAuth: %w", err)
	}
	result := IntentResult{
		JobID: jobID.String(), AuthURL: authURL,
		ConfirmationDID: params.Owner, ExpiresAt: expiresAt,
	}
	if billingOwner {
		result.Warning = &DeletionWarning{
			Code:    "provider_billing_not_canceled",
			Message: "Deleting your CraftSky account does not cancel provider billing and may end CraftSky access.",
		}
	}
	return result, nil
}

func (service *AppService) Accept(ctx context.Context, params AcceptParams) error {
	jobID, err := uuid.Parse(params.JobID)
	if err != nil {
		return ErrOperationNotFound
	}
	request := AcceptanceRequest{
		JobID: jobID, Owner: params.Owner,
		ReauthProof: params.ReauthProof, ConfirmationDID: params.ConfirmationDID,
	}
	binding, err := service.store.AcceptanceBinding(ctx, request)
	if err != nil {
		return err
	}
	current, err := service.owners.Get(ctx, params.Owner)
	if err != nil {
		return err
	}
	sessionParticipant := service.sessions.OwnerTransitionParticipant(
		&binding,
		service.store.AcceptParticipant(request, binding),
	)
	participant := sessionParticipant
	billingClosed := false
	if service.billing != nil {
		participant = func(participantCtx context.Context, tx pgx.Tx, before, after ownerlifecycle.Lifecycle) error {
			var err error
			billingClosed, err = service.billing.ConfirmDeletion(participantCtx, tx, params.Owner, service.now().UTC())
			if err != nil {
				return err
			}
			return sessionParticipant(participantCtx, tx, before, after)
		}
	}
	_, err = service.owners.TransitionWith(ctx, ownerlifecycle.TransitionRequest{
		Owner: params.Owner, ExpectedGeneration: current.Generation,
		To: ownerlifecycle.StateDeleting, Reason: "accountDeletionAccepted",
	}, participant)
	if service.observer != nil {
		switch {
		case err == nil && billingClosed:
			service.observer.ObserveBillingClosure(ctx, "closed")
		case errors.Is(err, ErrProviderBillingMustBeResolved):
			service.observer.ObserveBillingClosure(ctx, "blocked")
		}
	}
	return err
}

func (service *AppService) CancelIntent(ctx context.Context, rawJobID string, owner syntax.DID) error {
	jobID, err := uuid.Parse(rawJobID)
	if err != nil {
		return ErrOperationNotFound
	}
	return service.cancelIntent(ctx, jobID, owner, "accountDeletionCanceled")
}

func (service *AppService) ExpireIntent(ctx context.Context, jobID uuid.UUID, owner syntax.DID) error {
	if jobID == uuid.Nil || owner == "" {
		return ErrOperationNotFound
	}
	return service.cancelIntent(ctx, jobID, owner, "accountDeletionIntentExpired")
}

func (service *AppService) cancelIntent(
	ctx context.Context,
	jobID uuid.UUID,
	owner syntax.DID,
	reason string,
) error {
	for attempt := 0; attempt < 2; attempt++ {
		current, err := service.owners.Get(ctx, owner)
		if err != nil {
			return err
		}
		if current.State != ownerlifecycle.StateDeletionPending {
			return ErrPointOfNoReturn
		}
		target, err := service.store.CancellationTarget(ctx, owner)
		if err != nil {
			return err
		}
		participant := service.store.CancelIntentParticipant(jobID, owner)
		if target == ownerlifecycle.StateDeparted {
			participant = service.sessions.OwnerTransitionParticipant(nil, participant)
		}
		_, err = service.owners.TransitionWith(ctx, ownerlifecycle.TransitionRequest{
			Owner: owner, ExpectedGeneration: current.Generation,
			To: target, Reason: reason,
		}, participant)
		if !errors.Is(err, ownerlifecycle.ErrGenerationChanged) {
			return err
		}
	}
	return ownerlifecycle.ErrGenerationChanged
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
	return auth.AccountDeletionOAuthResult{}, ErrReauthenticationRequired
}

func (service *AppService) Reject(ctx context.Context, did syntax.DID, sessionID string) error {
	return ErrReauthenticationRequired
}

func (service *AppService) CompleteAttempt(
	ctx context.Context,
	request auth.AccountDeletionAuthRequest,
	attempt auth.CallbackAttempt,
) (auth.AccountDeletionOAuthResult, error) {
	jobID, err := uuid.Parse(request.JobID)
	if err != nil || request.Purpose != auth.AccountDeletionOAuthPurpose ||
		request.Owner != attempt.Owner || attempt.Purpose != auth.AccountDeletionOAuthPurpose ||
		attempt.State == "" {
		return auth.AccountDeletionOAuthResult{}, ErrReauthenticationRequired
	}
	credentialGeneration, err := service.store.ReauthenticationGeneration(ctx, jobID, attempt.Owner)
	if err != nil {
		return auth.AccountDeletionOAuthResult{}, err
	}
	proofBytes := make([]byte, 32)
	if _, err := io.ReadFull(service.random, proofBytes); err != nil {
		return auth.AccountDeletionOAuthResult{}, err
	}
	proof := base64.RawURLEncoding.EncodeToString(proofBytes)
	if err := service.oauthStore.BindDeletionCredential(
		ctx,
		attempt,
		jobID,
		credentialGeneration,
		service.store.BindReauthentication(
			jobID, attempt.Owner, attempt.State, credentialGeneration, HashSecret(proof),
		),
	); err != nil {
		return auth.AccountDeletionOAuthResult{}, err
	}
	return auth.AccountDeletionOAuthResult{JobID: jobID.String(), Proof: proof}, nil
}

func (service *AppService) CompleteAccepted(ctx context.Context, operation ClaimedOperation) error {
	current, err := service.owners.Get(ctx, operation.Owner)
	if err != nil {
		return err
	}
	if current.State != ownerlifecycle.StateDeleting {
		return ErrOperationNotFound
	}
	participant := service.sessions.OwnerTransitionParticipant(
		nil,
		service.store.CompleteParticipant(operation),
	)
	_, err = service.owners.TransitionWith(ctx, ownerlifecycle.TransitionRequest{
		Owner: operation.Owner, ExpectedGeneration: current.Generation,
		To: ownerlifecycle.StateDeparted, Reason: "accountDeletionCompleted",
	}, participant)
	return err
}

var (
	_ Service                                   = (*AppService)(nil)
	_ auth.AccountDeletionOAuthCallbacks        = (*AppService)(nil)
	_ auth.AccountDeletionOAuthAttemptCallbacks = (*AppService)(nil)
	_ auth.AccountDeletionPendingLoginPolicy    = (*AppService)(nil)
)
