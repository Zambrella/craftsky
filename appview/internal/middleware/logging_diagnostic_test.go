package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/observability"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/routes"
)

// AT-004 / FR-004, RULE-004 / AC-004, AC-020: no wall-clock sleeps; the
// held handler proves arrival is safe and observable before completion.
func TestLoggingActualPathAtArrivalBeforeHandlerCompletes(t *testing.T) {
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvDev, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/posts/{did}/{rkey}", func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; _, _ = w.Write([]byte("ok")) })
	handler := middleware.Logging(logger, catalogue)(catalogue.RoutingHandler(mux))
	response := httptest.NewRecorder()
	go func() {
		handler.ServeHTTP(response, httptest.NewRequest("GET", "/v1/posts/did:plc:target/record?cursor=credential", nil))
		close(done)
	}()
	released := false
	defer func() {
		if !released {
			close(release)
		}
		<-done
	}()
	<-entered
	var arrival map[string]any
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	if err := decoder.Decode(&arrival); err != nil {
		t.Fatal(err)
	}
	if arrival["incoming_path"] != "/v1/posts/did:plc:target/record" || arrival["method"] != "GET" || arrival["run_id"] == "" || arrival["level"] != "INFO" {
		t.Fatalf("arrival missing safe actual path/correlation: %#v", arrival)
	}
	var unexpected map[string]any
	if err := decoder.Decode(&unexpected); err != io.EOF {
		t.Fatalf("unexpected early completion: %#v %v", unexpected, err)
	}
	close(release)
	released = true
	<-done
	decoder = json.NewDecoder(bytes.NewReader(logs.Bytes()))
	var first, completion map[string]any
	_ = decoder.Decode(&first)
	if err := decoder.Decode(&completion); err != nil {
		t.Fatal(err)
	}
	if completion["msg"] != "Request completed" || completion["level"] != "INFO" || completion["run_id"] != arrival["run_id"] || completion["incoming_path"] != arrival["incoming_path"] || completion["route_pattern"] != "/v1/posts/{did}/{rkey}" || completion["status"] != float64(200) || completion["bytes"] != float64(2) || completion["duration"] == nil {
		t.Fatalf("completion missing useful fields: %#v", completion)
	}
}

// IT-001 / FR-004, RULE-004 / AC-004, AC-020.
func TestLoggingLifecycleOutcomesAtINFO(t *testing.T) {
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvDev, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, path string
		status     int
		canceled   bool
	}{
		{"success", "/v1/posts/did:plc:target/record?cursor=credential", 200, false},
		{"client rejection", "/v1/posts/did:plc:target/record", 422, false},
		{"server failure", "/v1/posts/did:plc:target/record", 500, false},
		{"cancellation", "/v1/posts/did:plc:target/record", 200, true},
		{"unmatched", "/v1/unclassified/private-value?search=private-term", 404, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			observer := observability.New(observability.Config{Env: "test", Logger: logger})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			mux := http.NewServeMux()
			mux.HandleFunc("GET /v1/posts/{did}/{rkey}", func(w http.ResponseWriter, r *http.Request) {
				if tc.canceled {
					cancel()
					return
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte("ok"))
			})
			handler := middleware.Logging(logger, catalogue)(middleware.HTTPMetrics(observer)(catalogue.RoutingHandler(mux)))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest("GET", tc.path, nil).WithContext(ctx))
			var arrival, completion map[string]any
			errorLogs := 0
			decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
			for {
				var record map[string]any
				if err := decoder.Decode(&record); err == io.EOF {
					break
				} else if err != nil {
					t.Fatal(err)
				}
				switch record["msg"] {
				case "Request received":
					arrival = record
				case "Request completed":
					completion = record
				}
				if record["level"] == "ERROR" {
					errorLogs++
				}
			}
			expectedPath := catalogue.Resolve("GET", httptest.NewRequest("GET", tc.path, nil).URL).Path
			if arrival["incoming_path"] != expectedPath || arrival["run_id"] == nil || completion["run_id"] != arrival["run_id"] || completion["incoming_path"] != expectedPath || completion["level"] != "INFO" {
				t.Fatalf("lifecycle correlation/path missing: %#v %#v", arrival, completion)
			}
			status := tc.status
			if tc.canceled {
				status = 499
			}
			if completion["status"] != float64(status) || completion["bytes"] != float64(response.Body.Len()) || completion["duration"].(float64) < 0 {
				t.Fatalf("completion outcome=%#v", completion)
			}
			if tc.status >= 500 && errorLogs == 0 {
				t.Fatal("5xx has no associated local ERROR cause diagnostic")
			}
			if strings.Contains(logs.String(), "credential") || strings.Contains(logs.String(), "private-value") || strings.Contains(logs.String(), "private-term") {
				t.Fatal("protected query/unclassified value leaked")
			}
			if response.Code != tc.status {
				t.Fatalf("response behavior changed: %d", response.Code)
			}
		})
	}
}

// IT-002 / FR-004 / AC-004: ordinary INFO does not contain successful probes.
func TestLoggingHealthProbeSeverity(t *testing.T) {
	for _, path := range []string{"/health", "/healthz"} {
		for _, threshold := range []slog.Level{slog.LevelInfo, slog.LevelDebug} {
			for _, status := range []int{200, 400, 500} {
				t.Run(fmt.Sprintf("%s/%s/%d", path, threshold, status), func(t *testing.T) {
					var logs bytes.Buffer
					logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: threshold}))
					handler := middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
					handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))
					if threshold == slog.LevelInfo && status == 200 {
						if logs.Len() != 0 {
							t.Fatalf("successful health noise at INFO: %s", logs.String())
						}
						return
					}
					wantLevel := "DEBUG"
					if status >= 500 {
						wantLevel = "ERROR"
					} else if status >= 400 {
						wantLevel = "WARN"
					}
					decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
					found := false
					for {
						var record map[string]any
						if err := decoder.Decode(&record); err == io.EOF {
							break
						} else if err != nil {
							t.Fatal(err)
						}
						if record["msg"] == "Request received" && record["level"] != "DEBUG" {
							t.Fatalf("probe arrival level=%v", record["level"])
						}
						if record["msg"] == "Request completed" {
							found = true
							if record["level"] != wantLevel || record["status"] != float64(status) {
								t.Fatalf("probe completion=%#v", record)
							}
						}
					}
					if !found {
						t.Fatal("probe completion missing")
					}
				})
			}
		}
	}
}

func TestLoggingHealthPanicAfterStartedResponseIsERROR(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := middleware.Logging(logger)(middleware.Recovery(logger, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
		panic("opaque private prose")
	})))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/healthz", nil))
	if response.Code != 200 || response.Body.String() != "ok" {
		t.Fatalf("started response rewritten: %d %s", response.Code, response.Body.String())
	}
	found := false
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	for {
		var record map[string]any
		if err := decoder.Decode(&record); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if record["msg"] == "Request completed" {
			found = true
			if record["level"] != "ERROR" {
				t.Fatalf("unexpected probe completion=%#v", record)
			}
		}
	}
	if !found {
		t.Fatal("unexpected probe completion missing at INFO")
	}
	if strings.Contains(logs.String(), "opaque private prose") {
		t.Fatal("unknown recovered prose leaked")
	}
}

// IT-013: ordinary lifecycle independently exports with tracing off/zero-rate.
func TestLoggingProtectedLifecycleExport(t *testing.T) {
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvDev, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tracing := range []bool{false, true} {
		t.Run(fmt.Sprint(tracing), func(t *testing.T) {
			var local bytes.Buffer
			transport := &sentry.MockTransport{}
			logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true, TracingEnabled: tracing, TracesSampleRate: 0})
			handler := middleware.Logging(logger, catalogue)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/posts/did:plc:target/record?cursor=private-cursor", nil))
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/health", nil))
			observer.Flush(time.Second)
			var exported []sentry.Log
			for _, event := range transport.Events() {
				if len(event.Exception) > 0 || event.Type == "transaction" {
					t.Fatal("lifecycle implied capture or trace")
				}
				exported = append(exported, event.Logs...)
			}
			if len(exported) != 2 {
				t.Fatalf("export=%d want2", len(exported))
			}
			for _, record := range exported {
				if record.Level != sentry.LogLevelInfo || record.Attributes["incoming_path"].AsString() != "/v1/posts/did:plc:target/record" {
					t.Fatalf("lifecycle lost safe path/severity: %#v", record)
				}
			}
			data, _ := json.Marshal(exported)
			if strings.Contains(local.String()+string(data), "private-cursor") || strings.Contains(local.String(), "/health") {
				t.Fatal("query or routine health noise")
			}
		})
	}
}
