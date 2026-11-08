package guestvisibility

import (
	"encoding/json"
	"errors"
)

// SetPreference changes only the shared visibility self-label. Raw fields avoid
// rounding extension numbers or discarding fields unknown to this AppView.
func SetPreference(record json.RawMessage, hidden bool) (json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if len(record) > 0 {
		if err := json.Unmarshal(record, &fields); err != nil || fields == nil {
			return nil, errors.New("profile record is not an object")
		}
	}
	if fields["$type"] == nil {
		fields["$type"] = json.RawMessage(`"app.bsky.actor.profile"`)
	}
	labels := map[string]json.RawMessage{
		"$type":  json.RawMessage(`"com.atproto.label.defs#selfLabels"`),
		"values": json.RawMessage(`[]`),
	}
	if raw, exists := fields["labels"]; exists {
		if ParsePreference(record) == Unknown || json.Unmarshal(raw, &labels) != nil {
			return nil, errors.New("profile self-labels are not understood")
		}
	}
	var values []json.RawMessage
	if err := json.Unmarshal(labels["values"], &values); err != nil {
		return nil, err
	}
	retained := make([]json.RawMessage, 0, len(values)+1)
	for _, raw := range values {
		var label struct {
			Value string `json:"val"`
		}
		if err := json.Unmarshal(raw, &label); err != nil {
			return nil, err
		}
		if label.Value != Label {
			retained = append(retained, raw)
		}
	}
	if hidden {
		retained = append(retained, json.RawMessage(`{"val":"!no-unauthenticated"}`))
	}
	var err error
	labels["values"], err = json.Marshal(retained)
	if err != nil {
		return nil, err
	}
	fields["labels"], err = json.Marshal(labels)
	if err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}
