package tap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ReplayEnvelope reclassifies one bounded, operator-selected Tap envelope
// through the same typed decoder and durable ingestor as the live consumer. It
// deliberately has no WebSocket or ACK capability and does not quarantine a
// still-invalid envelope again: returning a retryable outcome leaves the
// existing quarantine row pending for a later code or data correction.
func ReplayEnvelope(ctx context.Context, raw []byte, ingestor DurableIngestor) (Outcome, error) {
	if ingestor == nil {
		return Retryable(ReasonDurableIngestorRequired), errors.New("tap replay: durable ingestor is required")
	}
	if len(raw) == 0 || len(raw) > MaxFrameBytes {
		return Retryable(ReasonInvalidEnvelope), errors.New("tap replay: envelope size is invalid")
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return Retryable(ReasonInvalidEnvelope), fmt.Errorf("tap replay: decode envelope: %w", err)
	}
	if env.ID == 0 {
		return Retryable(ReasonInvalidEnvelope), errors.New("tap replay: envelope has no event id")
	}
	switch env.Type {
	case "record":
		if env.Record == nil {
			return Retryable(ReasonMissingRecord), errors.New("tap replay: record envelope has no record")
		}
		event, reason, err := decodeRecordEvent(env)
		if err != nil {
			return Retryable(reason), fmt.Errorf("tap replay: decode record: %w", err)
		}
		return ingestor.IngestRecord(ctx, event)
	case "identity":
		event, reason, err := decodeIdentityEvent(env)
		if err != nil {
			return Retryable(reason), fmt.Errorf("tap replay: decode identity: %w", err)
		}
		return ingestor.IngestIdentity(ctx, event)
	default:
		return Retryable(ReasonUnsupportedEventType), fmt.Errorf("tap replay: unsupported event type %q", env.Type)
	}
}
