// appview/internal/api/post_store.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/observability"
)

// ErrPostNotFound is returned by PostStore.ReadOne when no row matches.
var ErrPostNotFound = errors.New("post: not found")

// ErrInteractionNotFound is returned when no active like/repost matches.
var ErrInteractionNotFound = errors.New("interaction: not found")

// PostRow is the joined view of craftsky_posts plus author display fields
// from bluesky_profiles. Reply/quote pointers are kept as separate
// pointers so handlers can decide nesting at the JSON layer.
type PostRow struct {
	URI                  string
	DID                  string
	Rkey                 string
	CID                  string
	Text                 string
	Facets               json.RawMessage
	Images               json.RawMessage
	RawEmbed             json.RawMessage
	ReplyRootURI         *string
	ReplyRootCID         *string
	ReplyParentURI       *string
	ReplyParentCID       *string
	QuoteURI             *string
	QuoteCID             *string
	Tags                 []string
	Langs                []string
	CreatedAt            time.Time
	IndexedAt            time.Time
	ExternalImportSource *string
	ProfileSortAt        time.Time
	IsProject            bool
	ProjectCraftType     *string
	RawProject           json.RawMessage

	AuthorDisplayName *string
	AuthorAvatarCID   *string
	AuthorAvatarMime  *string

	ModerationWarningKind *string

	Project *Project
}

func (row *PostRow) IsRoot() bool {
	return row != nil && row.ReplyRootURI == nil && row.ReplyParentURI == nil
}

func (row *PostRow) IsComment() bool {
	return row != nil && row.ReplyRootURI != nil && row.ReplyParentURI != nil && *row.ReplyRootURI == *row.ReplyParentURI
}

// PostAuthorRow is the slim author-hydration view used when we need to
// build a synthetic response for a freshly-created post (the post row
// itself doesn't exist yet at that moment, but the author's bsky
// profile may).
type PostAuthorRow struct {
	DisplayName *string
	AvatarCID   *string
	AvatarMime  *string
}

// PostTargetRef is the indexed post identity needed before writing an
// interaction against a post.
type PostTargetRef struct {
	URI string
	CID string
}

// ShareTargetRef is the indexed post identity and eligibility metadata for
// amplification actions such as reposts and quote posts.
type ShareTargetRef struct {
	URI       string
	CID       string
	IsReply   bool
	IsProject bool
}

// InteractionRow is an active indexed like or repost record.
type InteractionRow struct {
	URI        string
	DID        string
	Rkey       string
	CID        string
	SubjectURI string
	SubjectCID string
	CreatedAt  time.Time
	IndexedAt  time.Time
}

// ViewerInteractionState is the current viewer's active state for one post.
type ViewerInteractionState struct {
	HasLiked    bool
	HasReposted bool
}

// ViewerReplyState is the current viewer's authored direct-reply state for one post.
type ViewerReplyState struct {
	HasReplied bool
}

// ViewerSavedState is the authenticated viewer's private state for one post.
type ViewerSavedState struct {
	HasSaved bool
	FolderID *string
}

// EngagementSummary is the batch-friendly read model used to augment posts.
type EngagementSummary struct {
	LikeCount           int
	RepostCount         int
	QuoteCount          int
	ReplyCount          int
	ViewerHasLiked      bool
	ViewerHasReposted   bool
	ViewerHasReplied    bool
	ViewerHasSaved      bool
	ViewerSavedFolderID *string
}

type RelationshipPair struct {
	First  syntax.DID
	Second syntax.DID
}

// PostStore is the Postgres-backed implementation.
type PostStore struct {
	pool          *pgxpool.Pool
	observer      *observability.Observer
	videoPlayback PlaybackURLBuilder
}

func NewPostStore(pool *pgxpool.Pool, observer ...*observability.Observer) *PostStore {
	store := &PostStore{pool: pool}
	if len(observer) > 0 {
		store.observer = observer[0]
	}
	return store
}

func NewPostStoreWithPlayback(pool *pgxpool.Pool, observer *observability.Observer, playback PlaybackURLBuilder) *PostStore {
	store := NewPostStore(pool, observer)
	store.videoPlayback = playback
	return store
}

func (s *PostStore) PostPlaybackURLBuilder() PlaybackURLBuilder {
	return s.videoPlayback
}

const postSelectColumns = `
	p.uri, p.did, p.rkey, p.cid, p.text, p.facets, p.images, p.record -> 'embed',
	p.reply_root_uri, p.reply_root_cid, p.reply_parent_uri, p.reply_parent_cid,
	p.quote_uri, p.quote_cid, p.tags, p.langs, p.created_at, p.indexed_at,
	p.external_import_source, p.profile_sort_at,
	p.is_project, p.project_craft_type, pp.raw_project,
	bp.display_name, bp.avatar_cid, bp.avatar_mime,
	CASE
		WHEN EXISTS (
			SELECT 1
			FROM moderation_outputs mo
			WHERE mo.action = 'apply'
			  AND NOT appview_owner_is_terminal(mo.source_did)
			  AND mo.subject_type = 'post'
			  AND mo.subject_uri = p.uri
			  AND mo.value = 'warn'
			  AND (mo.expires_at IS NULL OR mo.expires_at > now())
			  AND NOT EXISTS (
				SELECT 1
				FROM moderation_outputs neg
				WHERE neg.action = 'negate'
				  AND neg.source_did = mo.source_did
				  AND neg.subject_type = mo.subject_type
				  AND neg.subject_did = mo.subject_did
				  AND neg.subject_uri = mo.subject_uri
				  AND neg.value = mo.value
				  AND (neg.expires_at IS NULL OR neg.expires_at > now())
				  AND neg.indexed_at > mo.indexed_at
			  )
		) THEN 'post'
		WHEN EXISTS (
			SELECT 1
			FROM moderation_outputs mo
			WHERE mo.action = 'apply'
			  AND NOT appview_owner_is_terminal(mo.source_did)
			  AND mo.subject_type = 'account'
			  AND mo.subject_did = p.did
			  AND mo.value = 'warn'
			  AND (mo.expires_at IS NULL OR mo.expires_at > now())
			  AND NOT EXISTS (
				SELECT 1
				FROM moderation_outputs neg
				WHERE neg.action = 'negate'
				  AND neg.source_did = mo.source_did
				  AND neg.subject_type = mo.subject_type
				  AND neg.subject_did = mo.subject_did
				  AND neg.value = mo.value
				  AND (neg.expires_at IS NULL OR neg.expires_at > now())
				  AND neg.indexed_at > mo.indexed_at
			  )
		) THEN 'author'
		ELSE NULL
	END AS moderation_warning_kind
`

const postVisibleModerationPredicate = `
		  AND NOT appview_owner_is_terminal(p.did)
		  AND NOT EXISTS (
			SELECT 1
			FROM moderation_outputs mo
			WHERE mo.action = 'apply'
			  AND NOT appview_owner_is_terminal(mo.source_did)
			  AND mo.value IN ('hide', 'takedown')
			  AND (mo.expires_at IS NULL OR mo.expires_at > now())
			  AND (
				(mo.subject_type = 'post' AND mo.subject_uri = p.uri)
				OR (mo.subject_type = 'account' AND mo.subject_did = p.did)
			  )
			  AND NOT EXISTS (
				SELECT 1
				FROM moderation_outputs neg
				WHERE neg.action = 'negate'
				  AND neg.source_did = mo.source_did
				  AND neg.subject_type = mo.subject_type
				  AND neg.subject_did = mo.subject_did
				  AND neg.value = mo.value
				  AND (neg.expires_at IS NULL OR neg.expires_at > now())
				  AND neg.indexed_at > mo.indexed_at
				  AND (mo.subject_type = 'account' OR neg.subject_uri = mo.subject_uri)
			  )
		  )
`

func postAuthorBlockedPredicate(alias, viewerParam string) string {
	return `EXISTS (
		SELECT 1 FROM atproto_blocks block
		WHERE ((block.blocker_did = ` + viewerParam + ` AND block.subject_did = ` + alias + `.did)
		   OR (block.blocker_did = ` + alias + `.did AND block.subject_did = ` + viewerParam + `))
		  AND NOT appview_owner_is_terminal(block.blocker_did)
		  AND NOT appview_owner_is_terminal(block.subject_did)
	)`
}

func postAuthorMutedPredicate(alias, viewerParam string) string {
	return `EXISTS (
		SELECT 1 FROM actor_mutes mute
		WHERE mute.owner_did = ` + viewerParam + `
		  AND mute.subject_did = ` + alias + `.did
		  AND NOT appview_owner_is_terminal(mute.owner_did)
		  AND NOT appview_owner_is_terminal(mute.subject_did)
	)`
}

func postReplyAuthorBlockedPredicate(alias string) string {
	return `EXISTS (
		SELECT 1
		FROM craftsky_posts parent_post
		JOIN atproto_blocks block ON
			(block.blocker_did = parent_post.did AND block.subject_did = ` + alias + `.did)
			OR (block.blocker_did = ` + alias + `.did AND block.subject_did = parent_post.did)
		WHERE parent_post.uri = ` + alias + `.reply_parent_uri
		  AND NOT appview_owner_is_terminal(parent_post.did)
		  AND NOT appview_owner_is_terminal(block.blocker_did)
		  AND NOT appview_owner_is_terminal(block.subject_did)
	)`
}

func postMentionAuthorBlockedPredicate(alias string) string {
	return `EXISTS (
		SELECT 1
		FROM craftsky_post_mentions mention
		JOIN atproto_blocks block ON
			(block.blocker_did = mention.mentioned_did AND block.subject_did = ` + alias + `.did)
			OR (block.blocker_did = ` + alias + `.did AND block.subject_did = mention.mentioned_did)
		WHERE mention.post_uri = ` + alias + `.uri
		  AND NOT appview_owner_is_terminal(mention.mentioned_did)
		  AND NOT appview_owner_is_terminal(block.blocker_did)
		  AND NOT appview_owner_is_terminal(block.subject_did)
	)`
}

func postQuoteAuthorBlockedPredicate(alias string) string {
	return `EXISTS (
		SELECT 1
		FROM craftsky_posts quoted_post
		JOIN atproto_blocks block ON
			(block.blocker_did = quoted_post.did AND block.subject_did = ` + alias + `.did)
			OR (block.blocker_did = ` + alias + `.did AND block.subject_did = quoted_post.did)
		WHERE quoted_post.uri = ` + alias + `.quote_uri
		  AND NOT appview_owner_is_terminal(quoted_post.did)
		  AND NOT appview_owner_is_terminal(block.blocker_did)
		  AND NOT appview_owner_is_terminal(block.subject_did)
	)`
}

func scanPostRow(scanner pgx.Row) (*PostRow, error) {
	return scanPostRowWithExtra(scanner)
}

func scanPostRowWithExtra(scanner pgx.Row, extraDestinations ...any) (*PostRow, error) {
	out := &PostRow{}
	var rawProject *json.RawMessage
	destinations := []any{
		&out.URI, &out.DID, &out.Rkey, &out.CID, &out.Text, &out.Facets, &out.Images, &out.RawEmbed,
		&out.ReplyRootURI, &out.ReplyRootCID, &out.ReplyParentURI, &out.ReplyParentCID,
		&out.QuoteURI, &out.QuoteCID, &out.Tags, &out.Langs, &out.CreatedAt, &out.IndexedAt,
		&out.ExternalImportSource, &out.ProfileSortAt,
		&out.IsProject, &out.ProjectCraftType, &rawProject,
		&out.AuthorDisplayName, &out.AuthorAvatarCID, &out.AuthorAvatarMime,
		&out.ModerationWarningKind,
	}
	destinations = append(destinations, extraDestinations...)
	err := scanner.Scan(destinations...)
	if err != nil {
		return out, err
	}
	if rawProject != nil && len(*rawProject) > 0 {
		out.RawProject = append(json.RawMessage(nil), (*rawProject)...)
		var project Project
		if err := json.Unmarshal(*rawProject, &project); err != nil {
			return out, err
		}
		out.Project = &project
	}
	return out, err
}

func decodeSeekCursor(cursor, timeKey string) (any, any, error) {
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	if cursor == "" {
		return nil, nil, nil
	}
	if len(cur) != 2 {
		return nil, nil, envelope.ErrInvalidCursor
	}
	timeValue, ok := cur[timeKey].(string)
	if !ok || timeValue == "" {
		return nil, nil, envelope.ErrInvalidCursor
	}
	parsedTime, err := time.Parse(time.RFC3339Nano, timeValue)
	if err != nil {
		return nil, nil, envelope.ErrInvalidCursor
	}
	uri, ok := cur["uri"].(string)
	if !ok || uri == "" {
		return nil, nil, envelope.ErrInvalidCursor
	}
	return parsedTime, uri, nil
}

func (s *PostStore) observeDB(ctx context.Context, operation, routePattern string, fn func(context.Context) error) error {
	if s == nil || s.observer == nil {
		return fn(ctx)
	}
	return s.observer.ObserveDB(ctx, observability.DBOperation{
		Operation:    operation,
		RoutePattern: routePattern,
	}, fn)
}
