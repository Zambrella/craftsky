package instagrammeta

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPClientUsesFixedDefaultsAndRejectsRaisedLimits(t *testing.T) {
	t.Parallel()

	base := HTTPClientConfig{
		HTTPClient:        http.DefaultClient,
		APIVersion:        "v99.0",
		AccessToken:       "synthetic-token",
		OfficialAccountID: "official",
	}
	client, err := NewHTTPClient(base)
	if err != nil {
		t.Fatalf("NewHTTPClient(defaults): %v", err)
	}
	if client.timeout != MaxProviderTimeout || client.responseLimit != MaxProviderResponseBytes || cap(client.concurrency) != MaxProviderConcurrentCalls {
		t.Fatalf("defaults = timeout %s, response %d, concurrency %d", client.timeout, client.responseLimit, cap(client.concurrency))
	}

	for name, mutate := range map[string]func(*HTTPClientConfig){
		"timeout":            func(config *HTTPClientConfig) { config.RequestTimeout = MaxProviderTimeout + time.Nanosecond },
		"response":           func(config *HTTPClientConfig) { config.ResponseLimit = MaxProviderResponseBytes + 1 },
		"concurrency":        func(config *HTTPClientConfig) { config.MaxConcurrent = MaxProviderConcurrentCalls + 1 },
		"insecure base URL":  func(config *HTTPClientConfig) { config.BaseURL = "http://graph.instagram.com" },
		"invalid version":    func(config *HTTPClientConfig) { config.APIVersion = "latest" },
		"invalid account ID": func(config *HTTPClientConfig) { config.OfficialAccountID = "official/messages" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			config := base
			mutate(&config)
			if _, err := NewHTTPClient(config); err == nil {
				t.Fatal("NewHTTPClient accepted unsafe configuration")
			}
		})
	}
}

func TestHTTPClientEnforcesResponseByteCap(t *testing.T) {
	t.Parallel()

	validPrefix := `{"username":"synthetic_user"}`
	validAtLimit := validPrefix + strings.Repeat(" ", MaxProviderResponseBytes-len(validPrefix))
	overLimit := validAtLimit + " "
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/over") {
			_, _ = io.WriteString(w, overLimit)
			return
		}
		_, _ = io.WriteString(w, validAtLimit)
	}))
	t.Cleanup(server.Close)
	client := newTestHTTPClient(t, server, HTTPClientConfig{
		APIVersion:        "v99.0",
		AccessToken:       "synthetic-token",
		OfficialAccountID: "official",
	})

	username, err := client.LookupUsername(context.Background(), "at-limit")
	if err != nil || username != "synthetic_user" {
		t.Fatalf("LookupUsername(at limit) = (%q, %v)", username, err)
	}
	if _, err := client.LookupUsername(context.Background(), "over"); err == nil {
		t.Fatal("LookupUsername accepted a response over 64 KiB")
	} else if kind, _, ok := ProviderErrorDetails(err); !ok || kind != ProviderErrorInvalidResponse {
		t.Fatalf("over-limit error = %v, want invalid response", err)
	}
}

func TestHTTPClientEnforcesTotalTimeoutAndConcurrencyWhileWaiting(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			close(started)
		}
		select {
		case <-release:
			_, _ = io.WriteString(w, `{"username":"synthetic_user"}`)
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(server.Close)
	client := newTestHTTPClient(t, server, HTTPClientConfig{
		APIVersion:        "v99.0",
		AccessToken:       "synthetic-token",
		OfficialAccountID: "official",
		RequestTimeout:    time.Second,
		MaxConcurrent:     1,
	})

	firstResult := make(chan error, 1)
	go func() {
		_, err := client.LookupUsername(context.Background(), "first")
		firstResult <- err
	}()
	<-started
	secondCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := client.LookupUsername(secondCtx, "second"); err == nil || !IsRetryableProviderError(err) {
		t.Fatalf("second lookup while concurrency full = %v, want transient timeout", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("provider request count = %d, want 1 while concurrency full", got)
	}
	close(release)
	if err := <-firstResult; err != nil {
		t.Fatalf("first lookup: %v", err)
	}
}

func TestProviderStatusClassification(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		status    int
		wantKind  ProviderErrorKind
		retryable bool
	}{
		{http.StatusBadRequest, ProviderErrorPermanent, false},
		{http.StatusUnauthorized, ProviderErrorAuthentication, false},
		{http.StatusForbidden, ProviderErrorAuthentication, false},
		{http.StatusNotFound, ProviderErrorNotFound, false},
		{http.StatusRequestTimeout, ProviderErrorTransient, true},
		{http.StatusTooManyRequests, ProviderErrorRateLimited, true},
		{http.StatusInternalServerError, ProviderErrorTransient, true},
		{http.StatusServiceUnavailable, ProviderErrorTransient, true},
	} {
		err := classifyStatus(test.status, "1")
		kind, _, ok := ProviderErrorDetails(err)
		if !ok || kind != test.wantKind {
			t.Errorf("status %d kind = %q, want %q", test.status, kind, test.wantKind)
		}
		if got := IsRetryableProviderError(err); got != test.retryable {
			t.Errorf("status %d retryable = %t, want %t", test.status, got, test.retryable)
		}
	}
	if err := classifyContextError(context.Canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v, want context.Canceled", err)
	}
}

func TestProviderStructuredErrorsClassifyWithoutRetainingMessages(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name      string
		body      string
		wantKind  ProviderErrorKind
		retryable bool
	}{
		{name: "expired token", body: `{"error":{"message":"private token details","code":190}}`, wantKind: ProviderErrorAuthentication},
		{name: "permission", body: `{"error":{"message":"private permission details","code":10}}`, wantKind: ProviderErrorAuthentication},
		{name: "provider throttle", body: `{"error":{"message":"private rate details","code":613}}`, wantKind: ProviderErrorRateLimited, retryable: true},
		{name: "transient flag", body: `{"error":{"message":"private server details","code":2,"is_transient":true}}`, wantKind: ProviderErrorTransient, retryable: true},
		{name: "missing object", body: `{"error":{"message":"private identity details","code":100,"error_subcode":33}}`, wantKind: ProviderErrorNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := classifyResponse(http.StatusBadRequest, "", []byte(test.body))
			kind, _, ok := ProviderErrorDetails(err)
			if !ok || kind != test.wantKind {
				t.Fatalf("kind = %q, want %q", kind, test.wantKind)
			}
			if got := IsRetryableProviderError(err); got != test.retryable {
				t.Fatalf("retryable = %t, want %t", got, test.retryable)
			}
			if strings.Contains(err.Error(), "private") {
				t.Fatalf("provider error retained upstream message: %v", err)
			}
		})
	}
}
