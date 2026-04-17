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
