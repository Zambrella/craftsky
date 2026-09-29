package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/safetyincident"
)

type ImageSafetyHealthReader interface {
	Health(context.Context) (imagesafety.QueueHealth, error)
}

type SafetyWorkItem = safetyincident.WorkItem

type SafetyWorkReader interface {
	SafeWork(context.Context) ([]SafetyWorkItem, error)
}

type safetyAdminStatusResponse struct {
	ImageScans safetyImageScanStatus `json:"imageScans"`
	Work       []safetyWorkStatus    `json:"work"`
}

type safetyImageScanStatus struct {
	Queued           int   `json:"queued"`
	Leased           int   `json:"leased"`
	DeadLettered     int   `json:"deadLettered"`
	OldestDueSeconds int64 `json:"oldestDueSeconds"`
	MaxAttempts      int   `json:"maxAttempts"`
	Alert            bool  `json:"alert"`
}

type safetyWorkStatus struct {
	Reference  string `json:"reference"`
	Kind       string `json:"kind"`
	State      string `json:"state"`
	Priority   int    `json:"priority"`
	AgeSeconds int64  `json:"ageSeconds"`
	Deadline   string `json:"deadline,omitempty"`
	Owner      string `json:"owner,omitempty"`
	Cover      string `json:"cover,omitempty"`
	Alert      bool   `json:"alert"`
}

func SafetyAdminStatusHandler(
	health ImageSafetyHealthReader,
	now func() time.Time,
	imageAlertAge time.Duration,
	workReaders ...SafetyWorkReader,
) http.Handler {
	if now == nil {
		now = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		if health == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "safety_status_unavailable", "safety status unavailable", runID, nil)
			return
		}
		queue, err := health.Health(r.Context())
		if err != nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "safety_status_unavailable", "safety status unavailable", runID, nil)
			return
		}
		current := now().UTC()
		response := safetyAdminStatusResponse{ImageScans: safetyImageScanStatus{
			Queued: queue.Queued, Leased: queue.Leased, DeadLettered: queue.DeadLettered,
			OldestDueSeconds: int64(queue.OldestDueAge / time.Second), MaxAttempts: queue.MaxAttempts,
			Alert: queue.DeadLettered > 0 || imageAlertAge > 0 && queue.OldestDueAge >= imageAlertAge,
		}, Work: make([]safetyWorkStatus, 0)}
		items := make([]SafetyWorkItem, 0)
		for _, reader := range workReaders {
			if reader == nil {
				continue
			}
			loaded, err := reader.SafeWork(r.Context())
			if err != nil {
				envelope.WriteError(w, http.StatusServiceUnavailable, "safety_status_unavailable", "safety status unavailable", runID, nil)
				return
			}
			items = append(items, loaded...)
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].Priority != items[j].Priority {
				return items[i].Priority < items[j].Priority
			}
			if items[i].Deadline != nil && items[j].Deadline != nil && !items[i].Deadline.Equal(*items[j].Deadline) {
				return items[i].Deadline.Before(*items[j].Deadline)
			}
			if items[i].Deadline != nil && items[j].Deadline == nil {
				return true
			}
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		})
		for _, item := range items {
			safe, ok := safetyincident.RedactForSink(safetyincident.RestrictedRecord{
				Reference: item.Reference,
				Kind:      item.Kind,
				State:     item.State,
				Priority:  item.Priority,
				Alert:     item.Alert,
				Sensitive: item.Sensitive,
			}, safetyincident.SinkOwnerAPI)
			if !ok {
				envelope.WriteError(w, http.StatusServiceUnavailable, "safety_status_unavailable", "safety status unavailable", runID, nil)
				return
			}
			status := safetyWorkStatus{
				Reference: safe.Reference, Kind: safe.Kind, State: safe.State, Priority: safe.Priority,
				AgeSeconds: int64(current.Sub(item.CreatedAt.UTC()) / time.Second),
				Owner:      item.Owner, Cover: item.Cover, Alert: safe.Alert,
			}
			if status.AgeSeconds < 0 {
				status.AgeSeconds = 0
			}
			if item.Deadline != nil {
				status.Deadline = item.Deadline.UTC().Format(time.RFC3339)
				status.Alert = status.Alert || !current.Before(item.Deadline.UTC())
			}
			response.Work = append(response.Work, status)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
}
