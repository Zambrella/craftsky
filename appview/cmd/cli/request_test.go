package main

import (
	"bytes"
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"hello":"world"}`)
	}))
	defer srv.Close()

	var out, errOut bytes.Buffer
	code, err := doRequest(requestArgs{
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

func TestDoRequest_4xxReturnsExit1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	var out, errOut bytes.Buffer
	code, err := doRequest(requestArgs{Method: "GET", Path: "/x", BaseURL: srv.URL, Out: &out, ErrOut: &errOut})
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
