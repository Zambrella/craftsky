package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoRequest_200WritesStatusThenBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer dev" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer dev")
		}
		if got := r.Header.Get("X-Dev-DID"); got != "did:plc:test-caller" {
			t.Errorf("X-Dev-DID = %q, want %q", got, "did:plc:test-caller")
		}
		if got := r.Header.Get("X-Craftsky-Device-Id"); got != "cli-dev" {
			t.Errorf("X-Craftsky-Device-Id = %q, want %q", got, "cli-dev")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"hello":"world"}`)
	}))
	defer srv.Close()

	var out, errOut bytes.Buffer
	code, err := doRequest(requestArgs{
		Ctx:     context.Background(),
		Method:  "GET",
		Path:    "/x",
		BaseURL: srv.URL,
		DevDID:  "did:plc:test-caller",
		Out:     &out,
		ErrOut:  &errOut,
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	outStr := out.String()
	if !strings.HasPrefix(outStr, "200 OK\n") {
		t.Errorf("out should start with '200 OK\\n', got %q", outStr)
	}
	if !strings.Contains(outStr, `{"hello":"world"}`) {
		t.Errorf("out missing body: %q", outStr)
	}
	if errOut.Len() != 0 {
		t.Errorf("errOut should be empty on success, got %q", errOut.String())
	}
}

func TestDoRequestRemoteDevAddsConfiguredCredential(t *testing.T) {
	const secret = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("X-Craftsky-Dev-Authorization"); got != secret {
			t.Fatalf("dev authorization header = %q", got)
		}
		if request.Host != "appview-dev.example.net" {
			t.Fatalf("Host = %q", request.Host)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	code, err := doRequest(requestArgs{
		Ctx:              context.Background(),
		Method:           http.MethodGet,
		Path:             "/v1/whoami",
		BaseURL:          server.URL,
		RequestHost:      "appview-dev.example.net",
		DevDID:           "did:plc:test-caller",
		DevAuthorization: secret,
		Out:              io.Discard,
		ErrOut:           io.Discard,
	})
	if err != nil || code != 0 {
		t.Fatalf("doRequest code=%d err=%v", code, err)
	}
}

func TestDoRequestRejectsDevCredentialHeaderArgumentWithoutEchoingIt(t *testing.T) {
	const secret = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"
	_, err := doRequest(requestArgs{
		Ctx:     context.Background(),
		Method:  http.MethodGet,
		Path:    "/v1/whoami",
		BaseURL: "http://127.0.0.1:1",
		Headers: []string{"X-Craftsky-Dev-Authorization: " + secret},
		Out:     io.Discard,
		ErrOut:  io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "validated configuration") {
		t.Fatalf("doRequest error = %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("doRequest error exposed the dev credential")
	}
}

func TestDoRequest_4xxReturnsExit1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	var out, errOut bytes.Buffer
	code, err := doRequest(requestArgs{Ctx: context.Background(), Method: "GET", Path: "/x", BaseURL: srv.URL, Out: &out, ErrOut: &errOut})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1 for 401 response", code)
	}
}

func TestDoRequest_TransportErrorReturnsExit2(t *testing.T) {
	// Port 1 is reserved; connect will fail.
	var out, errOut bytes.Buffer
	code, err := doRequest(requestArgs{
		Ctx:     context.Background(),
		Method:  "GET",
		Path:    "/x",
		BaseURL: "http://127.0.0.1:1",
		Out:     &out,
		ErrOut:  &errOut,
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for transport error", code)
	}
	if !strings.Contains(errOut.String(), "transport error:") {
		t.Errorf("errOut should contain 'transport error:', got %q", errOut.String())
	}
	if out.Len() != 0 {
		t.Errorf("out should be empty on transport error, got %q", out.String())
	}
}
