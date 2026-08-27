package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/followergrowth"
)

func TestFollowerGrowthResponseJSONContract(t *testing.T) {
	start := time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	dateRange := followergrowth.DateRange{Start: start, End: end}

	t.Run("no history keeps nullable keys and all calendar points", func(t *testing.T) {
		body := marshalGrowthResponse(t, newFollowerGrowthResponse(
			followergrowth.PeriodThirtyDays,
			followergrowth.BuildSeries(followergrowth.History{}, dateRange),
		))
		assertGrowthResponseKeys(t, body)
		for _, key := range []string{
			"availableFrom",
			"latestSnapshotDate",
			"latestCapturedAt",
			"latestFollowerCount",
			"netChange",
		} {
			if body[key] != nil {
				t.Errorf("%s = %v, want null", key, body[key])
			}
		}
		points, ok := body["points"].([]any)
		if !ok || len(points) != 2 {
			t.Fatalf("points = %#v, want two calendar points", body["points"])
		}
		for i, point := range points {
			pointMap := point.(map[string]any)
			if pointMap["count"] != nil {
				t.Errorf("point %d count = %v, want null", i, pointMap["count"])
			}
		}
	})

	t.Run("populated response uses date-only and RFC3339 values", func(t *testing.T) {
		availableFrom := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
		latestDate := end
		capturedAt := time.Date(2026, time.August, 25, 0, 0, 2, 0, time.UTC)
		latestCount := int64(42)
		netChange := int64(5)
		pointCount := int64(42)
		body := marshalGrowthResponse(t, newFollowerGrowthResponse(
			followergrowth.PeriodThirtyDays,
			followergrowth.Growth{
				Range:               dateRange,
				AvailableFrom:       &availableFrom,
				LatestSnapshotDate:  &latestDate,
				LatestCapturedAt:    &capturedAt,
				LatestFollowerCount: &latestCount,
				NetChange:           &netChange,
				Points: []followergrowth.Point{
					{Date: start},
					{Date: end, Count: &pointCount},
				},
			},
		))
		assertGrowthResponseKeys(t, body)
		assertJSONValue(t, body, "period", "30d")
		assertJSONValue(t, body, "rangeStart", "2026-08-24")
		assertJSONValue(t, body, "rangeEnd", "2026-08-25")
		assertJSONValue(t, body, "availableFrom", "2026-07-01")
		assertJSONValue(t, body, "latestSnapshotDate", "2026-08-25")
		assertJSONValue(t, body, "latestCapturedAt", "2026-08-25T00:00:02Z")
		assertJSONValue(t, body, "latestFollowerCount", float64(42))
		assertJSONValue(t, body, "netChange", float64(5))
	})
}

func marshalGrowthResponse(t *testing.T, response followerGrowthResponse) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if strings.Contains(string(encoded), "_") {
		t.Fatalf("response contains snake_case key: %s", encoded)
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

func assertGrowthResponseKeys(t *testing.T, body map[string]any) {
	t.Helper()
	want := []string{
		"period",
		"rangeStart",
		"rangeEnd",
		"availableFrom",
		"latestSnapshotDate",
		"latestCapturedAt",
		"latestFollowerCount",
		"netChange",
		"points",
	}
	if len(body) != len(want) {
		t.Fatalf("response keys = %v, want %v", body, want)
	}
	for _, key := range want {
		if _, ok := body[key]; !ok {
			t.Errorf("response key %q missing", key)
		}
	}
}

func assertJSONValue(t *testing.T, body map[string]any, key string, want any) {
	t.Helper()
	if body[key] != want {
		t.Errorf("%s = %v, want %v", key, body[key], want)
	}
}
