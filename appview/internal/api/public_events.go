package api

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/business"
)

// PublicEvent deliberately omits owner suppression and management metadata.
type PublicEvent struct {
	URI             syntax.ATURI         `json:"uri"`
	CID             syntax.CID           `json:"cid"`
	DID             syntax.DID           `json:"did"`
	Rkey            syntax.RecordKey     `json:"rkey"`
	Author          PublicAuthor         `json:"author"`
	Name            string               `json:"name"`
	StartsAt        string               `json:"startsAt"`
	EndsAt          string               `json:"endsAt"`
	Roles           []business.OpenValue `json:"roles"`
	Mode            *business.OpenValue  `json:"mode,omitempty"`
	Status          business.OpenValue   `json:"status"`
	TimeZone        string               `json:"timeZone,omitempty"`
	IsAllDay        bool                 `json:"isAllDay"`
	Summary         string               `json:"summary,omitempty"`
	VenueName       string               `json:"venueName,omitempty"`
	EventURI        string               `json:"eventUri,omitempty"`
	RegistrationURI string               `json:"registrationUri,omitempty"`
	Image           *BusinessImageView   `json:"image,omitempty"`
	CreatedAt       string               `json:"createdAt"`
	Past            bool                 `json:"past"`
}

type PublicEventPage struct {
	Items  []PublicEvent `json:"items"`
	Cursor string        `json:"cursor,omitempty"`
}

type publicEventReader interface {
	ReadEvent(context.Context, business.EventReadInput) (business.EventView, error)
	ListUpcomingEvents(context.Context, business.UpcomingEventListInput) ([]business.EventView, error)
}

func publicEvent(view business.EventView, author PublicAuthor) PublicEvent {
	return PublicEvent{URI: view.URI, CID: view.CID, DID: view.DID, Rkey: view.Rkey, Author: author, Name: view.Name, StartsAt: view.StartsAt, EndsAt: view.EndsAt, Roles: view.Roles, Mode: view.Mode, Status: view.Status, TimeZone: view.TimeZone, IsAllDay: view.IsAllDay, Summary: view.Summary, VenueName: view.VenueName, EventURI: view.EventURI, RegistrationURI: view.RegistrationURI, Image: buildBusinessImageView(view.DID, view.Image), CreatedAt: view.CreatedAt, Past: view.Past}
}

func (s *PublicPageStore) PublicEvent(ctx context.Context, did syntax.DID, key syntax.RecordKey) (*PublicEvent, error) {
	profile, err := s.PublicProfile(ctx, did)
	if err != nil {
		return nil, err
	}
	store, ok := s.accountTypes.(publicEventReader)
	if !ok {
		return nil, ErrPostNotFound
	}
	event, err := store.ReadEvent(ctx, business.EventReadInput{OwnerDID: did, Rkey: key, AsOf: time.Now().UTC(), Guest: true})
	if errors.Is(err, business.ErrEventNotFound) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}
	out := publicEvent(event, profile.PublicAuthor)
	return &out, nil
}

func (s *PublicPageStore) PublicEvents(ctx context.Context, did syntax.DID, limit int, cursor string) (*PublicEventPage, error) {
	if limit < 1 || limit > 20 {
		return nil, ErrSearchValidation
	}
	profile, err := s.PublicProfile(ctx, did)
	if err != nil {
		return nil, err
	}
	store, ok := s.accountTypes.(publicEventReader)
	if !ok {
		return nil, ErrPostNotFound
	}
	scope := publicCursorScope("events/"+did.String(), nil)
	inner, err := decodePublicCursor(cursor, scope)
	if err != nil {
		return nil, err
	}
	var seek *business.UpcomingEventSeek
	if inner != "" {
		at, uri, err := decodeSeekCursor(inner, "startsAt")
		if err != nil {
			return nil, err
		}
		if at == nil || uri == nil {
			return nil, envelope.ErrInvalidCursor
		}
		seek = &business.UpcomingEventSeek{StartsAt: at.(time.Time), URI: syntax.ATURI(uri.(string))}
	}
	rows, err := store.ListUpcomingEvents(ctx, business.UpcomingEventListInput{OwnerDID: did, AsOf: time.Now().UTC(), Limit: limit, Seek: seek, Guest: true})
	if err != nil {
		return nil, err
	}
	out := &PublicEventPage{Items: []PublicEvent{}}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	for _, row := range rows {
		out.Items = append(out.Items, publicEvent(row, profile.PublicAuthor))
	}
	if more {
		last := rows[len(rows)-1]
		inner, err := envelope.EncodeCursor(map[string]any{"startsAt": last.StartsAt, "uri": last.URI.String()})
		if err != nil {
			return nil, err
		}
		out.Cursor, err = encodePublicCursor(inner, scope)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
