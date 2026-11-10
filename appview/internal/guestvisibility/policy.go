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

func (preference Preference) AllowsGuest() bool { return preference != Hidden }

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
	hidden, valid := parseSelfLabels(raw)
	if hidden {
		return Hidden
	}
	if !valid {
		return Unknown
	}
	return Public
}

// An explicit opt-out wins even when another label is not understood. Validity
// remains separate so preference writes never replace an unknown label payload.
func parseSelfLabels(raw json.RawMessage) (hidden, valid bool) {
	var labels struct {
		Type   json.RawMessage   `json:"$type"`
		Values []json.RawMessage `json:"values"`
	}
	if json.Unmarshal(raw, &labels) != nil {
		return false, false
	}
	var labelType string
	valid = json.Unmarshal(labels.Type, &labelType) == nil && labelType == "com.atproto.label.defs#selfLabels" && labels.Values != nil
	for _, rawLabel := range labels.Values {
		var label struct {
			Value string `json:"val"`
		}
		if json.Unmarshal(rawLabel, &label) != nil || label.Value == "" {
			valid = false
			continue
		}
		if label.Value == Label {
			hidden = true
		}
	}
	return hidden, valid
}
