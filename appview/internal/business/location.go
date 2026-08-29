package business

import (
	"errors"
	"strings"
)

var ErrInvalidLocation = errors.New("business: invalid location")

type Location struct {
	Country  string  `json:"country"`
	Locality *string `json:"locality,omitempty"`
}

func NormalizeCountry(value string) (string, error) {
	normalized := strings.ToUpper(value)
	if len(value) != 2 {
		return "", ErrInvalidLocation
	}
	if _, assigned := assignedCountryCodes[normalized]; !assigned {
		return "", ErrInvalidLocation
	}
	return normalized, nil
}

func HydrateIndependentLocation(country string, locality *string) *Location {
	normalized, err := NormalizeCountry(country)
	if err != nil {
		return nil
	}
	hydrated := &Location{Country: normalized}
	if locality != nil && ValidateText(TextFieldLocality, *locality) == nil {
		hydrated.Locality = locality
	}
	return hydrated
}
