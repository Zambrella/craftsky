package instagram

import "time"

const (
	WebhookLeaseDuration    = 60 * time.Second
	WebhookMaxAttempts      = 5
	WebhookInitialBackoff   = time.Second
	WebhookMaxBackoff       = 5 * time.Minute
	WebhookMaxProcessingAge = 15 * time.Minute
)

type WebhookRetryPolicy struct {
	MaxAttempts      int
	InitialBackoff   time.Duration
	MaxBackoff       time.Duration
	MaxProcessingAge time.Duration
}

func DefaultWebhookRetryPolicy() WebhookRetryPolicy {
	return WebhookRetryPolicy{
		MaxAttempts: WebhookMaxAttempts, InitialBackoff: WebhookInitialBackoff,
		MaxBackoff: WebhookMaxBackoff, MaxProcessingAge: WebhookMaxProcessingAge,
	}
}

func (p WebhookRetryPolicy) valid() bool {
	return p.MaxAttempts > 0 && p.MaxAttempts <= WebhookMaxAttempts &&
		p.InitialBackoff > 0 && p.InitialBackoff <= WebhookInitialBackoff &&
		p.MaxBackoff >= p.InitialBackoff && p.MaxBackoff <= WebhookMaxBackoff &&
		p.MaxProcessingAge > 0 && p.MaxProcessingAge <= WebhookMaxProcessingAge
}

// NextWebhookRetry computes deterministic provider backoff. A retry that would
// start at or after the fixed processing deadline is not scheduled.
func NextWebhookRetry(now, processingStartedAt time.Time, attempts int, providerDelay time.Duration) (time.Time, bool) {
	return nextWebhookRetry(DefaultWebhookRetryPolicy(), now, processingStartedAt, attempts, providerDelay)
}

func nextWebhookRetry(policy WebhookRetryPolicy, now, processingStartedAt time.Time, attempts int, providerDelay time.Duration) (time.Time, bool) {
	if !policy.valid() {
		return time.Time{}, false
	}
	if now.IsZero() || processingStartedAt.IsZero() || now.Before(processingStartedAt) ||
		attempts <= 0 || attempts >= policy.MaxAttempts {
		return time.Time{}, false
	}
	deadline := processingStartedAt.Add(policy.MaxProcessingAge)
	if !now.Before(deadline) {
		return time.Time{}, false
	}

	backoff := policy.InitialBackoff << (attempts - 1)
	if backoff > policy.MaxBackoff {
		backoff = policy.MaxBackoff
	}
	if providerDelay < 0 {
		providerDelay = 0
	}
	if providerDelay > policy.MaxBackoff {
		providerDelay = policy.MaxBackoff
	}
	if providerDelay > backoff {
		backoff = providerDelay
	}
	next := now.Add(backoff)
	if !next.Before(deadline) {
		return time.Time{}, false
	}
	return next.UTC(), true
}
