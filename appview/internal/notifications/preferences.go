package notifications

import "fmt"

type Scope string

const (
	Everyone      Scope = "everyone"
	PeopleIFollow Scope = "peopleIFollow"
)

func (s Scope) Valid() bool {
	return s == Everyone || s == PeopleIFollow
}

type Preference struct {
	Scope       Scope `json:"scope"`
	PushEnabled bool  `json:"pushEnabled"`
}

type PreferencePatch struct {
	Scope       *Scope `json:"scope,omitempty"`
	PushEnabled *bool  `json:"pushEnabled,omitempty"`
}

var defaultPreference = Preference{Scope: Everyone, PushEnabled: true}

// ResolvePreferences returns all effective category values after applying a
// partial patch. It validates the complete input before returning any state so
// callers can persist the patch atomically.
func ResolvePreferences(persisted map[Category]Preference, patch map[Category]PreferencePatch) (map[Category]Preference, error) {
	for category, preference := range persisted {
		if !category.Valid() {
			return nil, fmt.Errorf("invalid notification category %q", category)
		}
		if !preference.Scope.Valid() {
			return nil, fmt.Errorf("invalid notification scope %q", preference.Scope)
		}
	}
	for category, update := range patch {
		if !category.Valid() {
			return nil, fmt.Errorf("invalid notification category %q", category)
		}
		if update.Scope != nil && !update.Scope.Valid() {
			return nil, fmt.Errorf("invalid notification scope %q", *update.Scope)
		}
	}

	resolved := make(map[Category]Preference, len(categories))
	for _, category := range categories {
		preference := defaultPreference
		if saved, ok := persisted[category]; ok {
			preference = saved
		}
		if update, ok := patch[category]; ok {
			if update.Scope != nil {
				preference.Scope = *update.Scope
			}
			if update.PushEnabled != nil {
				preference.PushEnabled = *update.PushEnabled
			}
		}
		resolved[category] = preference
	}
	return resolved, nil
}
