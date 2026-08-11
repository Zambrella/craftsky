package accountdeletion

import (
	"hash/fnv"
	"strconv"
	"time"
)

type ErrorCategory string

const (
	ErrorCategoryPrivateCleanup   ErrorCategory = "privateCleanup"
	ErrorCategoryReauthentication ErrorCategory = "reauthentication"
	ErrorCategoryPDS              ErrorCategory = "pds"
	ErrorCategoryTerminal         ErrorCategory = "terminal"
)

type RetryPolicy struct {
	Delays    []time.Duration
	MaxJitter time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		Delays: []time.Duration{
			0,
			time.Minute,
			5 * time.Minute,
			15 * time.Minute,
			time.Hour,
			6 * time.Hour,
		},
		MaxJitter: 30 * time.Second,
	}
}

// Next returns a server-owned retry time. Once the schedule is exhausted the
// final delay is reused, so a deletion never enters a client-recovery state.
func (policy RetryPolicy) Next(now time.Time, jobID string, attempt int) time.Time {
	if len(policy.Delays) == 0 {
		policy = DefaultRetryPolicy()
	}
	index := attempt
	if index < 0 {
		index = 0
	}
	if index >= len(policy.Delays) {
		index = len(policy.Delays) - 1
	}
	delay := policy.Delays[index]
	if index > 0 && policy.MaxJitter > 0 {
		delay += deterministicJitter(jobID, attempt, policy.MaxJitter)
	}
	return now.Add(delay)
}

func deterministicJitter(jobID string, attempt int, maximum time.Duration) time.Duration {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(jobID))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(strconv.Itoa(attempt)))
	width := uint64(2*maximum + 1)
	return time.Duration(hasher.Sum64()%width) - maximum
}
