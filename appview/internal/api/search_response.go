package api

import (
	"encoding/json"
	"time"
)

type SearchPostPageResponse struct {
	Hashtag string          `json:"hashtag,omitempty"`
	Items   []*PostResponse `json:"items"`
	Cursor  string          `json:"cursor,omitempty"`
}

type SearchProfilePageResponse struct {
	Items  []ProfileSearchSummary `json:"items"`
	Cursor string                 `json:"cursor,omitempty"`
}

type HashtagSearchPageResponse struct {
	Items  []HashtagSearchResult `json:"items"`
	Cursor string                `json:"cursor,omitempty"`
}

type HashtagSearchResult struct {
	Tag             string `json:"tag"`
	PostsLast28Days int    `json:"postsLast28Days"`
}

type SearchSuggestionsResponse struct {
	Profiles SuggestionProfileSection `json:"profiles"`
	Hashtags SuggestionHashtagSection `json:"hashtags"`
}

type SuggestionProfileSection struct {
	Items   []ProfileSearchSummary `json:"items"`
	HasMore bool                   `json:"hasMore"`
}

type SuggestionHashtagSection struct {
	Items   []HashtagSearchResult `json:"items"`
	HasMore bool                  `json:"hasMore"`
}

type ProfileSearchSummary struct {
	ProfileAccountSummary
	ViewerIsFollowing bool     `json:"viewerIsFollowing"`
	Crafts            []string `json:"crafts"`
}

type TopHashtagsResponse struct {
	Groups []TopHashtagGroup `json:"groups"`
}

type TopHashtagGroup struct {
	CraftType string           `json:"craftType"`
	Items     []TopHashtagItem `json:"items"`
}

type TopHashtagItem struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type RecentSearchPageResponse struct {
	Items []RecentSearchResponse `json:"items"`
}

type RecentSearchResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	DisplayLabel string `json:"displayLabel"`
	Payload      any    `json:"payload"`
	UpdatedAt    string `json:"updatedAt"`
}

func BuildRecentSearchResponse(row RecentSearchRow) (RecentSearchResponse, error) {
	var payload any
	if err := json.Unmarshal(row.NormalizedPayload, &payload); err != nil {
		return RecentSearchResponse{}, err
	}
	return RecentSearchResponse{ID: row.ID, Type: row.Type, DisplayLabel: row.DisplayLabel, Payload: payload, UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}, nil
}
