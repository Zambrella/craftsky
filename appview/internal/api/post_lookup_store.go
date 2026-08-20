// appview/internal/api/post_lookup_store.go
package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

// ReadPostByURI returns the post identified by AT-URI.
func (s *PostStore) ReadPostByURI(ctx context.Context, uri string) (*PostRow, error) {
	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.uri = $1
		` + postVisibleModerationPredicate + `
	`
	row, err := scanPostRow(s.pool.QueryRow(ctx, q, uri))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("post read uri %s: %w", uri, err)
	}
	return row, nil
}

// ReadEligiblePostsByURI batch-loads current canonical posts for the private
// saved surface. Missing membership, moderation-hidden, and directly blocked
// targets are omitted without deleting their private save rows.
func (s *PostStore) ReadEligiblePostsByURI(ctx context.Context, viewer syntax.DID, uris []syntax.ATURI) (map[syntax.ATURI]*PostRow, error) {
	out := make(map[syntax.ATURI]*PostRow, len(uris))
	if len(uris) == 0 {
		return out, nil
	}
	values := make([]string, 0, len(uris))
	for _, uri := range uris {
		values = append(values, uri.String())
	}
	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		JOIN craftsky_profiles member ON member.did = p.did
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.uri = ANY($1::text[])
		  AND NOT ` + postAuthorBlockedPredicate("p", "$2") + `
		` + postVisibleModerationPredicate + `
	`
	rows, err := s.pool.Query(ctx, q, values, viewer)
	if err != nil {
		return nil, fmt.Errorf("saved post batch read: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		row, err := scanPostRow(rows)
		if err != nil {
			return nil, fmt.Errorf("saved post batch scan: %w", err)
		}
		out[syntax.ATURI(row.URI)] = row
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("saved post batch iter: %w", err)
	}
	return out, nil
}

// RequiredContextStates validates reply navigation context in one bounded
// recursive query. It never deletes saves; callers omit invalid contexts and
// allow them to reappear if the indexed chain becomes eligible again.
func (s *PostStore) RequiredContextStates(ctx context.Context, viewer syntax.DID, uris []syntax.ATURI) (map[syntax.ATURI]bool, error) {
	out := make(map[syntax.ATURI]bool, len(uris))
	if len(uris) == 0 {
		return out, nil
	}
	values := make([]string, 0, len(uris))
	for _, uri := range uris {
		values = append(values, uri.String())
		out[uri] = false
	}
	parentModeration := strings.ReplaceAll(postVisibleModerationPredicate, "p.", "parent.")
	rootModeration := strings.ReplaceAll(postVisibleModerationPredicate, "p.", "root.")
	q := `
		WITH RECURSIVE input(uri) AS (
			SELECT unnest($1::text[])
		), targets AS (
			SELECT input.uri, p.reply_root_uri, p.reply_parent_uri
			FROM input
			LEFT JOIN craftsky_posts p
			  ON p.uri = input.uri
			 AND NOT appview_owner_is_terminal(p.did)
		), walk(target_uri, current_uri, root_uri, path, depth) AS (
			SELECT uri, reply_parent_uri, reply_root_uri, ARRAY[uri]::text[], 0
			FROM targets
			WHERE reply_root_uri IS NOT NULL AND reply_parent_uri IS NOT NULL
			UNION ALL
			SELECT walk.target_uri, parent.reply_parent_uri, walk.root_uri,
			       walk.path || parent.uri, walk.depth + 1
			FROM walk
			JOIN craftsky_posts parent ON parent.uri = walk.current_uri
			JOIN craftsky_profiles parent_member ON parent_member.did = parent.did
			WHERE walk.current_uri <> walk.root_uri
			  AND walk.depth < 64
			  AND NOT parent.uri = ANY(walk.path)
			  AND NOT ` + postAuthorBlockedPredicate("parent", "$2") + `
			  ` + parentModeration + `
		), valid_replies AS (
			SELECT DISTINCT walk.target_uri
			FROM walk
			JOIN craftsky_posts root
			  ON root.uri = walk.current_uri
			 AND root.uri = walk.root_uri
			 AND root.reply_root_uri IS NULL
			 AND root.reply_parent_uri IS NULL
			JOIN craftsky_profiles root_member ON root_member.did = root.did
			WHERE NOT ` + postAuthorBlockedPredicate("root", "$2") + `
			` + rootModeration + `
		)
		SELECT targets.uri,
		       CASE
				WHEN p.uri IS NULL THEN false
				WHEN targets.reply_root_uri IS NULL AND targets.reply_parent_uri IS NULL THEN true
				WHEN targets.reply_root_uri IS NULL OR targets.reply_parent_uri IS NULL THEN false
				ELSE valid_replies.target_uri IS NOT NULL
		       END
		FROM targets
		LEFT JOIN craftsky_posts p
		  ON p.uri = targets.uri
		 AND NOT appview_owner_is_terminal(p.did)
		LEFT JOIN valid_replies ON valid_replies.target_uri = targets.uri
	`
	rows, err := s.pool.Query(ctx, q, values, viewer)
	if err != nil {
		return nil, fmt.Errorf("saved post context states: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rawURI string
		var valid bool
		if err := rows.Scan(&rawURI, &valid); err != nil {
			return nil, fmt.Errorf("saved post context states scan: %w", err)
		}
		out[syntax.ATURI(rawURI)] = valid
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("saved post context states iter: %w", err)
	}
	return out, nil
}

// ReadOne returns the post identified by (did, rkey). Returns
// ErrPostNotFound when no row exists.
func (s *PostStore) ReadOne(ctx context.Context, did, rkey string) (*PostRow, error) {
	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1 AND p.rkey = $2
		` + postVisibleModerationPredicate + `
	`
	row, err := scanPostRow(s.pool.QueryRow(ctx, q, did, rkey))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("post read %s/%s: %w", did, rkey, err)
	}
	return row, nil
}

// ReadAuthor returns the bluesky_profiles display fields for did.
// Returns (&PostAuthorRow{nil, nil}, nil) — not an error — when the
// user has no bluesky_profiles row yet. The post-create path uses this
// to hydrate authors whose row hasn't been indexed yet.
func (s *PostStore) ReadAuthor(ctx context.Context, did string) (*PostAuthorRow, error) {
	const q = `
		SELECT display_name, avatar_cid, avatar_mime
		FROM bluesky_profiles
		WHERE did = $1 AND NOT appview_owner_is_terminal(did)
	`
	out := &PostAuthorRow{}
	err := s.pool.QueryRow(ctx, q, did).Scan(&out.DisplayName, &out.AvatarCID, &out.AvatarMime)
	if errors.Is(err, pgx.ErrNoRows) {
		return &PostAuthorRow{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("post read author %s: %w", did, err)
	}
	return out, nil
}

// ResolvePostTarget returns the URI/CID identity for a post addressed by
// author DID and rkey. It shares ReadOne's ErrPostNotFound contract.
func (s *PostStore) ResolvePostTarget(ctx context.Context, did, rkey string) (*PostTargetRef, error) {
	const q = `
		SELECT uri, cid
		FROM craftsky_posts
		WHERE did = $1 AND rkey = $2
		  AND NOT appview_owner_is_terminal(did)
	`
	out := &PostTargetRef{}
	err := s.pool.QueryRow(ctx, q, did, rkey).Scan(&out.URI, &out.CID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("post resolve target %s/%s: %w", did, rkey, err)
	}
	return out, nil
}

// ResolveSavedPostTarget applies the saved surface's current direct-access
// policy before private state is created. Muted targets remain eligible;
// missing membership, moderation-hidden content, and either block direction
// are indistinguishable from a missing post.
func (s *PostStore) ResolveSavedPostTarget(ctx context.Context, viewer, did syntax.DID, rkey syntax.RecordKey) (syntax.ATURI, error) {
	uri := syntax.ATURI("at://" + did.String() + "/social.craftsky.feed.post/" + rkey.String())
	rows, err := s.ReadEligiblePostsByURI(ctx, viewer, []syntax.ATURI{uri})
	if err != nil {
		return "", err
	}
	if rows[uri] == nil {
		return "", ErrPostNotFound
	}
	contexts, err := s.RequiredContextStates(ctx, viewer, []syntax.ATURI{uri})
	if err != nil {
		return "", err
	}
	if !contexts[uri] {
		return "", ErrPostNotFound
	}
	return uri, nil
}

// ResolveShareTarget returns visible indexed target metadata for share actions.
func (s *PostStore) ResolveShareTarget(ctx context.Context, did, rkey string) (*ShareTargetRef, error) {
	q := `
		SELECT p.uri, p.cid,
		       (p.reply_root_uri IS NOT NULL OR p.reply_parent_uri IS NOT NULL) AS is_reply,
		       p.is_project
		FROM craftsky_posts p
		WHERE p.did = $1 AND p.rkey = $2
		` + postVisibleModerationPredicate + `
	`
	out := &ShareTargetRef{}
	err := s.pool.QueryRow(ctx, q, did, rkey).Scan(&out.URI, &out.CID, &out.IsReply, &out.IsProject)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("post resolve share target %s/%s: %w", did, rkey, err)
	}
	return out, nil
}

// ResolvePostReportTarget returns the canonical indexed post snapshot used for
// report eligibility and private report persistence. Platform moderation does
// not make a current member's addressable post ineligible, but membership loss
// hides the retained public record from all user-facing report targets.
func (s *PostStore) ResolvePostReportTarget(ctx context.Context, did syntax.DID, rkey syntax.RecordKey) (*PostReportTarget, error) {
	const q = `
		SELECT p.uri, p.cid
		FROM craftsky_posts p
		JOIN craftsky_profiles profile ON profile.did = p.did
		WHERE p.did = $1 AND p.rkey = $2
		  AND NOT appview_owner_is_terminal(p.did)
	`
	out := &PostReportTarget{DID: did.String(), Rkey: rkey.String()}
	err := s.pool.QueryRow(ctx, q, did, rkey).Scan(&out.URI, &out.CIDSnapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("post resolve report target %s/%s: %w", did, rkey, err)
	}
	return out, nil
}
