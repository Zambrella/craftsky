// appview/internal/api/notification_store.go
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"social.craftsky/appview/internal/api/envelope"
)

type NotificationType string

const (
	NotificationTypeFollow NotificationType = "follow"
	NotificationTypeLike   NotificationType = "like"
	NotificationTypeRepost NotificationType = "repost"
	NotificationTypeReply  NotificationType = "reply"
)

type NotificationReplyRef struct {
	URI  string
	CID  string
	Rkey string
}

type NotificationRow struct {
	Type NotificationType
	URI  string
	CID  string
	Rkey string

	ActorDID         string
	ActorDisplayName *string
	ActorAvatarCID   *string

	CreatedAt time.Time
	IndexedAt time.Time

	SubjectPost *PostRow
	Reply       *NotificationReplyRef
}

func (s *PostStore) ListNotifications(ctx context.Context, viewerDID string, limit int, cursor string) ([]*NotificationRow, string, error) {
	curIndexedAt, curURI, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		WITH events AS (
			SELECT
				'follow'::text AS event_type,
				f.uri, f.cid, f.rkey,
				f.did AS actor_did,
				f.created_at, f.indexed_at,
				NULL::text AS subject_uri
			FROM atproto_follows f
			WHERE f.subject_did = $1
			  AND f.did <> $1
			UNION ALL
			SELECT
				'like'::text AS event_type,
				l.uri, l.cid, l.rkey,
				l.did AS actor_did,
				l.created_at, l.indexed_at,
				p.uri AS subject_uri
			FROM craftsky_likes l
			JOIN craftsky_posts p ON p.uri = l.subject_uri
			WHERE p.did = $1
			  AND l.did <> $1
			  AND l.deleted_at IS NULL
			UNION ALL
			SELECT
				'repost'::text AS event_type,
				r.uri, r.cid, r.rkey,
				r.did AS actor_did,
				r.created_at, r.indexed_at,
				p.uri AS subject_uri
			FROM craftsky_reposts r
			JOIN craftsky_posts p ON p.uri = r.subject_uri
			WHERE p.did = $1
			  AND r.did <> $1
			  AND r.deleted_at IS NULL
			UNION ALL
			SELECT
				'reply'::text AS event_type,
				reply.uri, reply.cid, reply.rkey,
				reply.did AS actor_did,
				reply.created_at, reply.indexed_at,
				parent.uri AS subject_uri
			FROM craftsky_posts reply
			JOIN craftsky_posts parent ON parent.uri = reply.reply_parent_uri
			WHERE parent.did = $1
			  AND reply.did <> $1
		)
		SELECT
			e.event_type,
			e.uri, e.cid, e.rkey,
			e.actor_did,
			actor_bp.display_name AS actor_display_name,
			actor_bp.avatar_cid AS actor_avatar_cid,
			e.created_at, e.indexed_at,
			sp.uri, sp.did, sp.rkey, sp.cid, sp.text, sp.facets, sp.images,
			sp.reply_root_uri, sp.reply_root_cid, sp.reply_parent_uri, sp.reply_parent_cid,
			sp.quote_uri, sp.quote_cid, sp.tags, sp.created_at, sp.indexed_at,
			sbp.display_name, sbp.avatar_cid
		FROM events e
		LEFT JOIN bluesky_profiles actor_bp ON actor_bp.did = e.actor_did
		LEFT JOIN craftsky_posts sp ON sp.uri = e.subject_uri
		LEFT JOIN bluesky_profiles sbp ON sbp.did = sp.did
		WHERE ($2::timestamptz IS NULL
		       OR (e.indexed_at, e.uri) < ($2::timestamptz, $3::text))
		ORDER BY e.indexed_at DESC, e.uri DESC
		LIMIT $4
	`
	queryLimit := limit + 1
	rows, err := s.pool.Query(ctx, q, viewerDID, curIndexedAt, curURI, queryLimit)
	if err != nil {
		return nil, "", fmt.Errorf("notification list %s: %w", viewerDID, err)
	}
	defer rows.Close()

	out := make([]*NotificationRow, 0, queryLimit)
	for rows.Next() {
		row := &NotificationRow{}
		var eventType string
		var subject notificationSubjectScan
		if err := rows.Scan(
			&eventType,
			&row.URI, &row.CID, &row.Rkey,
			&row.ActorDID, &row.ActorDisplayName, &row.ActorAvatarCID,
			&row.CreatedAt, &row.IndexedAt,
			&subject.URI, &subject.DID, &subject.Rkey, &subject.CID, &subject.Text, &subject.Facets, &subject.Images,
			&subject.ReplyRootURI, &subject.ReplyRootCID, &subject.ReplyParentURI, &subject.ReplyParentCID,
			&subject.QuoteURI, &subject.QuoteCID, &subject.Tags, &subject.CreatedAt, &subject.IndexedAt,
			&subject.AuthorDisplayName, &subject.AuthorAvatarCID,
		); err != nil {
			return nil, "", fmt.Errorf("notification list scan: %w", err)
		}
		row.Type = NotificationType(eventType)
		if subject.URI.Valid {
			row.SubjectPost = subject.postRow()
		}
		if row.Type == NotificationTypeReply {
			row.Reply = &NotificationReplyRef{URI: row.URI, CID: row.CID, Rkey: row.Rkey}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("notification list iter: %w", err)
	}
	if len(out) <= limit {
		return out, "", nil
	}
	out = out[:limit]
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": last.IndexedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode notification cursor: %w", err)
	}
	return out, next, nil
}

type notificationSubjectScan struct {
	URI            sql.NullString
	DID            sql.NullString
	Rkey           sql.NullString
	CID            sql.NullString
	Text           sql.NullString
	Facets         json.RawMessage
	Images         json.RawMessage
	ReplyRootURI   *string
	ReplyRootCID   *string
	ReplyParentURI *string
	ReplyParentCID *string
	QuoteURI       *string
	QuoteCID       *string
	Tags           []string
	CreatedAt      sql.NullTime
	IndexedAt      sql.NullTime

	AuthorDisplayName *string
	AuthorAvatarCID   *string
}

func (s notificationSubjectScan) postRow() *PostRow {
	return &PostRow{
		URI:               s.URI.String,
		DID:               s.DID.String,
		Rkey:              s.Rkey.String,
		CID:               s.CID.String,
		Text:              s.Text.String,
		Facets:            s.Facets,
		Images:            s.Images,
		ReplyRootURI:      s.ReplyRootURI,
		ReplyRootCID:      s.ReplyRootCID,
		ReplyParentURI:    s.ReplyParentURI,
		ReplyParentCID:    s.ReplyParentCID,
		QuoteURI:          s.QuoteURI,
		QuoteCID:          s.QuoteCID,
		Tags:              s.Tags,
		CreatedAt:         s.CreatedAt.Time,
		IndexedAt:         s.IndexedAt.Time,
		AuthorDisplayName: s.AuthorDisplayName,
		AuthorAvatarCID:   s.AuthorAvatarCID,
	}
}
