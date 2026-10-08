package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"social.craftsky/appview/internal/tap"
)

// Pinger matches *pgxpool.Pool's Ping signature without depending on pgx.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Stater returns Tap connection state. Matches tap.Consumer.State.
type Stater interface {
	State() tap.ConnState
}

type healthResponse struct {
	Status string         `json:"status"`
	DB     string         `json:"db"`
	Tap    healthTapBlock `json:"tap"`
}

type healthTapBlock struct {
	Telemetry        *tap.Telemetry `json:"telemetry,omitempty"`
	Connected        bool           `json:"connected"`
	LastEventAt      string         `json:"last_event_at"`
	ReconnectAttempt int            `json:"reconnect_attempt"`
	LastError        string         `json:"last_error"`
}

// NewHealthHandler returns a handler for GET /healthz. Unlike the
// shallow HealthHandler (which only checks DB liveness), this is the
// deep health check that also reports Tap consumer state and cached telemetry.
// With telemetry, "ok" requires DB readiness, a connected consumer and recent
// global cursor progress, not activity from the small tracked member feed.
// This does not establish relay-head freshness. HTTP status is always 200.
func NewHealthHandler(pinger Pinger, stater Stater) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pinger.Ping(ctx); err != nil {
			dbStatus = "error"
		}
		tapState := stater.State()

		resp := healthResponse{
			DB: dbStatus,
			Tap: healthTapBlock{
				Connected:        tapState.Connected,
				ReconnectAttempt: tapState.ReconnectAttempt,
				LastError:        tapState.LastError,
			},
		}
		if !tapState.LastEventAt.IsZero() {
			resp.Tap.LastEventAt = tapState.LastEventAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		if dbStatus == "ok" && tapState.Connected && !tapState.LastEventAt.IsZero() {
			resp.Status = "ok"
		} else {
			resp.Status = "degraded"
		}
		if provider, ok := stater.(interface{ Telemetry() tap.Telemetry }); ok {
			telemetry := provider.Telemetry()
			resp.Tap.Telemetry = &telemetry
			// A quiet tracked feed is normal. Use global cursor progress instead.
			resp.Status = "degraded"
			if dbStatus == "ok" && tapState.Connected && telemetry.Status == "progressing" {
				resp.Status = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
