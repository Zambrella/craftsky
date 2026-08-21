package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestNewHTTPServerRejectsUnsafeAdmissionConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*HTTPAdmissionConfig)
	}{
		{name: "connection capacity", mutate: func(cfg *HTTPAdmissionConfig) { cfg.MaxConnections = 0 }},
		{name: "request capacity", mutate: func(cfg *HTTPAdmissionConfig) { cfg.MaxInFlightRequests = 0 }},
		{name: "header timeout", mutate: func(cfg *HTTPAdmissionConfig) { cfg.ReadHeaderTimeout = 0 }},
		{name: "read timeout", mutate: func(cfg *HTTPAdmissionConfig) { cfg.ReadTimeout = 0 }},
		{name: "write timeout", mutate: func(cfg *HTTPAdmissionConfig) { cfg.WriteTimeout = 0 }},
		{name: "idle timeout", mutate: func(cfg *HTTPAdmissionConfig) { cfg.IdleTimeout = 0 }},
		{name: "header bytes", mutate: func(cfg *HTTPAdmissionConfig) { cfg.MaxHeaderBytes = 0 }},
		{name: "header budget", mutate: func(cfg *HTTPAdmissionConfig) { cfg.ReadHeaderTimeout = cfg.ReadTimeout + time.Second }},
		{name: "write budget", mutate: func(cfg *HTTPAdmissionConfig) { cfg.WriteTimeout = cfg.ReadTimeout }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := DefaultHTTPAdmissionConfig()
			tt.mutate(&cfg)
			if _, err := NewHTTPServer(http.NotFoundHandler(), cfg); err == nil {
				t.Fatal("NewHTTPServer() error = nil, want invalid admission configuration")
			}
		})
	}
}

func TestNewHTTPServerAppliesFiniteCeilings(t *testing.T) {
	t.Parallel()

	cfg := DefaultHTTPAdmissionConfig()
	server, err := NewHTTPServer(http.NotFoundHandler(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if server.ReadHeaderTimeout != cfg.ReadHeaderTimeout ||
		server.ReadTimeout != cfg.ReadTimeout ||
		server.WriteTimeout != cfg.WriteTimeout ||
		server.IdleTimeout != cfg.IdleTimeout ||
		server.MaxHeaderBytes != cfg.MaxHeaderBytes {
		t.Fatalf("server ceilings = %#v, want %#v", server, cfg)
	}
}

func TestConnectionLimitListenerAcquiresBeforeAcceptAndReleasesExactlyOnce(t *testing.T) {
	t.Parallel()

	inner := newScriptedListener()
	limited, err := NewConnectionLimitListener(inner, 1)
	if err != nil {
		t.Fatal(err)
	}

	serverOne, clientOne := net.Pipe()
	inner.offer(serverOne)
	acceptedOne, err := limited.Accept()
	if err != nil {
		t.Fatal(err)
	}
	<-inner.acceptCalled

	secondResult := make(chan error, 1)
	go func() {
		_, err := limited.Accept()
		secondResult <- err
	}()

	select {
	case <-inner.acceptCalled:
		t.Fatal("second inner Accept ran before the first connection released its slot")
	case <-time.After(30 * time.Millisecond):
	}

	if err := acceptedOne.Close(); err != nil {
		t.Fatal(err)
	}
	if err := acceptedOne.Close(); err != nil {
		t.Fatal(err)
	}
	_ = clientOne.Close()

	select {
	case <-inner.acceptCalled:
	case <-time.After(time.Second):
		t.Fatal("second inner Accept did not begin after slot release")
	}

	serverTwo, clientTwo := net.Pipe()
	inner.offer(serverTwo)
	select {
	case err := <-secondResult:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("second Accept did not finish")
	}
	_ = serverTwo.Close()
	_ = clientTwo.Close()
	_ = limited.Close()
}

func TestConnectionLimitListenerCloseCancelsSaturatedAccept(t *testing.T) {
	t.Parallel()

	inner := newScriptedListener()
	limited, err := NewConnectionLimitListener(inner, 1)
	if err != nil {
		t.Fatal(err)
	}

	server, client := net.Pipe()
	inner.offer(server)
	accepted, err := limited.Accept()
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := limited.Accept()
		done <- err
	}()
	if err := limited.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("Accept() error = %v, want net.ErrClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("saturated Accept did not stop on listener close")
	}
	_ = accepted.Close()
	_ = client.Close()
}

type scriptedListener struct {
	connections  chan net.Conn
	acceptCalled chan struct{}
	closed       chan struct{}
	closeOnce    sync.Once
}

func newScriptedListener() *scriptedListener {
	return &scriptedListener{
		connections:  make(chan net.Conn),
		acceptCalled: make(chan struct{}, 4),
		closed:       make(chan struct{}),
	}
}

func (l *scriptedListener) offer(conn net.Conn) {
	go func() {
		select {
		case l.connections <- conn:
		case <-l.closed:
			_ = conn.Close()
		}
	}()
}

func (l *scriptedListener) Accept() (net.Conn, error) {
	l.acceptCalled <- struct{}{}
	select {
	case conn := <-l.connections:
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *scriptedListener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

func (l *scriptedListener) Addr() net.Addr { return testAddr("scripted") }

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }

func TestServeHTTPUsesBoundedListener(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	server, err := NewHTTPServer(http.NotFoundHandler(), DefaultHTTPAdmissionConfig())
	if err != nil {
		t.Fatal(err)
	}
	listener := newScriptedListener()
	if err := ServeHTTP(ctx, server, listener, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("ServeHTTP() error = %v, want context.Canceled", err)
	}
}
