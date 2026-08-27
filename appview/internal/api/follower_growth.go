package api

import (
	"context"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/followergrowth"
	"social.craftsky/appview/internal/middleware"
)

type FollowerGrowthReader interface {
	Read(context.Context, syntax.DID, followergrowth.DateRange) (followergrowth.History, error)
}

func GetFollowerGrowthHandler(reader FollowerGrowthReader, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		query := r.URL.Query()
		values, exists := query["period"]
		if len(query) != 1 || !exists || len(values) != 1 {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_period", "period must be one of 7d, 30d, or 1y", runID, nil)
			return
		}
		period, err := followergrowth.ParsePeriod(values[0])
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_period", "period must be one of 7d, 30d, or 1y", runID, nil)
			return
		}
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "authenticated owner unavailable", runID, nil)
			return
		}
		dateRange := period.Range(now())
		history, err := reader.Read(r.Context(), owner, dateRange)
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load follower growth", runID, nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, newFollowerGrowthResponse(period, followergrowth.BuildSeries(history, dateRange)))
	})
}

type followerGrowthResponse struct {
	Period              followergrowth.Period `json:"period"`
	RangeStart          string                `json:"rangeStart"`
	RangeEnd            string                `json:"rangeEnd"`
	AvailableFrom       *string               `json:"availableFrom"`
	LatestSnapshotDate  *string               `json:"latestSnapshotDate"`
	LatestCapturedAt    *string               `json:"latestCapturedAt"`
	LatestFollowerCount *int64                `json:"latestFollowerCount"`
	NetChange           *int64                `json:"netChange"`
	Points              []followerGrowthPoint `json:"points"`
}

type followerGrowthPoint struct {
	Date  string `json:"date"`
	Count *int64 `json:"count"`
}

func newFollowerGrowthResponse(period followergrowth.Period, growth followergrowth.Growth) followerGrowthResponse {
	response := followerGrowthResponse{
		Period:              period,
		RangeStart:          growth.Range.Start.Format(time.DateOnly),
		RangeEnd:            growth.Range.End.Format(time.DateOnly),
		AvailableFrom:       dateStringPointer(growth.AvailableFrom),
		LatestSnapshotDate:  dateStringPointer(growth.LatestSnapshotDate),
		LatestFollowerCount: growth.LatestFollowerCount,
		NetChange:           growth.NetChange,
		Points:              make([]followerGrowthPoint, len(growth.Points)),
	}
	if growth.LatestCapturedAt != nil {
		value := growth.LatestCapturedAt.UTC().Format(time.RFC3339Nano)
		response.LatestCapturedAt = &value
	}
	for i, point := range growth.Points {
		response.Points[i] = followerGrowthPoint{
			Date:  point.Date.Format(time.DateOnly),
			Count: point.Count,
		}
	}
	return response
}

func dateStringPointer(date *time.Time) *string {
	if date == nil {
		return nil
	}
	value := date.UTC().Format(time.DateOnly)
	return &value
}
