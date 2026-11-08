package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

var ErrPublicAuthenticationRequired = errors.New("public content requires authentication")

// These allowlists deliberately do not embed authenticated response types.
// Adding a member-only field elsewhere cannot expand the anonymous surface.
type PublicAuthor struct {
	AccountType   business.AccountType  `json:"accountType"`
	Customisation *ProfileCustomisation `json:"customisation,omitempty"`
	DID           syntax.DID            `json:"did"`
	Handle        syntax.Handle         `json:"handle"`
	DisplayName   *string               `json:"displayName,omitempty"`
	Avatar        *string               `json:"avatar,omitempty"`
	Business      bool                  `json:"business"`
}
type PublicLink struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}
type PublicProfile struct {
	CreatedAt         time.Time                `json:"createdAt"`
	PostsLast7Days    int                      `json:"postsLast7Days"`
	PostCount         int                      `json:"postCount"`
	ProjectCount      int                      `json:"projectCount"`
	FollowerCount     int                      `json:"followerCount"`
	FollowingCount    int                      `json:"followingCount"`
	BusinessProfile   *BusinessProfileResponse `json:"businessProfile,omitempty"`
	HasUpcomingEvents bool                     `json:"hasUpcomingEvents"`
	PublicAuthor
	Description *string      `json:"description,omitempty"`
	Pronouns    *string      `json:"pronouns,omitempty"`
	Banner      *string      `json:"banner,omitempty"`
	Crafts      []string     `json:"crafts"`
	Links       []PublicLink `json:"links"`
}
type PublicPost struct {
	ExternalImport *ExternalImportResponse `json:"externalImport,omitempty"`
	URI            syntax.ATURI            `json:"uri"`
	CID            syntax.CID              `json:"cid"`
	Facets         json.RawMessage         `json:"facets,omitempty"`
	Tags           []string                `json:"tags"`
	Langs          []string                `json:"langs"`
	LikeCount      int                     `json:"likeCount"`
	RepostCount    int                     `json:"repostCount"`
	QuoteCount     int                     `json:"quoteCount"`
	ReplyCount     int                     `json:"replyCount"`
	External       *ExternalResponse       `json:"external,omitempty"`
	QuoteView      *PublicEmbed            `json:"quoteView,omitempty"`
	Rkey           syntax.RecordKey        `json:"rkey"`
	Text           string                  `json:"text"`
	Author         PublicAuthor            `json:"author"`
	CreatedAt      time.Time               `json:"createdAt"`
	EditedAt       *time.Time              `json:"editedAt,omitempty"`
	Images         []PostImageView         `json:"images"`
	Video          *PostVideoView          `json:"video,omitempty"`
	Project        *Project                `json:"project,omitempty"`
	Sponsored      bool                    `json:"sponsored"`
}
type PublicEmbed struct {
	State string      `json:"state"`
	Post  *PublicPost `json:"post,omitempty"`
}

type PublicThread struct {
	Post      *PublicPost    `json:"post"`
	Ancestors []PublicEmbed  `json:"ancestors"`
	Replies   PublicPostPage `json:"replies"`
}

type PublicPostPage struct {
	Items  []PublicPost `json:"items"`
	Cursor string       `json:"cursor,omitempty"`
}

type PublicPageReader interface {
	PublicProfile(context.Context, syntax.DID) (*PublicProfile, error)
	PublicPost(context.Context, syntax.DID, syntax.RecordKey) (*PublicPost, error)
	PublicPosts(context.Context, syntax.DID, bool, int, string) (*PublicPostPage, error)
}

// PublicPagesHandler serves only the explicit anonymous read contract. Bearer
// tokens and cookies never select a viewer or enable owner moderation bypasses.
func PublicPagesHandler(store PublicPageReader, resolver HandleResolver, resource string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		fail := func(status int, code, message string) {
			envelope.WriteError(w, status, code, message, middleware.GetRunID(ctx), nil)
		}
		allowed := []string{}
		if resource != "profile" && resource != "post" {
			allowed = []string{"cursor", "limit"}
		}
		limit, validationErr := validatePublicQuery(r.URL.Query(), allowed...)
		if validationErr != nil {
			fail(400, "invalid_request", "invalid public request")
			return
		}
		raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
		if resource == "post" {
			raw = r.PathValue("did")
		}
		if raw == "me" || raw == "recent" {
			fail(404, "not_found", "page not found")
			return
		}
		did, err := resolveToDID(ctx, raw, resolver)
		if err != nil {
			if errors.Is(err, errInvalidIdentifier) {
				fail(400, "invalid_identifier", "invalid profile identifier")
			} else {
				fail(503, "identity_unavailable", "identity temporarily unavailable")
			}
			return
		}
		var result any
		switch resource {
		case "profile":
			result, err = store.PublicProfile(ctx, did)
		case "post":
			key, parseErr := syntax.ParseRecordKey(r.PathValue("rkey"))
			if parseErr != nil {
				fail(400, "invalid_identifier", "invalid post identifier")
				return
			}
			result, err = store.PublicPost(ctx, did, key)
		default:
			cursor := r.URL.Query().Get("cursor")
			if len(cursor) > 2048 {
				fail(400, "invalid_cursor", "invalid cursor")
				return
			}
			if _, _, e := decodeSeekCursor(cursor, "indexedAt"); e != nil {
				fail(400, "invalid_cursor", "invalid cursor")
				return
			}
			result, err = store.PublicPosts(ctx, did, resource == "projects", limit, cursor)
		}
		if errors.Is(err, ErrPublicAuthenticationRequired) {
			fail(403, "authentication_required", "Sign in to view this content.")
			return
		}
		if errors.Is(err, ErrProfileNotFound) || errors.Is(err, ErrPostNotFound) {
			fail(404, "not_found", "page not found")
			return
		}
		if err != nil {
			observability.ReportRequestFailure(ctx, err, "api.PublicPagesHandler", "read")
			fail(503, "unavailable", "page temporarily unavailable")
			return
		}
		writeJSON(w, 200, result)
	})
}

// validatePublicQuery bounds work and rejects parameters outside the public
// contract. A guest request cannot select a member viewer or advanced search.
func validatePublicQuery(query url.Values, allowed ...string) (int, error) {
	for key, values := range query {
		valid := false
		for _, candidate := range allowed {
			if key == candidate {
				valid = true
				break
			}
		}
		if !valid || len(values) != 1 {
			return 0, fmt.Errorf("unsupported public parameter")
		}
	}
	limit := 20
	if raw, ok := query["limit"]; ok {
		parsed, err := strconv.Atoi(raw[0])
		if err != nil || parsed < 1 || parsed > 20 {
			return 0, fmt.Errorf("invalid public limit")
		}
		limit = parsed
	}
	if len(query.Get("cursor")) > 2048 {
		return 0, fmt.Errorf("invalid public cursor")
	}
	return limit, nil
}
