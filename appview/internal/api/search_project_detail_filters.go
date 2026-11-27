package api

import (
	"strconv"
	"strings"
)

type craftDetailBrowseFilter struct {
	key   string
	craft string
	match func(string) string
}

// These are browse-only query parameters. Each entry points to the matching
// field in the primary craft's typed detail object.
var craftDetailBrowseFilters = []craftDetailBrowseFilter{
	// The parent/child pairs are matched together by projectDetailFilterSQL.
	{key: "technique"},
	{key: "techniqueSubtype", craft: "knitting"},
	{key: "embroideryMethod", craft: "embroidery"},
	{key: "embroideryStyle", craft: "embroidery"},
	{key: "crochetMethod", craft: "crochet", match: func(param string) string {
		return methodsArrayFilterSQL(crochetMethodsArraySQL, "crochet", param)
	}},
	detailScalarBrowseFilter("spinningTool", "spinning", true, "tool"),
	detailScalarBrowseFilter("fibrePreparation", "spinning", true, "fibrePreparation"),
	detailRowsBrowseFilter("draftingTechnique", "spinning", "draftingTechniques", "technique", true),
	detailRowsBrowseFilter("plyingTechnique", "spinning", "plyingTechniques", "technique", true),
	detailScalarBrowseFilter("finishedYarnMassUnit", "spinning", false, "finishedYarn", "mass", "unit"),
	detailScalarBrowseFilter("finishedYarnLengthUnit", "spinning", false, "finishedYarn", "length", "unit"),
	detailScalarBrowseFilter("materialForm", "dyeing", true, "materialForm"),
	detailScalarBrowseFilter("dyeType", "dyeing", true, "dyeType"),
	detailScalarBrowseFilter("applicationMethod", "dyeing", true, "applicationMethod"),
	detailScalarBrowseFilter("colorEffect", "dyeing", true, "colorEffect"),
	detailScalarBrowseFilter("loopHeightUnit", "punchNeedle", false, "loopHeight", "unit"),
	detailRowsBrowseFilter("cordConstruction", "macrame", "cords", "construction", true),
	detailRowsBrowseFilter("knot", "macrame", "knots", "knot", true),
	detailScalarBrowseFilter("loomType", "weaving", true, "loomType"),
	detailScalarBrowseFilter("weaveStructure", "weaving", true, "structure"),
	detailRowsBrowseFilter("materialRole", "weaving", "materials", "role", true),
	detailScalarBrowseFilter("settUnit", "weaving", false, "sett", "unit"),
	detailScalarBrowseFilter("feltingForm", "felting", true, "form"),
	detailRowsBrowseFilter("needleGauge", "felting", "needleGauges", "gauge", false),
	{key: "needleSizeMm", craft: "knitting", match: func(param string) string {
		return toolSizeFilterSQL("knitting", "needleSizeMm", "needleSizesMm", "pp.knitting_needle_size_mm", param)
	}},
	{key: "hookSizeMm", craft: "crochet", match: func(param string) string {
		return toolSizeFilterSQL("crochet", "hookSizeMm", "hookSizesMm", "pp.crochet_hook_size_mm", param)
	}},
	{key: "gaugeUnit", match: func(param string) string {
		return "(" + gaugeUnitFilterSQL("knitting", "pp.knitting_gauge", param) + " OR " +
			gaugeUnitFilterSQL("crochet", "pp.crochet_gauge", param) + ")"
	}},
}

func isAllowedCraftDetailBrowseFilter(key string) bool {
	for _, filter := range craftDetailBrowseFilters {
		if key == filter.key {
			return filter.craft == "" || IsSupportedProjectCraftType(craftTypePrefix+filter.craft)
		}
	}
	return false
}

func detailScalarBrowseFilter(key, craft string, token bool, path ...string) craftDetailBrowseFilter {
	return craftDetailBrowseFilter{key, craft, func(param string) string {
		object := detailObjectSQL(craft)
		for _, field := range path[:len(path)-1] {
			object += "->'" + field + "'"
		}
		return detailBrowseValueSQL(craft, jsonStringSQL(object, path[len(path)-1]), param, token)
	}}
}

func detailRowsBrowseFilter(key, craft, list, field string, token bool) craftDetailBrowseFilter {
	return craftDetailBrowseFilter{key, craft, func(param string) string {
		return "EXISTS (SELECT 1 FROM jsonb_array_elements(" +
			guardedArraySQL(detailObjectSQL(craft)+"->'"+list+"'") +
			") entry WHERE " + detailBrowseValueSQL(craft, jsonStringSQL("entry", field), param, token) + ")"
	}}
}

func detailBrowseValueSQL(craft, value, param string, token bool) string {
	if !token {
		return "lower(" + value + ") = ANY(" + param + "::text[])"
	}
	nsid := "social[.]craftsky[.]project[.]" + craft
	switch craft {
	case "knitting", "crochet", "sewing", "embroidery":
		nsid += "[.]defs"
	}
	return tokenFilterSQL(value, param, nsid)
}

func projectDetailFilterSQL(filters map[string][]string, args *[]any) string {
	bind := func(key string) string {
		values := filters[key]
		if len(values) == 0 {
			return ""
		}
		*args = append(*args, values)
		return "$" + strconv.Itoa(len(*args))
	}

	var clause strings.Builder
	and := func(predicate string) { clause.WriteString("\n AND " + predicate) }
	technique, subtype := bind("technique"), bind("techniqueSubtype")
	if subtype != "" {
		and(detailGroupFilterSQL("knitting", "techniques", "technique", technique, "subtypes", "subtype", subtype))
	} else if technique != "" {
		and("(" + strings.Join([]string{
			detailGroupFilterSQL("knitting", "techniques", "technique", technique, "", "", ""),
			detailGroupFilterSQL("crochet", "techniques", "technique", technique, "", "", ""),
			detailGroupFilterSQL("sewing", "techniques", "technique", technique, "", "", ""),
		}, " OR ") + ")")
	}
	method, style := bind("embroideryMethod"), bind("embroideryStyle")
	if method != "" || style != "" {
		and(embroideryBrowseFilterSQL(method, style))
	}
	for _, filter := range craftDetailBrowseFilters {
		if filter.match != nil {
			if param := bind(filter.key); param != "" {
				and(filter.match(param))
			}
		}
	}
	return clause.String()
}

func methodsArrayFilterSQL(array, craft, param string) string {
	return "EXISTS (SELECT 1 FROM jsonb_array_elements(" + array + ") method WHERE jsonb_typeof(method)='string' AND " +
		detailBrowseValueSQL(craft, "method#>>'{}'", param, true) + ")"
}

func embroideryBrowseFilterSQL(method, style string) string {
	details := detailObjectSQL("embroidery")
	legacy := func(methodParam, styleParam string) string {
		return detailGroupFilterSQL("embroidery", "methods", "method", methodParam, "styles", "style", styleParam)
	}
	field := func(key, param, fallback string) string {
		return "(CASE WHEN " + details + " ? '" + key + "' THEN " +
			detailBrowseValueSQL("embroidery", jsonStringSQL(details, key), param, true) + " ELSE " + fallback + " END)"
	}
	if method == "" {
		return field("style", style, legacy("", style))
	}
	if style == "" {
		return field("method", method, legacy(method, ""))
	}
	// Keep the group constraint for records written with the grouped schema.
	// Flat values supersede their old counterparts and are independent.
	return "(CASE WHEN " + details + " ?| ARRAY['method','style'] THEN " + field("method", method, legacy(method, "")) +
		" AND " + field("style", style, legacy("", style)) + " ELSE " + legacy(method, style) + " END)"
}

func detailGroupFilterSQL(craft, list, parent, parentParam, childList, child, childParam string) string {
	var conditions []string
	if parentParam != "" {
		conditions = append(conditions, detailBrowseValueSQL(craft, jsonStringSQL("entry", parent), parentParam, true))
	}
	if childParam != "" {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM jsonb_array_elements("+
			guardedArraySQL("entry->'"+childList+"'")+") item WHERE "+
			detailBrowseValueSQL(craft, jsonStringSQL("item", child), childParam, true)+")")
	}
	return "EXISTS (SELECT 1 FROM jsonb_array_elements(" +
		guardedArraySQL(detailObjectSQL(craft)+"->'"+list+"'") +
		") entry WHERE " + strings.Join(conditions, " AND ") + ")"
}

// Older projections can have the typed columns without the raw detail object.
// Their values still belong only to the primary craft and never override a
// stored details variant belonging to another craft.
func projectedDetailMatchesCraftSQL(craft string) string {
	return "(lower(pp.common_craft_type)='" + craftTypePrefix + craft + "' AND (pp.raw_details IS NULL OR " +
		detailObjectSQL(craft) + " <> '{}'::jsonb))"
}

func toolSizeFilterSQL(craft, singular, plural, legacyColumn, param string) string {
	details := detailObjectSQL(craft)
	return "(" + projectedDetailMatchesCraftSQL(craft) + " AND CASE WHEN " + details + " ? '" + plural +
		"' THEN EXISTS (SELECT 1 FROM jsonb_array_elements(" + guardedArraySQL(details+"->'"+plural+"'") +
		") size WHERE jsonb_typeof(size)='string' AND " + detailBrowseValueSQL(craft, "size#>>'{}'", param, false) +
		") ELSE " + detailBrowseValueSQL(craft, "coalesce("+jsonStringSQL(details, singular)+", "+legacyColumn+")", param, false) +
		" END)"
}

func gaugeUnitFilterSQL(craft, legacyColumn, param string) string {
	unit := "coalesce(" + jsonStringSQL(detailObjectSQL(craft)+"->'gauge'", "unit") + ", " +
		jsonStringSQL(legacyColumn, "unit") + ")"
	return "(" + projectedDetailMatchesCraftSQL(craft) + " AND " + detailBrowseValueSQL(craft, unit, param, false) + ")"
}
