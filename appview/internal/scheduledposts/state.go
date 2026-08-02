package scheduledposts

import "errors"

type Status string

const (
	StatusScheduled      Status = "scheduled"
	StatusPublishing     Status = "publishing"
	StatusRetrying       Status = "retrying"
	StatusNeedsAttention Status = "needs_attention"
	StatusPublished      Status = "published"
	StatusDeleted        Status = "deleted"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid scheduled post status transition")
	ErrStaleWorkerVersion      = errors.New("stale scheduled post worker version")
)

func (status Status) CountsTowardCapacity() bool {
	switch status {
	case StatusScheduled, StatusPublishing, StatusRetrying, StatusNeedsAttention:
		return true
	default:
		return false
	}
}

func ValidateStatusTransition(from, to Status) error {
	allowed := false
	switch from {
	case StatusScheduled:
		allowed = to == StatusPublishing || to == StatusDeleted
	case StatusPublishing:
		allowed = to == StatusRetrying || to == StatusNeedsAttention || to == StatusPublished
	case StatusRetrying:
		allowed = to == StatusPublishing || to == StatusScheduled ||
			to == StatusNeedsAttention || to == StatusDeleted
	case StatusNeedsAttention:
		allowed = to == StatusScheduled || to == StatusPublishing || to == StatusDeleted
	}
	if !allowed {
		return ErrInvalidStatusTransition
	}
	return nil
}

func (status Status) AllowsMemberMutation() bool {
	return status == StatusScheduled || status == StatusRetrying || status == StatusNeedsAttention
}

func (status Status) AutomaticClaimable() bool {
	return status == StatusScheduled || status == StatusRetrying
}

func ValidateWorkerVersion(current, claimed int64) error {
	if current != claimed {
		return ErrStaleWorkerVersion
	}
	return nil
}
