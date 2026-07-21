package instagram

import "time"

const (
	WebhookLeaseDuration    = 60 * time.Second
	WebhookMaxAttempts      = 5
	WebhookInitialBackoff   = time.Second
	WebhookMaxBackoff       = 5 * time.Minute
	WebhookMaxProcessingAge = 15 * time.Minute
)

// NextWebhookRetry computes deterministic provider backoff. A retry that would
// start at or after the fixed processing deadline is not scheduled.
func NextWebhookRetry(now, processingStartedAt time.Time, attempts int, providerDelay time.Duration) (time.Time, bool) {
	if now.IsZero() || processingStartedAt.IsZero() || now.Before(processingStartedAt) ||
		attempts <= 0 || attempts >= WebhookMaxAttempts {
		return time.Time{}, false
	}
	deadline := processingStartedAt.Add(WebhookMaxProcessingAge)
	if !now.Before(deadline) {
		return time.Time{}, false
	}

	backoff := WebhookInitialBackoff << (attempts - 1)
	if backoff > WebhookMaxBackoff {
		backoff = WebhookMaxBackoff
	}
	if providerDelay < 0 {
		providerDelay = 0
	}
	if providerDelay > WebhookMaxBackoff {
		providerDelay = WebhookMaxBackoff
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
