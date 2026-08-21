package observability

import (
	"strings"
	"testing"
	"time"
)

func TestScheduledImageValidationMetricsAreContentFree(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{Env: "test", MetricRecorder: recorder})
	observer.ObserveScheduledImageValidation(
		"started",
		"unknown",
		0,
		1,
	)
	observer.ObserveScheduledImageValidation(
		"success",
		"jpeg",
		5*time.Millisecond,
		0,
	)

	calls := recorder.Calls()
	wantCounts := map[string]int{
		"craftsky_appview_scheduled_image_validations_total":           1,
		"craftsky_appview_scheduled_image_validation_duration_seconds": 1,
		"craftsky_appview_scheduled_image_validation_in_flight":        2,
	}
	gotCounts := map[string]int{}
	var inFlightValues []float64
	for _, call := range calls {
		if _, ok := wantCounts[call.Name]; ok {
			gotCounts[call.Name]++
		}
		if call.Name == "craftsky_appview_scheduled_image_validation_in_flight" {
			inFlightValues = append(inFlightValues, call.Value)
		}
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("invalid scheduled image metric: %v; call=%#v", err, call)
		}
		encoded := call.Name
		for key, value := range call.Attributes {
			encoded += key + value
		}
		for _, forbidden := range []string{
			"did:", "mediaId", "object", "width", "height", "private-image",
		} {
			if strings.Contains(encoded, forbidden) {
				t.Fatalf("scheduled image metric contains %q: %#v", forbidden, call)
			}
		}
	}
	for name, want := range wantCounts {
		if got := gotCounts[name]; got != want {
			t.Errorf("metric %q calls=%d, want %d; calls=%#v", name, got, want, calls)
		}
	}
	if len(inFlightValues) != 2 || inFlightValues[0] != 1 || inFlightValues[1] != 0 {
		t.Errorf("in-flight gauge values=%v, want [1 0]", inFlightValues)
	}
}
