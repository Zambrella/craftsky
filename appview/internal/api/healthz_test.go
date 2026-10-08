package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/tap"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(ctx context.Context) error { return f.err }

type fakeStater struct{ state tap.ConnState }

type telemetryStater struct {
	fakeStater
	telemetry tap.Telemetry
}

func (f *telemetryStater) Telemetry() tap.Telemetry { return f.telemetry }

func TestHealthzCachedTelemetry(t *testing.T) {
	for _, status := range []string{"progressing", "stalled", "unknown"} {
		t.Run(status, func(t *testing.T) {
			// No member events are needed to establish global cursor progress.
			stater := &telemetryStater{fakeStater: fakeStater{state: tap.ConnState{Connected: true}}, telemetry: tap.Telemetry{Status: status}}
			h := api.NewHealthHandler(fakePinger{}, stater)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
			var doc struct {
				Status string
				Tap    struct{ Telemetry tap.Telemetry }
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			want := "degraded"
			if status == "progressing" {
				want = "ok"
			}
			if doc.Status != want || doc.Tap.Telemetry.Status != status || rr.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("unexpected health: %s", rr.Body.String())
			}
		})
	}
}

func (f *fakeStater) State() tap.ConnState { return f.state }

func TestHealthz_AllOK(t *testing.T) {
	t.Parallel()
	h := api.NewHealthHandler(fakePinger{}, &fakeStater{state: tap.ConnState{Connected: true, LastEventAt: time.Unix(1700000000, 0)}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))

	if rr.Code != 200 {
		t.Fatalf("code = %d", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v", body["status"])
	}
	if body["db"] != "ok" {
		t.Errorf("db = %v", body["db"])
	}
	tapBlock, ok := body["tap"].(map[string]any)
	if !ok {
		t.Fatalf("tap block missing: %+v", body)
	}
	if tapBlock["connected"] != true {
		t.Errorf("tap.connected = %v", tapBlock["connected"])
	}
}

func TestHealthz_TapDisconnectedDegraded(t *testing.T) {
	t.Parallel()
	h := api.NewHealthHandler(fakePinger{}, &fakeStater{state: tap.ConnState{Connected: false, LastError: "dial timeout"}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))

	if rr.Code != 200 {
		t.Fatalf("code = %d (degraded should still be 200)", rr.Code)
	}
	var body map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body["status"] != "degraded" {
		t.Errorf("status = %v", body["status"])
	}
}

func TestHealthz_TapConnectedWithoutEventsDegraded(t *testing.T) {
	t.Parallel()
	h := api.NewHealthHandler(fakePinger{}, &fakeStater{state: tap.ConnState{Connected: true}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))

	if rr.Code != 200 {
		t.Fatalf("code = %d (degraded should still be 200)", rr.Code)
	}
	var body map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body["status"] != "degraded" {
		t.Errorf("status = %v", body["status"])
	}
}

func TestHealthz_DBErrorDegraded(t *testing.T) {
	t.Parallel()
	h := api.NewHealthHandler(fakePinger{err: errors.New("ping failed")}, &fakeStater{state: tap.ConnState{Connected: true}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != 200 {
		t.Errorf("code = %d", rr.Code)
	}
	var body map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body["db"] != "error" {
		t.Errorf("db = %v", body["db"])
	}
	if body["status"] != "degraded" {
		t.Errorf("status = %v", body["status"])
	}
}

func TestHealthz_ImageSafetyReadinessFailsClosedWithoutSensitiveDetail(t *testing.T) {
	t.Parallel()
	h := api.NewHealthHandler(
		fakePinger{},
		&telemetryStater{
			fakeStater: fakeStater{state: tap.ConnState{Connected: true, LastEventAt: time.Unix(1700000000, 0)}},
			telemetry:  tap.Telemetry{Status: "progressing"},
		},
		api.StaticReadiness(false),
	)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "degraded" {
		t.Fatalf("status=%v", body["status"])
	}
	imageSafety, ok := body["imageSafety"].(map[string]any)
	if !ok || imageSafety["ready"] != false || len(imageSafety) != 1 {
		t.Fatalf("unsafe image-safety readiness block=%v", body["imageSafety"])
	}
}
