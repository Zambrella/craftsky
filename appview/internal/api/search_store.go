package api

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

// Search query capability owners:
//   - SearchProfiles: search_profile_store.go
//   - SearchPostsWithLanguages: search_post_store.go
//
// Shared SQL predicates and row scanners remain here because post, project,
// and hashtag-post searches all rely on their exact semantics.

type SearchPostRow struct {
	Post  *PostRow
	Score float64
}

const (
	relevanceCursorKindPosts    = "searchPostsRelevance"
	relevanceCursorKindProjects = "searchProjectsRelevance"
)

func searchTSQuery(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}

func relationshipTopLevelPredicate(viewerParam string) string {
	return `
		  AND NOT EXISTS (
			SELECT 1 FROM actor_mutes mute
			WHERE mute.owner_did = ` + viewerParam + ` AND mute.subject_did = p.did
			  AND NOT appview_owner_is_terminal(mute.owner_did)
			  AND NOT appview_owner_is_terminal(mute.subject_did)
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM atproto_blocks block
			WHERE ((block.blocker_did = ` + viewerParam + ` AND block.subject_did = p.did)
			   OR (block.blocker_did = p.did AND block.subject_did = ` + viewerParam + `))
			  AND NOT appview_owner_is_terminal(block.blocker_did)
			  AND NOT appview_owner_is_terminal(block.subject_did)
		  )
	`
}

func scanSearchPostRow(scanner pgx.Row) (SearchPostRow, error) {
	row, err := scanPostRowWithExtraScore(scanner)
	return row, err
}

func scanPostRowWithExtraScore(scanner pgx.Row) (SearchPostRow, error) {
	post := &PostRow{}
	var rawProject *[]byte
	var score float64
	err := scanner.Scan(
		&post.URI, &post.DID, &post.Rkey, &post.CID, &post.Text, &post.Facets, &post.Images, &post.RawEmbed,
		&post.ReplyRootURI, &post.ReplyRootCID, &post.ReplyParentURI, &post.ReplyParentCID,
		&post.QuoteURI, &post.QuoteCID, &post.Tags, &post.Langs, &post.CreatedAt, &post.IndexedAt,
		&post.ExternalImportSource, &post.ProfileSortAt,
		&post.IsProject, &post.ProjectCraftType, &rawProject,
		&post.AuthorDisplayName, &post.AuthorAvatarCID, &post.AuthorAvatarMime,
		&post.ModerationWarningKind,
		&score,
	)
	if err != nil {
		return SearchPostRow{}, err
	}
	if rawProject != nil && len(*rawProject) > 0 {
		post.RawProject = append([]byte(nil), (*rawProject)...)
		var project Project
		if err := json.Unmarshal(post.RawProject, &project); err != nil {
			return SearchPostRow{}, err
		}
		post.Project = &project
	}
	return SearchPostRow{Post: post, Score: score}, nil
}

func (s *SearchStore) EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error) {
	return s.engagementReader.EngagementSummaries(ctx, viewerDID, postURIs)
}

func (s *SearchStore) QuoteViewRows(ctx context.Context, refs []ResponseStrongRef) (map[string]*QuoteViewRow, error) {
	return s.quoteReader.QuoteViewRows(ctx, refs)
}

func (s *SearchStore) observeDB(ctx context.Context, operation, routePattern string, fn func(context.Context) error) error {
	if s == nil || s.observer == nil {
		return fn(ctx)
	}
	return s.observer.ObserveDB(ctx, observability.DBOperation{
		Operation:    operation,
		RoutePattern: routePattern,
		RunID:        middleware.GetRunID(ctx),
	}, fn)
}
