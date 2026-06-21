package api

import "github.com/bluesky-social/indigo/atproto/syntax"

type FacetMentionSuggestionsResponse struct {
	Items []FacetMentionSuggestion `json:"items"`
}

type FacetMentionSuggestion struct {
	DID               syntax.DID    `json:"did"`
	Handle            syntax.Handle `json:"handle"`
	DisplayName       *string       `json:"displayName,omitempty"`
	Avatar            *string       `json:"avatar,omitempty"`
	IsCraftskyProfile bool          `json:"isCraftskyProfile"`
	ViewerIsFollowing bool          `json:"viewerIsFollowing"`
}

type FacetMentionResolveResponse struct {
	DID               syntax.DID    `json:"did"`
	Handle            syntax.Handle `json:"handle"`
	IsCraftskyProfile bool          `json:"isCraftskyProfile"`
}

type FacetHashtagSuggestionsResponse struct {
	Items []FacetHashtagSuggestion `json:"items"`
}

type FacetHashtagSuggestion struct {
	Tag             string `json:"tag"`
	PostsLast28Days int    `json:"postsLast28Days"`
}

type MentionSuggestionRow struct {
	DID               string
	Handle            string
	DisplayName       *string
	Description       *string
	AvatarCID         *string
	AvatarMime        *string
	Crafts            []string
	IsCraftskyProfile bool
	ViewerIsFollowing bool
}

type HashtagSuggestionRow struct {
	Tag             string
	PostsLast28Days int
}

func BuildFacetMentionSuggestion(row MentionSuggestionRow) FacetMentionSuggestion {
	out := FacetMentionSuggestion{
		DID:               syntax.DID(row.DID),
		Handle:            syntax.Handle(row.Handle),
		DisplayName:       row.DisplayName,
		IsCraftskyProfile: row.IsCraftskyProfile,
		ViewerIsFollowing: row.ViewerIsFollowing,
	}
	if avatar := synthBlobURL("avatar", row.DID, row.AvatarCID, row.AvatarMime); avatar != "" {
		out.Avatar = &avatar
	}
	return out
}

func BuildFacetMentionResolveResponse(row IdentityCacheRow) FacetMentionResolveResponse {
	return FacetMentionResolveResponse{
		DID:               row.DID,
		Handle:            row.Handle,
		IsCraftskyProfile: true,
	}
}
