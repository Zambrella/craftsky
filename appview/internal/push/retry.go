package push

import "time"

const maxRetryDelay = 15 * time.Minute

// NextRetry applies bounded exponential backoff. jitter is a fractional
// adjustment in [-1,1] supplied by the dispatcher for deterministic tests.
func NextRetry(now, deadline time.Time, attempt int, jitter float64) (time.Time, bool) {
	if attempt < 1 {
		attempt = 1
	}
	delay := time.Second
	for i := 1; i < attempt && delay < maxRetryDelay; i++ {
		delay *= 2
		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
	}
	if jitter < -1 {
		jitter = -1
	}
	if jitter > 1 {
		jitter = 1
	}
	delay += time.Duration(float64(delay) * 0.2 * jitter)
	next := now.Add(delay)
	if !next.Before(deadline) {
		return time.Time{}, false
	}
	return next, true
}

func ProviderTTL(now, deadline time.Time) (time.Duration, bool) {
	ttl := deadline.Sub(now)
	if ttl <= 0 {
		return 0, false
	}
	return ttl, true
}
