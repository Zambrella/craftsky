package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	defaultMaxConnections     = 512
	defaultMaxInFlightRequest = 256
	defaultMaxHeaderBytes     = 32 << 10
)

// HTTPAdmissionConfig defines the finite process budgets enforced before and
// by net/http. Route-specific body and handler budgets must fit inside these
// absolute server ceilings.
type HTTPAdmissionConfig struct {
	MaxConnections      int
	MaxInFlightRequests int
	ReadHeaderTimeout   time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	MaxHeaderBytes      int
}

func DefaultHTTPAdmissionConfig() HTTPAdmissionConfig {
	return HTTPAdmissionConfig{
		MaxConnections:      defaultMaxConnections,
		MaxInFlightRequests: defaultMaxInFlightRequest,
		ReadHeaderTimeout:   5 * time.Second,
		ReadTimeout:         90 * time.Second,
		WriteTimeout:        2 * time.Minute,
		IdleTimeout:         60 * time.Second,
		MaxHeaderBytes:      defaultMaxHeaderBytes,
	}
}

func (cfg HTTPAdmissionConfig) Validate() error {
	switch {
	case cfg.MaxConnections <= 0:
		return errors.New("HTTP max connections must be positive")
	case cfg.MaxInFlightRequests <= 0:
		return errors.New("HTTP max in-flight requests must be positive")
	case cfg.MaxInFlightRequests > cfg.MaxConnections:
		return errors.New("HTTP max in-flight requests cannot exceed max connections")
	case cfg.ReadHeaderTimeout <= 0:
		return errors.New("HTTP read-header timeout must be positive")
	case cfg.ReadTimeout <= 0:
		return errors.New("HTTP read timeout must be positive")
	case cfg.ReadHeaderTimeout > cfg.ReadTimeout:
		return errors.New("HTTP read-header timeout cannot exceed read timeout")
	case cfg.WriteTimeout <= cfg.ReadTimeout:
		return errors.New("HTTP write timeout must exceed read timeout")
	case cfg.IdleTimeout <= 0:
		return errors.New("HTTP idle timeout must be positive")
	case cfg.MaxHeaderBytes <= 0:
		return errors.New("HTTP max header bytes must be positive")
	case cfg.MaxHeaderBytes > http.DefaultMaxHeaderBytes:
		return fmt.Errorf("HTTP max header bytes cannot exceed %d", http.DefaultMaxHeaderBytes)
	default:
		return nil
	}
}

func NewHTTPServer(handler http.Handler, cfg HTTPAdmissionConfig) (*http.Server, error) {
	if handler == nil {
		return nil, errors.New("HTTP handler is required")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}, nil
}

// ConnectionLimitListener reserves a slot before calling the underlying
// listener's Accept. Saturation therefore remains in the kernel backlog and
// cannot allocate another accepted socket or net/http serve goroutine.
type ConnectionLimitListener struct {
	net.Listener
	slots     chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

func NewConnectionLimitListener(listener net.Listener, capacity int) (*ConnectionLimitListener, error) {
	if listener == nil {
		return nil, errors.New("HTTP listener is required")
	}
	if capacity <= 0 {
		return nil, errors.New("HTTP connection capacity must be positive")
	}
	return &ConnectionLimitListener{
		Listener: listener,
		slots:    make(chan struct{}, capacity),
		done:     make(chan struct{}),
	}, nil
}

func (l *ConnectionLimitListener) Accept() (net.Conn, error) {
	select {
	case l.slots <- struct{}{}:
	case <-l.done:
		return nil, net.ErrClosed
	}

	conn, err := l.Listener.Accept()
	if err != nil {
		<-l.slots
		return nil, err
	}
	return &connectionSlot{Conn: conn, release: func() { <-l.slots }}, nil
}

func (l *ConnectionLimitListener) Close() error {
	l.closeOnce.Do(func() {
		close(l.done)
		l.closeErr = l.Listener.Close()
	})
	return l.closeErr
}

type connectionSlot struct {
	net.Conn
	release   func()
	closeOnce sync.Once
	closeErr  error
}

func (c *connectionSlot) Close() error {
	c.closeOnce.Do(func() {
		c.closeErr = c.Conn.Close()
		c.release()
	})
	return c.closeErr
}

// ServeHTTP is the shutdown-aware server boundary used by real-listener tests
// and by callers that want cancellation to stop accepting immediately.
func ServeHTTP(ctx context.Context, server *http.Server, listener net.Listener, maxConnections int) error {
	if ctx == nil {
		return errors.New("HTTP server context is required")
	}
	if server == nil {
		return errors.New("HTTP server is required")
	}
	limited, err := NewConnectionLimitListener(listener, maxConnections)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		_ = limited.Close()
		return err
	}

	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = limited.Close()
		case <-stopped:
		}
	}()
	err = server.Serve(limited)
	close(stopped)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
