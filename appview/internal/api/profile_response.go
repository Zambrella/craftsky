// appview/internal/api/profile_response.go
package api

import (
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ProfileResponse is the JSON shape returned by all three profile
// endpoints. Fields tagged `omitempty` are omitted from the wire when nil.
//
// syntax.DID and syntax.Handle JSON-marshal via TextMarshaler — the
// wire shape is the same plain-string JSON it always was.
type ProfileResponse struct {
	DID                 syntax.DID          `json:"did"`
	Handle              syntax.Handle       `json:"handle"`
	ViewerIsFollowing   bool                `json:"viewerIsFollowing"`
	IsCraftskyProfile   bool                `json:"isCraftskyProfile"`
	FollowingCount      *int                `json:"followingCount,omitempty"`
	FollowerCount       *int                `json:"followerCount,omitempty"`
	MutualFollowerCount *int                `json:"mutualFollowerCount,omitempty"`
	PostCount           *int                `json:"postCount,omitempty"`
	PostsLast7Days      *int                `json:"postsLast7Days,omitempty"`
	ProjectCount        *int                `json:"projectCount,omitempty"`
	DisplayName         *string             `json:"displayName,omitempty"`
	Description         *string             `json:"description,omitempty"`
	Avatar              *string             `json:"avatar,omitempty"`
	Banner              *string             `json:"banner,omitempty"`
	Crafts              []string            `json:"crafts"`
	CreatedAt           *time.Time          `json:"createdAt,omitempty"`
	Moderation          *ModerationMetadata `json:"moderation,omitempty"`
}

type ProfileAccountPage struct {
	Items      []ProfileAccountSummary `json:"items"`
	Cursor     *string                 `json:"cursor,omitempty"`
	TotalCount int                     `json:"totalCount"`
}

type ProfileAccountSummary struct {
	DID               syntax.DID    `json:"did"`
	Handle            syntax.Handle `json:"handle"`
	DisplayName       *string       `json:"displayName,omitempty"`
	Description       *string       `json:"description,omitempty"`
	Avatar            *string       `json:"avatar,omitempty"`
	IsCraftskyProfile bool          `json:"isCraftskyProfile"`
}

func BuildProfileAccountSummary(row *ProfileAccountRow, handle syntax.Handle) ProfileAccountSummary {
	out := ProfileAccountSummary{
		DID:               syntax.DID(row.DID),
		Handle:            handle,
		DisplayName:       row.DisplayName,
		Description:       row.Description,
		IsCraftskyProfile: row.IsCraftskyProfile,
	}
	if avatar := synthBlobURL("avatar", row.DID, row.AvatarCID, row.AvatarMime); avatar != "" {
		out.Avatar = &avatar
	}
	return out
}

// mimeExt maps the MIME types we know Bluesky's CDN serves into the
// extension suffix it expects in the URL. Unknown MIME types cause the
// avatar/banner field to be omitted rather than produce a broken URL.
// See docs/superpowers/specs/2026-04-23-profile-onboarding-design.md §5.4.
var mimeExt = map[string]string{
	"image/jpeg": "jpeg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// BuildProfileResponse composes a ProfileResponse from a row and a
// freshly-resolved handle. When includeCreatedAt is false, CreatedAt is
// nil — used by the PUT response path, which must not emit this field
// (see §5.3 of the spec).
//
// row.DID is direct-cast to syntax.DID — we own the database, and DIDs
// are written from typed values via the indexer path.
func BuildProfileResponse(row *ProfileRow, handle syntax.Handle, includeCreatedAt bool) ProfileResponse {
	crafts := row.Crafts
	if crafts == nil {
		crafts = []string{}
	}
	out := ProfileResponse{
		DID:                 syntax.DID(row.DID),
		Handle:              handle,
		ViewerIsFollowing:   row.ViewerIsFollowing,
		IsCraftskyProfile:   row.IsCraftskyProfile,
		FollowingCount:      row.FollowingCount,
		FollowerCount:       row.FollowerCount,
		MutualFollowerCount: row.MutualFollowerCount,
		PostCount:           row.PostCount,
		PostsLast7Days:      row.PostsLast7Days,
		ProjectCount:        row.ProjectCount,
		DisplayName:         row.DisplayName,
		Description:         row.Description,
		Crafts:              crafts,
	}
	if avatar := synthBlobURL("avatar", row.DID, row.AvatarCID, row.AvatarMime); avatar != "" {
		out.Avatar = &avatar
	}
	if banner := synthBlobURL("banner", row.DID, row.BannerCID, row.BannerMime); banner != "" {
		out.Banner = &banner
	}
	if includeCreatedAt {
		t := row.CreatedAt
		out.CreatedAt = &t
	}
	if row.ModerationWarningKind != nil && *row.ModerationWarningKind != "" {
		out.Moderation = &ModerationMetadata{WarningKind: *row.ModerationWarningKind}
	}
	return out
}

func synthBlobURL(kind, did string, cid, mime *string) string {
	if cid == nil || mime == nil {
		return ""
	}
	if name, ok := strings.CutPrefix(*cid, "devmedia:"); ok {
		return devMediaURL(name)
	}
	ext, ok := mimeExt[*mime]
	if !ok {
		return ""
	}
	return "https://cdn.bsky.app/img/" + kind + "/plain/" + did + "/" + *cid + "@" + ext
}
