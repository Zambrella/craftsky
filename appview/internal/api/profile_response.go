// appview/internal/api/profile_response.go
package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/business"
)

// ProfileResponse is the JSON shape returned by all three profile
// endpoints. Fields tagged `omitempty` are omitted from the wire when nil.
//
// syntax.DID and syntax.Handle JSON-marshal via TextMarshaler — the
// wire shape is the same plain-string JSON it always was.
type ProfileResponse struct {
	DID                 syntax.DID               `json:"did"`
	Handle              syntax.Handle            `json:"handle"`
	ViewerIsFollowing   bool                     `json:"viewerIsFollowing"`
	Muted               bool                     `json:"muted"`
	Blocking            bool                     `json:"blocking"`
	BlockedBy           bool                     `json:"blockedBy"`
	IsCraftskyProfile   bool                     `json:"isCraftskyProfile"`
	FollowingCount      *int                     `json:"followingCount,omitempty"`
	FollowerCount       *int                     `json:"followerCount,omitempty"`
	MutualFollowerCount *int                     `json:"mutualFollowerCount,omitempty"`
	PostCount           *int                     `json:"postCount,omitempty"`
	PostsLast7Days      *int                     `json:"postsLast7Days,omitempty"`
	ProjectCount        *int                     `json:"projectCount,omitempty"`
	DisplayName         *string                  `json:"displayName,omitempty"`
	Description         *string                  `json:"description,omitempty"`
	Avatar              *string                  `json:"avatar,omitempty"`
	Banner              *string                  `json:"banner,omitempty"`
	Crafts              []string                 `json:"crafts"`
	CreatedAt           *time.Time               `json:"createdAt,omitempty"`
	Moderation          *ModerationMetadata      `json:"moderation,omitempty"`
	Customisation       *ProfileCustomisation    `json:"customisation,omitempty"`
	AccountType         *business.AccountType    `json:"accountType,omitempty"`
	Business            *BusinessProfileResponse `json:"business,omitempty"`
	HasUpcomingEvents   bool                     `json:"hasUpcomingEvents"`
}

type BusinessProfileResponse struct {
	CID           syntax.CID                `json:"cid"`
	BusinessTypes []business.OpenValue      `json:"businessTypes,omitempty"`
	Offerings     []business.OpenValue      `json:"offerings,omitempty"`
	Tagline       string                    `json:"tagline,omitempty"`
	HoursNote     string                    `json:"hoursNote,omitempty"`
	ServiceArea   string                    `json:"serviceArea,omitempty"`
	Location      *business.Location        `json:"location,omitempty"`
	PrimaryAction *business.Action          `json:"primaryAction,omitempty"`
	Products      []BusinessProductResponse `json:"products,omitempty"`
}

type BusinessProductResponse struct {
	Title string             `json:"title"`
	URI   string             `json:"uri,omitempty"`
	Image *BusinessImageView `json:"image,omitempty"`
	Price *business.Price    `json:"price,omitempty"`
}

type BusinessImageView struct {
	CID         string                `json:"cid"`
	MIME        string                `json:"mime"`
	Size        int64                 `json:"size"`
	Alt         string                `json:"alt"`
	AspectRatio *PostImageAspectRatio `json:"aspectRatio,omitempty"`
	Thumb       string                `json:"thumb"`
	Fullsize    string                `json:"fullsize"`
}

func BuildBusinessProfileResponse(owner syntax.DID, view *business.ProfileView) *BusinessProfileResponse {
	if view == nil {
		return nil
	}
	response := &BusinessProfileResponse{
		CID: view.CID, BusinessTypes: view.BusinessTypes, Offerings: view.Offerings,
		Tagline: view.Tagline, HoursNote: view.HoursNote, ServiceArea: view.ServiceArea,
		Location: view.Location, PrimaryAction: view.PrimaryAction,
	}
	for _, product := range view.Products {
		response.Products = append(response.Products, BusinessProductResponse{
			Title: product.Title, URI: product.URI,
			Image: buildBusinessImageView(owner, product.Image), Price: product.Price,
		})
	}
	return response
}

func buildBusinessImageView(owner syntax.DID, image *business.HydratedImage) *BusinessImageView {
	if image == nil {
		return nil
	}
	view := &BusinessImageView{
		CID: image.CID.String(), MIME: image.MIME, Size: image.Size, Alt: image.Alt,
		Thumb:    synthPostImageURL("feed_thumbnail", owner.String(), image.CID.String(), image.MIME),
		Fullsize: synthPostImageURL("feed_fullsize", owner.String(), image.CID.String(), image.MIME),
	}
	if view.Thumb == "" || view.Fullsize == "" {
		return nil
	}
	if image.AspectRatio != nil {
		view.AspectRatio = &PostImageAspectRatio{
			Width: int(image.AspectRatio.Width), Height: int(image.AspectRatio.Height),
		}
	}
	return view
}

// MarshalJSON keeps the ordinary profile contract unchanged while enforcing
// the blocked-profile information boundary on the wire.
func (p ProfileResponse) MarshalJSON() ([]byte, error) {
	if !p.Blocking && !p.BlockedBy {
		type ordinary ProfileResponse
		return json.Marshal(ordinary(p))
	}
	type blockedShell struct {
		DID               syntax.DID            `json:"did"`
		Handle            syntax.Handle         `json:"handle"`
		DisplayName       *string               `json:"displayName,omitempty"`
		Avatar            *string               `json:"avatar,omitempty"`
		IsCraftskyProfile bool                  `json:"isCraftskyProfile"`
		Muted             bool                  `json:"muted"`
		Blocking          bool                  `json:"blocking"`
		BlockedBy         bool                  `json:"blockedBy"`
		Customisation     *ProfileCustomisation `json:"customisation,omitempty"`
	}
	return json.Marshal(blockedShell{
		DID: p.DID, Handle: p.Handle, DisplayName: p.DisplayName, Avatar: p.Avatar,
		IsCraftskyProfile: p.IsCraftskyProfile,
		Muted:             p.Muted, Blocking: p.Blocking, BlockedBy: p.BlockedBy,
		Customisation: p.Customisation,
	})
}

type ProfileAccountPage struct {
	Items      []ProfileAccountSummary `json:"items"`
	Cursor     *string                 `json:"cursor,omitempty"`
	TotalCount int                     `json:"totalCount"`
}

type ProfileAccountSummary struct {
	DID               syntax.DID            `json:"did"`
	Handle            syntax.Handle         `json:"handle"`
	DisplayName       *string               `json:"displayName,omitempty"`
	Description       *string               `json:"description,omitempty"`
	Avatar            *string               `json:"avatar,omitempty"`
	IsCraftskyProfile bool                  `json:"isCraftskyProfile"`
	Muted             bool                  `json:"muted"`
	Blocking          bool                  `json:"blocking"`
	BlockedBy         bool                  `json:"blockedBy"`
	Customisation     *ProfileCustomisation `json:"customisation,omitempty"`
}

func BuildProfileAccountSummary(row *ProfileAccountRow, handle syntax.Handle) ProfileAccountSummary {
	out := ProfileAccountSummary{
		DID:               syntax.DID(row.DID),
		Handle:            handle,
		DisplayName:       row.DisplayName,
		Description:       row.Description,
		IsCraftskyProfile: row.IsCraftskyProfile,
		Muted:             row.Muted,
		Blocking:          row.Blocking,
		BlockedBy:         row.BlockedBy,
	}
	if row.IsCraftskyProfile {
		value := DefaultProfileCustomisation
		out.Customisation = &value
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
		Muted:               row.Muted,
		Blocking:            row.Blocking,
		BlockedBy:           row.BlockedBy,
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
	if row.IsCraftskyProfile {
		value := DefaultProfileCustomisation
		out.Customisation = &value
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
