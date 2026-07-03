// appview/internal/api/post_store.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	URI              string
	DID              string
	Rkey             string
	CID              string
	Text             string
	Facets           json.RawMessage
	Images           json.RawMessage
	ReplyRootURI     *string
	ReplyRootCID     *string
	ReplyParentURI   *string
	ReplyParentCID   *string
	QuoteURI         *string
	QuoteCID         *string
	Tags             []string
	CreatedAt        time.Time
	IndexedAt        time.Time
	IsProject        bool
	ProjectCraftType *string
	RawProject       json.RawMessage

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

// EngagementSummary is the batch-friendly read model used to augment posts.
type EngagementSummary struct {
	LikeCount         int
	RepostCount       int
	ReplyCount        int
	ViewerHasLiked    bool
	ViewerHasReposted bool
	ViewerHasReplied  bool
}

// PostReader is the read-side interface handlers depend on. Tests inject
// fakes; production uses *PostStore.
type PostReader interface {
	ReadOne(ctx context.Context, did, rkey string) (*PostRow, error)
	ReadPostByURI(ctx context.Context, uri string) (*PostRow, error)
	ListByAuthor(ctx context.Context, did string, limit int, cursor string) (rows []*PostRow, nextCursor string, err error)
	ListProjectsByAuthor(ctx context.Context, did string, limit int, cursor string) (rows []*PostRow, nextCursor string, err error)
	ListCommentsByAuthor(ctx context.Context, did string, limit int, cursor string) (rows []*PostRow, nextCursor string, err error)
	ResolvePostTarget(ctx context.Context, did, rkey string) (*PostTargetRef, error)
	ListRootComments(ctx context.Context, rootURI, viewerDID, sort string, limit int, cursor string) (rows []*PostRow, nextCursor string, err error)
	ListCommentBranchReplies(ctx context.Context, commentURI, rootURI string, limit int, cursor string) (rows []*PostRow, nextCursor string, err error)
	ListCommentBranchRepliesAround(ctx context.Context, commentURI, rootURI, focusURI string, limit int) (rows []*PostRow, nextCursor string, err error)
	ReadAuthor(ctx context.Context, did string) (*PostAuthorRow, error)
	EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error)
}

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

// ListRootComments returns direct replies to the root post in comment-section
// render order: viewer-authored comments first, then the selected sort within
// each group. The follows sort currently uses oldest-first ordering.
func (s *PostStore) ListRootComments(ctx context.Context, rootURI, viewerDID, sortValue string, limit int, cursor string) ([]*PostRow, string, error) {
	curCreatedAt, curURI, err := decodeSeekCursor(cursor, "createdAt")
	if err != nil {
		return nil, "", err
	}
	orderDirection := "ASC"
	seekComparator := ">"
	if sortValue == "newest" {
		orderDirection = "DESC"
		seekComparator = "<"
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.reply_parent_uri = $1
		` + postVisibleModerationPredicate + `
		  AND ($2::timestamptz IS NULL
		       OR (p.created_at, p.uri) ` + seekComparator + ` ($2::timestamptz, $3::text))
		ORDER BY CASE WHEN p.did = $5 THEN 0 ELSE 1 END ASC, p.created_at ` + orderDirection + `, p.uri ` + orderDirection + `
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, rootURI, curCreatedAt, curURI, limit, viewerDID)
	if err != nil {
		return nil, "", fmt.Errorf("comment list root %s: %w", rootURI, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("comment list scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("comment list iter: %w", err)
	}
	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"createdAt": last.CreatedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode comment cursor: %w", err)
	}
	return out, next, nil
}

func (s *PostStore) commentBranchHasRepliesAfter(ctx context.Context, commentURI, rootURI string, createdAt time.Time, uri string) (bool, error) {
	q := `
		WITH RECURSIVE branch(uri, created_at, depth) AS (
			SELECT p.uri, p.created_at, 1
			FROM craftsky_posts p
			WHERE p.reply_parent_uri = $1
			  AND p.reply_root_uri = $2
			UNION ALL
			SELECT child.uri, child.created_at, parent.depth + 1
			FROM craftsky_posts child
			JOIN branch parent ON child.reply_parent_uri = parent.uri
			WHERE child.reply_root_uri = $2
			  AND parent.depth < 64
		)
		SELECT EXISTS (
			SELECT 1
			FROM branch
			JOIN craftsky_posts p ON p.uri = branch.uri
			WHERE (branch.created_at, branch.uri) > ($3::timestamptz, $4::text)
			` + postVisibleModerationPredicate + `
		)
	`
	var hasMore bool
	if err := s.pool.QueryRow(ctx, q, commentURI, rootURI, createdAt, uri).Scan(&hasMore); err != nil {
		return false, fmt.Errorf("reply list branch has more comment=%s root=%s: %w", commentURI, rootURI, err)
	}
	return hasMore, nil
}

// PostStore is the Postgres-backed implementation.
type PostStore struct {
	pool     *pgxpool.Pool
	observer *observability.Observer
}

func NewPostStore(pool *pgxpool.Pool, observer ...*observability.Observer) *PostStore {
	store := &PostStore{pool: pool}
	if len(observer) > 0 {
		store.observer = observer[0]
	}
	return store
}

const postSelectColumns = `
	p.uri, p.did, p.rkey, p.cid, p.text, p.facets, p.images,
	p.reply_root_uri, p.reply_root_cid, p.reply_parent_uri, p.reply_parent_cid,
	p.quote_uri, p.quote_cid, p.tags, p.created_at, p.indexed_at,
	p.is_project, p.project_craft_type, pp.raw_project,
	bp.display_name, bp.avatar_cid, bp.avatar_mime,
	CASE
		WHEN EXISTS (
			SELECT 1
			FROM moderation_outputs mo
			WHERE mo.action = 'apply'
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
		  AND NOT EXISTS (
			SELECT 1
			FROM moderation_outputs mo
			WHERE mo.action = 'apply'
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

func scanPostRow(scanner pgx.Row) (*PostRow, error) {
	out := &PostRow{}
	var rawProject *json.RawMessage
	err := scanner.Scan(
		&out.URI, &out.DID, &out.Rkey, &out.CID, &out.Text, &out.Facets, &out.Images,
		&out.ReplyRootURI, &out.ReplyRootCID, &out.ReplyParentURI, &out.ReplyParentCID,
		&out.QuoteURI, &out.QuoteCID, &out.Tags, &out.CreatedAt, &out.IndexedAt,
		&out.IsProject, &out.ProjectCraftType, &rawProject,
		&out.AuthorDisplayName, &out.AuthorAvatarCID, &out.AuthorAvatarMime,
		&out.ModerationWarningKind,
	)
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

// ListByAuthor returns up to limit posts authored by did, ordered by
// (indexed_at DESC, uri DESC), starting after the cursor if non-empty.
// Returns the encoded next-page cursor when the result is full; empty
// string when this is the final page.
func (s *PostStore) ListByAuthor(ctx context.Context, did string, limit int, cursor string) ([]*PostRow, string, error) {
	curIndexedAt, curURI, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1
		  AND p.is_project = false
		  AND p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		` + postVisibleModerationPredicate + `
		  AND ($2::timestamptz IS NULL
		       OR (p.indexed_at, p.uri) < ($2::timestamptz, $3::text))
		ORDER BY p.indexed_at DESC, p.uri DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, did, curIndexedAt, curURI, limit)
	if err != nil {
		return nil, "", fmt.Errorf("post list %s: %w", did, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("post list scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("post list iter: %w", err)
	}

	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": last.IndexedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode cursor: %w", err)
	}
	return out, next, nil
}

// ListProjectsByAuthor returns root project posts authored by did, ordered by
// (indexed_at DESC, uri DESC), starting after the cursor if non-empty.
func (s *PostStore) ListProjectsByAuthor(ctx context.Context, did string, limit int, cursor string) ([]*PostRow, string, error) {
	curIndexedAt, curURI, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1
		  AND p.is_project = true
		  AND p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		  AND p.quote_uri IS NULL
		` + postVisibleModerationPredicate + `
		  AND ($2::timestamptz IS NULL
		       OR (p.indexed_at, p.uri) < ($2::timestamptz, $3::text))
		ORDER BY p.indexed_at DESC, p.uri DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, did, curIndexedAt, curURI, limit)
	if err != nil {
		return nil, "", fmt.Errorf("project list %s: %w", did, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("project list scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("project list iter: %w", err)
	}
	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": last.IndexedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode project cursor: %w", err)
	}
	return out, next, nil
}

// ListCommentsByAuthor returns authored comments and nested replies, ordered by
// (indexed_at DESC, uri DESC), starting after the cursor if non-empty.
func (s *PostStore) ListCommentsByAuthor(ctx context.Context, did string, limit int, cursor string) ([]*PostRow, string, error) {
	curIndexedAt, curURI, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1
		  AND p.reply_root_uri IS NOT NULL
		  AND p.reply_parent_uri IS NOT NULL
		` + postVisibleModerationPredicate + `
		  AND ($2::timestamptz IS NULL
		       OR (p.indexed_at, p.uri) < ($2::timestamptz, $3::text))
		ORDER BY p.indexed_at DESC, p.uri DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, did, curIndexedAt, curURI, limit)
	if err != nil {
		return nil, "", fmt.Errorf("comment list author %s: %w", did, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("comment list author scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("comment list author iter: %w", err)
	}

	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": last.IndexedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode comment author cursor: %w", err)
	}
	return out, next, nil
}

// ReadAuthor returns the bluesky_profiles display fields for did.
// Returns (&PostAuthorRow{nil, nil}, nil) — not an error — when the
// user has no bluesky_profiles row yet. The post-create path uses this
// to hydrate authors whose row hasn't been indexed yet.
func (s *PostStore) ReadAuthor(ctx context.Context, did string) (*PostAuthorRow, error) {
	const q = `
		SELECT display_name, avatar_cid, avatar_mime
		FROM bluesky_profiles
		WHERE did = $1
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

// ResolvePostReportTarget returns the canonical indexed post snapshot used for
// report eligibility and private report persistence. It intentionally checks
// indexed existence only; moderation visibility filters must not make stale or
// race report submissions ineligible.
func (s *PostStore) ResolvePostReportTarget(ctx context.Context, did syntax.DID, rkey syntax.RecordKey) (*PostReportTarget, error) {
	ref, err := s.ResolvePostTarget(ctx, did.String(), rkey.String())
	if err != nil {
		return nil, err
	}
	return &PostReportTarget{
		DID:         did.String(),
		Rkey:        rkey.String(),
		URI:         ref.URI,
		CIDSnapshot: ref.CID,
	}, nil
}

func scanInteractionRow(scanner pgx.Row) (*InteractionRow, error) {
	out := &InteractionRow{}
	err := scanner.Scan(
		&out.URI, &out.DID, &out.Rkey, &out.CID,
		&out.SubjectURI, &out.SubjectCID, &out.CreatedAt, &out.IndexedAt,
	)
	return out, err
}

func (s *PostStore) findActiveInteraction(ctx context.Context, table, label, did, subjectURI string) (*InteractionRow, error) {
	q := `
		SELECT uri, did, rkey, cid, subject_uri, subject_cid, created_at, indexed_at
		FROM ` + table + `
		WHERE did = $1 AND subject_uri = $2 AND deleted_at IS NULL
	`
	row, err := scanInteractionRow(s.pool.QueryRow(ctx, q, did, subjectURI))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInteractionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s find active %s/%s: %w", label, did, subjectURI, err)
	}
	return row, nil
}

// FindActiveLike returns the active like by did for subjectURI.
func (s *PostStore) FindActiveLike(ctx context.Context, did, subjectURI string) (*InteractionRow, error) {
	return s.findActiveInteraction(ctx, "craftsky_likes", "like", did, subjectURI)
}

// FindActiveRepost returns the active repost by did for subjectURI.
func (s *PostStore) FindActiveRepost(ctx context.Context, did, subjectURI string) (*InteractionRow, error) {
	return s.findActiveInteraction(ctx, "craftsky_reposts", "repost", did, subjectURI)
}

func (s *PostStore) countActiveInteractions(ctx context.Context, table, label string, postURIs []string) (map[string]int, error) {
	out := make(map[string]int, len(postURIs))
	if len(postURIs) == 0 {
		return out, nil
	}
	for _, uri := range postURIs {
		out[uri] = 0
	}
	q := `
		SELECT subject_uri, count(*)::int
		FROM ` + table + `
		WHERE deleted_at IS NULL AND subject_uri = ANY($1::text[])
		GROUP BY subject_uri
	`
	rows, err := s.pool.Query(ctx, q, postURIs)
	if err != nil {
		return nil, fmt.Errorf("%s count active: %w", label, err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err != nil {
			return nil, fmt.Errorf("%s count scan: %w", label, err)
		}
		out[uri] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s count iter: %w", label, err)
	}
	return out, nil
}

// CountActiveLikes returns active like counts keyed by post URI.
func (s *PostStore) CountActiveLikes(ctx context.Context, postURIs []string) (map[string]int, error) {
	return s.countActiveInteractions(ctx, "craftsky_likes", "like", postURIs)
}

// CountActiveReposts returns active repost counts keyed by post URI.
func (s *PostStore) CountActiveReposts(ctx context.Context, postURIs []string) (map[string]int, error) {
	return s.countActiveInteractions(ctx, "craftsky_reposts", "repost", postURIs)
}

// CountDescendantReplies returns all descendant reply counts keyed by ancestor
// post URI. Traversal is depth-capped to match branch rendering.
func (s *PostStore) CountDescendantReplies(ctx context.Context, postURIs []string) (map[string]int, error) {
	out := make(map[string]int, len(postURIs))
	if len(postURIs) == 0 {
		return out, nil
	}
	for _, uri := range postURIs {
		out[uri] = 0
	}
	const q = `
		WITH RECURSIVE descendants(subject_uri, uri, depth) AS (
			SELECT subjects.subject_uri, p.uri, 1
			FROM unnest($1::text[]) AS subjects(subject_uri)
			JOIN craftsky_posts p ON p.reply_parent_uri = subjects.subject_uri
			UNION ALL
			SELECT descendants.subject_uri, child.uri, descendants.depth + 1
			FROM descendants
			JOIN craftsky_posts child ON child.reply_parent_uri = descendants.uri
			WHERE descendants.depth < 64
		)
		SELECT subject_uri, count(*)::int
		FROM descendants
		GROUP BY subject_uri
	`
	rows, err := s.pool.Query(ctx, q, postURIs)
	if err != nil {
		return nil, fmt.Errorf("reply count descendants: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err != nil {
			return nil, fmt.Errorf("reply count descendant scan: %w", err)
		}
		out[uri] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reply count descendant iter: %w", err)
	}
	return out, nil
}

// ViewerInteractionStates returns current-viewer active like/repost booleans keyed by post URI.
func (s *PostStore) ViewerInteractionStates(ctx context.Context, viewerDID string, postURIs []string) (map[string]ViewerInteractionState, error) {
	out := make(map[string]ViewerInteractionState, len(postURIs))
	if len(postURIs) == 0 || viewerDID == "" {
		for _, uri := range postURIs {
			out[uri] = ViewerInteractionState{}
		}
		return out, nil
	}
	for _, uri := range postURIs {
		out[uri] = ViewerInteractionState{}
	}
	const q = `
		SELECT subject_uri, bool_or(kind = 'like'), bool_or(kind = 'repost')
		FROM (
			SELECT subject_uri, 'like' AS kind
			FROM craftsky_likes
			WHERE did = $1 AND deleted_at IS NULL AND subject_uri = ANY($2::text[])
			UNION ALL
			SELECT subject_uri, 'repost' AS kind
			FROM craftsky_reposts
			WHERE did = $1 AND deleted_at IS NULL AND subject_uri = ANY($2::text[])
		) interactions
		GROUP BY subject_uri
	`
	rows, err := s.pool.Query(ctx, q, viewerDID, postURIs)
	if err != nil {
		return nil, fmt.Errorf("viewer interaction states %s: %w", viewerDID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var liked, reposted bool
		if err := rows.Scan(&uri, &liked, &reposted); err != nil {
			return nil, fmt.Errorf("viewer interaction states scan: %w", err)
		}
		out[uri] = ViewerInteractionState{HasLiked: liked, HasReposted: reposted}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("viewer interaction states iter: %w", err)
	}
	return out, nil
}

// ViewerReplyStates returns whether the current viewer authored a direct child reply for each post URI.
func (s *PostStore) ViewerReplyStates(ctx context.Context, viewerDID string, postURIs []string) (map[string]ViewerReplyState, error) {
	out := make(map[string]ViewerReplyState, len(postURIs))
	for _, uri := range postURIs {
		out[uri] = ViewerReplyState{}
	}
	if len(postURIs) == 0 || viewerDID == "" {
		return out, nil
	}
	const q = `
		WITH RECURSIVE subjects(uri, reply_parent_uri) AS (
			SELECT uri, reply_parent_uri
			FROM craftsky_posts
			WHERE uri = ANY($2::text[])
		), descendants(subject_uri, uri, depth) AS (
			SELECT subjects.uri, child.uri, 1
			FROM subjects
			JOIN craftsky_posts child ON child.reply_parent_uri = subjects.uri
			UNION ALL
			SELECT descendants.subject_uri, child.uri, descendants.depth + 1
			FROM descendants
			JOIN craftsky_posts child ON child.reply_parent_uri = descendants.uri
			WHERE descendants.depth < 64
		)
		SELECT descendants.subject_uri, true
		FROM descendants
		JOIN subjects ON subjects.uri = descendants.subject_uri
		JOIN craftsky_posts viewer_reply ON viewer_reply.uri = descendants.uri
		WHERE viewer_reply.did = $1
		  AND (subjects.reply_parent_uri IS NOT NULL OR descendants.depth = 1)
		GROUP BY descendants.subject_uri
	`
	rows, err := s.pool.Query(ctx, q, viewerDID, postURIs)
	if err != nil {
		return nil, fmt.Errorf("viewer reply states %s: %w", viewerDID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var replied bool
		if err := rows.Scan(&uri, &replied); err != nil {
			return nil, fmt.Errorf("viewer reply states scan: %w", err)
		}
		out[uri] = ViewerReplyState{HasReplied: replied}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("viewer reply states iter: %w", err)
	}
	return out, nil
}

// EngagementSummaries returns counts and current-viewer state keyed by post URI.
func (s *PostStore) EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error) {
	var summaries map[string]EngagementSummary
	err := s.observeDB(ctx, "post.engagement_summaries", "unmatched", func(ctx context.Context) error {
		var err error
		summaries, err = s.engagementSummariesObserved(ctx, viewerDID, postURIs)
		return err
	})
	return summaries, err
}

func (s *PostStore) engagementSummariesObserved(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error) {
	out := make(map[string]EngagementSummary, len(postURIs))
	for _, uri := range postURIs {
		out[uri] = EngagementSummary{}
	}
	if len(postURIs) == 0 {
		return out, nil
	}
	likeCounts, err := s.CountActiveLikes(ctx, postURIs)
	if err != nil {
		return nil, err
	}
	repostCounts, err := s.CountActiveReposts(ctx, postURIs)
	if err != nil {
		return nil, err
	}
	replyCounts, err := s.CountDescendantReplies(ctx, postURIs)
	if err != nil {
		return nil, err
	}
	viewerStates, err := s.ViewerInteractionStates(ctx, viewerDID, postURIs)
	if err != nil {
		return nil, err
	}
	viewerReplyStates, err := s.ViewerReplyStates(ctx, viewerDID, postURIs)
	if err != nil {
		return nil, err
	}
	for _, uri := range postURIs {
		state := viewerStates[uri]
		replyState := viewerReplyStates[uri]
		out[uri] = EngagementSummary{
			LikeCount:         likeCounts[uri],
			RepostCount:       repostCounts[uri],
			ReplyCount:        replyCounts[uri],
			ViewerHasLiked:    state.HasLiked,
			ViewerHasReposted: state.HasReposted,
			ViewerHasReplied:  replyState.HasReplied,
		}
	}
	return out, nil
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

// ListCommentBranchReplies returns visual replies under a top-level comment,
// including deeper descendants flattened into chronological branch order.
func (s *PostStore) ListCommentBranchReplies(ctx context.Context, commentURI, rootURI string, limit int, cursor string) ([]*PostRow, string, error) {
	curCreatedAt, curURI, err := decodeSeekCursor(cursor, "createdAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		WITH RECURSIVE branch(uri, created_at, depth) AS (
			SELECT p.uri, p.created_at, 1
			FROM craftsky_posts p
			WHERE p.reply_parent_uri = $1
			  AND p.reply_root_uri = $2
			UNION ALL
			SELECT child.uri, child.created_at, parent.depth + 1
			FROM craftsky_posts child
			JOIN branch parent ON child.reply_parent_uri = parent.uri
			WHERE child.reply_root_uri = $2
			  AND parent.depth < 64
		), page AS (
			SELECT branch.uri, branch.created_at
			FROM branch
			JOIN craftsky_posts p ON p.uri = branch.uri
			WHERE ($3::timestamptz IS NULL
			       OR (branch.created_at, branch.uri) > ($3::timestamptz, $4::text))
			` + postVisibleModerationPredicate + `
			ORDER BY branch.created_at ASC, branch.uri ASC
			LIMIT $5
		)
		SELECT ` + postSelectColumns + `
		FROM page
		JOIN craftsky_posts p ON p.uri = page.uri
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE true
		` + postVisibleModerationPredicate + `
		ORDER BY page.created_at ASC, page.uri ASC
	`
	rows, err := s.pool.Query(ctx, q, commentURI, rootURI, curCreatedAt, curURI, limit)
	if err != nil {
		return nil, "", fmt.Errorf("reply list branch comment=%s root=%s: %w", commentURI, rootURI, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("reply list branch scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("reply list branch iter: %w", err)
	}
	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"createdAt": last.CreatedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode reply cursor: %w", err)
	}
	return out, next, nil
}

// ListCommentBranchRepliesAround returns a bounded visual reply page that
// includes focusURI, ending at the focused reply so deep links can render the
// target without loading every earlier branch reply.
func (s *PostStore) ListCommentBranchRepliesAround(ctx context.Context, commentURI, rootURI, focusURI string, limit int) ([]*PostRow, string, error) {
	q := `
		WITH RECURSIVE branch(uri, created_at, depth) AS (
			SELECT p.uri, p.created_at, 1
			FROM craftsky_posts p
			WHERE p.reply_parent_uri = $1
			  AND p.reply_root_uri = $2
			UNION ALL
			SELECT child.uri, child.created_at, parent.depth + 1
			FROM craftsky_posts child
			JOIN branch parent ON child.reply_parent_uri = parent.uri
			WHERE child.reply_root_uri = $2
			  AND parent.depth < 64
		), focus AS (
			SELECT uri, created_at
			FROM branch
			WHERE uri = $3
		), page AS (
			SELECT branch.uri, branch.created_at
			FROM branch
			JOIN focus ON true
			JOIN craftsky_posts p ON p.uri = branch.uri
			WHERE (branch.created_at, branch.uri) <= (focus.created_at, focus.uri)
			` + postVisibleModerationPredicate + `
			ORDER BY branch.created_at DESC, branch.uri DESC
			LIMIT $4
		), ordered_page AS (
			SELECT uri, created_at
			FROM page
			ORDER BY created_at ASC, uri ASC
		)
		SELECT ` + postSelectColumns + `
		FROM ordered_page
		JOIN craftsky_posts p ON p.uri = ordered_page.uri
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE true
		` + postVisibleModerationPredicate + `
		ORDER BY ordered_page.created_at ASC, ordered_page.uri ASC
	`
	rows, err := s.pool.Query(ctx, q, commentURI, rootURI, focusURI, limit)
	if err != nil {
		return nil, "", fmt.Errorf("reply list branch around comment=%s root=%s focus=%s: %w", commentURI, rootURI, focusURI, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("reply list branch around scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("reply list branch around iter: %w", err)
	}
	if len(out) == 0 {
		return out, "", nil
	}
	last := out[len(out)-1]
	hasMore, err := s.commentBranchHasRepliesAfter(ctx, commentURI, rootURI, last.CreatedAt, last.URI)
	if err != nil {
		return nil, "", err
	}
	if !hasMore {
		return out, "", nil
	}
	next, err := envelope.EncodeCursor(map[string]any{
		"createdAt": last.CreatedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode reply cursor: %w", err)
	}
	return out, next, nil
}
