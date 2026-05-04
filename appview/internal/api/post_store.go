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

// PostReader is the read-side interface handlers depend on. Tests inject
// fakes; production uses *PostStore.
type PostReader interface {
	ReadOne(ctx context.Context, did, rkey string) (*PostRow, error)
	ListByAuthor(ctx context.Context, did string, limit int, cursor string) (rows []*PostRow, nextCursor string, err error)
	ReadAuthor(ctx context.Context, did string) (*PostAuthorRow, error)
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
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var (
		curIndexedAt any
		curURI       any
	)
	if v, ok := cur["indexedAt"].(string); ok && v != "" {
		t, perr := time.Parse(time.RFC3339Nano, v)
		if perr != nil {
			return nil, "", envelope.ErrInvalidCursor
		}
		curIndexedAt = t
		uri, _ := cur["uri"].(string)
		curURI = uri
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
