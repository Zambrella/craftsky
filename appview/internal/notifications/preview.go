package notifications

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Preview contains only public, visible text. Media and routing facts are kept separate.
type Preview struct {
	Text         string `json:"text"`
	ProjectTitle string `json:"projectTitle,omitempty"`
	Role         string `json:"role"`
}

// ReadPreviews applies visibility at read time, including blocks between a
// source author and the authors of its reply, quote, and mention references.
func ReadPreviews(ctx context.Context, pool *pgxpool.Pool, viewer syntax.DID, uris []syntax.ATURI) (map[syntax.ATURI]*Preview, error) {
	out := make(map[syntax.ATURI]*Preview)
	if len(uris) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
 SELECT p.uri,p.text,COALESCE(project.raw_project#>>'{common,title}',''),
 CASE WHEN p.reply_root_uri IS NULL THEN 'post'
      WHEN p.reply_parent_uri=p.reply_root_uri THEN 'comment' ELSE 'reply' END
 FROM craftsky_posts p
 LEFT JOIN craftsky_project_posts project ON project.uri=p.uri
 WHERE p.uri=ANY($2) AND NOT appview_owner_is_terminal(p.did)
 AND NOT EXISTS (SELECT 1 FROM actor_mutes m WHERE m.owner_did=$1 AND m.subject_did=p.did)
 AND NOT EXISTS (SELECT 1 FROM pds_set_aggregates b WHERE b.kind='block'
   AND ((b.actor_did=$1 AND b.subject_did=p.did) OR (b.actor_did=p.did AND b.subject_did=$1)))
 AND NOT EXISTS (
   SELECT 1 FROM pds_set_aggregates b
   WHERE b.kind='block' AND NOT appview_owner_is_terminal(b.actor_did) AND NOT appview_owner_is_terminal(b.subject_did)
   AND ((b.actor_did=p.did AND b.subject_did IN (
     SELECT r.did FROM craftsky_posts r WHERE r.uri IN (p.reply_parent_uri,p.quote_uri)
     UNION SELECT m.mentioned_did FROM craftsky_post_mentions m WHERE m.post_uri=p.uri))
     OR (b.subject_did=p.did AND b.actor_did IN (
     SELECT r.did FROM craftsky_posts r WHERE r.uri IN (p.reply_parent_uri,p.quote_uri)
     UNION SELECT m.mentioned_did FROM craftsky_post_mentions m WHERE m.post_uri=p.uri)))
 )
 AND NOT EXISTS (
   SELECT 1 FROM moderation_outputs mo
   WHERE mo.action='apply' AND mo.value IN ('hide','takedown')
   AND NOT appview_owner_is_terminal(mo.source_did)
   AND (mo.expires_at IS NULL OR mo.expires_at>now())
   AND ((mo.subject_type='post' AND mo.subject_uri=p.uri) OR (mo.subject_type='account' AND mo.subject_did=p.did))
   AND NOT EXISTS (SELECT 1 FROM moderation_outputs neg
     WHERE neg.action='negate' AND neg.source_did=mo.source_did
     AND neg.subject_type=mo.subject_type AND neg.subject_did=mo.subject_did
     AND neg.value=mo.value AND (neg.expires_at IS NULL OR neg.expires_at>now())
     AND neg.indexed_at>mo.indexed_at AND (mo.subject_type='account' OR neg.subject_uri=mo.subject_uri))
 )`, viewer, uris)
	if err != nil {
		return nil, fmt.Errorf("notification previews: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri syntax.ATURI
		var preview Preview
		if err := rows.Scan(&uri, &preview.Text, &preview.ProjectTitle, &preview.Role); err != nil {
			return nil, fmt.Errorf("notification preview scan: %w", err)
		}
		out[uri] = &preview
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("notification preview rows: %w", err)
	}
	return out, nil
}
