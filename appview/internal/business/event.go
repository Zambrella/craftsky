package business

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

var ErrInvalidEventSource = errors.New("business: invalid event source")

type EventView struct {
	DID                      syntax.DID       `json:"did"`
	Rkey                     syntax.RecordKey `json:"rkey"`
	URI                      syntax.ATURI     `json:"uri"`
	CID                      syntax.CID       `json:"cid"`
	Name                     string           `json:"name"`
	StartsAt                 string           `json:"startsAt"`
	EndsAt                   string           `json:"endsAt"`
	Roles                    []OpenValue      `json:"roles"`
	Mode                     *OpenValue       `json:"mode,omitempty"`
	Status                   OpenValue        `json:"status"`
	TimeZone                 string           `json:"timeZone,omitempty"`
	IsAllDay                 bool             `json:"isAllDay"`
	Summary                  string           `json:"summary,omitempty"`
	VenueName                string           `json:"venueName,omitempty"`
	EventURI                 string           `json:"eventUri,omitempty"`
	RegistrationURI          string           `json:"registrationUri,omitempty"`
	Image                    json.RawMessage  `json:"image,omitempty"`
	CreatedAt                string           `json:"createdAt"`
	Past                     bool             `json:"past"`
	PublicSuppressionReasons []string         `json:"publicSuppressionReasons"`
	UpcomingExclusionReasons []string         `json:"upcomingExclusionReasons"`
}

func HydrateEvent(raw json.RawMessage) (EventView, error) {
	var source struct {
		Name            string          `json:"name"`
		StartsAt        string          `json:"startsAt"`
		EndsAt          string          `json:"endsAt"`
		Roles           []string        `json:"roles"`
		Mode            *string         `json:"mode"`
		Status          *string         `json:"status"`
		TimeZone        *string         `json:"timeZone"`
		IsAllDay        *bool           `json:"isAllDay"`
		Summary         string          `json:"summary"`
		VenueName       string          `json:"venueName"`
		EventURI        string          `json:"eventUri"`
		RegistrationURI string          `json:"registrationUri"`
		Image           json.RawMessage `json:"image"`
		CreatedAt       string          `json:"createdAt"`
	}
	if err := json.Unmarshal(raw, &source); err != nil || source.Name == "" || source.StartsAt == "" || source.EndsAt == "" || source.CreatedAt == "" {
		return EventView{}, ErrInvalidEventSource
	}
	defaults := HydrateIndependentEventDefaults(source.Status, source.Mode, source.TimeZone, source.IsAllDay)
	view := EventView{
		Name: source.Name, StartsAt: source.StartsAt, EndsAt: source.EndsAt,
		Roles:    ClassifyIndependentRoles(source.Roles),
		Status:   ClassifyOpenValue(defaults.Status, eventStatusCatalog),
		IsAllDay: defaults.IsAllDay, CreatedAt: source.CreatedAt,
		PublicSuppressionReasons: []string{}, UpcomingExclusionReasons: []string{},
	}
	if defaults.Mode != nil {
		mode := ClassifyOpenValue(*defaults.Mode, eventModeCatalog)
		view.Mode = &mode
	}
	if defaults.TimeZone != nil && validTimeZone(*defaults.TimeZone) {
		view.TimeZone = *defaults.TimeZone
	}
	if ValidateText(TextFieldEventSummary, source.Summary) == nil {
		view.Summary = source.Summary
	}
	if ValidateText(TextFieldVenueName, source.VenueName) == nil {
		view.VenueName = source.VenueName
	}
	destinations := HydrateIndependentEventDestinations(source.EventURI, source.RegistrationURI)
	view.EventURI = destinations.EventURI
	view.RegistrationURI = destinations.RegistrationURI
	if safeIndependentImage(source.Image) {
		view.Image = append(json.RawMessage(nil), source.Image...)
	}
	return view, nil
}

func validTimeZone(value string) bool {
	if value == "UTC" {
		return true
	}
	_, err := time.LoadLocation(value)
	return err == nil
}

func safeIndependentImage(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	decoded, err := atdata.UnmarshalJSON(raw)
	if err != nil {
		return false
	}
	source := decoded
	blob, ok := source["image"].(atdata.Blob)
	if !ok {
		return false
	}
	alt, _ := source["alt"].(string)
	var aspectRatio *AspectRatio
	if rawAspect, exists := source["aspectRatio"]; exists {
		aspect, ok := rawAspect.(map[string]any)
		width, widthOK := aspect["width"].(int64)
		height, heightOK := aspect["height"].(int64)
		if !ok || !widthOK || !heightOK {
			return false
		}
		aspectRatio = &AspectRatio{Width: width, Height: height}
	}
	return ValidateProductImage(&Image{
		MIMEType: blob.MimeType, Size: blob.Size,
		Alt: alt, AspectRatio: aspectRatio,
	}, false) == nil
}
