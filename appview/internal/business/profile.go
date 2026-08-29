package business

import (
	"encoding/json"
	"errors"
)

const businessProfileType = "social.craftsky.business.profile"

var ErrInvalidProfile = errors.New("business: invalid profile")

var profileKnownFields = map[string]struct{}{
	"businessTypes": {},
	"offerings":     {},
	"tagline":       {},
	"hoursNote":     {},
	"serviceArea":   {},
	"location":      {},
	"primaryAction": {},
	"products":      {},
}

type ProfileView struct {
	BusinessTypes []OpenValue   `json:"businessTypes,omitempty"`
	Offerings     []OpenValue   `json:"offerings,omitempty"`
	Tagline       string        `json:"tagline,omitempty"`
	HoursNote     string        `json:"hoursNote,omitempty"`
	ServiceArea   string        `json:"serviceArea,omitempty"`
	Location      *Location     `json:"location,omitempty"`
	PrimaryAction *Action       `json:"primaryAction,omitempty"`
	Products      []ProductView `json:"products,omitempty"`
}

type ProductView struct {
	Title string          `json:"title"`
	URI   string          `json:"uri,omitempty"`
	Image json.RawMessage `json:"image,omitempty"`
	Price *Price          `json:"price,omitempty"`
}

func MergeProfileReplacement(existingRaw, replacementKnown json.RawMessage) (json.RawMessage, error) {
	var existing map[string]json.RawMessage
	if err := json.Unmarshal(existingRaw, &existing); err != nil || existing == nil {
		return nil, ErrInvalidProfile
	}
	var replacement map[string]json.RawMessage
	if err := json.Unmarshal(replacementKnown, &replacement); err != nil || replacement == nil {
		return nil, ErrInvalidProfile
	}
	for field := range profileKnownFields {
		delete(existing, field)
	}
	for field, value := range replacement {
		if _, known := profileKnownFields[field]; known {
			existing[field] = value
		}
	}
	existing["$type"] = json.RawMessage(`"` + businessProfileType + `"`)
	merged, err := json.Marshal(existing)
	if err != nil {
		return nil, ErrInvalidProfile
	}
	return merged, nil
}

func HydrateProfile(raw json.RawMessage) (ProfileView, error) {
	var source struct {
		BusinessTypes []string `json:"businessTypes"`
		Offerings     []string `json:"offerings"`
		Tagline       string   `json:"tagline"`
		HoursNote     string   `json:"hoursNote"`
		ServiceArea   string   `json:"serviceArea"`
		Location      *struct {
			Country  string  `json:"country"`
			Locality *string `json:"locality"`
		} `json:"location"`
		PrimaryAction *Action `json:"primaryAction"`
		Products      []struct {
			Title string          `json:"title"`
			URI   string          `json:"uri"`
			Image json.RawMessage `json:"image"`
			Price *Price          `json:"price"`
		} `json:"products"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return ProfileView{}, ErrInvalidProfile
	}
	var view ProfileView
	view.BusinessTypes = classifyOpenCatalog(source.BusinessTypes, businessTypeCatalog)
	view.Offerings = classifyOpenCatalog(source.Offerings, offeringCatalog)
	if ValidateText(TextFieldTagline, source.Tagline) == nil {
		view.Tagline = source.Tagline
	}
	if ValidateText(TextFieldHoursNote, source.HoursNote) == nil {
		view.HoursNote = source.HoursNote
	}
	if ValidateText(TextFieldServiceArea, source.ServiceArea) == nil {
		view.ServiceArea = source.ServiceArea
	}
	if source.Location != nil {
		view.Location = HydrateIndependentLocation(source.Location.Country, source.Location.Locality)
	}
	if source.PrimaryAction != nil && catalogContains(actionCatalog, source.PrimaryAction.Type) {
		valid := ValidateWebDestination(source.PrimaryAction.Destination)
		if source.PrimaryAction.Type == "email" {
			valid = ValidateMailtoDestination(source.PrimaryAction.Destination)
		}
		if valid == nil {
			action := *source.PrimaryAction
			view.PrimaryAction = &action
		}
	}
	for _, product := range source.Products {
		if ValidateText(TextFieldProductTitle, product.Title) != nil {
			continue
		}
		hydrated := ProductView{Title: product.Title, Price: HydrateIndependentPrice(product.Price)}
		if ValidateWebDestination(product.URI) == nil {
			hydrated.URI = product.URI
		}
		if safeIndependentImage(product.Image) {
			hydrated.Image = append(json.RawMessage(nil), product.Image...)
		}
		view.Products = append(view.Products, hydrated)
	}
	return view, nil
}

func classifyOpenCatalog(values, catalog []string) []OpenValue {
	seen := make(map[string]struct{}, len(values))
	selected := make(map[string]struct{}, len(values))
	unknown := make([]string, 0)
	for _, value := range values {
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		if catalogContains(catalog, value) {
			selected[value] = struct{}{}
		} else {
			unknown = append(unknown, value)
		}
	}
	classified := make([]OpenValue, 0, len(seen))
	for _, value := range catalog {
		if _, ok := selected[value]; ok {
			classified = append(classified, OpenValue{Value: value, Known: true})
		}
	}
	for _, value := range unknown {
		classified = append(classified, OpenValue{Value: value})
	}
	return classified
}
