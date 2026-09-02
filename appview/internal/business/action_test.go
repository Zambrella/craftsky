package business

import (
	"errors"
	"reflect"
	"testing"
)

func TestActionCatalog(t *testing.T) {
	wantCatalog := []string{
		"shop",
		"browse-patterns",
		"request-custom-order",
		"book-class",
		"book-appointment",
		"view-event-calendar",
		"email",
		"visit-website",
		"wholesale-enquiries",
	}
	if got := ActionCatalog(); !reflect.DeepEqual(got, wantCatalog) {
		t.Fatalf("ActionCatalog() = %v, want %v", got, wantCatalog)
	}

	if err := ValidateActions(nil); err != nil {
		t.Fatalf("ValidateActions(nil): %v", err)
	}
	for _, actionType := range wantCatalog {
		if err := ValidateActions([]Action{{Type: actionType}}); err != nil {
			t.Errorf("ValidateActions(%q): %v", actionType, err)
		}
	}
	for name, actions := range map[string][]Action{
		"unknown":     {{Type: "unknown"}},
		"mixed case":  {{Type: "Shop"}},
		"two actions": {{Type: "shop"}, {Type: "email"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateActions(actions); !errors.Is(err, ErrInvalidActions) {
				t.Fatalf("ValidateActions(%v) error = %v, want ErrInvalidActions", actions, err)
			}
		})
	}
}
