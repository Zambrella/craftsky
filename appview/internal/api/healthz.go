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

type Readiness interface {
	Ready() bool
}

// AdvisoryReadiness reports a capability without making it a deployment probe
// prerequisite. The readiness value remains visible to operators.
type AdvisoryReadiness struct{ Readiness }

func (AdvisoryReadiness) Required() bool { return false }

type StaticReadiness bool

func (readiness StaticReadiness) Ready() bool { return bool(readiness) }

type healthResponse struct {
	Status      string                `json:"status"`
	DB          string                `json:"db"`
	Tap         healthTapBlock        `json:"tap"`
	ImageSafety *healthReadinessBlock `json:"imageSafety,omitempty"`
}

type healthReadinessBlock struct {
	Ready    bool  `json:"ready"`
	Required *bool `json:"required,omitempty"`
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
// Image safety is required unless explicitly supplied as advisory readiness.
// This does not establish relay-head freshness. HTTP status is always 200.
func NewHealthHandler(pinger Pinger, stater Stater, optional ...Readiness) http.Handler {
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
		imageSafetyReady := true
		if len(optional) > 0 && optional[0] != nil {
			imageSafetyReady = optional[0].Ready()
			resp.ImageSafety = &healthReadinessBlock{Ready: imageSafetyReady}
			if requirement, ok := optional[0].(interface{ Required() bool }); ok {
				required := requirement.Required()
				resp.ImageSafety.Required = &required
				imageSafetyReady = imageSafetyReady || !required
			}
		}
		if !tapState.LastEventAt.IsZero() {
			resp.Tap.LastEventAt = tapState.LastEventAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		if dbStatus == "ok" && tapState.Connected && !tapState.LastEventAt.IsZero() && imageSafetyReady {
			resp.Status = "ok"
		} else {
			resp.Status = "degraded"
		}
		if provider, ok := stater.(interface{ Telemetry() tap.Telemetry }); ok {
			telemetry := provider.Telemetry()
			resp.Tap.Telemetry = &telemetry
			// A quiet tracked feed is normal. Use global cursor progress instead.
			resp.Status = "degraded"
			if dbStatus == "ok" && tapState.Connected && telemetry.Status == "progressing" && imageSafetyReady {
				resp.Status = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
