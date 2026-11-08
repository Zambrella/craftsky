package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"social.craftsky/appview/internal/api/envelope"
)

func publicCursorScope(resource string, parameters any) string {
	raw, _ := json.Marshal(parameters)
	sum := sha256.Sum256(append([]byte(resource+":"), raw...))
	return hex.EncodeToString(sum[:])
}

func decodePublicCursor(cursor, scope string) (string, error) {
	if cursor == "" {
		return "", nil
	}
	if len(cursor) > 2048 {
		return "", envelope.ErrInvalidCursor
	}
	payload, err := envelope.DecodeCursor(cursor)
	if err != nil || len(payload) != 3 || payload["audience"] != "guest" || payload["scope"] != scope {
		return "", envelope.ErrInvalidCursor
	}
	inner, ok := payload["cursor"].(string)
	if !ok || inner == "" {
		return "", envelope.ErrInvalidCursor
	}
	return inner, nil
}

func encodePublicCursor(cursor, scope string) (string, error) {
	if cursor == "" {
		return "", nil
	}
	return envelope.EncodeCursor(map[string]any{"audience": "guest", "scope": scope, "cursor": cursor})
}

func bindPublicPageCursor(page any, scope string) error {
	var cursor *string
	switch result := page.(type) {
	case *PublicPostPage:
		cursor = &result.Cursor
	case *PublicProfilePage:
		cursor = &result.Cursor
	case *PublicHashtagPage:
		cursor = &result.Cursor
	default:
		return nil
	}
	encoded, err := encodePublicCursor(*cursor, scope)
	if err == nil {
		*cursor = encoded
	}
	return err
}
