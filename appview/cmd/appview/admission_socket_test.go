package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

func TestRealServerSlowHeaderConsumesOnlyOneConnectionSlotAndRecovers(t *testing.T) {
	states := make(chan http.ConnState, 16)
	cfg := DefaultHTTPAdmissionConfig()
	cfg.MaxConnections = 1
	cfg.MaxInFlightRequests = 1
	cfg.ReadHeaderTimeout = 200 * time.Millisecond
	cfg.ReadTimeout = time.Second
	cfg.WriteTimeout = 2 * time.Second
	cfg.IdleTimeout = time.Second
	address, stop := startAdmissionSocketServer(t, cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), func(_ net.Conn, state http.ConnState) {
		states <- state
	})
	defer stop()

	first, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := io.WriteString(first, "GET / HTTP/1.1\r\nHost: slow.example"); err != nil {
		t.Fatal(err)
	}
	waitForConnectionState(t, states, http.StateNew)

	second, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, err := io.WriteString(second, "GET / HTTP/1.1\r\nHost: appview.example\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	if err := second.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	probe := make([]byte, 1)
	if _, err := second.Read(probe); err == nil {
		t.Fatal("second connection was served while the slow-header connection still held the only slot")
	} else if networkError, ok := err.(net.Error); !ok || !networkError.Timeout() {
		t.Fatalf("second read error=%v, want temporary deadline while queued in the kernel backlog", err)
	}

	if err := second.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(second), &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatalf("read recovered second response: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("recovered response status=%d, want 204", response.StatusCode)
	}
}

func TestRealServerRejectsOversizedHeaders(t *testing.T) {
	cfg := DefaultHTTPAdmissionConfig()
	cfg.MaxConnections = 1
	cfg.MaxInFlightRequests = 1
	cfg.MaxHeaderBytes = 1024
	address, stop := startAdmissionSocketServer(t, cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("handler ran for oversized headers")
		w.WriteHeader(http.StatusNoContent)
	}), nil)
	defer stop()

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	request := "GET / HTTP/1.1\r\nHost: appview.example\r\nX-Oversized: " + strings.Repeat("a", 8<<10) + "\r\n\r\n"
	if _, err := io.WriteString(connection, request); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatalf("read oversized-header response: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("oversized-header status=%d, want 431", response.StatusCode)
	}
}

func TestRealServerBodyDeadlineReturnsCanonicalTimeoutEnvelope(t *testing.T) {
	cfg := DefaultHTTPAdmissionConfig()
	cfg.MaxConnections = 1
	cfg.MaxInFlightRequests = 1
	cfg.ReadHeaderTimeout = time.Second
	cfg.ReadTimeout = 2 * time.Second
	cfg.WriteTimeout = 3 * time.Second
	bodyCfg := middleware.BodyLimitConfig{
		DefaultJSONBytes:       1024,
		DefaultJSONReadTimeout: 50 * time.Millisecond,
	}
	handler := middleware.BodyLimit(bodyCfg, middleware.BodyDefaultJSON, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, "generic decoder failure", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	address, stop := startAdmissionSocketServer(t, cfg, handler, nil)
	defer stop()

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := io.WriteString(connection,
		"POST / HTTP/1.1\r\nHost: appview.example\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\n\r\n10\r\nabc"); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodPost})
	if err != nil {
		t.Fatalf("read slow-body response: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestTimeout {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("slow-body status=%d, want 408; body=%s", response.StatusCode, body)
	}
	if !response.Close {
		t.Fatalf("slow-body response did not disable connection reuse: headers=%v", response.Header)
	}
	var problem envelope.Error
	if err := json.NewDecoder(response.Body).Decode(&problem); err != nil {
		t.Fatalf("decode slow-body envelope: %v", err)
	}
	if problem.Error != "request_body_timeout" || problem.Message == "" {
		t.Fatalf("slow-body envelope=%+v", problem)
	}
}

func startAdmissionSocketServer(
	t *testing.T,
	cfg HTTPAdmissionConfig,
	handler http.Handler,
	connState func(net.Conn, http.ConnState),
) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	limited, err := NewConnectionLimitListener(listener, cfg.MaxConnections)
	if err != nil {
		listener.Close()
		t.Fatal(err)
	}
	server, err := NewHTTPServer(handler, cfg)
	if err != nil {
		limited.Close()
		t.Fatal(err)
	}
	server.ConnState = connState
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = server.Serve(limited)
	}()
	stop := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		_ = limited.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("real HTTP server did not stop")
		}
	}
	return listener.Addr().String(), stop
}

func waitForConnectionState(t *testing.T, states <-chan http.ConnState, wanted http.ConnState) {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case state := <-states:
			if state == wanted {
				return
			}
		case <-timer.C:
			t.Fatalf("connection did not reach state %v", wanted)
		}
	}
}
