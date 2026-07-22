package instagram

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/integrations/instagrammeta"
)

const (
	WebhookWorkerCount          = 4
	WebhookMaxReplyWindow       = 24 * time.Hour
	WebhookInvalidIGSIDLimit    = 10
	WebhookMetaLookupIGSIDLimit = 5
)

type WebhookWorkQueue interface {
	ClaimWebhookWork(ctx context.Context, limit int, now time.Time) ([]WebhookWork, error)
	CompleteWebhookWork(ctx context.Context, id, leaseToken uuid.UUID, now time.Time, reason WebhookTerminalReason) error
	IgnoreWebhookWork(ctx context.Context, id, leaseToken uuid.UUID, now time.Time, reason WebhookTerminalReason) error
	FailWebhookWork(ctx context.Context, id, leaseToken uuid.UUID, now time.Time, reason WebhookTerminalReason) error
	RetryWebhookWork(ctx context.Context, id, leaseToken uuid.UUID, nextAttemptAt, now time.Time) error
}

type WebhookRedemptionRequest struct {
	WorkID          uuid.UUID       `json:"-"`
	LeaseToken      uuid.UUID       `json:"-"`
	ChallengeDigest ChallengeDigest `json:"-"`
	SenderIGSID     string          `json:"-"`
	Now             time.Time       `json:"-"`
}

func (WebhookRedemptionRequest) String() string {
	return "Instagram webhook redemption request [REDACTED]"
}

func (WebhookRedemptionRequest) GoString() string {
	return "Instagram webhook redemption request [REDACTED]"
}

type WebhookRedemption struct {
	AttemptID uuid.UUID  `json:"-"`
	OwnerDID  syntax.DID `json:"-"`
}

func (WebhookRedemption) String() string {
	return "Instagram webhook redemption [REDACTED]"
}

func (WebhookRedemption) GoString() string {
	return "Instagram webhook redemption [REDACTED]"
}

type WebhookRedeemer interface {
	// RedeemWebhookChallenge must be idempotent for WorkID. A provider retry
	// occurs after challenge redemption, so the same durable work may call this
	// method again after a crash or transient lookup failure.
	RedeemWebhookChallenge(ctx context.Context, request WebhookRedemptionRequest) (WebhookRedemption, error)
	SetWebhookCandidate(ctx context.Context, attemptID uuid.UUID, username string, now time.Time) error
	InactivateWebhookOwner(ctx context.Context, attemptID uuid.UUID, owner syntax.DID, now time.Time) error
	RejectWebhookAttempt(ctx context.Context, attemptID uuid.UUID, retryCode AttemptRetryCode, now time.Time) error
}

type WebhookMembership interface {
	IsCurrentMember(ctx context.Context, did syntax.DID) (bool, error)
}

type WebhookMembershipInactivator interface {
	InactivateMembership(ctx context.Context, did syntax.DID) error
}

type WebhookIdentifierLimiter interface {
	AllowIdentifier(context.Context, RateLimitScope, []byte, time.Duration, int) (RateLimitDecision, error)
}

var _ WebhookMembership = (*MembershipStore)(nil)

type WebhookWorkerOptions struct {
	BatchSize                  int                          `json:"-"`
	Now                        func() time.Time             `json:"-"`
	ReplyText                  string                       `json:"-"`
	ReplyWindow                time.Duration                `json:"-"`
	RateLimiter                WebhookIdentifierLimiter     `json:"-"`
	InvalidIGSIDPer15Minutes   int                          `json:"-"`
	MetaLookupsPerIGSIDPerHour int                          `json:"-"`
	MembershipInactivator      WebhookMembershipInactivator `json:"-"`
	RetryPolicy                WebhookRetryPolicy           `json:"-"`
}

func (WebhookWorkerOptions) String() string {
	return "Instagram webhook worker options [REDACTED]"
}

func (WebhookWorkerOptions) GoString() string {
	return "Instagram webhook worker options [REDACTED]"
}

type WebhookWorker struct {
	queue       WebhookWorkQueue
	redeemer    WebhookRedeemer
	membership  WebhookMembership
	inactivator WebhookMembershipInactivator
	meta        instagrammeta.Client
	options     WebhookWorkerOptions
}

func NewWebhookWorker(queue WebhookWorkQueue, redeemer WebhookRedeemer, membership WebhookMembership, meta instagrammeta.Client, options WebhookWorkerOptions) (*WebhookWorker, error) {
	if queue == nil || redeemer == nil || membership == nil || meta == nil {
		return nil, errors.New("Instagram webhook worker dependencies are incomplete")
	}
	inactivator := options.MembershipInactivator
	if inactivator == nil {
		inactivator, _ = membership.(WebhookMembershipInactivator)
	}
	if inactivator == nil {
		return nil, errors.New("Instagram webhook worker membership inactivator is incomplete")
	}
	if options.BatchSize == 0 {
		options.BatchSize = 1
	}
	if options.BatchSize < 0 || options.BatchSize > instagrammeta.MaxSupportedEvents {
		return nil, errors.New("Instagram webhook worker batch size is invalid")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.RetryPolicy == (WebhookRetryPolicy{}) {
		options.RetryPolicy = DefaultWebhookRetryPolicy()
	}
	if !options.RetryPolicy.valid() {
		return nil, errors.New("Instagram webhook worker retry policy is invalid")
	}
	if options.RateLimiter != nil {
		if options.InvalidIGSIDPer15Minutes == 0 {
			options.InvalidIGSIDPer15Minutes = WebhookInvalidIGSIDLimit
		}
		if options.MetaLookupsPerIGSIDPerHour == 0 {
			options.MetaLookupsPerIGSIDPerHour = WebhookMetaLookupIGSIDLimit
		}
		if options.InvalidIGSIDPer15Minutes < 1 || options.InvalidIGSIDPer15Minutes > WebhookInvalidIGSIDLimit ||
			options.MetaLookupsPerIGSIDPerHour < 1 || options.MetaLookupsPerIGSIDPerHour > WebhookMetaLookupIGSIDLimit {
			return nil, errors.New("Instagram webhook identifier limits are invalid")
		}
	}
	if options.ReplyText != "" {
		if options.ReplyWindow == 0 {
			options.ReplyWindow = WebhookMaxReplyWindow
		}
		if options.ReplyWindow < 0 || options.ReplyWindow > WebhookMaxReplyWindow {
			return nil, errors.New("Instagram webhook reply window is invalid")
		}
	}
	return &WebhookWorker{
		queue:       queue,
		redeemer:    redeemer,
		membership:  membership,
		inactivator: inactivator,
		meta:        meta,
		options:     options,
	}, nil
}

func (*WebhookWorker) String() string {
	return "Instagram webhook worker [REDACTED]"
}

func (*WebhookWorker) GoString() string {
	return "Instagram webhook worker [REDACTED]"
}

func (w *WebhookWorker) ProcessBatch(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	now := w.options.Now().UTC()
	claimed, err := w.queue.ClaimWebhookWork(ctx, w.options.BatchSize, now)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, item := range claimed {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		if err := w.processOne(ctx, item); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (w *WebhookWorker) processOne(ctx context.Context, item WebhookWork) error {
	now, err := w.liveWorkTime(ctx, item)
	if err != nil {
		return err
	}
	if item.ProcessingStartedAt.IsZero() || !now.Before(item.ProcessingStartedAt.Add(w.options.RetryPolicy.MaxProcessingAge)) {
		return w.queue.FailWebhookWork(ctx, item.ID, item.LeaseToken, now, WebhookReasonMaxAge)
	}
	digest := ChallengeDigest{Version: item.ChallengeDigest.Version, Value: item.ChallengeDigest.Value}
	redemption, redeemErr := w.redeemer.RedeemWebhookChallenge(ctx, WebhookRedemptionRequest{
		WorkID:          item.ID,
		LeaseToken:      item.LeaseToken,
		ChallengeDigest: digest,
		SenderIGSID:     item.SenderIGSID,
		Now:             now,
	})
	now, err = w.liveWorkTime(ctx, item)
	if err != nil {
		return err
	}
	if redeemErr != nil {
		if errors.Is(redeemErr, context.Canceled) {
			return context.Canceled
		}
		if errors.Is(redeemErr, ErrInstagramResourceNotFound) || errors.Is(redeemErr, ErrInstagramStateTransition) {
			if w.options.RateLimiter != nil {
				decision, limitErr := w.options.RateLimiter.AllowIdentifier(
					ctx,
					RateLimitInvalidRedemptionIGSID,
					[]byte(item.SenderIGSID),
					15*time.Minute,
					w.options.InvalidIGSIDPer15Minutes,
				)
				if limitErr != nil {
					return w.retryWork(ctx, item, nil, now, 0)
				}
				if !decision.Allowed {
					return w.queue.IgnoreWebhookWork(ctx, item.ID, item.LeaseToken, now, WebhookReasonRateLimited)
				}
			}
			return w.queue.IgnoreWebhookWork(ctx, item.ID, item.LeaseToken, now, WebhookReasonChallengeUnavailable)
		}
		return w.retryWork(ctx, item, nil, now, 0)
	}
	if !now.Before(item.ProcessingStartedAt.Add(w.options.RetryPolicy.MaxProcessingAge)) {
		return w.retryWork(ctx, item, &redemption, now, 0)
	}
	current, membershipErr := w.membership.IsCurrentMember(ctx, redemption.OwnerDID)
	now, err = w.liveWorkTime(ctx, item)
	if err != nil {
		return err
	}
	if membershipErr != nil {
		if errors.Is(membershipErr, context.Canceled) {
			return context.Canceled
		}
		return w.retryWork(ctx, item, &redemption, now, 0)
	}
	if !now.Before(item.ProcessingStartedAt.Add(w.options.RetryPolicy.MaxProcessingAge)) {
		return w.retryWork(ctx, item, &redemption, now, 0)
	}
	if !current {
		if inactivateErr := w.inactivator.InactivateMembership(ctx, redemption.OwnerDID); inactivateErr != nil {
			if errors.Is(inactivateErr, context.Canceled) {
				return context.Canceled
			}
			return w.retryWork(ctx, item, &redemption, now, 0)
		}
		inactivateErr := w.redeemer.InactivateWebhookOwner(ctx, redemption.AttemptID, redemption.OwnerDID, now)
		now, err = w.liveWorkTime(ctx, item)
		if err != nil {
			return err
		}
		if inactivateErr != nil {
			if errors.Is(inactivateErr, context.Canceled) {
				return context.Canceled
			}
			return w.retryWork(ctx, item, &redemption, now, 0)
		}
		return w.queue.IgnoreWebhookWork(ctx, item.ID, item.LeaseToken, now, WebhookReasonMembershipInactive)
	}
	if w.options.RateLimiter != nil {
		decision, limitErr := w.options.RateLimiter.AllowIdentifier(
			ctx,
			RateLimitMetaLookupIGSID,
			[]byte(item.SenderIGSID),
			time.Hour,
			w.options.MetaLookupsPerIGSIDPerHour,
		)
		now, err = w.liveWorkTime(ctx, item)
		if err != nil {
			return err
		}
		if limitErr != nil {
			return w.retryWork(ctx, item, &redemption, now, 0)
		}
		if !decision.Allowed {
			return w.retryWork(ctx, item, &redemption, now, decision.RetryAfter)
		}
	}
	username, lookupErr := w.meta.LookupUsername(ctx, item.SenderIGSID)
	now, err = w.liveWorkTime(ctx, item)
	if err != nil {
		return err
	}
	if lookupErr != nil {
		if errors.Is(lookupErr, context.Canceled) {
			return context.Canceled
		}
		kind, retryAfter, classified := classifyMetaFailure(lookupErr)
		if !classified || kind == instagrammeta.ProviderErrorTransient || kind == instagrammeta.ProviderErrorRateLimited {
			return w.retryWork(ctx, item, &redemption, now, retryAfter)
		}
		return w.rejectProviderFailure(ctx, item, redemption, kind, now)
	}
	if !now.Before(item.ProcessingStartedAt.Add(w.options.RetryPolicy.MaxProcessingAge)) {
		return w.retryWork(ctx, item, &redemption, now, 0)
	}
	candidateErr := w.redeemer.SetWebhookCandidate(ctx, redemption.AttemptID, username, now)
	now, err = w.liveWorkTime(ctx, item)
	if err != nil {
		return err
	}
	if candidateErr != nil {
		if errors.Is(candidateErr, context.Canceled) {
			return context.Canceled
		}
		if errors.Is(candidateErr, ErrInvalidInstagramUsername) {
			if rejectErr := w.redeemer.RejectWebhookAttempt(ctx, redemption.AttemptID, RetryInvalidProfileResponse, now); rejectErr != nil {
				return rejectErr
			}
			return w.queue.FailWebhookWork(ctx, item.ID, item.LeaseToken, now, WebhookReasonInvalidProfile)
		}
		if errors.Is(candidateErr, ErrInstagramResourceNotFound) || errors.Is(candidateErr, ErrInstagramStateTransition) {
			return w.queue.IgnoreWebhookWork(ctx, item.ID, item.LeaseToken, now, WebhookReasonChallengeUnavailable)
		}
		return w.retryWork(ctx, item, &redemption, now, 0)
	}
	if err := w.queue.CompleteWebhookWork(ctx, item.ID, item.LeaseToken, now, WebhookReasonProcessed); err != nil {
		return err
	}
	replyNow := w.options.Now().UTC()
	if ctx.Err() == nil && w.options.ReplyText != "" && !replyNow.Before(item.EventAt) && replyNow.Before(item.EventAt.Add(w.options.ReplyWindow)) {
		_ = w.meta.SendReply(ctx, item.SenderIGSID, w.options.ReplyText)
	}
	return nil
}

func (w *WebhookWorker) liveWorkTime(ctx context.Context, item WebhookWork) (time.Time, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	now := w.options.Now().UTC()
	if item.LeaseExpiresAt.IsZero() || !now.Before(item.LeaseExpiresAt) {
		return time.Time{}, ErrWebhookLeaseLost
	}
	return now, nil
}

type classifiedMetaError interface {
	Kind() instagrammeta.ProviderErrorKind
	RetryAfter() time.Duration
}

func classifyMetaFailure(err error) (instagrammeta.ProviderErrorKind, time.Duration, bool) {
	var classified classifiedMetaError
	if !errors.As(err, &classified) {
		return "", 0, false
	}
	return classified.Kind(), classified.RetryAfter(), true
}

func (w *WebhookWorker) retryWork(ctx context.Context, item WebhookWork, redemption *WebhookRedemption, now time.Time, providerDelay time.Duration) error {
	next, retry := nextWebhookRetry(w.options.RetryPolicy, now, item.ProcessingStartedAt, item.Attempts, providerDelay)
	if retry {
		return w.queue.RetryWebhookWork(ctx, item.ID, item.LeaseToken, next, now)
	}
	if redemption != nil {
		if err := w.redeemer.RejectWebhookAttempt(ctx, redemption.AttemptID, RetryProfileLookupUnavailable, now); err != nil {
			return err
		}
	}
	reason := WebhookReasonMaxAttempts
	if !now.Before(item.ProcessingStartedAt.Add(w.options.RetryPolicy.MaxProcessingAge)) ||
		item.Attempts < w.options.RetryPolicy.MaxAttempts {
		reason = WebhookReasonMaxAge
	}
	return w.queue.FailWebhookWork(ctx, item.ID, item.LeaseToken, now, reason)
}

func (w *WebhookWorker) rejectProviderFailure(ctx context.Context, item WebhookWork, redemption WebhookRedemption, kind instagrammeta.ProviderErrorKind, now time.Time) error {
	retryCode := RetryInvalidProfileResponse
	reason := WebhookReasonInvalidProfile
	if kind == instagrammeta.ProviderErrorAuthentication {
		retryCode = RetryProfileLookupUnavailable
		reason = WebhookReasonProviderPermanent
	}
	if err := w.redeemer.RejectWebhookAttempt(ctx, redemption.AttemptID, retryCode, now); err != nil {
		return err
	}
	return w.queue.FailWebhookWork(ctx, item.ID, item.LeaseToken, now, reason)
}
