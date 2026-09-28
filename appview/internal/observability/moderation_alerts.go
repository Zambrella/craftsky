package observability

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ModerationAdminAuthFailureThreshold = 5
	ModerationAdminAuthFailureWindow    = 5 * time.Minute
	ModerationExpiryDeadline            = time.Hour
	ModerationNotificationMaxAge        = 15 * time.Minute
)

var safeModerationRequestID = regexp.MustCompile(`^[0-9a-f]{64}$`)

type ModerationOperationObservation struct {
	Operation     string
	Result        string
	ActorType     string
	RequestID     string
	CaseReference string
	OccurredAt    time.Time
}

func AdminAuthFailureAlert(failures int, window time.Duration) bool {
	return failures >= ModerationAdminAuthFailureThreshold && window <= ModerationAdminAuthFailureWindow
}

func ExpiryOverdueAlert(dueAt, now time.Time) bool {
	return now.Sub(dueAt) > ModerationExpiryDeadline
}

func ModerationNotificationStalledAlert(createdAt, now time.Time) bool {
	return now.Sub(createdAt) >= ModerationNotificationMaxAge
}

type moderationMetricRecorder interface {
	ModerationAuth(context.Context, string)
	ModerationOperation(context.Context, string, string)
	ModerationWork(context.Context, string, int, time.Duration, bool)
}

func (o *Observer) ObserveModeratorAuthSuccess(now time.Time) {
	o.observeModeratorAuth("success", now, false)
}

func (o *Observer) ObserveModeratorAuthFailure(now time.Time) bool {
	if o == nil {
		return false
	}
	now = now.UTC()
	cutoff := now.Add(-ModerationAdminAuthFailureWindow)
	o.moderationMu.Lock()
	kept := o.adminAuthFailures[:0]
	for _, at := range o.adminAuthFailures {
		if !at.Before(cutoff) {
			kept = append(kept, at)
		}
	}
	o.adminAuthFailures = append(kept, now)
	active := AdminAuthFailureAlert(len(o.adminAuthFailures), ModerationAdminAuthFailureWindow)
	o.moderationMu.Unlock()
	o.observeModeratorAuth("failure", now, active)
	return active
}

func (o *Observer) observeModeratorAuth(result string, now time.Time, alert bool) {
	if o == nil {
		return
	}
	if recorder, ok := o.metricRecorder.(moderationMetricRecorder); ok {
		recorder.ModerationAuth(context.Background(), result)
	}
	level := slog.LevelInfo
	if result == "failure" {
		level = slog.LevelWarn
	}
	if alert {
		level = slog.LevelError
	}
	o.Log(context.Background(), level, "moderation admin authentication observed", EventContext{
		"component": "moderation", "operation": "admin_auth", "result": result,
		"occurred_at": now.UTC().Format(time.RFC3339Nano),
	})
}

func (o *Observer) ObserveModerationOperation(ctx context.Context, observation ModerationOperationObservation) {
	if o == nil {
		return
	}
	operation := safeModerationOperation(observation.Operation)
	result := safeModerationResult(observation.Result)
	if recorder, ok := o.metricRecorder.(moderationMetricRecorder); ok {
		recorder.ModerationOperation(ctx, operation, result)
	}
	o.Log(ctx, slog.LevelInfo, "moderation operation completed", EventContext{
		"component":      "moderation",
		"operation":      operation,
		"result":         result,
		"actor_type":     safeModerationActorType(observation.ActorType),
		"request_id":     safeModerationRequestIdentifier(observation.RequestID),
		"case_reference": safeModerationCaseReference(observation.CaseReference),
		"occurred_at":    observation.OccurredAt.UTC().Format(time.RFC3339Nano),
	})
}

func (o *Observer) ObserveModerationCommand(ctx context.Context, operation, result, actorType, requestID, caseReference string, occurredAt time.Time) {
	o.ObserveModerationOperation(ctx, ModerationOperationObservation{
		Operation: operation, Result: result, ActorType: actorType,
		RequestID: requestID, CaseReference: caseReference, OccurredAt: occurredAt,
	})
}

func (o *Observer) ObserveModerationExpiry(dueAt, now time.Time) bool {
	active := ExpiryOverdueAlert(dueAt, now)
	if o != nil {
		age := nonNegativeDuration(now.Sub(dueAt))
		if recorder, ok := o.metricRecorder.(moderationMetricRecorder); ok {
			recorder.ModerationWork(context.Background(), "strike_expiry", 1, age, active)
		}
		if active {
			o.Log(context.Background(), slog.LevelError, "moderation strike expiry alert", EventContext{"component": "moderation", "operation": "strike_expiry", "result": "alert"})
		}
	}
	return active
}

func (o *Observer) ObserveModerationNotificationQueue(pending int, oldestAge time.Duration) bool {
	active := pending > 0 && oldestAge >= 15*time.Minute
	if o != nil {
		if recorder, ok := o.metricRecorder.(moderationMetricRecorder); ok {
			recorder.ModerationWork(context.Background(), "notification_delivery", max(pending, 0), nonNegativeDuration(oldestAge), active)
		}
		if active {
			o.Log(context.Background(), slog.LevelError, "moderation notification queue alert", EventContext{"component": "moderation", "operation": "notification_delivery", "result": "alert"})
		}
	}
	return active
}

func safeModerationOperation(value string) string {
	switch strings.TrimSpace(value) {
	case "decision", "appeal_confirmation", "appeal_resolution", "effect_change", "strike_expiry":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeModerationResult(value string) string {
	switch strings.TrimSpace(value) {
	case "success", "failure", "replayed":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeModerationActorType(value string) string {
	switch strings.TrimSpace(value) {
	case "moderator", "system":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeModerationRequestIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if safeModerationRequestID.MatchString(value) {
		return value
	}
	return "unknown"
}

func safeModerationCaseReference(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "MOD-") {
		return "unknown"
	}
	id, err := uuid.Parse(value[4:])
	if err != nil {
		return "unknown"
	}
	return "MOD-" + id.String()
}
