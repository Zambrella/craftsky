// appview/internal/api/post_response.go
package api

import (
	"encoding/json"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

var postImageMimeExt = map[string]string{
	"image/jpeg": "jpeg",
	"image/png":  "png",
	"image/webp": "webp",
}

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
	URI               string              `json:"uri"`
	CID               string              `json:"cid"`
	Rkey              string              `json:"rkey"`
	Text              string              `json:"text"`
	Images            []PostImageView     `json:"images,omitempty"`
	Facets            json.RawMessage     `json:"facets"`
	Tags              []string            `json:"tags"`
	LikeCount         int                 `json:"likeCount"`
	RepostCount       int                 `json:"repostCount"`
	ReplyCount        int                 `json:"replyCount"`
	ViewerHasLiked    bool                `json:"viewerHasLiked"`
	ViewerHasReposted bool                `json:"viewerHasReposted"`
	ViewerHasReplied  bool                `json:"viewerHasReplied"`
	Reply             *ResponseReply      `json:"reply"`
	Quote             *ResponseStrongRef  `json:"quote"`
	CreatedAt         time.Time           `json:"createdAt"`
	IndexedAt         time.Time           `json:"indexedAt"`
	Author            PostAuthor          `json:"author"`
	Moderation        *ModerationMetadata `json:"moderation,omitempty"`
	Project           *Project            `json:"project,omitempty"`
}

// ModerationMetadata is the safe, generic moderation response shape shared by
// post and profile responses. It intentionally carries only a warning intent
// for Flutter localization, never raw report details, internal reasons, source
// DIDs, output IDs, or counts.
type ModerationMetadata struct {
	WarningKind string `json:"warningKind"`
}

type PostImageView struct {
	CID         string                `json:"cid,omitempty"`
	MIME        string                `json:"mime,omitempty"`
	Size        int64                 `json:"size,omitempty"`
	Alt         string                `json:"alt"`
	AspectRatio *PostImageAspectRatio `json:"aspectRatio,omitempty"`
	Thumb       string                `json:"thumb,omitempty"`
	Fullsize    string                `json:"fullsize,omitempty"`
}

type storedPostImage struct {
	CID         string                `json:"cid"`
	MIME        string                `json:"mime"`
	Size        int64                 `json:"size,omitempty"`
	Alt         string                `json:"alt"`
	AspectRatio *PostImageAspectRatio `json:"aspectRatio,omitempty"`
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
		Images:    buildPostImageViews(row),
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
		Project: row.Project,
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
	if row.ModerationWarningKind != nil && *row.ModerationWarningKind != "" {
		resp.Moderation = &ModerationMetadata{WarningKind: *row.ModerationWarningKind}
	}
	return resp
}

func buildPostImageViews(row *PostRow) []PostImageView {
	if row == nil || len(row.Images) == 0 {
		return nil
	}
	var stored []storedPostImage
	if err := json.Unmarshal(row.Images, &stored); err != nil {
		return nil
	}
	out := make([]PostImageView, 0, len(stored))
	for _, img := range stored {
		view := PostImageView{
			CID:         img.CID,
			MIME:        img.MIME,
			Size:        img.Size,
			Alt:         img.Alt,
			AspectRatio: img.AspectRatio,
		}
		view.Thumb = synthPostImageURL("feed_thumbnail", row.DID, img.CID, img.MIME)
		view.Fullsize = synthPostImageURL("feed_fullsize", row.DID, img.CID, img.MIME)
		out = append(out, view)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func synthPostImageURL(kind, did, cid, mime string) string {
	if cid == "" || mime == "" {
		return ""
	}
	ext, ok := postImageMimeExt[mime]
	if !ok {
		return ""
	}
	return "https://cdn.bsky.app/img/" + kind + "/plain/" + did + "/" + cid + "@" + ext
}

func applyEngagementSummary(resp *PostResponse, summary EngagementSummary) {
	resp.LikeCount = summary.LikeCount
	resp.RepostCount = summary.RepostCount
	resp.ReplyCount = summary.ReplyCount
	resp.ViewerHasLiked = summary.ViewerHasLiked
	resp.ViewerHasReposted = summary.ViewerHasReposted
	resp.ViewerHasReplied = summary.ViewerHasReplied
}

func derefOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
