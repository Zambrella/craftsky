package imagesafety

import "time"

type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func (policy RetryPolicy) Next(attempt int) (time.Duration, bool) {
	if policy.MaxAttempts <= 0 || policy.InitialBackoff <= 0 || policy.MaxBackoff < policy.InitialBackoff ||
		attempt <= 0 || attempt >= policy.MaxAttempts {
		return 0, false
	}

	delay := policy.InitialBackoff
	for range attempt - 1 {
		if delay >= policy.MaxBackoff/2 {
			return policy.MaxBackoff, true
		}
		delay *= 2
	}
	if delay > policy.MaxBackoff {
		delay = policy.MaxBackoff
	}
	return delay, true
}
