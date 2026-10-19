package api

import "strings"

// SearchFieldContract is the executable allowlist for submitted keyword search.
// Unknown JSON, facets, URLs, identifiers, timestamps, commercial flags, and
// import provenance are deliberately excluded. See docs/development/search-fields.md.
type SearchFieldContract struct {
	Name       string
	SQL        string
	Weight     string
	Project    bool
	Vocabulary bool
}

var submittedSearchFields = []SearchFieldContract{
	{"text", "p.text", "1.0", false, false},
	{"tags", distinctSearchTagTextSQL("p.tags"), "0.2", false, false},
	{"project.common.title", "pp.common_title", "1.0", true, false},
	{"project.common.duration", "pp.common_duration", "0.3", true, false},
	{"project.common.pattern.name", "pp.pattern_name", "1.0", true, false},
	{"project.common.pattern.designer", "pp.pattern_designer", "0.7", true, false},
	{"project.common.pattern.publisher", "pp.pattern_publisher", "0.7", true, false},
	{"project.common.materials.text", "array_to_string(pp.materials,' ')", "0.4", true, false},
	{"project.common.tags", distinctSearchTagTextSQL("pp.project_tags"), "0.2", true, false},
	{"project.common.craftType/additionalCrafts", "array_to_string(" + projectCraftValuesSQL + ",' ')", "0.4", true, true},
	{"project.common.status", "pp.common_status", "0.4", true, true},
	{"project.common.pattern.difficulty", "pp.pattern_difficulty", "0.4", true, true},
	{"project.common.colors", "array_to_string(pp.colors,' ')", "0.4", true, true},
	{"project.common.designTags", "array_to_string(pp.design_tags,' ')", "0.2", true, true},
	{"project.common.pattern.selfDrafted", "CASE WHEN pp.pattern_self_drafted THEN 'self drafted' END", "0.4", true, false},
	{"project.details.projectType", "concat_ws(' ',pp.knitting_project_type,pp.crochet_project_type,pp.sewing_project_type,pp.quilting_project_type)", "0.4", true, true},
	{"project.details.projectSubtype", "concat_ws(' ',pp.knitting_project_subtype,pp.crochet_project_subtype,pp.sewing_project_subtype,pp.quilting_project_subtype)", "0.4", true, true},
	{"project.details.yarnWeight", "concat_ws(' ',pp.knitting_yarn_weight,pp.crochet_yarn_weight)", "0.4", true, true},
	{"project.details.needleSizeMm/hookSizeMm", "concat_ws(' ',pp.knitting_needle_size_mm,pp.crochet_hook_size_mm)", "0.3", true, false},
	{"project.details.finishedSize/size/sizeMade", "concat_ws(' ',pp.knitting_finished_size,pp.crochet_finished_size,pp.quilting_size,pp.sewing_size_made)", "0.3", true, false},
	{"project.details.fitNotes", "pp.sewing_fit_notes", "0.4", true, false},
	{"project.details.piecingTechnique", "pp.quilting_piecing_technique", "0.4", true, true},
	{"project.details.quiltingMethod", "pp.quilting_quilting_method", "0.4", true, true},
	{"project.details.gauge", "concat_ws(' ',pp.knitting_gauge->>'stitches',pp.knitting_gauge->>'rows',pp.knitting_gauge->>'measurement',pp.knitting_gauge->>'unit',pp.crochet_gauge->>'stitches',pp.crochet_gauge->>'rows',pp.crochet_gauge->>'measurement',pp.crochet_gauge->>'unit')", "0.3", true, false},
}

func distinctSearchTagTextSQL(column string) string {
	return `array_to_string(ARRAY(SELECT DISTINCT lower(tag) COLLATE "C" FROM unnest(coalesce(` + column + `,'{}'::text[])) tag WHERE tag IS NOT NULL ORDER BY lower(tag) COLLATE "C"),' ')`
}

// Raw project JSON is already retained by projection. This expression lets new
// additional crafts participate without duplicating or rewriting source data.
const projectCraftValuesSQL = `(ARRAY[lower(pp.common_craft_type)] || ARRAY(
 SELECT DISTINCT lower(craft->>'craftType')
 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(pp.raw_project#>'{common,additionalCrafts}')='array'
 THEN pp.raw_project#>'{common,additionalCrafts}' ELSE '[]'::jsonb END) craft
 WHERE jsonb_typeof(craft->'craftType')='string'))`

func submittedSearchFieldSQL(project bool) (string, string) {
	fields, weights := []string{}, []string{}
	for _, field := range submittedSearchFields {
		if field.Project && !project {
			continue
		}
		expression := field.SQL
		if field.Vocabulary {
			// Labels, aliases and ancestors come from the pinned application
			// catalog ($18), never arbitrary extension JSON or inferred data.
			expression = `array_to_string(ARRAY(SELECT term FROM unnest(string_to_array(coalesce(` + expression + `,''),' ')) value CROSS JOIN LATERAL jsonb_array_elements_text(coalesce($18::jsonb->lower(value),jsonb_build_array(value))) term),' ')`
		}
		fields = append(fields, expression)
		weights = append(weights, field.Weight)
	}
	// Alt text is weighted per image so phrases cannot span unrelated images.
	alts := `ARRAY(SELECT image->>'alt' FROM jsonb_array_elements(CASE WHEN jsonb_typeof(p.images)='array' THEN p.images ELSE '[]'::jsonb END) image WHERE jsonb_typeof(image->'alt')='string')`
	return `ARRAY[` + strings.Join(fields, ",") + `]::text[] || ` + alts,
		`ARRAY[` + strings.Join(weights, ",") + `]::real[] || array_fill(0.1::real,ARRAY[cardinality(` + alts + `)])`
}
