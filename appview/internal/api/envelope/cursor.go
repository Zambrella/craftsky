package envelope

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// ErrInvalidCursor is returned by DecodeCursor when the input is not a
// valid base64url-encoded JSON object. Handlers should map this to a
// 400 with error code "invalid_cursor".
var ErrInvalidCursor = errors.New("invalid cursor")

// EncodeCursor serialises payload as base64url-encoded JSON. Handlers
// use it to produce the "cursor" field on paginated responses. The
// format is deliberately opaque — clients must not inspect it.
func EncodeCursor(payload map[string]any) (string, error) {
	if len(payload) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeCursor is the inverse of EncodeCursor. An empty input returns
// an empty map (clients omit the cursor on the first page). Malformed
// input returns ErrInvalidCursor.
func DecodeCursor(cursor string) (map[string]any, error) {
	if cursor == "" {
		return map[string]any{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, ErrInvalidCursor
	}
	return out, nil
}
