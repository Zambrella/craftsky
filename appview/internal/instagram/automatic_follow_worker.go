package instagram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
)

const (
	AutomaticFollowBatchDefault = 20
	AutomaticFollowBatchMax     = 100

	AutomaticFollowProviderAttempts = 5
)

type AutomaticFollowOperation struct {
	ID               uuid.UUID
	OwnerDID         syntax.DID
	TargetDID        syntax.DID
	ImportedUsername string
	Rkey             syntax.RecordKey
	LeaseToken       uuid.UUID
	AttemptCount     int
	CreatedAt        time.Time
}

func (AutomaticFollowOperation) String() string {
	return "Instagram automatic follow operation [REDACTED]"
}

func (operation AutomaticFollowOperation) GoString() string {
	return operation.String()
}

func (operation AutomaticFollowOperation) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, operation.String())
}

type AutomaticFollowOperationStore interface {
	ClaimBatch(context.Context, int, time.Time) ([]AutomaticFollowOperation, error)
	CompleteFollowed(context.Context, AutomaticFollowOperation, time.Time) error
	CompleteAlreadyFollowing(context.Context, AutomaticFollowOperation, time.Time) error
	Invalidate(context.Context, AutomaticFollowOperation, string, time.Time) error
	Retry(context.Context, AutomaticFollowOperation, string, time.Time) error
}

type BackgroundSessionSelector interface {
	Select(context.Context, syntax.DID) (string, error)
	Invalidate(context.Context, syntax.DID, string) error
}

type AutomaticFollowWriter interface {
	Write(context.Context, syntax.DID, syntax.DID, string, *syntax.RecordKey, time.Time) error
	HasDeterministicFollow(
		context.Context,
		syntax.DID,
		syntax.DID,
		string,
		syntax.RecordKey,
	) (bool, error)
}

type AutomaticFollowWorkerOptions struct {
	Store                 AutomaticFollowOperationStore
	Policy                InstagramSuggestionEligibilityPolicy
	Sessions              BackgroundSessionSelector
	Writer                AutomaticFollowWriter
	Membership            WebhookMembership
	MembershipInactivator WebhookMembershipInactivator
	Now                   func() time.Time

	MaxProviderAttempts int
}

type AutomaticFollowWorker struct {
	store       AutomaticFollowOperationStore
	policy      InstagramSuggestionEligibilityPolicy
	sessions    BackgroundSessionSelector
	writer      AutomaticFollowWriter
	membership  WebhookMembership
	inactivator WebhookMembershipInactivator
	now         func() time.Time

	maxProviderAttempts int
}

func NewAutomaticFollowWorker(options AutomaticFollowWorkerOptions) (*AutomaticFollowWorker, error) {
	if options.Store == nil || options.Policy == nil || options.Sessions == nil || options.Writer == nil {
		return nil, errors.New("automatic follow worker dependencies are required")
	}
	if (options.Membership == nil) != (options.MembershipInactivator == nil) {
		return nil, errors.New("automatic follow membership dependencies are incomplete")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.MaxProviderAttempts == 0 {
		options.MaxProviderAttempts = AutomaticFollowProviderAttempts
	}
	if options.MaxProviderAttempts < 1 ||
		options.MaxProviderAttempts > AutomaticFollowProviderAttempts {
		return nil, errors.New("invalid automatic follow provider-attempt limit")
	}
	return &AutomaticFollowWorker{
		store:       options.Store,
		policy:      options.Policy,
		sessions:    options.Sessions,
		writer:      options.Writer,
		membership:  options.Membership,
		inactivator: options.MembershipInactivator,
		now:         options.Now,

		maxProviderAttempts: options.MaxProviderAttempts,
	}, nil
}

func (w *AutomaticFollowWorker) ProcessBatch(ctx context.Context, limit int) (int, error) {
	if w == nil || limit < 1 {
		return 0, errors.New("invalid automatic follow batch")
	}
	now := w.now().UTC()
	operations, err := w.store.ClaimBatch(ctx, limit, now)
	if err != nil {
		return 0, fmt.Errorf("claim automatic follow operations: %w", err)
	}
	for _, operation := range operations {
		if err := w.process(ctx, operation, now); err != nil {
			return 0, err
		}
	}
	return len(operations), nil
}

func (w *AutomaticFollowWorker) process(
	ctx context.Context,
	operation AutomaticFollowOperation,
	now time.Time,
) error {
	if departed, err := w.inactivateDepartedMember(ctx, operation, now); departed || err != nil {
		return err
	}
	decision, err := w.policy.Evaluate(ctx, EligibilityAtAutomaticFollow, SuggestionEligibilityRequest{
		ImporterDID:      operation.OwnerDID,
		TargetDID:        operation.TargetDID,
		ImportedUsername: operation.ImportedUsername,
	})
	if err != nil {
		return w.store.Retry(ctx, operation, "eligibilityUnavailable", now)
	}
	if decision.Reason == EligibilityAlreadyFollowing {
		if operation.AttemptCount > 1 {
			return w.recoverCompletedWrite(ctx, operation, now)
		}
		return w.store.CompleteAlreadyFollowing(ctx, operation, now)
	}
	if decision.Reason == EligibilityMembership {
		if departed, err := w.inactivateDepartedMember(ctx, operation, now); departed || err != nil {
			return err
		}
		return w.store.Retry(ctx, operation, "membershipUnavailable", now)
	}
	if !decision.Eligible {
		return w.store.Invalidate(ctx, operation, string(decision.Reason), now)
	}

	for range w.maxProviderAttempts {
		sessionID, err := w.sessions.Select(ctx, operation.OwnerDID)
		if errors.Is(err, auth.ErrNoUsableBackgroundSession) {
			return w.store.Retry(ctx, operation, "ownerSessionUnavailable", now)
		}
		if err != nil {
			return fmt.Errorf("select automatic follow session: %w", err)
		}
		rkey := operation.Rkey
		err = w.writer.Write(
			ctx,
			operation.OwnerDID,
			operation.TargetDID,
			sessionID,
			&rkey,
			operation.CreatedAt,
		)
		if errors.Is(err, auth.ErrPDSSessionExpired) {
			if invalidateErr := w.sessions.Invalidate(ctx, operation.OwnerDID, sessionID); invalidateErr != nil {
				return fmt.Errorf("invalidate automatic follow session: %w", invalidateErr)
			}
			continue
		}
		if err != nil {
			return w.store.Retry(ctx, operation, "followWriteUnavailable", now)
		}
		return w.store.CompleteFollowed(ctx, operation, now)
	}
	return w.store.Retry(ctx, operation, "ownerSessionUnavailable", now)
}

func (w *AutomaticFollowWorker) inactivateDepartedMember(
	ctx context.Context,
	operation AutomaticFollowOperation,
	now time.Time,
) (bool, error) {
	if w.membership == nil {
		return false, nil
	}
	for _, did := range []syntax.DID{operation.OwnerDID, operation.TargetDID} {
		current, err := w.membership.IsCurrentMember(ctx, did)
		if err != nil {
			return true, w.store.Retry(
				ctx,
				operation,
				"membershipUnavailable",
				now,
			)
		}
		if current {
			continue
		}
		if err := w.inactivator.InactivateMembership(ctx, did); err != nil {
			return true, w.store.Retry(
				ctx,
				operation,
				"membershipInactivationUnavailable",
				now,
			)
		}
		return true, nil
	}
	return false, nil
}

func (w *AutomaticFollowWorker) recoverCompletedWrite(
	ctx context.Context,
	operation AutomaticFollowOperation,
	now time.Time,
) error {
	for range w.maxProviderAttempts {
		sessionID, err := w.sessions.Select(ctx, operation.OwnerDID)
		if errors.Is(err, auth.ErrNoUsableBackgroundSession) {
			return w.store.Retry(ctx, operation, "ownerSessionUnavailable", now)
		}
		if err != nil {
			return fmt.Errorf("select automatic follow recovery session: %w", err)
		}
		exists, err := w.writer.HasDeterministicFollow(
			ctx,
			operation.OwnerDID,
			operation.TargetDID,
			sessionID,
			operation.Rkey,
		)
		if errors.Is(err, auth.ErrPDSSessionExpired) {
			if invalidateErr := w.sessions.Invalidate(
				ctx,
				operation.OwnerDID,
				sessionID,
			); invalidateErr != nil {
				return fmt.Errorf(
					"invalidate automatic follow recovery session: %w",
					invalidateErr,
				)
			}
			continue
		}
		if err != nil {
			return w.store.Retry(ctx, operation, "followReadUnavailable", now)
		}
		if exists {
			return w.store.CompleteFollowed(ctx, operation, now)
		}
		return w.store.CompleteAlreadyFollowing(ctx, operation, now)
	}
	return w.store.Retry(ctx, operation, "ownerSessionUnavailable", now)
}
