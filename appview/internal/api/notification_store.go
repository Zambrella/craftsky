// appview/internal/api/notification_store.go
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
)

type InstagramNotificationEligibility interface {
	RevalidateNotification(context.Context, uuid.UUID, instagram.EligibilityStage) (bool, error)
}

type NotificationType string

type NotificationKind string

type NotificationDestination string

const (
	NotificationKindSocial NotificationKind = "social"
	NotificationKindSystem NotificationKind = "system"

	NotificationTypeFollow         NotificationType = "follow"
	NotificationTypeLike           NotificationType = "like"
	NotificationTypeRepost         NotificationType = "repost"
	NotificationTypeReply          NotificationType = "reply"
	NotificationTypeMention        NotificationType = "mention"
	NotificationTypeQuote          NotificationType = "quote"
	NotificationTypeEverythingElse NotificationType = "everythingElse"
	NotificationTypeInstagramMatch NotificationType = "instagramMatch"

	NotificationDestinationInstagramMigration NotificationDestination = "instagramMigration"
)

type NotificationSystem struct {
	Count       int                     `json:"count"`
	CountCapped bool                    `json:"countCapped"`
	Destination NotificationDestination `json:"destination"`
}

type NotificationReplyRef struct {
	Available bool   `json:"available"`
	URI       string `json:"uri,omitempty"`
	CID       string `json:"cid,omitempty"`
	Rkey      string `json:"rkey,omitempty"`
}

type NotificationReference struct {
	Available bool   `json:"available"`
	URI       string `json:"uri,omitempty"`
	CID       string `json:"cid,omitempty"`
	Rkey      string `json:"rkey,omitempty"`
}

type NotificationReferences struct {
	Source  NotificationReference  `json:"source"`
	Subject *NotificationReference `json:"subject,omitempty"`
	Parent  *NotificationReference `json:"parent,omitempty"`
	Root    *NotificationReference `json:"root,omitempty"`
	Quoted  *NotificationReference `json:"quoted,omitempty"`
}

type NotificationRow struct {
	ID     string
	Kind   NotificationKind
	Type   NotificationType
	System *NotificationSystem
	URI    string
	CID    string
	Rkey   string

	ActorDID               string
	ActorDisplayName       *string
	ActorAvatarCID         *string
	ActorAvatarMime        *string
	ActorViewerIsFollowing bool

	CreatedAt time.Time
	IndexedAt time.Time

	SubjectPost *PostRow
	Reply       *NotificationReplyRef
	References  NotificationReferences
}

func (s *PostStore) NotificationHandles(ctx context.Context, dids []string) (map[string]syntax.Handle, error) {
	out := make(map[string]syntax.Handle)
	if len(dids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT did,handle FROM atproto_identity_cache WHERE did=ANY($1)`, dids)
	if err != nil {
		return nil, fmt.Errorf("notification handle batch: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var did string
		var handle syntax.Handle
		if err := rows.Scan(&did, &handle); err != nil {
			return nil, fmt.Errorf("notification handle scan: %w", err)
		}
		out[did] = handle
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("notification handle rows: %w", err)
	}
	return out, nil
}

func (s *PostStore) ListNotifications(ctx context.Context, viewerDID string, limit int, cursor string) ([]*NotificationRow, string, error) {
	curIndexedAt, curID, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		WITH page_events AS MATERIALIZED (
			SELECT e.*
			FROM notification_events e
			WHERE e.recipient_did = $1
			  AND e.state = 'active'
			  AND ($2::timestamptz IS NULL
			       OR (e.indexed_at, e.id) < ($2::timestamptz, $3::uuid))
			  AND NOT EXISTS (
				SELECT 1
				FROM moderation_outputs mo
				WHERE mo.action = 'apply'
				  AND mo.subject_type = 'account'
				  AND mo.subject_did = e.actor_did
				  AND mo.value IN ('hide', 'takedown')
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
			  )
			ORDER BY e.indexed_at DESC, e.id DESC
			LIMIT $4
		),
		reference_uris AS (
			SELECT source_uri AS uri FROM page_events WHERE category IN ('reply','mention','quote')
			UNION SELECT subject_uri FROM page_events WHERE subject_uri IS NOT NULL
			UNION SELECT parent_uri FROM page_events WHERE parent_uri IS NOT NULL
			UNION SELECT root_uri FROM page_events WHERE root_uri IS NOT NULL
			UNION SELECT quoted_uri FROM page_events WHERE quoted_uri IS NOT NULL
			UNION
			SELECT p.quote_uri
			FROM craftsky_posts p
			JOIN page_events e ON e.subject_uri=p.uri
			WHERE p.quote_uri IS NOT NULL
		),
		visible_posts AS (
			SELECT p.*
			FROM craftsky_posts p
			JOIN reference_uris refs ON refs.uri=p.uri
			WHERE NOT EXISTS (
				SELECT 1 FROM moderation_outputs mo
				WHERE mo.action='apply' AND mo.value IN ('hide','takedown')
				  AND (mo.expires_at IS NULL OR mo.expires_at>now())
				  AND ((mo.subject_type='post' AND mo.subject_uri=p.uri)
				       OR (mo.subject_type='account' AND mo.subject_did=p.did))
				  AND NOT EXISTS (
					SELECT 1 FROM moderation_outputs neg
					WHERE neg.action='negate' AND neg.source_did=mo.source_did
					  AND neg.subject_type=mo.subject_type AND neg.subject_did=mo.subject_did
					  AND neg.value=mo.value AND (neg.expires_at IS NULL OR neg.expires_at>now())
					  AND neg.indexed_at>mo.indexed_at
					  AND (mo.subject_type='account' OR neg.subject_uri=mo.subject_uri)
				  )
			)
		)
		SELECT
			e.id::text, COALESCE(to_jsonb(e)->>'kind', 'social'), e.category,
			e.source_uri, e.source_cid, e.source_rkey,
			CASE WHEN e.category IN ('reply','mention','quote') THEN source_post.uri IS NOT NULL ELSE true END,
			e.subject_uri,e.subject_cid,(e.subject_uri IS NOT NULL AND sp.uri IS NOT NULL),
			e.parent_uri,e.parent_cid,(e.parent_uri IS NOT NULL AND parent_post.uri IS NOT NULL),
			e.root_uri,e.root_cid,(e.root_uri IS NOT NULL AND root_post.uri IS NOT NULL),
			e.quoted_uri,e.quoted_cid,(e.quoted_uri IS NOT NULL AND quoted_post.uri IS NOT NULL),
			CASE WHEN sp.quote_uri IS NULL THEN true ELSE subject_quote.uri IS NOT NULL END,
			e.actor_did,
			actor_bp.display_name AS actor_display_name,
			actor_bp.avatar_cid AS actor_avatar_cid,
			actor_bp.avatar_mime AS actor_avatar_mime,
			EXISTS (
				SELECT 1
				FROM atproto_follows actor_follow
				WHERE actor_follow.did = $1
				  AND actor_follow.subject_did = e.actor_did
			) AS actor_viewer_is_following,
			CASE
				WHEN COALESCE(to_jsonb(e)->>'kind', 'social') = 'system'
				THEN e.first_activity_at
				ELSE e.activity_at
			END,
			e.indexed_at,
			NULLIF(to_jsonb(e)->>'system_count', '')::integer,
			NULLIF(to_jsonb(e)->>'system_count_capped', '')::boolean,
			NULLIF(to_jsonb(e)->>'system_destination', ''),
			sp.uri, sp.did, sp.rkey, sp.cid, sp.text, sp.facets, sp.images,
			sp.reply_root_uri, sp.reply_root_cid, sp.reply_parent_uri, sp.reply_parent_cid,
			sp.quote_uri, sp.quote_cid, sp.tags, sp.created_at, sp.indexed_at,
			sp.is_project, sp.project_craft_type, spp.raw_project,
			sbp.display_name, sbp.avatar_cid
		FROM page_events e
		LEFT JOIN bluesky_profiles actor_bp ON actor_bp.did = e.actor_did
		LEFT JOIN visible_posts source_post ON source_post.uri=e.source_uri
		LEFT JOIN visible_posts sp ON sp.uri=e.subject_uri
		LEFT JOIN visible_posts parent_post ON parent_post.uri=e.parent_uri
		LEFT JOIN visible_posts root_post ON root_post.uri=e.root_uri
		LEFT JOIN visible_posts quoted_post ON quoted_post.uri=e.quoted_uri
		LEFT JOIN visible_posts subject_quote ON subject_quote.uri=sp.quote_uri
		LEFT JOIN craftsky_project_posts spp ON spp.uri = sp.uri
		LEFT JOIN bluesky_profiles sbp ON sbp.did = sp.did
		ORDER BY e.indexed_at DESC, e.id DESC
	`
	queryLimit := limit + 1
	rows, err := s.pool.Query(ctx, q, viewerDID, curIndexedAt, curID, queryLimit)
	if err != nil {
		return nil, "", fmt.Errorf("notification list %s: %w", viewerDID, err)
	}
	defer rows.Close()

	out := make([]*NotificationRow, 0, queryLimit)
	for rows.Next() {
		row := &NotificationRow{}
		var eventKind, eventType string
		var subject notificationSubjectScan
		var sourceURI, sourceCID, sourceRkey, actorDID sql.NullString
		var systemCount sql.NullInt64
		var systemCountCapped sql.NullBool
		var systemDestination sql.NullString
		var sourceAvailable, subjectAvailable, parentAvailable, rootAvailable, quotedAvailable, subjectQuoteAvailable bool
		var subjectURI, subjectCID, parentURI, parentCID, rootURI, rootCID, quotedURI, quotedCID sql.NullString
		if err := rows.Scan(
			&row.ID,
			&eventKind,
			&eventType,
			&sourceURI, &sourceCID, &sourceRkey, &sourceAvailable,
			&subjectURI, &subjectCID, &subjectAvailable,
			&parentURI, &parentCID, &parentAvailable,
			&rootURI, &rootCID, &rootAvailable,
			&quotedURI, &quotedCID, &quotedAvailable,
			&subjectQuoteAvailable,
			&actorDID, &row.ActorDisplayName, &row.ActorAvatarCID, &row.ActorAvatarMime, &row.ActorViewerIsFollowing,
			&row.CreatedAt, &row.IndexedAt,
			&systemCount, &systemCountCapped, &systemDestination,
			&subject.URI, &subject.DID, &subject.Rkey, &subject.CID, &subject.Text, &subject.Facets, &subject.Images,
			&subject.ReplyRootURI, &subject.ReplyRootCID, &subject.ReplyParentURI, &subject.ReplyParentCID,
			&subject.QuoteURI, &subject.QuoteCID, &subject.Tags, &subject.CreatedAt, &subject.IndexedAt,
			&subject.IsProject, &subject.ProjectCraftType, &subject.RawProject,
			&subject.AuthorDisplayName, &subject.AuthorAvatarCID,
		); err != nil {
			return nil, "", fmt.Errorf("notification list scan: %w", err)
		}
		row.Kind = NotificationKind(eventKind)
		row.Type = NotificationType(eventType)
		row.ActorDID = actorDID.String
		if row.Kind == NotificationKindSystem {
			if systemCount.Valid && systemCountCapped.Valid && systemDestination.Valid {
				row.System = &NotificationSystem{
					Count:       int(systemCount.Int64),
					CountCapped: systemCountCapped.Bool,
					Destination: NotificationDestination(systemDestination.String),
				}
			}
			if row.Type == NotificationTypeInstagramMatch && s.instagramNotificationEligibility != nil {
				notificationID, parseErr := uuid.Parse(row.ID)
				if parseErr != nil {
					return nil, "", fmt.Errorf("parse Instagram notification id: %w", parseErr)
				}
				eligible, eligibilityErr := s.instagramNotificationEligibility.RevalidateNotification(ctx, notificationID, instagram.EligibilityAtFeed)
				if eligibilityErr != nil {
					return nil, "", fmt.Errorf("revalidate Instagram notification feed: %w", eligibilityErr)
				}
				if !eligible {
					continue
				}
			}
		} else {
			row.References = NotificationReferences{
				Source:  *notificationReference(sourceURI, sourceCID, sourceRkey.String, sourceAvailable),
				Subject: notificationReference(subjectURI, subjectCID, "", subjectAvailable),
				Parent:  notificationReference(parentURI, parentCID, "", parentAvailable),
				Root:    notificationReference(rootURI, rootCID, "", rootAvailable),
				Quoted:  notificationReference(quotedURI, quotedCID, "", quotedAvailable),
			}
			if sourceAvailable {
				row.URI, row.CID, row.Rkey = sourceURI.String, sourceCID.String, sourceRkey.String
			}
		}
		if subject.URI.Valid {
			row.SubjectPost = subject.postRow()
			if !subjectQuoteAvailable {
				row.SubjectPost.QuoteURI = nil
				row.SubjectPost.QuoteCID = nil
			}
		}
		if row.Type == NotificationTypeReply {
			row.Reply = &NotificationReplyRef{Available: sourceAvailable, URI: row.URI, CID: row.CID, Rkey: row.Rkey}
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
		"uri":       last.ID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode notification cursor: %w", err)
	}
	return out, next, nil
}

func notificationReference(uri, cid sql.NullString, rkey string, available bool) *NotificationReference {
	if !uri.Valid || uri.String == "" {
		return nil
	}
	ref := &NotificationReference{Available: available}
	if available {
		ref.URI = uri.String
		ref.CID = cid.String
		ref.Rkey = rkey
	}
	return ref
}

type notificationSubjectScan struct {
	URI              sql.NullString
	DID              sql.NullString
	Rkey             sql.NullString
	CID              sql.NullString
	Text             sql.NullString
	Facets           json.RawMessage
	Images           json.RawMessage
	ReplyRootURI     *string
	ReplyRootCID     *string
	ReplyParentURI   *string
	ReplyParentCID   *string
	QuoteURI         *string
	QuoteCID         *string
	Tags             []string
	CreatedAt        sql.NullTime
	IndexedAt        sql.NullTime
	IsProject        sql.NullBool
	ProjectCraftType *string
	RawProject       *json.RawMessage

	AuthorDisplayName *string
	AuthorAvatarCID   *string
}

func (s notificationSubjectScan) postRow() *PostRow {
	row := &PostRow{
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
	row.IsProject = s.IsProject.Valid && s.IsProject.Bool
	row.ProjectCraftType = s.ProjectCraftType
	if s.RawProject != nil && len(*s.RawProject) > 0 {
		row.RawProject = append(json.RawMessage(nil), (*s.RawProject)...)
		var project Project
		if err := json.Unmarshal(*s.RawProject, &project); err == nil {
			row.Project = &project
		}
	}
	return row
}
