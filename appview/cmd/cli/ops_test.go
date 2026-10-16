package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/ops"
)

func TestOpsStatusCommandFormatsAggregateSnapshot(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		var output bytes.Buffer
		cmd := newOpsCmd(func(ctx context.Context, origin string, retries int) (opsReport, error) {
			if origin != "http://127.0.0.1:8080" || retries != 5 {
				t.Fatalf("origin=%s retries=%d", origin, retries)
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("missing operation deadline")
			}
			return opsReport{Snapshot: ops.Snapshot{CollectedAt: time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC), RetryThreshold: 5,
				Queues: []ops.Queue{{Name: "tapRepository", Pending: 2, Due: 1, RepeatedAttempts: 1, States: map[string]int64{"pending": 2}}}}, Health: opsHealth{Status: "unavailable", DB: "unknown"}}, nil
		})
		args := []string{"status", "--url", "http://127.0.0.1:8080"}
		if jsonOutput {
			args = append(args, "--json")
		}
		cmd.SetArgs(args)
		cmd.SetOut(&output)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if jsonOutput {
			var report opsReport
			if err := json.Unmarshal(output.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.Queues[0].Pending != 2 || report.Health.Status != "unavailable" {
				t.Fatalf("report=%+v", report)
			}
		} else if !strings.Contains(output.String(), "tapRepository") || !strings.Contains(output.String(), "ATTEMPTS >= 5") || !strings.Contains(output.String(), "unavailable") {
			t.Fatalf("text=%s", output.String())
		}
	}
}

func TestOpsStatusRejectsInvalidOptionsBeforeLoading(t *testing.T) {
	for _, args := range [][]string{{"--timeout", "0s"}, {"--timeout", "2m"}, {"--retry-threshold", "0"}, {"--url", "https://user:secret@example.com"}, {"--url", "https://example.com/?private=secret"}, {"--url", "file:///tmp/secret"}} {
		cmd := newOpsCmd(func(context.Context, string, int) (opsReport, error) {
			t.Fatal("invalid options reached database")
			return opsReport{}, nil
		})
		cmd.SetArgs(append([]string{"status"}, args...))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestOpsDefaultOriginUsesCanonicalProductionHost(t *testing.T) {
	t.Setenv("PORT", "10000")
	if got := defaultOpsOrigin("https://appview.craftsky.social"); got != "https://appview.craftsky.social" {
		t.Fatalf("production origin=%s", got)
	}
	if got := defaultOpsOrigin(""); got != "http://127.0.0.1:10000" {
		t.Fatalf("local origin=%s", got)
	}
	t.Setenv("PORT", "")
	if got := defaultOpsOrigin(""); got != "http://127.0.0.1:8080" {
		t.Fatalf("default local origin=%s", got)
	}
}

func TestOpsHealthSelectsOnlyPublicHealthFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/healthz" || r.Header.Get("Authorization") != "" {
			t.Errorf("request=%s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"status":"ok","db":"ok","tap":{"connected":true,"last_error":"private-error-canary","telemetry":{"status":"progressing","outboxDepth":0}},"private":"private-payload-canary"}`))
	}))
	defer server.Close()
	health := readOpsHealth(t.Context(), server.URL)
	encoded, _ := json.Marshal(health)
	if health.Status != "ok" || health.Tap.Telemetry == nil || health.Tap.Telemetry.Status != "progressing" {
		t.Fatalf("health=%s", encoded)
	}
	if strings.Contains(string(encoded), "canary") || strings.Contains(string(encoded), "last_error") {
		t.Fatalf("unsafe health=%s", encoded)
	}
}

func TestOpsHealthUnavailableDoesNotLookHealthy(t *testing.T) {
	for _, body := range []string{`not json`, `{}`, `{"status":"ok","db":"ok"}`, strings.Repeat(" ", 65537)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(body)) }))
		health := readOpsHealth(t.Context(), server.URL)
		server.Close()
		if health.Status != "unavailable" || health.DB != "unknown" || health.Tap.Connected != nil {
			t.Fatalf("invalid response became %+v", health)
		}
	}
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.invalid/", http.StatusFound)
	}))
	defer redirect.Close()
	if readOpsHealth(t.Context(), redirect.URL).Status != "unavailable" {
		t.Fatal("followed health redirect")
	}
}
