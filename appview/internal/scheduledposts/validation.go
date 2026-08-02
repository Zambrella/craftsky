package scheduledposts

import (
	"errors"
	"time"
)

const (
	MinimumScheduleDelay = 5 * time.Minute
	MaximumScheduleDelay = 28 * 24 * time.Hour
)

var ErrInvalidScheduledAt = errors.New("invalid scheduled time")

var ErrIneligibleScheduledPost = errors.New("ineligible scheduled post")

type PostKind string

const (
	PostKindStandard PostKind = "standard"
	PostKindProject  PostKind = "project"
)

type PostShape struct {
	Kind              PostKind
	HasReplyReference bool
	HasQuoteEmbed     bool
}

func ValidateScheduledAt(now, scheduledAt time.Time) error {
	if now.IsZero() || scheduledAt.IsZero() ||
		scheduledAt.Second() != 0 || scheduledAt.Nanosecond() != 0 {
		return ErrInvalidScheduledAt
	}

	delay := scheduledAt.Sub(now)
	if delay < MinimumScheduleDelay || delay > MaximumScheduleDelay {
		return ErrInvalidScheduledAt
	}
	return nil
}

func ValidateScheduleEligibility(shape PostShape) error {
	if shape.HasReplyReference || shape.HasQuoteEmbed {
		return ErrIneligibleScheduledPost
	}
	if shape.Kind != PostKindStandard && shape.Kind != PostKindProject {
		return ErrIneligibleScheduledPost
	}
	return nil
}
