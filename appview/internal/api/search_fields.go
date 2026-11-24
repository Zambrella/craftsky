package api

import (
	"strings"

	"social.craftsky/appview/internal/projectcatalog"
)

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
	{"project.common.materials.text", "array_to_string(pp.materials,' ')", "0.4", true, false},
	{"project.common.tags", distinctSearchTagTextSQL("pp.project_tags"), "0.2", true, false},
	{"project.common.craftType/additionalCrafts", "array_to_string(" + projectCraftValuesSQL + ",' ')", "0.4", true, true},
	{"project.common.status", "pp.common_status", "0.4", true, true},
	{"project.common.colors", "array_to_string(pp.colors,' ')", "0.4", true, true},
	{"project.common.designTags", "array_to_string(pp.design_tags,' ')", "0.2", true, true},
	{"project.details.projectType", projectItemTypeSQL, "0.4", true, true},
	{"project.details.projectSubtype", projectItemSubtypeSQL, "0.4", true, true},
	{"project.details.yarnWeight", "concat_ws(' ',pp.knitting_yarn_weight,pp.crochet_yarn_weight)", "0.4", true, true},
	{"project.details.needleSizeMm/needleSizesMm", effectiveToolSizesSQL("knitting", "needleSizeMm", "needleSizesMm", "pp.knitting_needle_size_mm"), "0.3", true, false},
	{"project.details.hookSizeMm/hookSizesMm", effectiveToolSizesSQL("crochet", "hookSizeMm", "hookSizesMm", "pp.crochet_hook_size_mm"), "0.3", true, false},
	{"project.details.finishedSize/size/sizeMade", "concat_ws(' ',pp.knitting_finished_size,pp.crochet_finished_size,pp.quilting_size,pp.sewing_size_made," + macrameFinishedSizeSQL + "," + weavingFinishedSizeSQL + ")", "0.3", true, false},
	{"project.details.fitNotes", "pp.sewing_fit_notes", "0.4", true, false},
	{"project.details.piecingTechnique", "pp.quilting_piecing_technique", "0.4", true, true},
	{"project.details.quiltingMethod", "pp.quilting_quilting_method", "0.4", true, true},
	{"project.details.gauge", "concat_ws(' ',pp.knitting_gauge->>'stitches',pp.knitting_gauge->>'rows',pp.knitting_gauge->>'measurement',pp.knitting_gauge->>'unit',pp.crochet_gauge->>'stitches',pp.crochet_gauge->>'rows',pp.crochet_gauge->>'measurement',pp.crochet_gauge->>'unit')", "0.3", true, false},
	{"project.details.techniques.technique", "concat_ws(' '," + detailObjectTermsSQL("knitting", "techniques", "technique") + "," + detailObjectTermsSQL("crochet", "techniques", "technique") + "," + detailObjectTermsSQL("sewing", "techniques", "technique") + ")", "0.4", true, true},
	{"project.details.techniques.subtypes.subtype", detailNestedTermsSQL("knitting", "techniques", "subtypes", "subtype"), "0.4", true, true},
	{"project.details.methods", "concat_ws(' '," + textArraySearchValueSQL(knittingMethodsArraySQL) + "," + textArraySearchValueSQL(crochetMethodsArraySQL) + "," + textArraySearchValueSQL(feltingMethodsArraySQL) + ")", "0.4", true, true},
	{"project.details.method", embroideryEffectiveFieldSQL("method", detailObjectTermsSQL("embroidery", "methods", "method")), "0.4", true, true},
	{"project.details.style", embroideryEffectiveFieldSQL("style", detailNestedTermsSQL("embroidery", "methods", "styles", "style")), "0.4", true, true},
	{"project.details.draftingTechniques.technique", detailObjectTermsSQL("spinning", "draftingTechniques", "technique"), "0.4", true, true},
	{"project.details.plyingTechniques.technique", detailObjectTermsSQL("spinning", "plyingTechniques", "technique"), "0.4", true, true},
	{"project.details.applicationMethod", detailStringSQL("dyeing", "applicationMethod"), "0.4", true, true},
	{"project.details.knots.knot", detailObjectTermsSQL("macrame", "knots", "knot"), "0.4", true, true},
	{"project.details.loomType", detailStringSQL("weaving", "loomType"), "0.4", true, true},
	{"project.details.structure", detailStringSQL("weaving", "structure"), "0.4", true, true},
	{"project.details.form", detailStringSQL("felting", "form"), "0.4", true, true},
}

// Keep each pattern's fields separate, just like individual image alt texts.
var patternSearchFields = []SearchFieldContract{
	{"project.common.patterns.name", jsonStringSQL("pattern", "name"), "1.0", true, false},
	{"project.common.patterns.designer", jsonStringSQL("pattern", "designer"), "0.7", true, false},
	{"project.common.patterns.publisher", jsonStringSQL("pattern", "publisher"), "0.7", true, false},
	{"project.common.patterns.difficulty", jsonStringSQL("pattern", "difficulty"), "0.4", true, true},
	{"project.common.patterns.selfDrafted", "CASE WHEN pattern->'selfDrafted'='true'::jsonb THEN 'self drafted' END", "0.4", true, false},
}

const rawPatternsSQL = `pp.raw_project#>'{common,patterns}'`

const effectivePatternsSQL = `(CASE
	WHEN ` + rawPatternsSQL + ` IS NOT NULL THEN
		CASE WHEN jsonb_typeof(` + rawPatternsSQL + `)='array' THEN ` + rawPatternsSQL + ` ELSE '[]'::jsonb END
	WHEN jsonb_typeof(pp.raw_project#>'{common,pattern}')='object' THEN jsonb_build_array(pp.raw_project#>'{common,pattern}')
	ELSE jsonb_build_array(jsonb_build_object('name',pp.pattern_name,'designer',pp.pattern_designer,'publisher',pp.pattern_publisher,'difficulty',pp.pattern_difficulty,'selfDrafted',pp.pattern_self_drafted))
END)`

var patternNamesSQL = `(SELECT string_agg(` + jsonStringSQL("pattern", "name") + `, ' ') FROM jsonb_array_elements(` + effectivePatternsSQL + `) pattern)`

func patternFilterSQL(difficultyParam, selfDraftedParam string) string {
	return `(cardinality(` + difficultyParam + `::text[]) = 0 AND NOT ` + selfDraftedParam + `::boolean OR EXISTS (
		SELECT 1 FROM jsonb_array_elements(` + effectivePatternsSQL + `) pattern
		WHERE (cardinality(` + difficultyParam + `::text[]) = 0 OR ` + tokenFilterSQL(jsonStringSQL("pattern", "difficulty"), difficultyParam, `social[.]craftsky[.]feed[.]defs`) + `)
		AND (NOT ` + selfDraftedParam + `::boolean OR pattern->'selfDrafted'='true'::jsonb)
	))`
}

func tokenFilterSQL(value, param, tokenNSID string) string {
	strip := func(expression string) string {
		return `regexp_replace(lower(` + expression + `), '^` + tokenNSID + `#', '')`
	}
	return strip(value) + ` = ANY(ARRAY(SELECT ` + strip("filter_value") + ` FROM unnest(` + param + `::text[]) filter_value))`
}

var (
	knittingMethodsArraySQL = guardedArraySQL(detailObjectSQL("knitting") + `->'methods'`)
	feltingMethodsArraySQL  = effectiveDetailMethodsSQL("felting", `jsonb_build_array(`+detailStringSQL("felting", "method")+`)`)
	crochetMethodsArraySQL  = effectiveDetailMethodsSQL("crochet", `(SELECT coalesce(jsonb_agg(entry->'technique'),'[]'::jsonb)
		FROM jsonb_array_elements(`+guardedArraySQL(detailObjectSQL("crochet")+`->'techniques'`)+`) entry
		WHERE regexp_replace(lower(`+jsonStringSQL("entry", "technique")+`), '^social[.]craftsky[.]project[.]crochet[.]defs#', '') = 'tunisiancrochet')`)
	projectItemTypeSQL     = projectItemValueSQL("projectType", "_project_type")
	projectItemSubtypeSQL  = projectItemValueSQL("projectSubtype", "_project_subtype")
	macrameFinishedSizeSQL = detailStringSQL("macrame", "finishedSize")
	weavingFinishedSizeSQL = detailStringSQL("weaving", "finishedSize")
)

func effectiveDetailMethodsSQL(craft, fallback string) string {
	details := detailObjectSQL(craft)
	return `(CASE WHEN ` + details + ` ? 'methods' THEN ` + guardedArraySQL(details+`->'methods'`) + ` ELSE ` + fallback + ` END)`
}

func embroideryEffectiveFieldSQL(key, legacy string) string {
	details := detailObjectSQL("embroidery")
	return `(CASE WHEN ` + details + ` ? '` + key + `' THEN ` + jsonStringSQL(details, key) + ` ELSE ` + legacy + ` END)`
}

func projectItemValueSQL(field, projectedSuffix string) string {
	var sql strings.Builder
	sql.WriteString("(CASE")
	for _, craft := range []string{"knitting", "crochet", "sewing", "quilting", "embroidery", "punchNeedle", "macrame", "weaving", "felting", "dyeing"} {
		matchesCraft := "lower(pp.common_craft_type) = '" + strings.ToLower(craftTypePrefix+craft) + "'"
		value := detailStringSQL(craft, field)
		switch craft {
		case "knitting", "crochet", "sewing", "quilting":
			// Earlier projections can carry a literal craft with only the
			// typed columns. Raw details still require the canonical variant.
			matchesCraft += " OR (pp.raw_details IS NULL AND lower(pp.common_craft_type) = '" + craft + "')"
			value = "CASE WHEN pp.raw_details IS NULL THEN pp." + craft + projectedSuffix + " ELSE " + value + " END"
		case "dyeing":
			if field == "projectType" {
				legacy := `regexp_replace(` + detailStringSQL(craft, "materialForm") + `, '^([[:alnum:]]+)$', 'social.craftsky.project.dyeing#\1')`
				value = "CASE WHEN " + detailObjectSQL(craft) + " ? 'projectType' THEN " + value +
					" ELSE " + legacy + " END"
			}
		}
		sql.WriteString(" WHEN " + matchesCraft + " THEN " + value)
	}
	sql.WriteString(" END)")
	return sql.String()
}

func projectItemFilterSQL(param, field string) string {
	match := func(value string) string {
		return "nullif(lower(" + value + "),'') = ANY(" + param + "::text[])"
	}
	filter := match(projectItemSubtypeSQL)
	if field == "projectType" {
		// A known historical item can also match its current parent while the
		// original saved parent still matches directly.
		filter = match(projectItemTypeSQL) + " OR (" + projectItemTypeSQL + " IS NOT NULL AND " + filter + ")"
	}
	return "(cardinality(" + param + "::text[]) = 0 OR " + filter + ")"
}

func detailObjectSQL(craft string) string {
	entry, ok := projectcatalog.Craft(craftTypePrefix + craft)
	if !ok {
		panic("missing project search taxonomy for " + craft)
	}
	return `(CASE WHEN lower(pp.common_craft_type)='` + strings.ToLower(entry.ID) + `'
		AND pp.raw_details->>'$type'='` + entry.DetailsType + `' THEN pp.raw_details ELSE '{}'::jsonb END)`
}

func guardedArraySQL(value string) string {
	return `(CASE WHEN jsonb_typeof(` + value + `)='array' THEN ` + value + ` ELSE '[]'::jsonb END)`
}

func jsonStringSQL(object, field string) string {
	return `CASE WHEN jsonb_typeof(` + object + `->'` + field + `')='string' THEN ` + object + `->>'` + field + `' END`
}

func detailStringSQL(craft, field string) string {
	return jsonStringSQL(detailObjectSQL(craft), field)
}

func textArraySearchValueSQL(array string) string {
	return `(SELECT string_agg(value#>>'{}', ' ') FROM jsonb_array_elements(` + array + `) value WHERE jsonb_typeof(value)='string')`
}

func effectiveToolSizesSQL(craft, singular, plural, legacyColumn string) string {
	details := detailObjectSQL(craft)
	return `CASE WHEN ` + details + ` ? '` + plural + `' THEN ` + textArraySearchValueSQL(guardedArraySQL(details+`->'`+plural+`'`)) + ` ELSE coalesce(` + jsonStringSQL(details, singular) + `,` + legacyColumn + `) END`
}

func detailObjectTermsSQL(craft, array, field string) string {
	return `(SELECT string_agg(` + jsonStringSQL("entry", field) + `, ' ') FROM jsonb_array_elements(` + guardedArraySQL(detailObjectSQL(craft)+`->'`+array+`'`) + `) entry)`
}

func detailNestedTermsSQL(craft, array, items, field string) string {
	return `(SELECT string_agg(` + jsonStringSQL("item", field) + `, ' ') FROM jsonb_array_elements(` + guardedArraySQL(detailObjectSQL(craft)+`->'`+array+`'`) + `) entry CROSS JOIN LATERAL jsonb_array_elements(` + guardedArraySQL(`entry->'`+items+`'`) + `) item)`
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
			expression = vocabularySearchValueSQL(expression)
		}
		fields = append(fields, expression)
		weights = append(weights, field.Weight)
	}
	fieldArray := `ARRAY[` + strings.Join(fields, ",") + `]::text[]`
	weightArray := `ARRAY[` + strings.Join(weights, ",") + `]::real[]`
	if project {
		for _, field := range patternSearchFields {
			expression := field.SQL
			if field.Vocabulary {
				expression = vocabularySearchValueSQL(expression)
			}
			fieldArray += ` || ARRAY(SELECT ` + expression + ` FROM jsonb_array_elements(` + effectivePatternsSQL + `) pattern)`
			weightArray += ` || array_fill(` + field.Weight + `::real,ARRAY[jsonb_array_length(` + effectivePatternsSQL + `)])`
		}
	}
	// Alt text is weighted per image so phrases cannot span unrelated images.
	alts := `ARRAY(SELECT image->>'alt' FROM jsonb_array_elements(CASE WHEN jsonb_typeof(p.images)='array' THEN p.images ELSE '[]'::jsonb END) image WHERE jsonb_typeof(image->'alt')='string')`
	return fieldArray + ` || ` + alts,
		weightArray + ` || array_fill(0.1::real,ARRAY[cardinality(` + alts + `)])`
}

func vocabularySearchValueSQL(expression string) string {
	return `array_to_string(ARRAY(SELECT term FROM unnest(string_to_array(coalesce(` + expression + `,''),' ')) value CROSS JOIN LATERAL jsonb_array_elements_text(coalesce($18::jsonb->lower(value),jsonb_build_array(value))) term),' ')`
}
