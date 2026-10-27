// Package postpublication retains bounded original recipient facts, never post bodies.
package postpublication

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/postutil"
)

type Facts struct {
	Present                                                      bool
	Deleted                                                      bool
	ReplyEligible, QuoteEligible                                 bool
	Known                                                        bool
	CreatedAt                                                    *time.Time
	ParentURI, ParentCID, RootURI, RootCID, QuotedURI, QuotedCID string
	Mentions                                                     []string
}

type Reader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Source contains only the publication evidence needed before coalescing.
// Publication policy is independent of the Tap transport and its diagnostics.
type Source struct {
	URI    syntax.ATURI
	DID    syntax.DID
	Record json.RawMessage
	Live   bool
	Action string
}

func Read(ctx context.Context, tx Reader, uri syntax.ATURI) (Facts, error) {
	var facts Facts
	var status string
	err := tx.QueryRow(ctx, `SELECT origin_status,original_creation_time,
		COALESCE(parent_uri,''),COALESCE(parent_cid,''),COALESCE(root_uri,''),COALESCE(root_cid,''),COALESCE(quoted_uri,''),COALESCE(quoted_cid,''),reply_eligible,quote_eligible
		FROM post_publication_state WHERE uri=$1`, uri).Scan(&status, &facts.CreatedAt, &facts.ParentURI, &facts.ParentCID, &facts.RootURI, &facts.RootCID, &facts.QuotedURI, &facts.QuotedCID, &facts.ReplyEligible, &facts.QuoteEligible)
	if errors.Is(err, pgx.ErrNoRows) {
		return facts, nil
	}
	facts.Known = status == "established"
	facts.Deleted = status == "deleted"
	facts.Present = err == nil
	return facts, err
}

// Prepare reads the old source before coalescing can erase the original facts.
func Prepare(ctx context.Context, tx pgx.Tx, event Source) (*Facts, error) {
	var status string
	stateErr := tx.QueryRow(ctx, `SELECT origin_status FROM post_publication_state WHERE uri=$1`, event.URI).Scan(&status)
	if stateErr != nil && !errors.Is(stateErr, pgx.ErrNoRows) {
		return nil, stateErr
	}
	existing := stateErr == nil
	if existing && (status == "established" || status == "deleted") {
		return nil, nil
	}
	raw := event.Record
	live := event.Live
	action := event.Action
	var old json.RawMessage
	var oldLive bool
	var oldAction, structural, semantic string
	err := tx.QueryRow(ctx, `SELECT record,live,action,structural_validation_status,semantic_validation_status FROM tap_source_records WHERE uri=$1`, event.URI).Scan(&old, &oldLive, &oldAction, &structural, &semantic)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil && oldAction != "delete" && structural == "valid" && semantic == "valid" {
		raw, live, action = old, oldLive, oldAction
	}
	if !existing && live && action == "create" {
		facts, err := factsForRecord(raw, true)
		if err != nil {
			return nil, err
		}
		if facts.Known {
			return facts, nil
		}
	}
	var proven json.RawMessage
	err = tx.QueryRow(ctx, `SELECT s.record FROM pds_command_steps s
 JOIN pds_commands c ON c.id=s.command_id
 JOIN pds_command_dispatches d ON d.command_id=c.id AND d.plan_version=s.plan_version
 WHERE c.owner_did=$1 AND c.selected_uri=$2 AND c.operation_kind='post.create' AND c.state='accepted'
 AND s.selected_uri=$2 AND s.repo_did=$1 AND s.action='create' AND s.ordinal=0
 AND s.plan_version=c.active_plan_version AND d.outcome='accepted' AND d.result_records=s.record::jsonb
 ORDER BY c.created_at LIMIT 1`, event.DID, event.URI).Scan(&proven)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		facts, err := factsForRecord(proven, true)
		if err != nil {
			return nil, err
		}
		if facts.Known {
			return facts, nil
		}
	}
	if existing {
		return nil, nil
	}
	// Unknown candidates are a bounded applicability intersection only. They
	// cannot activate anything; later original proof must intersect this bound.
	return factsForRecord(raw, false)
}

func factsForRecord(raw json.RawMessage, known bool) (*Facts, error) {
	var rec craftskylex.FeedPost
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, err
	}
	if known && (rec.EditedAt != nil || rec.ExternalImport != nil) {
		return &Facts{}, nil
	}
	facts := &Facts{Known: known, Mentions: MentionDIDs(rec)}
	if known {
		created, err := time.Parse(time.RFC3339Nano, rec.CreatedAt)
		if err != nil {
			return nil, err
		}
		facts.CreatedAt = &created
	}
	if rec.Reply != nil && rec.Reply.Parent != nil && rec.Reply.Root != nil {
		facts.ParentURI = rec.Reply.Parent.Uri
		facts.ParentCID = rec.Reply.Parent.Cid
		facts.RootURI = rec.Reply.Root.Uri
		facts.RootCID = rec.Reply.Root.Cid
	}
	if rec.Embed != nil && rec.Embed.FeedPost_QuoteEmbed != nil && rec.Embed.FeedPost_QuoteEmbed.Record != nil {
		facts.QuotedURI = rec.Embed.FeedPost_QuoteEmbed.Record.Uri
		facts.QuotedCID = rec.Embed.FeedPost_QuoteEmbed.Record.Cid
	}
	return facts, nil
}

func Persist(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, owner syntax.DID, facts *Facts) error {
	if facts == nil {
		return nil
	}
	var previous string
	err := tx.QueryRow(ctx, `SELECT origin_status FROM post_publication_state WHERE uri=$1 FOR UPDATE`, uri).Scan(&previous)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	existing := err == nil
	if existing && (previous == "established" || previous == "deleted" || !facts.Known) {
		return nil
	}
	status := "unknown"
	if facts.Known {
		status = "established"
	}
	result, err := tx.Exec(ctx, `INSERT INTO post_publication_state(uri,owner_did,origin_status,original_creation_time,parent_uri,parent_cid,root_uri,root_cid,quoted_uri,quoted_cid)
 VALUES($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),NULLIF($10,''))
 ON CONFLICT(uri) DO UPDATE SET origin_status=EXCLUDED.origin_status,original_creation_time=EXCLUDED.original_creation_time,
 reply_eligible=post_publication_state.reply_eligible AND post_publication_state.parent_uri IS NOT DISTINCT FROM EXCLUDED.parent_uri
 AND post_publication_state.parent_cid IS NOT DISTINCT FROM EXCLUDED.parent_cid AND post_publication_state.root_uri IS NOT DISTINCT FROM EXCLUDED.root_uri AND post_publication_state.root_cid IS NOT DISTINCT FROM EXCLUDED.root_cid,
 quote_eligible=post_publication_state.quote_eligible AND post_publication_state.quoted_uri IS NOT DISTINCT FROM EXCLUDED.quoted_uri AND post_publication_state.quoted_cid IS NOT DISTINCT FROM EXCLUDED.quoted_cid,
 parent_uri=EXCLUDED.parent_uri,parent_cid=EXCLUDED.parent_cid,root_uri=EXCLUDED.root_uri,root_cid=EXCLUDED.root_cid,quoted_uri=EXCLUDED.quoted_uri,quoted_cid=EXCLUDED.quoted_cid
 WHERE post_publication_state.origin_status='unknown' AND EXCLUDED.origin_status='established'`, uri, owner, status, facts.CreatedAt, facts.ParentURI, facts.ParentCID, facts.RootURI, facts.RootCID, facts.QuotedURI, facts.QuotedCID)
	if err != nil || result.RowsAffected() == 0 {
		return err
	}
	if existing {
		// Discard unproven edit-added candidates. Original candidates not present
		// in the first observed bound remain ineligible, even when proof arrives.
		if _, err := tx.Exec(ctx, `DELETE FROM post_publication_mentions WHERE post_uri=$1 AND NOT(recipient_did=ANY($2::text[]))`, uri, facts.Mentions); err != nil {
			return err
		}
	}
	for _, did := range facts.Mentions {
		if _, err := tx.Exec(ctx, `INSERT INTO post_publication_mentions(post_uri,recipient_did,eligible) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, uri, did, !existing); err != nil {
			return err
		}
	}
	return nil
}

func MentionDIDs(rec craftskylex.FeedPost) []string {
	sets := [][]string{postutil.ExtractMentionDIDsForText(rec.Text, rec.Facets)}
	if rec.Project != nil && rec.Project.Common != nil {
		common := rec.Project.Common
		if p := common.Pattern; p != nil {
			if p.Name != nil {
				sets = append(sets, postutil.ExtractMentionDIDsForText(*p.Name, p.NameFacets))
			}
			if p.Designer != nil {
				sets = append(sets, postutil.ExtractMentionDIDsForText(*p.Designer, p.DesignerFacets))
			}
			if p.Publisher != nil {
				sets = append(sets, postutil.ExtractMentionDIDsForText(*p.Publisher, p.PublisherFacets))
			}
		}
		for _, material := range common.Materials {
			if material != nil {
				sets = append(sets, postutil.ExtractMentionDIDsForText(material.Text, material.Facets))
			}
		}
	}
	return postutil.MergeMentionDIDs(sets...)
}

func OriginalMentions(ctx context.Context, tx Reader, uri syntax.ATURI, current []string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT recipient_did FROM post_publication_mentions m JOIN post_publication_state s ON s.uri=m.post_uri WHERE m.post_uri=$1 AND m.eligible AND s.origin_status='established'`, uri)
	if err != nil {
		return nil, err
	}
	originals, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]bool, len(originals))
	for _, did := range originals {
		allowed[did] = true
	}
	var out []string
	for _, did := range current {
		if allowed[did] {
			out = append(out, did)
		}
	}
	return out, nil
}
