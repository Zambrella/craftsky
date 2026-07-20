package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
)

const (
	defaultRelationshipLimit = 50
	maxRelationshipLimit     = 100
	relationshipCursorKind   = "relationship"
)

var ErrInvalidRelationshipLimit = errors.New("invalid relationship limit")

type RelationshipListRequest struct {
	Limit        int
	Cursor       string
	AfterCreated time.Time
	AfterSubject syntax.DID
}

func ParseRelationshipListRequest(r *http.Request) (RelationshipListRequest, error) {
	limit := defaultRelationshipLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxRelationshipLimit {
			return RelationshipListRequest{}, ErrInvalidRelationshipLimit
		}
		limit = parsed
	}

	cursor := r.URL.Query().Get("cursor")
	createdAt, subject, err := DecodeRelationshipCursor(cursor)
	if err != nil {
		return RelationshipListRequest{}, err
	}
	return RelationshipListRequest{
		Limit:        limit,
		Cursor:       cursor,
		AfterCreated: createdAt,
		AfterSubject: subject,
	}, nil
}

func EncodeRelationshipCursor(createdAt time.Time, subject syntax.DID) (string, error) {
	return envelope.EncodeCursor(map[string]any{
		"kind": relationshipCursorKind,
		"at":   createdAt.UTC().Format(time.RFC3339Nano),
		"did":  subject.String(),
	})
}

func DecodeRelationshipCursor(cursor string) (time.Time, syntax.DID, error) {
	if cursor == "" {
		return time.Time{}, "", nil
	}
	payload, err := envelope.DecodeCursor(cursor)
	if err != nil {
		return time.Time{}, "", err
	}
	kind, kindOK := payload["kind"].(string)
	rawAt, atOK := payload["at"].(string)
	rawDID, didOK := payload["did"].(string)
	if !kindOK || kind != relationshipCursorKind || !atOK || !didOK {
		return time.Time{}, "", envelope.ErrInvalidCursor
	}
	createdAt, err := time.Parse(time.RFC3339Nano, rawAt)
	if err != nil {
		return time.Time{}, "", envelope.ErrInvalidCursor
	}
	subject, err := syntax.ParseDID(rawDID)
	if err != nil {
		return time.Time{}, "", envelope.ErrInvalidCursor
	}
	return createdAt, subject, nil
}
