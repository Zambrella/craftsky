package observability

import (
	"context"

	"github.com/getsentry/sentry-go"
)

// The SDK batches logs/metrics on its own goroutines. A caller-level recovery
// cannot guard those sends. Wrap explicitly supplied transports only, leaving
// the SDK's default transport and telemetry processor selection unchanged.
type guardedTransport struct {
	sentry.Transport
	observer *Observer
}

func (t guardedTransport) SendEvent(event *sentry.Event) {
	defer func() {
		if recover() == nil {
			return
		}
		message := "telemetry export failed"
		fields := EventContext{"component": "telemetry", "operation": "telemetry.export", "result": "error"}
		var causes []DiagnosticCause
		if len(event.Exception) > 0 {
			message = "telemetry capture failed"
			// BeforeSend has already replaced all issue fields with selected data.
			for key, value := range event.Tags {
				fields[key] = value
			}
			for key, value := range event.Contexts["correlation"] {
				fields[key] = value
			}
			for _, exception := range event.Exception {
				causes = append(causes, DiagnosticCause{Type: exception.Type, Message: exception.Value})
			}
		}
		t.observer.localFailure(context.Background(), message, fields, causes)
	}()
	t.Transport.SendEvent(event)
}
