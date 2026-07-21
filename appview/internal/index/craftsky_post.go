// appview/internal/index/craftsky_post.go
package index

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/postutil"
	"social.craftsky/appview/internal/tap"
)

// CraftskyPost indexes social.craftsky.feed.post events into craftsky_posts.
// Required invariant: idempotent on (URI, CID). Tap delivers at-least-once.
//
// Posts are gated on craftsky_profiles membership: events from non-members
// are dropped silently, matching BlueskyProfile's pattern. A post arriving
// before its author's craftsky_profiles row is dropped permanently for now;
// see the design spec for the post-backfiller follow-up.
type CraftskyPost struct {
	pool      *pgxpool.Pool
	logger    *slog.Logger
	lifecycle notifications.Lifecycle
}

var _ Indexer = (*CraftskyPost)(nil)

func NewCraftskyPost(pool *pgxpool.Pool, logger *slog.Logger, lifecycles ...notifications.Lifecycle) *CraftskyPost {
	if logger == nil {
		logger = slog.Default()
	}
	lifecycle := notifications.Lifecycle(notifications.NoopLifecycle{})
	if len(lifecycles) > 0 && lifecycles[0] != nil {
		lifecycle = lifecycles[0]
	}
	return &CraftskyPost{pool: pool, logger: logger, lifecycle: lifecycle}
}

const craftskyPostNSID syntax.NSID = "social.craftsky.feed.post"

func (c *CraftskyPost) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != craftskyPostNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		return c.handleUpsert(ctx, ev)
	case "delete":
		return c.handleDelete(ctx, ev)
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

func (c *CraftskyPost) handleUpsert(ctx context.Context, ev tap.Event) error {
	isMember, err := c.isMember(ctx, ev.DID)
	if err != nil {
		return fmt.Errorf("membership check %s: %w", ev.DID, err)
	}
	if !isMember {
		return nil
	}

	var rec indexedPostRecord
	if err := json.Unmarshal(ev.Record, &rec); err != nil {
		return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
	}
	createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("parse createdAt %q on %s: %w", rec.CreatedAt, ev.URI, err)
	}

	var facetsJSON json.RawMessage
	if len(rec.Facets) > 0 && string(rec.Facets) != "null" {
		facetsJSON = append(json.RawMessage(nil), rec.Facets...)
	}

	var imagesJSON []byte
	if flat := flattenImages(rec.Images); flat != nil {
		imagesJSON, err = json.Marshal(flat)
		if err != nil {
			return fmt.Errorf("marshal images %s: %w", ev.URI, err)
		}
	}

	project, err := extractProjectForIndex(ev.Record)
	if err != nil {
		return fmt.Errorf("extract project %s: %w", ev.URI, err)
	}
	// Reply and quote pointers are typed `any` (not `string`) so absent
	// fields stay nil and pgx writes SQL NULL. An empty string would still
	// satisfy the partial indexes' `WHERE ... IS NOT NULL` predicate and
	// defeat them.
	var (
		replyRootURI, replyRootCID     any
		replyParentURI, replyParentCID any
	)
	if rec.Reply != nil {
		if rec.Reply.Root != nil {
			replyRootURI = rec.Reply.Root.Uri
			replyRootCID = rec.Reply.Root.Cid
		}
		if rec.Reply.Parent != nil {
			replyParentURI = rec.Reply.Parent.Uri
			replyParentCID = rec.Reply.Parent.Cid
		}
	}

	var quoteURI, quoteCID any
	if rec.Embed != nil && rec.Embed.FeedPost_QuoteEmbed != nil &&
		rec.Embed.FeedPost_QuoteEmbed.Record != nil {
		quoteURI = rec.Embed.FeedPost_QuoteEmbed.Record.Uri
		quoteCID = rec.Embed.FeedPost_QuoteEmbed.Record.Cid
	}
	if project != nil && (rec.Reply != nil || quoteURI != nil) {
		project = nil
	}
	topLevelFacets := postutil.DecodeFacets(rec.Facets)
	tags := postutil.MergeTags(postutil.ExtractTagsForText(rec.Text, topLevelFacets), projectSearchTags(project))
	mentions := postutil.MergeMentionDIDs(postutil.ExtractMentionDIDsForText(rec.Text, topLevelFacets), projectMentionDIDs(project))

	var isProject bool
	var projectCraftType any
	if project != nil {
		isProject = true
		projectCraftType = project.Common.CraftType
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert %s: %w", ev.URI, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO craftsky_posts
			(uri, did, rkey, cid, text, facets, images,
			 reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid,
			 quote_uri, quote_cid, tags, is_project, project_craft_type, record, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (uri) DO UPDATE SET
			cid              = EXCLUDED.cid,
			text             = EXCLUDED.text,
			facets           = EXCLUDED.facets,
			images           = EXCLUDED.images,
			reply_root_uri   = EXCLUDED.reply_root_uri,
			reply_root_cid   = EXCLUDED.reply_root_cid,
			reply_parent_uri = EXCLUDED.reply_parent_uri,
			reply_parent_cid = EXCLUDED.reply_parent_cid,
			quote_uri        = EXCLUDED.quote_uri,
			quote_cid        = EXCLUDED.quote_cid,
			tags             = EXCLUDED.tags,
			is_project       = EXCLUDED.is_project,
			project_craft_type = EXCLUDED.project_craft_type,
			record           = EXCLUDED.record,
			created_at       = EXCLUDED.created_at,
			indexed_at       = now()
		WHERE craftsky_posts.cid IS DISTINCT FROM EXCLUDED.cid
	`
	_, err = tx.Exec(ctx, q,
		ev.URI, ev.DID, ev.Rkey, ev.CID,
		rec.Text,
		facetsJSON, imagesJSON,
		replyRootURI, replyRootCID,
		replyParentURI, replyParentCID,
		quoteURI, quoteCID,
		tags,
		isProject,
		projectCraftType,
		ev.Record,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("upsert %s: %w", ev.URI, err)
	}
	if project != nil {
		if err := upsertProjectMaterialization(ctx, tx, ev.URI, project, tags); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `DELETE FROM craftsky_project_posts WHERE uri = $1`, ev.URI); err != nil {
		return fmt.Errorf("delete stale project %s: %w", ev.URI, err)
	}
	if err := syncPostMentions(ctx, tx, ev.URI, mentions, createdAt); err != nil {
		return err
	}
	if err := c.activatePostNotifications(ctx, tx, ev, rec, mentions, createdAt); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert %s: %w", ev.URI, err)
	}
	return nil
}

func (c *CraftskyPost) activatePostNotifications(ctx context.Context, tx pgx.Tx, ev tap.Event, rec indexedPostRecord, mentions []string, createdAt time.Time) error {
	reasons := make(map[syntax.DID]notifications.PostReasons)
	addPostAuthor := func(uri any, apply func(notifications.PostReasons) notifications.PostReasons) error {
		value, ok := uri.(string)
		if !ok || value == "" {
			return nil
		}
		var did syntax.DID
		if err := tx.QueryRow(ctx, `SELECT did FROM craftsky_posts WHERE uri = $1`, value).Scan(&did); err != nil {
			if err == pgx.ErrNoRows {
				return nil
			}
			return err
		}
		reasons[did] = apply(reasons[did])
		return nil
	}
	if rec.Reply != nil && rec.Reply.Parent != nil {
		if err := addPostAuthor(rec.Reply.Parent.Uri, func(r notifications.PostReasons) notifications.PostReasons { r.Reply = true; return r }); err != nil {
			return fmt.Errorf("classify reply notification: %w", err)
		}
	}
	if rec.Embed != nil && rec.Embed.FeedPost_QuoteEmbed != nil && rec.Embed.FeedPost_QuoteEmbed.Record != nil {
		if err := addPostAuthor(rec.Embed.FeedPost_QuoteEmbed.Record.Uri, func(r notifications.PostReasons) notifications.PostReasons { r.Quote = true; return r }); err != nil {
			return fmt.Errorf("classify quote notification: %w", err)
		}
	}
	for _, rawDID := range mentions {
		did, err := syntax.ParseDID(rawDID)
		if err != nil {
			return fmt.Errorf("parse mention DID: %w", err)
		}
		r := reasons[did]
		r.Mention = true
		reasons[did] = r
	}
	for recipient, recipientReasons := range reasons {
		category, ok := notifications.ClassifyPostReason(recipientReasons)
		if !ok || recipient == ev.DID {
			continue
		}
		var recipientIsMember bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=$1)`, recipient).Scan(&recipientIsMember); err != nil {
			return fmt.Errorf("check notification recipient membership: %w", err)
		}
		if !recipientIsMember {
			continue
		}
		if err := c.lifecycle.Activate(ctx, tx, notifications.Activation{
			RecipientDID: recipient,
			ActorDID:     ev.DID,
			Category:     category,
			SubjectKey:   ev.URI.String(),
			SourceURI:    ev.URI,
			SourceCID:    ev.CID,
			SourceRkey:   ev.Rkey,
			SubjectURI:   postNotificationSubjectURI(category, ev, rec),
			SubjectCID:   postNotificationSubjectCID(category, ev, rec),
			ParentURI:    postReplyParentURI(rec),
			ParentCID:    postReplyParentCID(rec),
			RootURI:      postReplyRootURI(rec),
			RootCID:      postReplyRootCID(rec),
			QuotedURI:    postQuotedURI(rec),
			QuotedCID:    postQuotedCID(rec),
			ActivityAt:   createdAt,
		}); err != nil {
			return fmt.Errorf("activate %s notification: %w", category, err)
		}
	}
	return nil
}

func postReplyParentURI(rec indexedPostRecord) syntax.ATURI {
	if rec.Reply != nil && rec.Reply.Parent != nil {
		return syntax.ATURI(rec.Reply.Parent.Uri)
	}
	return ""
}
func postReplyParentCID(rec indexedPostRecord) syntax.CID {
	if rec.Reply != nil && rec.Reply.Parent != nil {
		return syntax.CID(rec.Reply.Parent.Cid)
	}
	return ""
}
func postReplyRootURI(rec indexedPostRecord) syntax.ATURI {
	if rec.Reply != nil && rec.Reply.Root != nil {
		return syntax.ATURI(rec.Reply.Root.Uri)
	}
	return ""
}
func postReplyRootCID(rec indexedPostRecord) syntax.CID {
	if rec.Reply != nil && rec.Reply.Root != nil {
		return syntax.CID(rec.Reply.Root.Cid)
	}
	return ""
}
func postQuotedURI(rec indexedPostRecord) syntax.ATURI {
	if rec.Embed != nil && rec.Embed.FeedPost_QuoteEmbed != nil && rec.Embed.FeedPost_QuoteEmbed.Record != nil {
		return syntax.ATURI(rec.Embed.FeedPost_QuoteEmbed.Record.Uri)
	}
	return ""
}
func postQuotedCID(rec indexedPostRecord) syntax.CID {
	if rec.Embed != nil && rec.Embed.FeedPost_QuoteEmbed != nil && rec.Embed.FeedPost_QuoteEmbed.Record != nil {
		return syntax.CID(rec.Embed.FeedPost_QuoteEmbed.Record.Cid)
	}
	return ""
}

func postNotificationSubjectURI(category notifications.Category, ev tap.Event, rec indexedPostRecord) syntax.ATURI {
	if category == notifications.Reply && rec.Reply != nil && rec.Reply.Parent != nil {
		return syntax.ATURI(rec.Reply.Parent.Uri)
	}
	return ev.URI
}

func postNotificationSubjectCID(category notifications.Category, ev tap.Event, rec indexedPostRecord) syntax.CID {
	if category == notifications.Reply && rec.Reply != nil && rec.Reply.Parent != nil {
		return syntax.CID(rec.Reply.Parent.Cid)
	}
	return ev.CID
}

type indexedProject struct {
	RawProject json.RawMessage
	RawDetails json.RawMessage
	Common     indexedProjectCommon
	Details    indexedProjectDetails
}

type indexedPostRecord struct {
	CreatedAt string                         `json:"createdAt"`
	Embed     *craftskylex.FeedPost_Embed    `json:"embed,omitempty"`
	Facets    json.RawMessage                `json:"facets,omitempty"`
	Images    []*craftskylex.FeedPost_Image  `json:"images,omitempty"`
	Reply     *craftskylex.FeedPost_ReplyRef `json:"reply,omitempty"`
	Text      string                         `json:"text"`
}

type indexedProjectCommon struct {
	CraftType  string                   `json:"craftType"`
	Status     *string                  `json:"status"`
	Title      *string                  `json:"title"`
	Duration   *string                  `json:"duration"`
	Pattern    *indexedProjectPattern   `json:"pattern"`
	Materials  []indexedProjectMaterial `json:"materials"`
	Colors     []string                 `json:"colors"`
	DesignTags []string                 `json:"designTags"`
	Tags       []string                 `json:"tags"`
}

type indexedProjectPattern struct {
	URL             *string         `json:"url"`
	Name            *string         `json:"name"`
	NameFacets      json.RawMessage `json:"nameFacets"`
	Difficulty      *string         `json:"difficulty"`
	Designer        *string         `json:"designer"`
	DesignerFacets  json.RawMessage `json:"designerFacets"`
	Publisher       *string         `json:"publisher"`
	PublisherFacets json.RawMessage `json:"publisherFacets"`
}

type indexedProjectMaterial struct {
	Text   string          `json:"text"`
	Facets json.RawMessage `json:"facets"`
}

type indexedProjectDetails struct {
	Type string
	Map  map[string]json.RawMessage
}

func extractProjectForIndex(raw json.RawMessage) (*indexedProject, error) {
	var root struct {
		Project json.RawMessage `json:"project"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	if len(root.Project) == 0 || string(root.Project) == "null" {
		return nil, nil
	}
	var payload struct {
		Common  json.RawMessage `json:"common"`
		Details json.RawMessage `json:"details"`
	}
	if err := json.Unmarshal(root.Project, &payload); err != nil {
		return nil, err
	}
	if len(payload.Common) == 0 || string(payload.Common) == "null" {
		return nil, nil
	}
	var common indexedProjectCommon
	if err := json.Unmarshal(payload.Common, &common); err != nil {
		return nil, err
	}
	common.CraftType = strings.TrimSpace(common.CraftType)
	if common.CraftType == "" {
		return nil, nil
	}
	out := &indexedProject{RawProject: append(json.RawMessage(nil), root.Project...), Common: common}
	if len(payload.Details) > 0 && string(payload.Details) != "null" {
		out.RawDetails = append(json.RawMessage(nil), payload.Details...)
		var detailsMap map[string]json.RawMessage
		if err := json.Unmarshal(payload.Details, &detailsMap); err != nil {
			return nil, err
		}
		out.Details.Map = detailsMap
		if rawType, ok := detailsMap["$type"]; ok {
			_ = json.Unmarshal(rawType, &out.Details.Type)
		}
	}
	return out, nil
}

func projectSearchTags(project *indexedProject) []string {
	if project == nil {
		return nil
	}
	return postutil.ExtractProjectTags(
		project.Common.Tags,
		indexedPatternFacetedTexts(project.Common.Pattern),
		indexedMaterialFacetedTexts(project.Common.Materials),
	)
}

func projectMentionDIDs(project *indexedProject) []string {
	if project == nil {
		return nil
	}
	return postutil.ExtractProjectMentionDIDs(
		indexedPatternFacetedTexts(project.Common.Pattern),
		indexedMaterialFacetedTexts(project.Common.Materials),
	)
}

func indexedPatternFacetedTexts(pattern *indexedProjectPattern) []postutil.FacetedText {
	if pattern == nil {
		return nil
	}
	return []postutil.FacetedText{
		{Text: stringPtrValue(pattern.Name), Facets: postutil.DecodeFacets(pattern.NameFacets)},
		{Text: stringPtrValue(pattern.Designer), Facets: postutil.DecodeFacets(pattern.DesignerFacets)},
		{Text: stringPtrValue(pattern.Publisher), Facets: postutil.DecodeFacets(pattern.PublisherFacets)},
	}
}

func indexedMaterialFacetedTexts(materials []indexedProjectMaterial) []postutil.FacetedText {
	out := make([]postutil.FacetedText, 0, len(materials))
	for _, material := range materials {
		out = append(out, postutil.FacetedText{
			Text:   material.Text,
			Facets: postutil.DecodeFacets(material.Facets),
		})
	}
	return out
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func upsertProjectMaterialization(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, project *indexedProject, tags []string) error {
	common := project.Common
	var patternURL, patternName, patternDifficulty, patternDesigner, patternPublisher *string
	var patternNameFacets, patternDesignerFacets, patternPublisherFacets json.RawMessage
	if common.Pattern != nil {
		patternURL = common.Pattern.URL
		patternName = common.Pattern.Name
		patternDifficulty = common.Pattern.Difficulty
		patternDesigner = common.Pattern.Designer
		patternPublisher = common.Pattern.Publisher
		patternNameFacets = common.Pattern.NameFacets
		patternDesignerFacets = common.Pattern.DesignerFacets
		patternPublisherFacets = common.Pattern.PublisherFacets
	}
	detailCols := craftDetailColumnsFor(project)
	const q = `
		INSERT INTO craftsky_project_posts (
			uri, raw_project, common_craft_type, common_status, common_title, common_duration,
			pattern_url, pattern_name, pattern_name_facets, pattern_difficulty, pattern_designer, pattern_designer_facets, pattern_publisher, pattern_publisher_facets,
			materials, colors, design_tags, project_tags, details_type, raw_details,
			knitting_project_type, knitting_project_subtype, knitting_yarn_weight, knitting_needle_size_mm, knitting_gauge, knitting_finished_size,
			crochet_project_type, crochet_project_subtype, crochet_yarn_weight, crochet_hook_size_mm, crochet_gauge, crochet_finished_size,
			quilting_project_type, quilting_project_subtype, quilting_piecing_technique, quilting_quilting_method, quilting_size,
			sewing_project_type, sewing_project_subtype, sewing_size_made, sewing_fit_notes
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26,
			$27, $28, $29, $30, $31, $32,
			$33, $34, $35, $36, $37,
			$38, $39, $40, $41
		)
		ON CONFLICT (uri) DO UPDATE SET
			raw_project = EXCLUDED.raw_project,
			common_craft_type = EXCLUDED.common_craft_type,
			common_status = EXCLUDED.common_status,
			common_title = EXCLUDED.common_title,
			common_duration = EXCLUDED.common_duration,
			pattern_url = EXCLUDED.pattern_url,
			pattern_name = EXCLUDED.pattern_name,
			pattern_name_facets = EXCLUDED.pattern_name_facets,
			pattern_difficulty = EXCLUDED.pattern_difficulty,
			pattern_designer = EXCLUDED.pattern_designer,
			pattern_designer_facets = EXCLUDED.pattern_designer_facets,
			pattern_publisher = EXCLUDED.pattern_publisher,
			pattern_publisher_facets = EXCLUDED.pattern_publisher_facets,
			materials = EXCLUDED.materials,
			colors = EXCLUDED.colors,
			design_tags = EXCLUDED.design_tags,
			project_tags = EXCLUDED.project_tags,
			details_type = EXCLUDED.details_type,
			raw_details = EXCLUDED.raw_details,
			knitting_project_type = EXCLUDED.knitting_project_type,
			knitting_project_subtype = EXCLUDED.knitting_project_subtype,
			knitting_yarn_weight = EXCLUDED.knitting_yarn_weight,
			knitting_needle_size_mm = EXCLUDED.knitting_needle_size_mm,
			knitting_gauge = EXCLUDED.knitting_gauge,
			knitting_finished_size = EXCLUDED.knitting_finished_size,
			crochet_project_type = EXCLUDED.crochet_project_type,
			crochet_project_subtype = EXCLUDED.crochet_project_subtype,
			crochet_yarn_weight = EXCLUDED.crochet_yarn_weight,
			crochet_hook_size_mm = EXCLUDED.crochet_hook_size_mm,
			crochet_gauge = EXCLUDED.crochet_gauge,
			crochet_finished_size = EXCLUDED.crochet_finished_size,
			quilting_project_type = EXCLUDED.quilting_project_type,
			quilting_project_subtype = EXCLUDED.quilting_project_subtype,
			quilting_piecing_technique = EXCLUDED.quilting_piecing_technique,
			quilting_quilting_method = EXCLUDED.quilting_quilting_method,
			quilting_size = EXCLUDED.quilting_size,
			sewing_project_type = EXCLUDED.sewing_project_type,
			sewing_project_subtype = EXCLUDED.sewing_project_subtype,
			sewing_size_made = EXCLUDED.sewing_size_made,
			sewing_fit_notes = EXCLUDED.sewing_fit_notes,
			indexed_at = now()
		WHERE craftsky_project_posts.raw_project IS DISTINCT FROM EXCLUDED.raw_project
		   OR craftsky_project_posts.project_tags IS DISTINCT FROM EXCLUDED.project_tags
	`
	_, err := tx.Exec(ctx, q,
		uri, project.RawProject, common.CraftType, common.Status, common.Title, common.Duration,
		patternURL, patternName, nullableJSON(patternNameFacets), patternDifficulty, patternDesigner, nullableJSON(patternDesignerFacets), patternPublisher, nullableJSON(patternPublisherFacets),
		materialTexts(common.Materials), nonNilStrings(common.Colors), nonNilStrings(common.DesignTags), nonNilStrings(tags), nullableString(project.Details.Type), nullableJSON(project.RawDetails),
		detailCols.knittingProjectType, detailCols.knittingProjectSubtype, detailCols.knittingYarnWeight, detailCols.knittingNeedleSizeMM, detailCols.knittingGauge, detailCols.knittingFinishedSize,
		detailCols.crochetProjectType, detailCols.crochetProjectSubtype, detailCols.crochetYarnWeight, detailCols.crochetHookSizeMM, detailCols.crochetGauge, detailCols.crochetFinishedSize,
		detailCols.quiltingProjectType, detailCols.quiltingProjectSubtype, detailCols.quiltingPiecingTechnique, detailCols.quiltingQuiltingMethod, detailCols.quiltingSize,
		detailCols.sewingProjectType, detailCols.sewingProjectSubtype, detailCols.sewingSizeMade, detailCols.sewingFitNotes,
	)
	if err != nil {
		return fmt.Errorf("upsert project %s: %w", uri, err)
	}
	return nil
}

func syncPostMentions(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, mentionedDIDs []string, createdAt time.Time) error {
	if len(mentionedDIDs) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM craftsky_post_mentions WHERE post_uri = $1`, uri); err != nil {
			return fmt.Errorf("delete mentions %s: %w", uri, err)
		}
		return nil
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM craftsky_post_mentions
		WHERE post_uri = $1 AND NOT (mentioned_did = ANY($2::text[]))
	`, uri, mentionedDIDs); err != nil {
		return fmt.Errorf("delete removed mentions %s: %w", uri, err)
	}
	for _, did := range mentionedDIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO craftsky_post_mentions (post_uri, mentioned_did, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (post_uri, mentioned_did) DO NOTHING
		`, uri, did, createdAt); err != nil {
			return fmt.Errorf("insert mention %s -> %s: %w", uri, did, err)
		}
	}
	return nil
}

type craftDetailColumns struct {
	knittingProjectType    any
	knittingProjectSubtype any
	knittingYarnWeight     any
	knittingNeedleSizeMM   any
	knittingGauge          any
	knittingFinishedSize   any

	crochetProjectType    any
	crochetProjectSubtype any
	crochetYarnWeight     any
	crochetHookSizeMM     any
	crochetGauge          any
	crochetFinishedSize   any

	quiltingProjectType      any
	quiltingProjectSubtype   any
	quiltingPiecingTechnique any
	quiltingQuiltingMethod   any
	quiltingSize             any

	sewingProjectType    any
	sewingProjectSubtype any
	sewingSizeMade       any
	sewingFitNotes       any
}

func craftDetailColumnsFor(project *indexedProject) craftDetailColumns {
	var cols craftDetailColumns
	if project == nil {
		return cols
	}
	d := project.Details.Map
	switch project.Details.Type {
	case "social.craftsky.project.knitting#details":
		cols.knittingProjectType = jsonString(d, "projectType")
		cols.knittingProjectSubtype = jsonString(d, "projectSubtype")
		cols.knittingYarnWeight = jsonString(d, "yarnWeight")
		cols.knittingNeedleSizeMM = jsonString(d, "needleSizeMm")
		cols.knittingGauge = jsonRaw(d, "gauge")
		cols.knittingFinishedSize = jsonString(d, "finishedSize")
	case "social.craftsky.project.crochet#details":
		cols.crochetProjectType = jsonString(d, "projectType")
		cols.crochetProjectSubtype = jsonString(d, "projectSubtype")
		cols.crochetYarnWeight = jsonString(d, "yarnWeight")
		cols.crochetHookSizeMM = jsonString(d, "hookSizeMm")
		cols.crochetGauge = jsonRaw(d, "gauge")
		cols.crochetFinishedSize = jsonString(d, "finishedSize")
	case "social.craftsky.project.quilting#details":
		cols.quiltingProjectType = jsonString(d, "projectType")
		cols.quiltingProjectSubtype = jsonString(d, "projectSubtype")
		cols.quiltingPiecingTechnique = jsonString(d, "piecingTechnique")
		cols.quiltingQuiltingMethod = jsonString(d, "quiltingMethod")
		cols.quiltingSize = jsonString(d, "size")
	case "social.craftsky.project.sewing#details":
		cols.sewingProjectType = jsonString(d, "projectType")
		cols.sewingProjectSubtype = jsonString(d, "projectSubtype")
		cols.sewingSizeMade = jsonString(d, "sizeMade")
		cols.sewingFitNotes = jsonString(d, "fitNotes")
	}
	return cols
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func materialTexts(in []indexedProjectMaterial) []string {
	out := make([]string, 0, len(in))
	for _, material := range in {
		text := strings.TrimSpace(material.Text)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func nullableString(in string) any {
	if in == "" {
		return nil
	}
	return in
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

func jsonString(m map[string]json.RawMessage, key string) any {
	if len(m) == 0 {
		return nil
	}
	raw, ok := m[key]
	if !ok {
		return nil
	}
	var out string
	if err := json.Unmarshal(raw, &out); err != nil || out == "" {
		return nil
	}
	return out
}

func jsonRaw(m map[string]json.RawMessage, key string) any {
	if len(m) == 0 {
		return nil
	}
	raw, ok := m[key]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

func (c *CraftskyPost) handleDelete(ctx context.Context, ev tap.Event) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete %s: %w", ev.URI, err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		WITH RECURSIVE descendants(uri, path) AS (
			SELECT p.uri, ARRAY[$1::text, p.uri]
			FROM craftsky_posts p
			WHERE p.uri <> $1
			  AND (
				p.reply_parent_uri = $1
				OR (
					p.reply_root_uri = $1
					AND NOT EXISTS (
						SELECT 1
						FROM craftsky_posts parent
						WHERE parent.uri = p.reply_parent_uri
					)
				)
			  )

			UNION ALL

			SELECT child.uri, descendants.path || child.uri
			FROM craftsky_posts child
			JOIN descendants ON child.reply_parent_uri = descendants.uri
			WHERE NOT child.uri = ANY(descendants.path)
		), affected(uri) AS (
			SELECT $1::text
			UNION
			SELECT uri FROM descendants
		)
		DELETE FROM saved_posts
		WHERE post_uri IN (SELECT uri FROM affected)
	`, ev.URI); err != nil {
		return fmt.Errorf("delete saved state for post and descendants: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM craftsky_posts WHERE uri = $1`, ev.URI); err != nil {
		return fmt.Errorf("delete %s: %w", ev.URI, err)
	}
	if err := c.lifecycle.Retract(ctx, tx, notifications.Retraction{SourceURI: ev.URI, Reason: "sourceDeleted"}); err != nil {
		return fmt.Errorf("retract notification for %s: %w", ev.URI, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete %s: %w", ev.URI, err)
	}
	return nil
}

func (c *CraftskyPost) isMember(ctx context.Context, did syntax.DID) (bool, error) {
	var exists bool
	err := c.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`, did).
		Scan(&exists)
	return exists, err
}

// flattenImages turns the lexicon's [{image: LexBlob, alt?, aspectRatio?}, ...] array
// into the storage shape [{cid, mime, size, alt, aspectRatio?}, ...]. Returns nil when there
// are no images, so the caller can pass nil to the JSONB column for SQL NULL.
func flattenImages(images []*craftskylex.FeedPost_Image) []map[string]any {
	if len(images) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(images))
	for _, img := range images {
		if img == nil || img.Image == nil {
			continue
		}
		alt := ""
		if img.Alt != nil {
			alt = *img.Alt
		}
		one := map[string]any{
			"cid":  img.Image.Ref.String(),
			"mime": img.Image.MimeType,
			"size": img.Image.Size,
			"alt":  alt,
		}
		if img.AspectRatio != nil {
			one["aspectRatio"] = map[string]any{
				"width":  img.AspectRatio.Width,
				"height": img.AspectRatio.Height,
			}
		}
		out = append(out, one)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
