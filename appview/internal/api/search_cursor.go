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

type ProfileCursor struct {
	FollowedRank  int
	RelevanceRank int
	HandleLower   string
	DID           string
}

type HashtagCursor struct {
	Query  string
	Offset int
}

type RelevanceCursor struct {
	Kind      string
	Query     string
	Score     float64
	CreatedAt time.Time
	URI       string
}

func (c RelevanceCursor) ScorePtr() any {
	if c.URI == "" {
		return nil
	}
	return c.Score
}
func (c RelevanceCursor) CreatedAtPtr() any {
	if c.URI == "" {
		return nil
	}
	return c.CreatedAt
}
func (c RelevanceCursor) URIPtr() any {
	if c.URI == "" {
		return nil
	}
	return c.URI
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

func EncodeProfileSearchCursor(followedRank, relevanceRank int, handleLower, did string) (string, error) {
	return envelope.EncodeCursor(map[string]any{
		"kind":          "profile",
		"followedRank":  followedRank,
		"relevanceRank": relevanceRank,
		"handleLower":   handleLower,
		"did":           did,
	})
}

func DecodeProfileSearchCursor(cursor string) (ProfileCursor, error) {
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil || cursor == "" {
		return ProfileCursor{}, err
	}
	if cur["kind"] != "profile" {
		return ProfileCursor{}, envelope.ErrInvalidCursor
	}
	followedRank, ok := numberAsInt(cur["followedRank"])
	if !ok {
		return ProfileCursor{}, envelope.ErrInvalidCursor
	}
	relevanceRank, ok := numberAsInt(cur["relevanceRank"])
	if !ok {
		return ProfileCursor{}, envelope.ErrInvalidCursor
	}
	handleLower, ok := cur["handleLower"].(string)
	if !ok || handleLower == "" {
		return ProfileCursor{}, envelope.ErrInvalidCursor
	}
	did, ok := cur["did"].(string)
	if !ok || did == "" {
		return ProfileCursor{}, envelope.ErrInvalidCursor
	}
	return ProfileCursor{FollowedRank: followedRank, RelevanceRank: relevanceRank, HandleLower: handleLower, DID: did}, nil
}

func EncodeHashtagSearchCursor(query string, offset int) (string, error) {
	return envelope.EncodeCursor(map[string]any{
		"kind":   "hashtag",
		"query":  query,
		"offset": offset,
	})
}

func DecodeHashtagSearchCursor(cursor, query string) (HashtagCursor, error) {
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil || cursor == "" {
		return HashtagCursor{}, err
	}
	if cur["kind"] != "hashtag" || cur["query"] != query {
		return HashtagCursor{}, envelope.ErrInvalidCursor
	}
	offset, ok := numberAsInt(cur["offset"])
	if !ok || offset < 0 {
		return HashtagCursor{}, envelope.ErrInvalidCursor
	}
	return HashtagCursor{Query: query, Offset: offset}, nil
}

func EncodeRelevanceSearchCursor(kind, query string, score float64, createdAt time.Time, uri string) (string, error) {
	return envelope.EncodeCursor(map[string]any{
		"kind":      kind,
		"query":     query,
		"score":     score,
		"createdAt": createdAt.UTC().Format(time.RFC3339Nano),
		"uri":       uri,
	})
}

func DecodeRelevanceSearchCursor(cursor, kind, query string) (RelevanceCursor, error) {
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil || cursor == "" {
		return RelevanceCursor{}, err
	}
	if cur["kind"] != kind || cur["query"] != query {
		return RelevanceCursor{}, envelope.ErrInvalidCursor
	}
	score, ok := cur["score"].(float64)
	if !ok {
		return RelevanceCursor{}, envelope.ErrInvalidCursor
	}
	createdAt, ok := cur["createdAt"].(string)
	if !ok || createdAt == "" {
		return RelevanceCursor{}, envelope.ErrInvalidCursor
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return RelevanceCursor{}, envelope.ErrInvalidCursor
	}
	uri, ok := cur["uri"].(string)
	if !ok || uri == "" {
		return RelevanceCursor{}, envelope.ErrInvalidCursor
	}
	return RelevanceCursor{Kind: kind, Query: query, Score: score, CreatedAt: parsedCreatedAt, URI: uri}, nil
}

func numberAsInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		if n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	case int:
		return n, true
	default:
		return 0, false
	}
}
