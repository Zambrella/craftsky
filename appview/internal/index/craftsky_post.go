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
	pool   *pgxpool.Pool
	logger *slog.Logger
}

var _ Indexer = (*CraftskyPost)(nil)

func NewCraftskyPost(pool *pgxpool.Pool, logger *slog.Logger) *CraftskyPost {
	if logger == nil {
		logger = slog.Default()
	}
	return &CraftskyPost{pool: pool, logger: logger}
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

	var rec craftskylex.FeedPost
	if err := json.Unmarshal(ev.Record, &rec); err != nil {
		return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
	}
	createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("parse createdAt %q on %s: %w", rec.CreatedAt, ev.URI, err)
	}

	var facetsJSON []byte
	if len(rec.Facets) > 0 {
		facetsJSON, err = json.Marshal(rec.Facets)
		if err != nil {
			return fmt.Errorf("marshal facets %s: %w", ev.URI, err)
		}
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
	tags := postutil.MergeTags(postutil.ExtractTags(rec.Facets), projectSearchTags(project))

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
		if err := upsertProjectMaterialization(ctx, tx, ev.URI, project); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `DELETE FROM craftsky_project_posts WHERE uri = $1`, ev.URI); err != nil {
		return fmt.Errorf("delete stale project %s: %w", ev.URI, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert %s: %w", ev.URI, err)
	}
	return nil
}

type indexedProject struct {
	RawProject json.RawMessage
	RawDetails json.RawMessage
	Common     indexedProjectCommon
	Details    indexedProjectDetails
}

type indexedProjectCommon struct {
	CraftType string  `json:"craftType"`
	Status    *string `json:"status"`
	Title     *string `json:"title"`
	Duration  *string `json:"duration"`
	Pattern   *struct {
		URL        *string `json:"url"`
		Name       *string `json:"name"`
		Difficulty *string `json:"difficulty"`
		Designer   *string `json:"designer"`
		Publisher  *string `json:"publisher"`
	} `json:"pattern"`
	Materials  []string `json:"materials"`
	Colors     []string `json:"colors"`
	DesignTags []string `json:"designTags"`
	Tags       []string `json:"tags"`
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
	return project.Common.Tags
}

func upsertProjectMaterialization(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, project *indexedProject) error {
	common := project.Common
	var patternURL, patternName, patternDifficulty, patternDesigner, patternPublisher *string
	if common.Pattern != nil {
		patternURL = common.Pattern.URL
		patternName = common.Pattern.Name
		patternDifficulty = common.Pattern.Difficulty
		patternDesigner = common.Pattern.Designer
		patternPublisher = common.Pattern.Publisher
	}
	detailCols := craftDetailColumnsFor(project)
	const q = `
		INSERT INTO craftsky_project_posts (
			uri, raw_project, common_craft_type, common_status, common_title, common_duration,
			pattern_url, pattern_name, pattern_difficulty, pattern_designer, pattern_publisher,
			materials, colors, design_tags, project_tags, details_type, raw_details,
			knitting_project_type, knitting_project_subtype, knitting_yarn_weight, knitting_needle_size_mm, knitting_gauge, knitting_finished_size,
			crochet_project_type, crochet_project_subtype, crochet_yarn_weight, crochet_hook_size_mm, crochet_gauge, crochet_finished_size,
			quilting_project_type, quilting_project_subtype, quilting_piecing_technique, quilting_quilting_method, quilting_size,
			sewing_project_type, sewing_project_subtype, sewing_size_made, sewing_fit_notes
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23,
			$24, $25, $26, $27, $28, $29,
			$30, $31, $32, $33, $34,
			$35, $36, $37, $38
		)
		ON CONFLICT (uri) DO UPDATE SET
			raw_project = EXCLUDED.raw_project,
			common_craft_type = EXCLUDED.common_craft_type,
			common_status = EXCLUDED.common_status,
			common_title = EXCLUDED.common_title,
			common_duration = EXCLUDED.common_duration,
			pattern_url = EXCLUDED.pattern_url,
			pattern_name = EXCLUDED.pattern_name,
			pattern_difficulty = EXCLUDED.pattern_difficulty,
			pattern_designer = EXCLUDED.pattern_designer,
			pattern_publisher = EXCLUDED.pattern_publisher,
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
	`
	_, err := tx.Exec(ctx, q,
		uri, project.RawProject, common.CraftType, common.Status, common.Title, common.Duration,
		patternURL, patternName, patternDifficulty, patternDesigner, patternPublisher,
		nonNilStrings(common.Materials), nonNilStrings(common.Colors), nonNilStrings(common.DesignTags), nonNilStrings(common.Tags), nullableString(project.Details.Type), nullableJSON(project.RawDetails),
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

func nullableString(in string) any {
	if in == "" {
		return nil
	}
	return in
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
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
	if _, err := c.pool.Exec(ctx,
		`DELETE FROM craftsky_posts WHERE uri = $1`, ev.URI); err != nil {
		return fmt.Errorf("delete %s: %w", ev.URI, err)
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
