package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/business"
)

type PublicAccountTypeReader interface {
	ReadAccountType(context.Context, syntax.DID) (business.AccountType, error)
}

type PublicPageStore struct {
	accountTypes PublicAccountTypeReader
	pool         *pgxpool.Pool
	posts        *PostStore
	resolver     HandleResolver
}

func NewPublicPageStore(pool *pgxpool.Pool, posts *PostStore, resolver HandleResolver, accountTypes PublicAccountTypeReader) *PublicPageStore {
	return &PublicPageStore{pool: pool, posts: posts, resolver: resolver, accountTypes: accountTypes}
}

// Membership, active lifecycle, account moderation and suspension are checked
// for every anonymous resource, including author feeds and post detail.
var publicOwnerPredicate = `
 AND appview_owner_is_active(cp.did)
 AND NOT EXISTS (SELECT 1 FROM moderation_account_standings standing WHERE standing.owner_did=cp.did AND standing.effective_suspended)
 ` + strings.ReplaceAll(profileVisibleModerationPredicate, "('hide', 'takedown')", "('hide', 'takedown', 'warn')")

func (s *PublicPageStore) PublicProfile(ctx context.Context, did syntax.DID) (*PublicProfile, error) {
	var row ProfileRow
	var imageClear bool
	err := s.pool.QueryRow(ctx, `
 SELECT cp.did,cp.crafts,bp.display_name,bp.description,bp.pronouns,
 bp.avatar_cid,bp.avatar_mime,bp.banner_cid,bp.banner_mime,
 COALESCE(appview_image_subject_is_clear('at://' || cp.did || '/app.bsky.actor.profile/self',bp.record_cid),false)
 FROM craftsky_profiles cp LEFT JOIN bluesky_profiles bp ON bp.did=cp.did
 WHERE cp.did=$1 `+publicOwnerPredicate, did).Scan(&row.DID, &row.Crafts, &row.DisplayName, &row.Description, &row.Pronouns, &row.AvatarCID, &row.AvatarMime, &row.BannerCID, &row.BannerMime, &imageClear)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("public profile read: %w", err)
	}
	handle, err := s.resolver.ResolveHandle(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("public profile identity: %w", err)
	}
	accountType, err := s.accountTypes.ReadAccountType(ctx, did)
	if err != nil {
		return nil, err
	}
	out := &PublicProfile{PublicAuthor: PublicAuthor{DID: did, Handle: handle, DisplayName: row.DisplayName, Business: accountType == business.AccountTypeBusiness}, Description: row.Description, Pronouns: row.Pronouns, Crafts: nonNilStrings(row.Crafts), Links: []PublicLink{}}
	if imageClear {
		if v := synthBlobURL("avatar", row.DID, row.AvatarCID, row.AvatarMime); v != "" {
			out.Avatar = &v
		}
		if v := synthBlobURL("banner", row.DID, row.BannerCID, row.BannerMime); v != "" {
			out.Banner = &v
		}
	}
	links, _, err := NewProfileStore(s.pool).readProfileLinks(ctx, did, "")
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		out.Links = append(out.Links, PublicLink{URL: link.URL, Title: link.Title})
	}
	return out, nil
}
func (s *PublicPageStore) PublicPost(ctx context.Context, did syntax.DID, key syntax.RecordKey) (*PublicPost, error) {
	profile, err := s.PublicProfile(ctx, did)
	if err != nil {
		return nil, err
	}
	row, err := scanPostRow(s.pool.QueryRow(ctx, `SELECT `+postSelectColumns+`
 FROM craftsky_posts p LEFT JOIN craftsky_project_posts pp ON pp.uri=p.uri
 LEFT JOIN bluesky_profiles bp ON bp.did=p.did
 WHERE p.did=$1 AND p.rkey=$2 AND p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL
 `+publicPostPredicate, did, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("public post read: %w", err)
	}
	return s.publicPost(row, profile.PublicAuthor), nil
}
func (s *PublicPageStore) PublicPosts(ctx context.Context, did syntax.DID, projects bool, limit int, cursor string) (*PublicPostPage, error) {
	profile, err := s.PublicProfile(ctx, did)
	if err != nil {
		return nil, err
	}
	if limit < 1 || limit > 20 {
		limit = 20
	}
	at, uri, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+postSelectColumns+`
 FROM craftsky_posts p LEFT JOIN craftsky_project_posts pp ON pp.uri=p.uri
 LEFT JOIN bluesky_profiles bp ON bp.did=p.did
 WHERE p.did=$1 AND p.is_project=$2 AND p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL
 AND (NOT $2 OR p.quote_uri IS NULL)
 `+publicPostPredicate+`
 AND ($3::timestamptz IS NULL OR (p.profile_sort_at,p.uri)<($3::timestamptz,$4::text))
 ORDER BY p.profile_sort_at DESC,p.uri DESC LIMIT $5`, did, projects, at, uri, limit+1)
	if err != nil {
		return nil, fmt.Errorf("public author posts: %w", err)
	}
	defer rows.Close()
	out := &PublicPostPage{Items: []PublicPost{}}
	var last *PostRow
	for rows.Next() {
		row, err := scanPostRow(rows)
		if err != nil {
			return nil, err
		}
		if len(out.Items) == limit {
			out.Cursor, err = envelope.EncodeCursor(map[string]any{"indexedAt": last.ProfileSortAt.UTC().Format(time.RFC3339Nano), "uri": last.URI})
			if err != nil {
				return nil, err
			}
			break
		}
		out.Items = append(out.Items, *s.publicPost(row, profile.PublicAuthor))
		last = row
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
func (s *PublicPageStore) publicPost(row *PostRow, author PublicAuthor) *PublicPost {
	response := buildPostResponse(row, author.Handle, s.posts)
	images := response.Images
	if images == nil {
		images = []PostImageView{}
	}
	return &PublicPost{Rkey: syntax.RecordKey(row.Rkey), Text: row.Text, Author: author, CreatedAt: response.CreatedAt, EditedAt: response.EditedAt, Images: images, Video: response.Video, Project: response.Project, Sponsored: row.Sponsored}
}

var publicPostPredicate = strings.ReplaceAll(postVisibleModerationPredicate, "('hide', 'takedown')", "('hide', 'takedown', 'warn')")
