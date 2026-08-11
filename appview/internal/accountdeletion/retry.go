package accountdeletion

import (
	"hash/fnv"
	"strconv"
	"time"
)

type FailureKind string

const (
	FailureTransient     FailureKind = "transient"
	FailureOAuthUnusable FailureKind = "oauthUnusable"
	FailurePermanent     FailureKind = "permanent"
)

type RetryAction string

const (
	RetrySchedule              RetryAction = "schedule"
	RetryNeedsAttention        RetryAction = "needsAttention"
	RetryNeedsReauthentication RetryAction = "needsReauthentication"
)

type AttentionReason string

const (
	AttentionRetriesExhausted AttentionReason = "retriesExhausted"
	AttentionPermanentFailure AttentionReason = "permanentFailure"
	AttentionOAuthUnusable    AttentionReason = "oauthUnusable"
)

type RetryDecision struct {
	Action RetryAction
	At     time.Time
	Reason AttentionReason
}

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

// Decide returns the next automatic action. attempt is zero-based and counts
// the attempt being scheduled; callers reset it to zero for an authorized
// manual retry of the same deletion job.
func (policy RetryPolicy) Decide(now time.Time, jobID string, attempt int, failure FailureKind) RetryDecision {
	switch failure {
	case FailureOAuthUnusable:
		return RetryDecision{Action: RetryNeedsReauthentication, Reason: AttentionOAuthUnusable}
	case FailurePermanent:
		return RetryDecision{Action: RetryNeedsAttention, Reason: AttentionPermanentFailure}
	case FailureTransient:
		// Continue to the bounded automatic schedule below.
	default:
		return RetryDecision{Action: RetryNeedsAttention, Reason: AttentionPermanentFailure}
	}

	if attempt < 0 || attempt >= len(policy.Delays) {
		return RetryDecision{Action: RetryNeedsAttention, Reason: AttentionRetriesExhausted}
	}

	delay := policy.Delays[attempt]
	if attempt > 0 && policy.MaxJitter > 0 {
		delay += deterministicJitter(jobID, attempt, policy.MaxJitter)
	}
	return RetryDecision{Action: RetrySchedule, At: now.Add(delay)}
}

func deterministicJitter(jobID string, attempt int, maximum time.Duration) time.Duration {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(jobID))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(strconv.Itoa(attempt)))

	width := uint64(2*maximum + 1)
	return time.Duration(hasher.Sum64()%width) - maximum
}
