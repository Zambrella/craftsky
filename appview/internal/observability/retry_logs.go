package observability

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const retryLogWindow = 30 * time.Second
const retryLogScopeLimit = 128

type retryLogState struct {
	started    time.Time
	lastSeen   time.Time
	suppressed int
}
type retryLogLimiter struct {
	mu     sync.Mutex
	now    func() time.Time
	scopes map[string]retryLogState
}

// Retry-only coalescing never owns issues or changes business attempts. Overflow
// emits normally, preserving independent failures with bounded limiter memory.
func (l *retryLogLimiter) selectRecord(record slog.Record) (bool, int) {
	fields := EventContext{}
	record.Attrs(func(attr slog.Attr) bool { fields[attr.Key] = attr.Value.Any(); return true })
	outcome, _ := fields["result"].(string)
	if outcome == "" {
		return true, 0
	}
	parts := []string{fmt.Sprint(fields["component"]), fmt.Sprint(fields["operation"]), fmt.Sprint(fields["failure_stage"])}
	if workflow, ok := fields["diagnostic"].(EventContext); ok {
		for _, key := range []string{"workflow_ref", "event_id", "record_uri"} {
			if value, ok := workflow[key].(string); ok && value != "" {
				parts = append(parts[:2], key, value)
				break
			}
		}
	}
	scope := strings.Join(parts, "\x00")
	if len(scope) > 512 {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for key, state := range l.scopes {
		if now.Before(state.lastSeen) || now.Sub(state.lastSeen) > 2*retryLogWindow {
			delete(l.scopes, key)
		}
	}
	state, found := l.scopes[scope]
	if outcome != "retry" {
		if found {
			delete(l.scopes, scope)
			return true, state.suppressed
		}
		return true, 0
	}
	if found && now.Sub(state.started) < retryLogWindow && !now.Before(state.started) {
		state.suppressed++
		state.lastSeen = now
		l.scopes[scope] = state
		return false, 0
	}
	if !found && len(l.scopes) >= retryLogScopeLimit {
		return true, 0
	}
	l.scopes[scope] = retryLogState{started: now, lastSeen: now}
	return true, state.suppressed
}
