package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
)

var ErrInvalidEventLimit = errors.New("invalid event limit")

const eventCursorKeyBytes = 32

type EventCursorCodec struct {
	key []byte
}

func NewEventCursorCodec(key []byte) (*EventCursorCodec, error) {
	if len(key) < eventCursorKeyBytes {
		return nil, errors.New("event cursor key must contain at least 32 bytes")
	}
	return &EventCursorCodec{key: append([]byte(nil), key...)}, nil
}

type EventCursorKind string

const (
	EventCursorUpcoming   EventCursorKind = "business-event-upcoming"
	EventCursorManagement EventCursorKind = "business-event-management"
)

type EventCursor struct {
	Kind     EventCursorKind
	AsOf     time.Time
	StartsAt time.Time
	URI      syntax.ATURI
}

func (codec *EventCursorCodec) Encode(cursor EventCursor, scope syntax.DID) (string, error) {
	payload := map[string]any{
		"kind":     string(cursor.Kind),
		"startsAt": cursor.StartsAt.UTC().Format(time.RFC3339Nano),
		"uri":      string(cursor.URI),
	}
	if cursor.Kind == EventCursorUpcoming {
		payload["asOf"] = cursor.AsOf.UTC().Format(time.RFC3339Nano)
	}
	encoded, err := envelope.EncodeCursor(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, codec.key)
	_, _ = mac.Write([]byte("business-event-cursor\x00"))
	_, _ = mac.Write([]byte(scope))
	_, _ = mac.Write([]byte("\x00" + encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (codec *EventCursorCodec) Decode(encoded string, expectedKind EventCursorKind, scope syntax.DID) (EventCursor, error) {
	parts := strings.Split(encoded, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return EventCursor{}, envelope.ErrInvalidCursor
	}
	provided, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return EventCursor{}, envelope.ErrInvalidCursor
	}
	mac := hmac.New(sha256.New, codec.key)
	_, _ = mac.Write([]byte("business-event-cursor\x00"))
	_, _ = mac.Write([]byte(scope))
	_, _ = mac.Write([]byte("\x00" + parts[0]))
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return EventCursor{}, envelope.ErrInvalidCursor
	}
	payload, err := envelope.DecodeCursor(parts[0])
	if err != nil || payload["kind"] != string(expectedKind) {
		return EventCursor{}, envelope.ErrInvalidCursor
	}
	startsAt, ok := parseCursorTime(payload["startsAt"])
	if !ok {
		return EventCursor{}, envelope.ErrInvalidCursor
	}
	rawURI, ok := payload["uri"].(string)
	if !ok || rawURI == "" {
		return EventCursor{}, envelope.ErrInvalidCursor
	}
	uri, err := syntax.ParseATURI(rawURI)
	if err != nil {
		return EventCursor{}, envelope.ErrInvalidCursor
	}
	cursor := EventCursor{Kind: expectedKind, StartsAt: startsAt, URI: uri}
	if expectedKind == EventCursorUpcoming {
		asOf, ok := parseCursorTime(payload["asOf"])
		if !ok {
			return EventCursor{}, envelope.ErrInvalidCursor
		}
		cursor.AsOf = asOf
	}
	return cursor, nil
}

func NormalizeEventLimit(raw string, defaultLimit int) (int, error) {
	if raw == "" {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 {
		return 0, ErrInvalidEventLimit
	}
	if limit > 50 {
		return 50, nil
	}
	return limit, nil
}

func parseCursorTime(value any) (time.Time, bool) {
	raw, ok := value.(string)
	if !ok || raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	return parsed, err == nil
}
