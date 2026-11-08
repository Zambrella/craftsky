// Package guestvisibility applies the shared profile preference to CraftSky's
// anonymous read boundary. Public repository storage is unaffected.
package guestvisibility

import "encoding/json"

type Preference uint8

const (
	Unknown Preference = iota
	Public
	Hidden
)

const Label = "!no-unauthenticated"

func (preference Preference) AllowsGuest() bool { return preference == Public }

// ParsePreference interprets an authoritative profile record. The caller must
// establish its ordering evidence before this value can become serving state.
func ParsePreference(record json.RawMessage) Preference {
	var fields map[string]json.RawMessage
	if json.Unmarshal(record, &fields) != nil || fields == nil {
		return Unknown
	}
	raw, exists := fields["labels"]
	if !exists {
		return Public
	}
	var labels struct {
		Type   string `json:"$type"`
		Values []struct {
			Value string `json:"val"`
		} `json:"values"`
	}
	if json.Unmarshal(raw, &labels) != nil || labels.Type != "com.atproto.label.defs#selfLabels" || labels.Values == nil {
		return Unknown
	}
	preference := Public
	for _, label := range labels.Values {
		if label.Value == "" {
			return Unknown
		}
		if label.Value == Label {
			preference = Hidden
		}
	}
	return preference
}
