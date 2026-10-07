package middleware

import (
	"errors"
	"github.com/getsentry/sentry-go"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/testlog"
	"testing"
	"time"
)

// IT-003: real recovery/metrics chain, response bytes and owner count.
func TestRecoveryAndFallbackOccurrenceOwnership(t *testing.T) {
	for _, mode := range []string{"captured", "fallback", "panic", "started_panic"} {
		t.Run(mode, func(t *testing.T) {
			transport := &sentry.MockTransport{}
			observer := observability.New(observability.Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			mux := http.NewServeMux()
			mux.HandleFunc("GET /v1/failure", func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "captured":
					observability.CaptureRequestDiagnostic(r.Context(), observability.DiagnosticInput{Error: errors.New("opaque private cause"), Context: observability.EventContext{"operation": "post.read"}})
					w.WriteHeader(500)
				case "fallback":
					w.WriteHeader(500)
				case "panic":
					panic("opaque private panic")
				case "started_panic":
					w.Write([]byte("started"))
					panic("opaque private panic")
				}
			})
			chain := Logging(testlog.Discard())(HTTPMetrics(observer)(Recovery(testlog.Discard(), observer)(mux)))
			for i := 0; i < 2; i++ {
				response := httptest.NewRecorder()
				chain.ServeHTTP(response, httptest.NewRequest("GET", "/v1/failure", nil))
				if mode == "started_panic" && (response.Code != 200 || response.Body.String() != "started") {
					t.Fatalf("rewrote started response: %d %q", response.Code, response.Body.String())
				}
			}
			observer.Flush(time.Second)
			events := transport.Events()
			if len(events) != 2 {
				t.Fatalf("%d issues, want independent occurrences2", len(events))
			}
			for _, event := range events {
				if mode == "panic" || mode == "started_panic" {
					if event.Exception[0].Stacktrace == nil || len(event.Exception[0].Stacktrace.Frames) == 0 {
						t.Fatal("missing recovery stack")
					}
				} else if event.Exception[0].Stacktrace != nil {
					t.Fatal("fabricated ordinary error stack")
				}
			}
		})
	}
}
