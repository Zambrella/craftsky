package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTapStatusExitConnected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","db":"ok","tap":{"connected":true,"last_event_at":"2026-04-17T14:23:11Z","reconnect_attempt":0,"last_error":""}}`))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestTapStatusExitDisconnected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"degraded","db":"ok","tap":{"connected":false,"last_event_at":"","reconnect_attempt":3,"last_error":"dial tcp: ..."}}`))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestTapStatusExitTransport(t *testing.T) {
	// Point at a closed port.
	code := tapStatus("http://127.0.0.1:1", nil)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestTapStatusExitGarbageBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 2 {
		t.Errorf("exit code = %d, want 2 (parse error)", code)
	}
}
