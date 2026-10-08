package safetyincident

import (
	"errors"
	"sort"
	"time"
)

var ErrDeadlineTargetNotConfigured = errors.New("approved incident deadline target is not configured")

type WorkflowClass string

const (
	WorkflowIntimateImage    WorkflowClass = "intimateImage"
	WorkflowCredibleThreat   WorkflowClass = "credibleThreat"
	WorkflowAuthorityRequest WorkflowClass = "authorityRequest"
)

type DeadlineTarget struct {
	ResponseWithin time.Duration
	AlertBefore    []time.Duration
}

type Deadline struct {
	DueAt   time.Time
	AlertAt []time.Time
}

type DeadlinePolicy struct {
	targets map[WorkflowClass]DeadlineTarget
}

func NewDeadlinePolicy(targets map[WorkflowClass]DeadlineTarget) (DeadlinePolicy, error) {
	policy := DeadlinePolicy{targets: make(map[WorkflowClass]DeadlineTarget, len(targets))}
	for class, target := range targets {
		if class != WorkflowIntimateImage || target.ResponseWithin <= 0 {
			return DeadlinePolicy{}, ErrDeadlineTargetNotConfigured
		}
		for _, before := range target.AlertBefore {
			if before <= 0 || before >= target.ResponseWithin {
				return DeadlinePolicy{}, ErrDeadlineTargetNotConfigured
			}
		}
		policy.targets[class] = DeadlineTarget{
			ResponseWithin: target.ResponseWithin,
			AlertBefore:    append([]time.Duration(nil), target.AlertBefore...),
		}
	}
	return policy, nil
}

func (policy DeadlinePolicy) Calculate(class WorkflowClass, receivedAt time.Time) (Deadline, error) {
	if receivedAt.IsZero() {
		return Deadline{}, ErrDeadlineTargetNotConfigured
	}
	target, ok := policy.targets[class]
	if !ok || target.ResponseWithin <= 0 {
		return Deadline{}, ErrDeadlineTargetNotConfigured
	}
	dueAt := receivedAt.UTC().Add(target.ResponseWithin)
	alerts := make([]time.Time, 0, len(target.AlertBefore))
	for _, before := range target.AlertBefore {
		alerts = append(alerts, dueAt.Add(-before))
	}
	sort.Slice(alerts, func(i, j int) bool { return alerts[i].Before(alerts[j]) })
	return Deadline{DueAt: dueAt, AlertAt: alerts}, nil
}
