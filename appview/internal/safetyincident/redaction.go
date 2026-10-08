package safetyincident

import (
	"encoding/json"
	"log/slog"
)

// RestrictedFields groups values that must never enter ordinary operational
// sinks. Keeping them nested prevents accidental JSON promotion from restricted
// storage models into safe read models.
type RestrictedFields struct {
	Content            string
	Hash               string
	Evidence           string
	Contact            string
	Credential         string
	InternalJudgment   string
	ProviderPayload    string
	EvidenceObjectKey  string
	AuthorityReference string
}

func (RestrictedFields) MarshalJSON() ([]byte, error) { return []byte(`{}`), nil }
func (RestrictedFields) String() string               { return "[REDACTED]" }
func (RestrictedFields) LogValue() slog.Value         { return slog.StringValue("[REDACTED]") }

type Sink string

const (
	SinkLog      Sink = "log"
	SinkMetric   Sink = "metric"
	SinkError    Sink = "error"
	SinkPush     Sink = "push"
	SinkOwnerAPI Sink = "ownerApi"
)

func (sink Sink) Valid() bool {
	switch sink {
	case SinkLog, SinkMetric, SinkError, SinkPush, SinkOwnerAPI:
		return true
	default:
		return false
	}
}

type RestrictedRecord struct {
	Reference string
	Kind      string
	State     string
	Priority  int
	Alert     bool
	Sensitive RestrictedFields
}

func (record RestrictedRecord) MarshalJSON() ([]byte, error) {
	projection, _ := RedactForSink(record, SinkOwnerAPI)
	return json.Marshal(projection)
}

func (record RestrictedRecord) LogValue() slog.Value {
	projection, _ := RedactForSink(record, SinkLog)
	return slog.GroupValue(
		slog.String("reference", projection.Reference),
		slog.String("kind", projection.Kind),
		slog.String("state", projection.State),
		slog.Int("priority", projection.Priority),
		slog.Bool("alert", projection.Alert),
	)
}

// SafeProjection is the complete allowlist shared by ordinary sinks. Restricted
// administrator APIs use their own purpose-specific read models.
type SafeProjection struct {
	Reference string `json:"reference"`
	Kind      string `json:"kind"`
	State     string `json:"state"`
	Priority  int    `json:"priority"`
	Alert     bool   `json:"alert"`
}

func RedactForSink(record RestrictedRecord, sink Sink) (SafeProjection, bool) {
	if !sink.Valid() {
		return SafeProjection{}, false
	}
	return SafeProjection{
		Reference: record.Reference,
		Kind:      record.Kind,
		State:     record.State,
		Priority:  record.Priority,
		Alert:     record.Alert,
	}, true
}
