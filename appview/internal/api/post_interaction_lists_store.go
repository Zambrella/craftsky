package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/relationships"
)

var ErrPostInteractionIdentityUnavailable = errors.New("post interaction identity unavailable")

type PostInteractionKind string

const (
	PostInteractionLikes   PostInteractionKind = "likes"
	PostInteractionReposts PostInteractionKind = "reposts"
	PostInteractionQuotes  PostInteractionKind = "quotes"
)

type PostInteractionTarget struct {
	URI     syntax.ATURI
	Author  syntax.DID
	IsReply bool
}

type PostInteractionCursor struct {
	Kind      PostInteractionKind
	Target    syntax.ATURI
	CreatedAt time.Time
	URI       syntax.ATURI
}

func encodePostInteractionCursor(cursor PostInteractionCursor) (string, error) {
	if !validPostInteractionKind(cursor.Kind) || cursor.CreatedAt.IsZero() {
		return "", envelope.ErrInvalidCursor
	}
	if _, err := syntax.ParseATURI(cursor.Target.String()); err != nil {
		return "", envelope.ErrInvalidCursor
	}
	if _, err := syntax.ParseATURI(cursor.URI.String()); err != nil {
		return "", envelope.ErrInvalidCursor
	}
	return envelope.EncodeCursor(map[string]any{
		"kind":      string(cursor.Kind),
		"target":    cursor.Target.String(),
		"createdAt": cursor.CreatedAt.UTC().Format(time.RFC3339Nano),
		"uri":       cursor.URI.String(),
	})
}

func decodePostInteractionCursor(encoded string, kind PostInteractionKind, target syntax.ATURI) (*PostInteractionCursor, error) {
	if encoded == "" {
		return nil, nil
	}
	if !validPostInteractionKind(kind) {
		return nil, envelope.ErrInvalidCursor
	}
	payload, err := envelope.DecodeCursor(encoded)
	if err != nil || len(payload) != 4 || payload["kind"] != string(kind) || payload["target"] != target.String() {
		return nil, envelope.ErrInvalidCursor
	}
	rawCreatedAt, ok := payload["createdAt"].(string)
	if !ok || rawCreatedAt == "" {
		return nil, envelope.ErrInvalidCursor
	}
	createdAt, err := time.Parse(time.RFC3339Nano, rawCreatedAt)
	if err != nil || createdAt.IsZero() {
		return nil, envelope.ErrInvalidCursor
	}
	rawURI, ok := payload["uri"].(string)
	if !ok || rawURI == "" {
		return nil, envelope.ErrInvalidCursor
	}
	uri, err := syntax.ParseATURI(rawURI)
	if err != nil {
		return nil, envelope.ErrInvalidCursor
	}
	return &PostInteractionCursor{Kind: kind, Target: target, CreatedAt: createdAt, URI: uri}, nil
}

func validPostInteractionKind(kind PostInteractionKind) bool {
	switch kind {
	case PostInteractionLikes, PostInteractionReposts, PostInteractionQuotes:
		return true
	default:
		return false
	}
}

func eligibleAccountInteractionPredicate(actorAlias, targetAuthor, viewerParam string) string {
	moderation := strings.ReplaceAll(profileVisibleModerationPredicate, "cp.", actorAlias+".")
	return `
		  AND appview_owner_is_active(` + actorAlias + `.did)
		  AND NOT appview_owner_is_terminal(` + targetAuthor + `)
		  AND NOT ` + postAuthorBlockedPredicate(actorAlias, viewerParam) + `
		  AND NOT EXISTS (
			SELECT 1 FROM atproto_blocks block
			WHERE ((block.blocker_did = ` + actorAlias + `.did AND block.subject_did = ` + targetAuthor + `)
			    OR (block.blocker_did = ` + targetAuthor + ` AND block.subject_did = ` + actorAlias + `.did))
			  AND NOT appview_owner_is_terminal(block.blocker_did)
			  AND NOT appview_owner_is_terminal(block.subject_did)
		  )
		  ` + moderation
}

func eligibleQuotePredicate(alias, viewerParam, languagesParam string) string {
	moderation := strings.ReplaceAll(postVisibleModerationPredicate, "p.", alias+".")
	language := strings.ReplaceAll(languageVisibilityPredicate("p", viewerParam, languagesParam), "p.", alias+".")
	return moderation + `
		  AND NOT ` + postAuthorBlockedPredicate(alias, viewerParam) + `
		  AND NOT ` + postQuoteAuthorBlockedPredicate(alias) + language
}

func (s *PostStore) ResolveInteractionTarget(ctx context.Context, viewer, owner syntax.DID, rkey syntax.RecordKey, kind PostInteractionKind) (*PostInteractionTarget, error) {
	uri := syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/" + rkey.String())
	posts, err := s.ReadEligiblePostsByURI(ctx, viewer, []syntax.ATURI{uri})
	if err != nil {
		return nil, err
	}
	post := posts[uri]
	if post == nil {
		return nil, ErrPostNotFound
	}
	contexts, err := s.RequiredContextStates(ctx, viewer, []syntax.ATURI{uri})
	if err != nil {
		return nil, err
	}
	if !contexts[uri] {
		return nil, ErrPostNotFound
	}
	target := &PostInteractionTarget{
		URI:     uri,
		Author:  syntax.DID(post.DID),
		IsReply: !post.IsRoot(),
	}
	if kind != PostInteractionLikes && target.IsReply {
		return nil, ErrPostNotFound
	}
	return target, nil
}

type postInteractionAccountRow struct {
	profile     ProfileAccountRow
	handle      syntax.Handle
	createdAt   time.Time
	interaction syntax.ATURI
}

func PostInteractionAccountListQuery(kind PostInteractionKind) (string, error) {
	var interactionTable string
	switch kind {
	case PostInteractionLikes:
		interactionTable = "craftsky_likes"
	case PostInteractionReposts:
		interactionTable = "craftsky_reposts"
	default:
		return "", fmt.Errorf("unsupported post interaction kind %q", kind)
	}
	return `
		WITH eligible AS (
			SELECT interaction.uri, interaction.created_at, actor.did,
			       bp.display_name, bp.description, bp.avatar_cid, bp.avatar_mime,
			       identity.handle,
			       EXISTS (
					SELECT 1 FROM actor_mutes mute
					WHERE mute.owner_did = $2 AND mute.subject_did = actor.did
					  AND NOT appview_owner_is_terminal(mute.owner_did)
					  AND NOT appview_owner_is_terminal(mute.subject_did)
			       ) AS muted
			FROM ` + interactionTable + ` interaction
			JOIN craftsky_profiles actor ON actor.did = interaction.did
			LEFT JOIN bluesky_profiles bp ON bp.did = actor.did
			LEFT JOIN atproto_identity_cache identity ON identity.did = actor.did
			WHERE interaction.subject_uri = $1
			  AND interaction.deleted_at IS NULL
			  ` + eligibleAccountInteractionPredicate("actor", "$3", "$2") + `
		), totals AS (
			SELECT count(*) AS total_count
			FROM eligible
		), page AS (
			SELECT *
			FROM eligible
			WHERE ($4::timestamptz IS NULL OR (created_at, uri) < ($4, $5))
			ORDER BY created_at DESC, uri DESC
			LIMIT $6
		)
		SELECT page.uri, page.created_at, page.did, page.display_name, page.description,
		       page.avatar_cid, page.avatar_mime, page.handle, page.muted, totals.total_count
		FROM totals
		LEFT JOIN page ON true
		ORDER BY page.created_at DESC, page.uri DESC
	`, nil
}

func (s *PostStore) ListPostInteractionAccounts(ctx context.Context, viewer syntax.DID, target *PostInteractionTarget, kind PostInteractionKind, limit int, cursor string) (ProfileAccountPage, error) {
	var page ProfileAccountPage
	err := s.observeDB(ctx, postInteractionListDBOperation(kind), postInteractionListRoutePattern(kind), func(ctx context.Context) error {
		var listErr error
		page, listErr = s.listPostInteractionAccounts(ctx, viewer, target, kind, limit, cursor)
		return listErr
	})
	return page, err
}

func (s *PostStore) listPostInteractionAccounts(ctx context.Context, viewer syntax.DID, target *PostInteractionTarget, kind PostInteractionKind, limit int, cursor string) (ProfileAccountPage, error) {
	if target == nil {
		return ProfileAccountPage{}, fmt.Errorf("unsupported post interaction kind %q", kind)
	}
	query, err := PostInteractionAccountListQuery(kind)
	if err != nil {
		return ProfileAccountPage{}, err
	}
	decodedCursor, err := decodePostInteractionCursor(cursor, kind, target.URI)
	if err != nil {
		return ProfileAccountPage{}, err
	}
	var cursorTime, cursorURI any
	if decodedCursor != nil {
		cursorTime = decodedCursor.CreatedAt
		cursorURI = decodedCursor.URI
	}
	if limit < 1 {
		limit = 1
	}

	rows, err := s.pool.Query(ctx, query, target.URI, viewer, target.Author, cursorTime, cursorURI, limit+1)
	if err != nil {
		return ProfileAccountPage{}, fmt.Errorf("post interaction accounts query: %w", err)
	}
	defer rows.Close()

	result := make([]postInteractionAccountRow, 0, limit+1)
	page := ProfileAccountPage{Items: []ProfileAccountSummary{}}
	for rows.Next() {
		var row postInteractionAccountRow
		var interaction *syntax.ATURI
		var createdAt *time.Time
		var did, displayName, description, avatarCID, avatarMime, rawHandle *string
		var muted *bool
		if err := rows.Scan(
			&interaction, &createdAt, &did, &displayName, &description, &avatarCID, &avatarMime,
			&rawHandle, &muted, &page.TotalCount,
		); err != nil {
			return ProfileAccountPage{}, fmt.Errorf("post interaction accounts scan: %w", err)
		}
		if interaction == nil {
			continue
		}
		row.interaction = *interaction
		row.createdAt = *createdAt
		row.profile.DID = *did
		row.profile.DisplayName = displayName
		row.profile.Description = description
		row.profile.AvatarCID = avatarCID
		row.profile.AvatarMime = avatarMime
		row.profile.Muted = *muted
		row.profile.IsCraftskyProfile = true
		if rawHandle == nil {
			return ProfileAccountPage{}, ErrPostInteractionIdentityUnavailable
		}
		row.handle, err = syntax.ParseHandle(*rawHandle)
		if err != nil {
			return ProfileAccountPage{}, fmt.Errorf("%w: invalid cached handle", ErrPostInteractionIdentityUnavailable)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return ProfileAccountPage{}, fmt.Errorf("post interaction accounts iter: %w", err)
	}

	hasMore := len(result) > limit
	if hasMore {
		result = result[:limit]
	}
	for i := range result {
		page.Items = append(page.Items, BuildProfileAccountSummary(&result[i].profile, result[i].handle))
	}
	if hasMore {
		last := result[len(result)-1]
		encoded, encodeErr := encodePostInteractionCursor(PostInteractionCursor{
			Kind:      kind,
			Target:    target.URI,
			CreatedAt: last.createdAt,
			URI:       last.interaction,
		})
		if encodeErr != nil {
			return ProfileAccountPage{}, encodeErr
		}
		page.Cursor = &encoded
	}
	return page, nil
}

func PostQuoteListQuery() string {
	return `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.quote_uri = $1
		  AND p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		` + eligibleQuotePredicate("p", "$2", "$3") + `
		  AND ($4::timestamptz IS NULL OR (p.created_at, p.uri) < ($4, $5))
		ORDER BY p.created_at DESC, p.uri DESC
		LIMIT $6
	`
}

func (s *PostStore) ListQuotePosts(ctx context.Context, viewer syntax.DID, target *PostInteractionTarget, contentLanguages []string, limit int, cursor string) ([]*PostRow, string, error) {
	var rows []*PostRow
	var next string
	err := s.observeDB(ctx, postInteractionListDBOperation(PostInteractionQuotes), postInteractionListRoutePattern(PostInteractionQuotes), func(ctx context.Context) error {
		var listErr error
		rows, next, listErr = s.listQuotePosts(ctx, viewer, target, contentLanguages, limit, cursor)
		return listErr
	})
	return rows, next, err
}

func (s *PostStore) listQuotePosts(ctx context.Context, viewer syntax.DID, target *PostInteractionTarget, contentLanguages []string, limit int, cursor string) ([]*PostRow, string, error) {
	if target == nil || target.IsReply {
		return nil, "", ErrPostNotFound
	}
	decodedCursor, err := decodePostInteractionCursor(cursor, PostInteractionQuotes, target.URI)
	if err != nil {
		return nil, "", err
	}
	var cursorTime, cursorURI any
	if decodedCursor != nil {
		cursorTime = decodedCursor.CreatedAt
		cursorURI = decodedCursor.URI
	}
	if limit < 1 {
		limit = 1
	}
	if contentLanguages == nil {
		contentLanguages = []string{}
	}

	rows, err := s.pool.Query(ctx, PostQuoteListQuery(), target.URI, viewer, contentLanguages, cursorTime, cursorURI, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("quote posts query: %w", err)
	}
	defer rows.Close()

	result := make([]*PostRow, 0, limit+1)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("quote posts scan: %w", scanErr)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("quote posts iter: %w", err)
	}

	if len(result) <= limit {
		return result, "", nil
	}
	result = result[:limit]
	last := result[len(result)-1]
	next, err := encodePostInteractionCursor(PostInteractionCursor{
		Kind:      PostInteractionQuotes,
		Target:    target.URI,
		CreatedAt: last.CreatedAt,
		URI:       syntax.ATURI(last.URI),
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode quote posts cursor: %w", err)
	}
	return result, next, nil
}

func postInteractionListDBOperation(kind PostInteractionKind) string {
	switch kind {
	case PostInteractionLikes, PostInteractionReposts, PostInteractionQuotes:
		return "post.interactions." + string(kind)
	default:
		return "post.interactions.unknown"
	}
}

func postInteractionListRoutePattern(kind PostInteractionKind) string {
	switch kind {
	case PostInteractionLikes, PostInteractionReposts, PostInteractionQuotes:
		return "/v1/posts/{did}/{rkey}/" + string(kind)
	default:
		return "unmatched"
	}
}

func (s *PostStore) ListQuotePostResponses(
	ctx context.Context,
	viewer syntax.DID,
	target *PostInteractionTarget,
	contentLanguages []string,
	limit int,
	cursor string,
	_ HandleResolver,
) ([]*PostResponse, string, error) {
	rows, next, err := s.ListQuotePosts(ctx, viewer, target, contentLanguages, limit, cursor)
	if err != nil || len(rows) == 0 {
		return []*PostResponse{}, next, err
	}

	postURIs := make([]string, 0, len(rows))
	subjects := make([]syntax.DID, 0, len(rows))
	seenSubjects := make(map[syntax.DID]struct{}, len(rows))
	for _, row := range rows {
		postURIs = append(postURIs, row.URI)
		did, parseErr := syntax.ParseDID(row.DID)
		if parseErr != nil {
			return nil, "", parseErr
		}
		if _, seen := seenSubjects[did]; !seen {
			seenSubjects[did] = struct{}{}
			subjects = append(subjects, did)
		}
	}

	summaries, err := s.EngagementSummaries(ctx, viewer.String(), contentLanguages, postURIs)
	if err != nil {
		return nil, "", err
	}
	states, err := s.RelationshipStates(ctx, viewer, subjects)
	if err != nil {
		return nil, "", err
	}
	handleDIDs := append(append([]syntax.DID(nil), subjects...), target.Author)
	handles, err := s.postInteractionHandles(ctx, handleDIDs)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrPostInteractionIdentityUnavailable, err)
	}
	resolver := fixedHandleResolver(handles)

	items := make([]*PostResponse, 0, len(rows))
	for _, row := range rows {
		response := buildPostResponse(row, handles[row.DID], s)
		applyEngagementSummary(response, summaries[row.URI])
		state := states[syntax.DID(row.DID)]
		ApplyPostAuthorViewerState(response, state)
		ApplyPostRelationshipPolicy(response, state, relationships.SurfaceQuote)
		items = append(items, response)
	}
	if err := attachQuoteViews(ctx, s, resolver, items); err != nil {
		if errors.Is(err, ErrHandleUnavailable) {
			return nil, "", fmt.Errorf("%w: %v", ErrPostInteractionIdentityUnavailable, err)
		}
		return nil, "", err
	}
	return items, next, nil
}

type fixedHandleResolver map[string]syntax.Handle

func (r fixedHandleResolver) ResolveHandle(_ context.Context, did syntax.DID) (syntax.Handle, error) {
	handle, ok := r[did.String()]
	if !ok {
		return "", ErrPostInteractionIdentityUnavailable
	}
	return handle, nil
}

func (r fixedHandleResolver) ResolveDID(_ context.Context, handle syntax.Handle) (syntax.DID, error) {
	for did, candidate := range r {
		if candidate == handle {
			return syntax.DID(did), nil
		}
	}
	return "", ErrPostInteractionIdentityUnavailable
}

func (s *PostStore) postInteractionHandles(ctx context.Context, dids []syntax.DID) (map[string]syntax.Handle, error) {
	unique := make([]string, 0, len(dids))
	seen := make(map[syntax.DID]struct{}, len(dids))
	for _, did := range dids {
		if _, ok := seen[did]; ok {
			continue
		}
		seen[did] = struct{}{}
		unique = append(unique, did.String())
	}
	rows, err := s.pool.Query(ctx, `
		SELECT did, handle
		FROM atproto_identity_cache
		WHERE did = ANY($1::text[])
	`, unique)
	if err != nil {
		return nil, fmt.Errorf("post interaction identity query: %w", err)
	}
	defer rows.Close()
	handles := make(map[string]syntax.Handle, len(unique))
	for rows.Next() {
		var did string
		var rawHandle string
		if err := rows.Scan(&did, &rawHandle); err != nil {
			return nil, fmt.Errorf("post interaction identity scan: %w", err)
		}
		handle, err := syntax.ParseHandle(rawHandle)
		if err != nil {
			return nil, fmt.Errorf("invalid cached handle")
		}
		handles[did] = handle
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("post interaction identity iter: %w", err)
	}
	if len(handles) != len(unique) {
		return nil, fmt.Errorf("cached identity missing")
	}
	return handles, nil
}
