package api

import (
	"context"
	"encoding/json"
	"net/http"

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
	Connected        bool   `json:"connected"`
	LastEventAt      string `json:"last_event_at"`
	ReconnectAttempt int    `json:"reconnect_attempt"`
	LastError        string `json:"last_error"`
}

// NewHealthHandler returns a handler for GET /healthz. Unlike the
// shallow HealthHandler (which only checks DB liveness), this is the
// deep health check that also reports Tap consumer state. Status is
// "ok" only when both DB ping succeeds and the Tap consumer is
// connected; otherwise "degraded". HTTP status is always 200.
func NewHealthHandler(pinger Pinger, stater Stater) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		if err := pinger.Ping(r.Context()); err != nil {
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
		if dbStatus == "ok" && tapState.Connected {
			resp.Status = "ok"
		} else {
			resp.Status = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
