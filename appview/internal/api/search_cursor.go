package api

import (
	"time"

	"social.craftsky/appview/internal/api/envelope"
)

type PopularityCursor struct {
	RankedAt  time.Time
	Score     float64
	CreatedAt time.Time
	URI       string
}

func (c PopularityCursor) ScorePtr() any {
	if c.URI == "" {
		return nil
	}
	return c.Score
}
func (c PopularityCursor) CreatedAtPtr() any {
	if c.URI == "" {
		return nil
	}
	return c.CreatedAt
}
func (c PopularityCursor) URIPtr() any {
	if c.URI == "" {
		return nil
	}
	return c.URI
}

func EncodeChronologicalSearchCursor(createdAt time.Time, uri string) (string, error) {
	return envelope.EncodeCursor(map[string]any{
		"sort":      string(SearchSortChronological),
		"createdAt": createdAt.UTC().Format(time.RFC3339Nano),
		"uri":       uri,
	})
}

func DecodeChronologicalSearchCursor(cursor string) (any, any, error) {
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil || cursor == "" {
		return nil, nil, err
	}
	if cur["sort"] != string(SearchSortChronological) {
		return nil, nil, envelope.ErrInvalidCursor
	}
	createdAt, ok := cur["createdAt"].(string)
	if !ok || createdAt == "" {
		return nil, nil, envelope.ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, nil, envelope.ErrInvalidCursor
	}
	uri, ok := cur["uri"].(string)
	if !ok || uri == "" {
		return nil, nil, envelope.ErrInvalidCursor
	}
	return parsed, uri, nil
}

func EncodePopularityCursor(rankedAt time.Time, score float64, createdAt time.Time, uri string) (string, error) {
	return envelope.EncodeCursor(map[string]any{
		"sort":      string(SearchSortPopular),
		"rankedAt":  rankedAt.UTC().Format(time.RFC3339Nano),
		"score":     score,
		"createdAt": createdAt.UTC().Format(time.RFC3339Nano),
		"uri":       uri,
	})
}

func DecodePopularityCursor(cursor string) (PopularityCursor, error) {
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil || cursor == "" {
		return PopularityCursor{}, err
	}
	if cur["sort"] != string(SearchSortPopular) {
		return PopularityCursor{}, envelope.ErrInvalidCursor
	}
	rankedAt, ok := cur["rankedAt"].(string)
	if !ok || rankedAt == "" {
		return PopularityCursor{}, envelope.ErrInvalidCursor
	}
	parsedRankedAt, err := time.Parse(time.RFC3339Nano, rankedAt)
	if err != nil {
		return PopularityCursor{}, envelope.ErrInvalidCursor
	}
	createdAt, ok := cur["createdAt"].(string)
	if !ok || createdAt == "" {
		return PopularityCursor{}, envelope.ErrInvalidCursor
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return PopularityCursor{}, envelope.ErrInvalidCursor
	}
	score, ok := cur["score"].(float64)
	if !ok {
		return PopularityCursor{}, envelope.ErrInvalidCursor
	}
	uri, ok := cur["uri"].(string)
	if !ok || uri == "" {
		return PopularityCursor{}, envelope.ErrInvalidCursor
	}
	return PopularityCursor{RankedAt: parsedRankedAt, Score: score, CreatedAt: parsedCreatedAt, URI: uri}, nil
}
