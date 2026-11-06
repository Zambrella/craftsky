package api

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// PublicProjects reuses project browsing's primary-craft/filter/sort semantics,
// selecting the anonymous policy before the SQL candidate window is formed.
func (s *PublicPageStore) PublicProjects(ctx context.Context, request ProjectListRequest) (*PublicPostPage, error) {
	ctx = withReadAudience(ctx, GuestAudience)
	if request.Limit < 1 || request.Limit > 20 {
		return nil, ErrSearchValidation
	}
	parameters := request
	parameters.Cursor = ""
	parameters.Limit = 0
	if parameters.Sort == "" {
		parameters.Sort = SearchSortChronological
	}
	scope := publicCursorScope("projects", parameters)
	inner, err := decodePublicCursor(request.Cursor, scope)
	if err != nil {
		return nil, err
	}
	request.Cursor = inner
	rows, cursor, err := NewSearchStore(s.pool, nil).SearchProjectsWithLanguages(ctx, "", []string{}, ProjectSearchRequest{Sort: request.Sort, Limit: request.Limit, Cursor: request.Cursor, Filters: request.Filters, SelfDrafted: request.SelfDrafted}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	page, err := s.publicSearchPosts(ctx, rows, cursor)
	if err != nil {
		return nil, err
	}
	if err := bindPublicPageCursor(page, scope); err != nil {
		return nil, err
	}
	return page, nil
}

func (s *PublicPageStore) publicSearchPosts(ctx context.Context, rows []SearchPostRow, cursor string) (*PublicPostPage, error) {
	posts := make([]*PostRow, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, row.Post)
	}
	return s.publicPostRows(ctx, posts, cursor)
}

func (s *PublicPageStore) publicPostRows(ctx context.Context, rows []*PostRow, cursor string) (*PublicPostPage, error) {
	out := &PublicPostPage{Items: []PublicPost{}, Cursor: cursor}
	authors := map[syntax.DID]PublicAuthor{}
	for _, row := range rows {
		did := syntax.DID(row.DID)
		author, ok := authors[did]
		if !ok {
			profile, err := s.PublicProfile(ctx, did)
			if err != nil {
				return nil, err
			}
			author = profile.PublicAuthor
			authors[did] = author
		}
		out.Items = append(out.Items, *s.publicPost(row, author))
	}
	posts := make([]*PublicPost, len(out.Items))
	for i := range out.Items {
		posts[i] = &out.Items[i]
	}
	if err := s.enrichPublicPosts(ctx, posts, rows); err != nil {
		return nil, err
	}
	return out, nil
}

type PublicProfilePage struct {
	Items  []PublicProfile `json:"items"`
	Cursor string          `json:"cursor,omitempty"`
}

type PublicHashtagPage struct {
	Items  []HashtagSearchResult `json:"items"`
	Cursor string                `json:"cursor,omitempty"`
}

// PublicSearch has no discovery/history side effects. Category selection is a
// closed read contract over an explicitly submitted query.
func (s *PublicPageStore) PublicSearch(ctx context.Context, category, query string, limit int, cursor string) (any, error) {
	scope := publicCursorScope("search/"+category, strings.ToLower(strings.TrimSpace(query)))
	inner, err := decodePublicCursor(cursor, scope)
	if err != nil {
		return nil, err
	}
	result, err := s.publicSearch(ctx, category, query, limit, inner)
	if err != nil {
		return nil, err
	}
	if err := bindPublicPageCursor(result, scope); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicPageStore) publicSearch(ctx context.Context, category, query string, limit int, cursor string) (any, error) {
	ctx = withReadAudience(ctx, GuestAudience)
	query, err := parseSearchQuery(url.Values{"q": {query}}, true)
	if err != nil || limit < 1 || limit > 20 {
		return nil, ErrSearchValidation
	}
	store := NewSearchStore(s.pool, nil)
	switch category {
	case "posts":
		rows, next, err := store.SearchPostsWithLanguages(ctx, "", []string{}, PostSearchRequest{Query: query, Limit: limit, Cursor: cursor}, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		return s.publicSearchPosts(ctx, rows, next)
	case "projects":
		rows, next, err := store.SearchProjectsWithLanguages(ctx, "", []string{}, ProjectSearchRequest{Query: query, Limit: limit, Cursor: cursor}, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		return s.publicSearchPosts(ctx, rows, next)
	case "profiles":
		rows, next, err := store.SearchProfiles(ctx, "", ProfileSearchRequest{Query: query, Limit: limit, Cursor: cursor})
		if err != nil {
			return nil, err
		}
		out := &PublicProfilePage{Items: []PublicProfile{}, Cursor: next}
		for _, row := range rows {
			profile, err := s.PublicProfile(ctx, syntax.DID(row.DID))
			if err != nil {
				return nil, err
			}
			out.Items = append(out.Items, *profile)
		}
		return out, nil
	case "hashtags":
		rows, next, err := store.SearchHashtags(ctx, HashtagSearchRequest{Query: query, Limit: limit, Cursor: cursor}, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		return &PublicHashtagPage{Items: rows, Cursor: next}, nil
	default:
		return nil, ErrSearchValidation
	}
}

func (s *PublicPageStore) enrichPublicPosts(ctx context.Context, posts []*PublicPost, rows []*PostRow) error {
	if len(rows) == 0 {
		return nil
	}
	uris := make([]string, len(rows))
	refs := []ResponseStrongRef{}
	for i, row := range rows {
		uris[i] = row.URI
		if row.QuoteURI != nil {
			refs = append(refs, ResponseStrongRef{URI: *row.QuoteURI})
		}
	}
	likes, err := s.posts.CountActiveLikes(ctx, uris)
	if err != nil {
		return err
	}
	reposts, err := s.posts.CountActiveReposts(ctx, uris)
	if err != nil {
		return err
	}
	quotes, err := s.posts.CountVisibleQuotes(ctx, uris)
	if err != nil {
		return err
	}
	replies, err := s.posts.CountDescendantReplies(ctx, uris)
	if err != nil {
		return err
	}
	nested, err := s.posts.QuoteViewRows(ctx, refs)
	if err != nil {
		return err
	}
	authors := map[syntax.DID]PublicAuthor{}
	for i, row := range rows {
		post := posts[i]
		post.LikeCount = likes[row.URI]
		post.RepostCount = reposts[row.URI]
		post.QuoteCount = quotes[row.URI]
		post.ReplyCount = replies[row.URI]
		if row.QuoteURI == nil {
			continue
		}
		post.QuoteView = &PublicEmbed{State: "unavailable"}
		view := nested[*row.QuoteURI]
		if view == nil || view.Post == nil || view.State != "visible" {
			continue
		}
		did := syntax.DID(view.Post.DID)
		author, ok := authors[did]
		if !ok {
			profile, err := s.PublicProfile(ctx, did)
			if errors.Is(err, ErrProfileNotFound) || errors.Is(err, ErrPublicAuthenticationRequired) {
				continue
			}
			if err != nil {
				return err
			}
			author = profile.PublicAuthor
			authors[did] = author
		}
		post.QuoteView = &PublicEmbed{State: "visible", Post: s.publicPost(view.Post, author)}
	}
	return nil
}

func (s *PublicPageStore) PublicProfilePosts(ctx context.Context, did syntax.DID, tab string, limit int, cursor string) (*PublicPostPage, error) {
	ctx = withReadAudience(ctx, GuestAudience)
	if limit < 1 || limit > 20 {
		return nil, ErrSearchValidation
	}
	if tab == "posts" || tab == "projects" {
		return s.PublicPosts(ctx, did, tab == "projects", limit, cursor)
	}
	if _, err := s.PublicProfile(ctx, did); err != nil {
		return nil, err
	}
	scope := publicCursorScope("profile/"+did.String()+"/"+tab, nil)
	inner, err := decodePublicCursor(cursor, scope)
	if err != nil {
		return nil, err
	}
	var rows []*PostRow
	var next string
	switch tab {
	case "comments":
		rows, next, err = s.posts.ListCommentsByAuthorWithLanguages(ctx, "", did.String(), []string{}, limit, inner)
	case "reposts":
		rows, next, err = s.posts.ListRepostsByAuthorWithLanguages(ctx, "", did.String(), []string{}, limit, inner)
	default:
		return nil, ErrSearchValidation
	}
	if err != nil {
		return nil, err
	}
	page, err := s.publicPostRows(ctx, rows, next)
	if err != nil {
		return nil, err
	}
	if err := bindPublicPageCursor(page, scope); err != nil {
		return nil, err
	}
	return page, nil
}

func (s *PublicPageStore) PublicThread(ctx context.Context, did syntax.DID, key syntax.RecordKey, limit int, cursor string) (*PublicThread, error) {
	ctx = withReadAudience(ctx, GuestAudience)
	if limit < 1 || limit > 20 {
		return nil, ErrSearchValidation
	}
	post, err := s.PublicPost(ctx, did, key)
	if err != nil {
		return nil, err
	}
	row, err := s.posts.ReadPostByURI(ctx, post.URI.String())
	if err != nil {
		return nil, err
	}
	out := &PublicThread{Post: post, Ancestors: []PublicEmbed{}}
	seen := map[string]bool{}
	for _, uri := range []*string{row.ReplyRootURI, row.ReplyParentURI} {
		if uri == nil || seen[*uri] {
			continue
		}
		seen[*uri] = true
		ancestor, err := s.posts.ReadPostByURI(ctx, *uri)
		if errors.Is(err, ErrPostNotFound) {
			out.Ancestors = append(out.Ancestors, PublicEmbed{State: "unavailable"})
			continue
		}
		if err != nil {
			return nil, err
		}
		page, err := s.publicPostRows(ctx, []*PostRow{ancestor}, "")
		if err != nil {
			return nil, err
		}
		out.Ancestors = append(out.Ancestors, PublicEmbed{State: "visible", Post: &page.Items[0]})
	}
	scope := publicCursorScope("thread/"+post.URI.String(), nil)
	inner, err := decodePublicCursor(cursor, scope)
	if err != nil {
		return nil, err
	}
	budget := min(limit, 20-1-len(out.Ancestors))
	rows, next, err := s.posts.ListRootComments(ctx, post.URI.String(), "", "oldest", budget, inner)
	if err != nil {
		return nil, err
	}
	page, err := s.publicPostRows(ctx, rows, next)
	if err != nil {
		return nil, err
	}
	if err := bindPublicPageCursor(page, scope); err != nil {
		return nil, err
	}
	out.Replies = *page
	return out, nil
}

func (s *PublicPageStore) PublicInteractions(ctx context.Context, did syntax.DID, key syntax.RecordKey, kind PostInteractionKind, limit int, cursor string) (any, error) {
	ctx = withReadAudience(ctx, GuestAudience)
	if limit < 1 || limit > 20 || !validPostInteractionKind(kind) {
		return nil, ErrSearchValidation
	}
	post, err := s.PublicPost(ctx, did, key)
	if err != nil {
		return nil, err
	}
	row, err := s.posts.ReadPostByURI(ctx, post.URI.String())
	if err != nil {
		return nil, err
	}
	target := &PostInteractionTarget{URI: post.URI, Author: did, IsReply: !row.IsRoot()}
	scope := publicCursorScope("interactions/"+post.URI.String()+"/"+string(kind), nil)
	inner, err := decodePublicCursor(cursor, scope)
	if err != nil {
		return nil, err
	}
	if kind == PostInteractionQuotes {
		rows, next, err := s.posts.ListQuotePosts(ctx, "", target, []string{}, limit, inner)
		if err != nil {
			return nil, err
		}
		page, err := s.publicPostRows(ctx, rows, next)
		if err != nil {
			return nil, err
		}
		if err := bindPublicPageCursor(page, scope); err != nil {
			return nil, err
		}
		return page, nil
	}
	page, err := s.posts.ListPostInteractionAccounts(ctx, "", target, kind, limit, inner)
	if err != nil {
		return nil, err
	}
	out := &PublicProfilePage{Items: []PublicProfile{}}
	if page.Cursor != nil {
		out.Cursor = *page.Cursor
	}
	for _, item := range page.Items {
		profile, err := s.PublicProfile(ctx, item.DID)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *profile)
	}
	if err := bindPublicPageCursor(out, scope); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PublicPageStore) PublicTagPosts(ctx context.Context, tag string, limit int, cursor string) (*PublicPostPage, error) {
	ctx = withReadAudience(ctx, GuestAudience)
	tag, err := NormalizeHashtagPathValue(tag)
	if err != nil || limit < 1 || limit > 20 {
		return nil, ErrSearchValidation
	}
	scope := publicCursorScope("tag", strings.ToLower(tag))
	inner, err := decodePublicCursor(cursor, scope)
	if err != nil {
		return nil, err
	}
	rows, next, err := NewSearchStore(s.pool, nil).SearchHashtagPostsWithLanguages(ctx, "", []string{}, tag, SearchSortChronological, limit, inner, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	page, err := s.publicSearchPosts(ctx, rows, next)
	if err != nil {
		return nil, err
	}
	if err := bindPublicPageCursor(page, scope); err != nil {
		return nil, err
	}
	return page, nil
}
