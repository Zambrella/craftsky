// appview/internal/api/post_store.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api/envelope"
)

// ErrPostNotFound is returned by PostStore.ReadOne when no row matches.
var ErrPostNotFound = errors.New("post: not found")

// ErrInteractionNotFound is returned when no active like/repost matches.
var ErrInteractionNotFound = errors.New("interaction: not found")

// PostRow is the joined view of craftsky_posts plus author display fields
// from bluesky_profiles. Reply/quote pointers are kept as separate
// pointers so handlers can decide nesting at the JSON layer.
type PostRow struct {
	URI            string
	DID            string
	Rkey           string
	CID            string
	Text           string
	Facets         json.RawMessage
	ReplyRootURI   *string
	ReplyRootCID   *string
	ReplyParentURI *string
	ReplyParentCID *string
	QuoteURI       *string
	QuoteCID       *string
	Tags           []string
	CreatedAt      time.Time
	IndexedAt      time.Time

	AuthorDisplayName *string
	AuthorAvatarCID   *string
}

// PostAuthorRow is the slim author-hydration view used when we need to
// build a synthetic response for a freshly-created post (the post row
// itself doesn't exist yet at that moment, but the author's bsky
// profile may).
type PostAuthorRow struct {
	DisplayName *string
	AvatarCID   *string
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

// EngagementSummary is the batch-friendly read model used to augment posts.
type EngagementSummary struct {
	LikeCount         int
	RepostCount       int
	ReplyCount        int
	ViewerHasLiked    bool
	ViewerHasReposted bool
}

// PostReader is the read-side interface handlers depend on. Tests inject
// fakes; production uses *PostStore.
type PostReader interface {
	ReadOne(ctx context.Context, did, rkey string) (*PostRow, error)
	ListByAuthor(ctx context.Context, did string, limit int, cursor string) (rows []*PostRow, nextCursor string, err error)
	ResolvePostTarget(ctx context.Context, did, rkey string) (*PostTargetRef, error)
	ListDirectReplies(ctx context.Context, parentURI string, limit int, cursor string) (rows []*PostRow, nextCursor string, err error)
	LoadThreadCandidates(ctx context.Context, rootURI, targetURI string, limit int) ([]*PostRow, error)
	ReadAuthor(ctx context.Context, did string) (*PostAuthorRow, error)
	EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error)
}

// PostStore is the Postgres-backed implementation.
type PostStore struct {
	pool *pgxpool.Pool
}

func NewPostStore(pool *pgxpool.Pool) *PostStore {
	return &PostStore{pool: pool}
}

const postSelectColumns = `
	p.uri, p.did, p.rkey, p.cid, p.text, p.facets,
	p.reply_root_uri, p.reply_root_cid, p.reply_parent_uri, p.reply_parent_cid,
	p.quote_uri, p.quote_cid, p.tags, p.created_at, p.indexed_at,
	bp.display_name, bp.avatar_cid
`

func scanPostRow(scanner pgx.Row) (*PostRow, error) {
	out := &PostRow{}
	err := scanner.Scan(
		&out.URI, &out.DID, &out.Rkey, &out.CID, &out.Text, &out.Facets,
		&out.ReplyRootURI, &out.ReplyRootCID, &out.ReplyParentURI, &out.ReplyParentCID,
		&out.QuoteURI, &out.QuoteCID, &out.Tags, &out.CreatedAt, &out.IndexedAt,
		&out.AuthorDisplayName, &out.AuthorAvatarCID,
	)
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
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1 AND p.rkey = $2
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
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1
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

// ReadAuthor returns the bluesky_profiles display fields for did.
// Returns (&PostAuthorRow{nil, nil}, nil) — not an error — when the
// user has no bluesky_profiles row yet. The post-create path uses this
// to hydrate authors whose row hasn't been indexed yet.
func (s *PostStore) ReadAuthor(ctx context.Context, did string) (*PostAuthorRow, error) {
	const q = `
		SELECT display_name, avatar_cid
		FROM bluesky_profiles
		WHERE did = $1
	`
	out := &PostAuthorRow{}
	err := s.pool.QueryRow(ctx, q, did).Scan(&out.DisplayName, &out.AvatarCID)
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

// CountDirectReplies returns direct child reply counts keyed by parent post URI.
func (s *PostStore) CountDirectReplies(ctx context.Context, postURIs []string) (map[string]int, error) {
	out := make(map[string]int, len(postURIs))
	if len(postURIs) == 0 {
		return out, nil
	}
	for _, uri := range postURIs {
		out[uri] = 0
	}
	const q = `
		SELECT reply_parent_uri, count(*)::int
		FROM craftsky_posts
		WHERE reply_parent_uri = ANY($1::text[])
		GROUP BY reply_parent_uri
	`
	rows, err := s.pool.Query(ctx, q, postURIs)
	if err != nil {
		return nil, fmt.Errorf("reply count direct: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err != nil {
			return nil, fmt.Errorf("reply count scan: %w", err)
		}
		out[uri] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reply count iter: %w", err)
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

// EngagementSummaries returns counts and current-viewer state keyed by post URI.
func (s *PostStore) EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error) {
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
	replyCounts, err := s.CountDirectReplies(ctx, postURIs)
	if err != nil {
		return nil, err
	}
	viewerStates, err := s.ViewerInteractionStates(ctx, viewerDID, postURIs)
	if err != nil {
		return nil, err
	}
	for _, uri := range postURIs {
		state := viewerStates[uri]
		out[uri] = EngagementSummary{
			LikeCount:         likeCounts[uri],
			RepostCount:       repostCounts[uri],
			ReplyCount:        replyCounts[uri],
			ViewerHasLiked:    state.HasLiked,
			ViewerHasReposted: state.HasReposted,
		}
	}
	return out, nil
}

// ListDirectReplies returns direct child replies oldest-first by (created_at, uri).
func (s *PostStore) ListDirectReplies(ctx context.Context, parentURI string, limit int, cursor string) ([]*PostRow, string, error) {
	curCreatedAt, curURI, err := decodeSeekCursor(cursor, "createdAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.reply_parent_uri = $1
		  AND ($2::timestamptz IS NULL
		       OR (p.created_at, p.uri) > ($2::timestamptz, $3::text))
		ORDER BY p.created_at ASC, p.uri ASC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, parentURI, curCreatedAt, curURI, limit)
	if err != nil {
		return nil, "", fmt.Errorf("reply list direct %s: %w", parentURI, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("reply list direct scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("reply list direct iter: %w", err)
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

// LoadThreadCandidates returns the target row plus descendants below that target.
func (s *PostStore) LoadThreadCandidates(ctx context.Context, rootURI, targetURI string, limit int) ([]*PostRow, error) {
	q := `
		WITH RECURSIVE target_uri AS (
			SELECT uri, created_at
			FROM craftsky_posts
			WHERE uri = $2
		), tree(uri, created_at, depth) AS (
			SELECT uri, created_at, 1
			FROM craftsky_posts
			WHERE reply_root_uri = $1 AND reply_parent_uri = $2
			UNION ALL
			SELECT child.uri, child.created_at, tree.depth + 1
			FROM craftsky_posts child
			JOIN tree ON child.reply_parent_uri = tree.uri
			WHERE child.reply_root_uri = $1 AND tree.depth < 7
		), descendant_uris AS (
			SELECT uri, created_at
			FROM tree
			ORDER BY created_at ASC, uri ASC
			LIMIT GREATEST($3 - 1, 0)
		), candidate_uris AS (
			SELECT uri, created_at FROM target_uri
			UNION ALL
			SELECT uri, created_at FROM descendant_uris
		)
		SELECT ` + postSelectColumns + `
		FROM candidate_uris c
		JOIN craftsky_posts p ON p.uri = c.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		ORDER BY c.created_at ASC, c.uri ASC
	`
	rows, err := s.pool.Query(ctx, q, rootURI, targetURI, limit)
	if err != nil {
		return nil, fmt.Errorf("thread candidates root=%s target=%s: %w", rootURI, targetURI, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("thread candidates scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("thread candidates iter: %w", err)
	}
	return out, nil
}
