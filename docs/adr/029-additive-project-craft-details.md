# Additive project craft details and vocabulary

- Status: Accepted for implementation
- Date: 2026-11-20
- Aspect: Public project record contracts

## Context

CraftSky needs to capture multiple patterns for one project and tool sizes, techniques and methods that apply to a particular craft. It also needs specialist details for six additional crafts. Existing projects can already contain one pattern, scalar needle or hook sizes and shared design tags. Externally authored projects can use valid values outside the first-party catalog. [Project requirements](../changes/2026-11-20-project-craft-schema/01-requirements.md) · [Project evolution ADR](020-project-evolution-search-and-multiple-crafts.md)

## Simplified authoring decision

The schema preview now prioritizes what someone made and a small number of practice distinctions that can support a feed. This decision supersedes the original authoring vocabulary. The original decision describes the public record representations that existing projects can still contain.

Keep shared project identity, status, title, patterns, tags, and additional crafts. Use **Pattern name** for the `name` field and preserve its hashtag facets. Deprecate duration, materials, colors, and design tags. The post description and regular hashtags cover those details for new projects.

| Craft | Fields for first-party authoring |
|---|---|
| Knitting | Project type and subtype. Optional Hand, Machine, and Loom methods, which can be combined. |
| Crochet | Project type and subtype. Optional Standard crochet and Tunisian crochet methods, which can be combined. |
| Sewing | Project type and subtype. |
| Quilting | Project type and subtype. |
| Embroidery | Project type and subtype. Two independent optional single choices for embroidery. `method` suggests Hand and Machine. `style` suggests Cross-stitch and Needlepoint. |
| Felting | Needle and Wet felting methods, which can be combined. Project type and subtype describe the item independently, such as Accessory and Bag. |
| Punch Needle | Project type. |
| Macramé | Project type. |
| Weaving | Project type and subtype, plus Loom type. Keep the existing rigid heddle, multi-shaft, and frame loom identities. |
| Dyeing | Project type shows what was dyed. |
| Spinning | Shared project fields. |

Retire other specialist fields from first-party authoring and display. Show the same current fields on project cards and details for old and new posts. Preserve all existing field types, limits, and definitions in the public record schema. Deprecation is documented in the field description, and the schema explorer shows those fields when **Show legacy fields** is selected. Preserve legacy grouped embroidery methods and their styles without collapsing mixed selections into one value. The new flat choices use open suggestions.

Use optional `methods` lists for Crochet and Felting. A present list, including an empty list, takes precedence. On older records fall back to an explicit Tunisian crochet technique or the single Felting method. AppView also recognizes the older Dyeing material forms in its new item browse. The preview is the review point for the field inventory and suggested values before changing clients or AppView behavior.

The [approved item vocabulary](../changes/2026-11-20-project-craft-schema/project-type-proposal.md) governs catalogue suggestions. The open subtype string accepts one catalogue identifier or bounded user-authored text. First-party pickers offer new custom item names under Other. Preserve previous catalogue identities and saved names for existing records and directional browse.

## Original additive decision

Keep `social.craftsky.feed.post` with one optional project details variant for the primary craft. Keep `project.common.designTags` as shared open values and scope first-party suggestions in the application catalog. Keep the existing scalar `needleSizeMm` and `hookSizeMm` fields valid. Add optional `needleSizesMm` and `hookSizesMm` string lists, each limited to ten items. Each item preserves the corresponding scalar's bounds of 20 bytes and 20 graphemes. The size items and Knitting method IDs are string lists because the individual selections carry no associated detail. Keep absence and an explicit empty array distinguishable wherever an old scalar could coexist with a list.

Add optional `project.common.patterns`, an ordered list of up to five existing `#pattern` objects for any craft. Each entry keeps its name, link, difficulty, self-drafted flag, designer, publisher and associated rich-text facets. Keep the legacy `project.common.pattern` object valid. A present `patterns` list takes precedence, including an empty list. If absent, read the legacy object as one entry. New writes and explicit edits use the plural field and remove the legacy field from the same record. Retain untouched entries and their extensions across edits. During the transition, AppView exposes a legacy object only for an effective singleton, converts valid legacy singleton writes and protects multi-pattern records from destructive legacy updates.

Use optional object arrays for techniques and for Embroidery methods with styles. First-party Knitting authoring can independently select Hand and Machine. Machine-specific fields describe the machine used and remain separate from measured stitch gauge. Knitting technique subtypes and Embroidery styles belong to their selected parent. Open token references document suggested values and support future extension.

Phase 1 adds multiple patterns, multiple sizes, Knitting, Crochet and Sewing techniques, Machine Knitting details, Embroidery details and shared design tag suggestions. Phase 2 adds Spinning, Dyeing, Punch Needle, Macramé, Weaving and Felting details and craft tokens. Each new details variant is optional. Spinning and other crafts with useful measured quantities use named units and bounded decimal strings. Weaving keeps each material's role with the material, and Macramé keeps each diameter with its cord. Existing shared Materials covers equivalent optional free text.

## Field bounds

| Public fields | Limit and representation |
|---|---|
| Shared project `patterns` | Five entries in saved order, each with the existing `#pattern` definition and field limits. |
| Knitting, Crochet and Sewing `techniques` | 20 entries per list. Knitting `subtypes` allow ten per technique. |
| Knitting and Embroidery `methods`, Embroidery `styles` | Ten per list. Each Embroidery method has its own styles list. |
| Spinning drafting and plying techniques, Macramé cords and knots, Weaving materials, Felting needle gauges | Ten per list. |
| Tool sizes, `machineGaugeMm`, cord `diameterMm`, Felting needle `gauge` | 20 bytes and 20 graphemes per string. Machine gauge and cord diameter are decimal strings in millimetres. |
| Spinning `finishedYarn.mass` and `.length`, Punch Needle `loopHeight`, Weaving `sett` | Objects with required `amount` and `unit`. Amount is a decimal string limited to 20 bytes and 20 graphemes. Mass and length units have the same limit. Sett units allow up to 64 bytes and 64 graphemes, with `ends-per-inch` and `ends-per-centimetre` as known values. |
| Spinning `plyCount` and `finishedYarn.wrapsPerInch` | Integers with a minimum of one. |
| Technique, method, style and other catalog selections | 100 bytes and 100 graphemes per value. |
| Short free-text details and finished dimensions | 500 bytes and 100 graphemes. Weaving material descriptions allow 1,000 bytes and 100 graphemes, matching shared materials. |
| Dyeing `recipeNotes` | 6,000 bytes and 600 graphemes, matching Sewing fit notes. |

The Lexicon limits string length and documents decimal values and known units. The application validates decimal syntax and eligible units when it adds first-party authoring.

## Alternatives

- Replacing the single-pattern object or a size scalar would invalidate the existing record shape. Optional plural fields preserve the earlier data.
- A shared techniques field could simplify a picker but would carry different meanings and dependent groups across crafts. Craft-specific details retain explicit semantics.
- Separate collections would change post identities and feed participation. New variants in the existing union keep the established project contract.

## Consequences

Existing posts and Postgres projections require no bulk rewrite or schema migration. Existing pattern search keeps its field weights and considers each effective entry. Combined difficulty and self-drafted filters match the same entry. Pattern facets remain associated with their own text and share the existing post-level hashtag budget and notification rules. AppView and clients adopt the reviewed shapes in each release phase. Generated Go, CBOR and the embedded validation catalog follow the Lexicons. The maintainer confirmed the public fields in the local explorer before authorizing behaviour implementation. Publication follows the release process.
