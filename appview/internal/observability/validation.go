package observability

import (
	"fmt"
	"sort"
	"strings"
)

func ValidateMetricCall(call MetricCall) error {
	var problems []string
	if !strings.HasPrefix(call.Name, "craftsky_appview_") {
		problems = append(problems, "name")
	}
	if call.Kind == "" {
		problems = append(problems, "kind")
	}
	for key, value := range call.Attributes {
		if forbiddenMetricAttributeKey(key) {
			problems = append(problems, key)
			continue
		}
		if forbiddenTelemetryValue(value) {
			problems = append(problems, key)
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("invalid metric telemetry: %s", strings.Join(problems, ", "))
	}
	return nil
}

func forbiddenMetricAttributeKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" || key == "run_id" {
		return true
	}
	return strings.Contains(key, "token") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "email") ||
		strings.Contains(key, "device") ||
		strings.Contains(key, "session")
}

func forbiddenTelemetryValue(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "did:") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "bearer ") ||
		strings.Contains(lower, "select ") ||
		strings.Contains(lower, " from ") ||
		strings.Contains(value, "?") ||
		strings.Contains(value, "@") {
		return true
	}
	return false
}
