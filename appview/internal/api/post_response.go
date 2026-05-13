// appview/internal/api/post_response.go
package api

import (
	"encoding/json"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// PostAuthor is the embedded author shape on every post-shaped response.
// Display name and avatar may be null when the user has no Bluesky
// profile mirror.
type PostAuthor struct {
	DID         string  `json:"did"`
	Handle      string  `json:"handle"`
	DisplayName *string `json:"displayName"`
	AvatarCID   *string `json:"avatarCid"`
}

// ResponseStrongRef is the JSON wire shape of a strongRef on a response
// body. Same fields as request StrongRef but kept distinct so request
// and response shapes can evolve independently.
type ResponseStrongRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// ResponseReply mirrors the lexicon's #replyRef on response bodies.
type ResponseReply struct {
	Root   ResponseStrongRef `json:"root"`
	Parent ResponseStrongRef `json:"parent"`
}

// PostResponse is the canonical wire shape returned by every
// post-shaped endpoint (POST, GET single, list items).
type PostResponse struct {
	URI               string             `json:"uri"`
	CID               string             `json:"cid"`
	Rkey              string             `json:"rkey"`
	Text              string             `json:"text"`
	Facets            json.RawMessage    `json:"facets"`
	Tags              []string           `json:"tags"`
	LikeCount         int                `json:"likeCount"`
	RepostCount       int                `json:"repostCount"`
	ReplyCount        int                `json:"replyCount"`
	ViewerHasLiked    bool               `json:"viewerHasLiked"`
	ViewerHasReposted bool               `json:"viewerHasReposted"`
	Reply             *ResponseReply     `json:"reply"`
	Quote             *ResponseStrongRef `json:"quote"`
	CreatedAt         time.Time          `json:"createdAt"`
	IndexedAt         time.Time          `json:"indexedAt"`
	Author            PostAuthor         `json:"author"`
}

// ThreadResponse is the root response for nested thread reads.
type ThreadResponse struct {
	Post      *PostResponse   `json:"post"`
	Ancestors []*PostResponse `json:"ancestors"`
	Replies   []*ThreadNode   `json:"replies"`
	Truncated bool            `json:"truncated"`
}

// ThreadNode is a nested reply node in a thread response.
type ThreadNode struct {
	Post    *PostResponse `json:"post"`
	Replies []*ThreadNode `json:"replies"`
}

// CommentSectionResponse is the root-post comment-section read surface.
type CommentSectionResponse struct {
	Post     *PostResponse `json:"post"`
	Comments CommentPage   `json:"comments"`
	Sort     string        `json:"sort"`
	Focus    *FocusContext `json:"focus,omitempty"`
}

// FocusContext reports backend focus resolution for a requested focus AT-URI.
type FocusContext struct {
	URI        string `json:"uri"`
	Status     string `json:"status"`
	Kind       string `json:"kind,omitempty"`
	CommentURI string `json:"commentUri,omitempty"`
}

// CommentPage carries the ordered top-level comment render list.
type CommentPage struct {
	Items  []CommentItem `json:"items"`
	Cursor string        `json:"cursor,omitempty"`
}

// CommentItem is a direct reply to the root post plus its action-loaded reply state.
type CommentItem struct {
	Post      *PostResponse `json:"post"`
	Placement string        `json:"placement"`
	Replies   ReplyPage     `json:"replies"`
}

// ReplyPage carries the per-comment reply list state.
type ReplyPage struct {
	Loaded bool        `json:"loaded"`
	Items  []ReplyItem `json:"items"`
	Cursor string      `json:"cursor,omitempty"`
}

// ReplyItem is a visual second-level reply under a comment branch.
type ReplyItem struct {
	Post       *PostResponse     `json:"post"`
	Flattened  bool              `json:"flattened"`
	ReplyingTo *ReplyingToAuthor `json:"replyingTo,omitempty"`
}

// ReplyingToAuthor describes the true backend parent for flattened replies.
type ReplyingToAuthor struct {
	URI         string  `json:"uri"`
	DID         string  `json:"did"`
	Handle      string  `json:"handle"`
	DisplayName *string `json:"displayName,omitempty"`
}

// BuildPostResponse converts a PostRow + resolved handle into the wire
// response. Reply and quote pointers are flattened from the row's
// pointer columns into the lexicon-shaped nested objects.
func BuildPostResponse(row *PostRow, handle syntax.Handle) *PostResponse {
	tags := row.Tags
	if tags == nil {
		tags = []string{}
	}
	resp := &PostResponse{
		URI:       row.URI,
		CID:       row.CID,
		Rkey:      row.Rkey,
		Text:      row.Text,
		Facets:    row.Facets,
		Tags:      tags,
		CreatedAt: row.CreatedAt.UTC(),
		IndexedAt: row.IndexedAt.UTC(),
		Author: PostAuthor{
			DID:         row.DID,
			Handle:      handle.String(),
			DisplayName: row.AuthorDisplayName,
			AvatarCID:   row.AuthorAvatarCID,
		},
	}
	if row.ReplyRootURI != nil && row.ReplyParentURI != nil {
		resp.Reply = &ResponseReply{
			Root: ResponseStrongRef{
				URI: *row.ReplyRootURI,
				CID: derefOrEmpty(row.ReplyRootCID),
			},
			Parent: ResponseStrongRef{
				URI: *row.ReplyParentURI,
				CID: derefOrEmpty(row.ReplyParentCID),
			},
		}
	}
	if row.QuoteURI != nil {
		resp.Quote = &ResponseStrongRef{
			URI: *row.QuoteURI,
			CID: derefOrEmpty(row.QuoteCID),
		}
	}
	return resp
}

func applyEngagementSummary(resp *PostResponse, summary EngagementSummary) {
	resp.LikeCount = summary.LikeCount
	resp.RepostCount = summary.RepostCount
	resp.ReplyCount = summary.ReplyCount
	resp.ViewerHasLiked = summary.ViewerHasLiked
	resp.ViewerHasReposted = summary.ViewerHasReposted
}

func derefOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
