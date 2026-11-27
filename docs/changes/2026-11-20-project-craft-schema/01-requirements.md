# Project craft schema requirements

## Approved simplification

The maintainer approved the [item taxonomy and custom entry proposal](project-type-proposal.md), the revised craft fields, and the sequence of schema preview, documentation, and implementation. The requirements in this section govern new authoring. The original scope below records how the older records and fields were introduced.

| ID | Requirement |
|---|---|
| SIM-001 | Offer the approved project type and subtype choices by primary craft. Use the same catalogue in both clients and the schema preview. Use short flat type lists for Punch Needle, Macramé, and Dyeing. Spinning uses shared project fields. |
| SIM-002 | Make type and subtype optional. Use listed subtypes under named project types and offer a bounded custom item name only under Other. Preserve previously saved custom values under named types on unrelated edits. Capitalize project composer and craft filter choices consistently and alphabetize choices and visible aliases. Put Other last wherever offered. Order Primary craft choices like the Home tabs, with the author's profile interests first in saved order. |
| SIM-003 | Keep up to five pattern entries with their text, links, and facets across immediate posts, drafts, scheduling, and published edits. Retain hashtag input in the pattern name field. |
| SIM-004 | Retire Materials and style, Duration, Design tags, and the superseded craft-specific fields from first-party authoring, filters, and public project displays. Use the same displayed fields for old and new posts. Keep legacy data and annotations in the record and preserve them on unrelated edits. |
| SIM-005 | Keep optional Hand, Machine, and Loom in Knitting. Offer Standard and Tunisian crochet. Offer Needle and Wet felting. Allow mixed method choices in these crafts. Offer independent optional single Method and Style choices in Embroidery. Keep Loom type in Weaving. |
| SIM-006 | Include compatible older values and narrower known item choices in broad browse filters. Custom item text is keyword-searchable, and the selected type governs the type filter. Dropdowns offer catalogue choices. Keep craft eligibility based on primary craft. |
| SIM-007 | Retain public record identity and existing shapes. A current list takes precedence over a legacy scalar or object wherever the schema defines a fallback. Use older literal selections for discovery where they express the same intent. Preserve unknown valid extensions. No bulk record rewrite or database migration is required. |
| SIM-008 | Keep existing craft availability, profile interests, onboarding, craft feeds, and grouped hashtag discovery across both release phases. |
| SIM-009 | Put Additional crafts last in Craft details. Require a Status in first-party composers before publishing, saving a published edit, or saving or publishing a schedule. Give new projects the existing Finished default and let authors change their status using the picker. Older records without a status require an explicit selection before those writes. Private drafts may be saved with an empty status. The public Lexicon keeps the field optional. |
| SIM-010 | Show Pattern details and Pattern name by default. For a primary craft of Spinning or Dyeing, show the section only when an additional craft uses patterns or at least one pattern has content. For Weaving, use Pattern or draft and Pattern or draft name. For Felting, use Pattern or template and Pattern or template name. Keep the shared pattern data model. |
| SIM-011 | In Craft details, put the Project type picker and Other name field before the craft-specific fields and Additional crafts. Show the selected item labels on the Craft details row in the main composer, with a middle dot between type and subtype. |
| SIM-012 | Use one searchable Project type picker in both composers and craft filters. In a picker with subtypes, give all top-level labels more visual weight, indent subtype rows, and leave a small gap between groups. A search for a type shows its whole branch. A search for a subtype or its alias shows the matching entry with its parent. The composer saves one choice. The filter can select multiple types and subtypes and includes projects matching any of them. Applying an existing filter link preserves its selection behavior until the Project type selection changes. |

The public vision lists structured materials among its original discussion prompts. This revision follows the maintainer's later direction to put materials in the searchable description. It retains data portability, pattern discovery, and chronological feeds. [Public vision](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit?usp=sharing)

## Earlier requirements and decisions

## 1. Initial request

Make project options more relevant to each craft, allow multiple knitting needle and crochet hook sizes, and capture techniques and subtypes from the craft review workbook. Follow-up discussion added sewing techniques, coverage for Spinning, Dyeing, Machine Knitting, Punch Needle, Macramé, Weaving and Needle Felting, and up to five patterns per project. Include the new primary crafts in profile interests and onboarding. The experience applies to Flutter on iOS and Android and to the Svelte web client. [Craft schema review workbook](https://docs.google.com/spreadsheets/d/1k0Ymrb7FPNLhdVd53A_zy-i4kBXnGxmN/edit?gid=1470412807)

## 2. Current codebase findings

- The composer offers five primary crafts. Sewing, Knitting, Crochet and Quilting have typed details. Embroidery uses shared fields today. Spinning, Weaving, Felting and Macramé have existing profile compatibility IDs. [Project lexicon](../../../lexicon/social/craftsky/project/defs.json) · [Profile craft catalog](../../../app/lib/profile/data/crafts_catalog.dart)
- The primary craft owns the optional details object and determines craft-feed eligibility. Additional crafts participate in keyword and tag discovery. AppView indexes an explicit set of public fields and retains the raw project data. [Project craft ADR](../../adr/020-project-evolution-search-and-multiple-crafts.md) · [Searchable-field contract](../../development/search-fields.md)
- Both composers have searchable multi-selects. The size fields are currently single strings. Draft restoration, scheduling and published editing have separate entry points that need coverage. [Flutter draft adapter](../../../app/lib/projects/composer/project_draft_snapshot_adapter.dart) · [Web project editor](../../../web-client/src/lib/features/composer/ProjectEditor.svelte) · [AppView edit merge](../../../appview/internal/api/post_edit_merge.go)
- The shared pattern field holds one structured object, including rich-text facets. AppView supports pattern keyword search, hashtag discovery, and existing difficulty and self-drafted filters. [Project Lexicon](../../../lexicon/social/craftsky/project/defs.json) · [Searchable-field contract](../../development/search-fields.md)
- AppView's versioned catalog already supplies stable values, labels, aliases and parent relationships. Flutter consumes generated data, and web currently has separate catalog inputs. [Project catalog](../../../appview/internal/projectcatalog/catalog-v1.json) · [Catalog generator](../../../scripts/project-catalog)
- Both Home implementations offer supported craft feeds and prioritise saved interests. Flutter has an explicit five-craft profile and onboarding picker and an explicit grouped hashtag list. Web sources its profile interests from its social craft catalog and relies on AppView for default hashtag groups. [Flutter Home](../../../app/lib/home/pages/home_page.dart) · [Flutter hashtag groups](../../../app/lib/search/providers/blank_search_provider.dart) · [Web craft feeds](../../../web-client/src/lib/features/social/craft-feeds.ts) · [Web profile editor](../../../web-client/src/lib/features/social/ProfileEditor.svelte) · [AppView craft catalogue](../../../appview/internal/api/craft_type.go)
- Verification entry points for later stages include `just lexgen-check`, `just appview-check`, `just app-test` and the web client's `check`, `test` and authenticated Playwright targets. [Task recipes](../../../justfile) · [Web scripts](../../../web-client/package.json)

## 3. Clarifying questions and decisions

| Question | Decision and status |
|---|---|
| How should multiple tool sizes be stored? | Confirmed by the maintainer. Use `needleSizesMm` and `hookSizesMm` as lists of strings such as `["4.0mm", "4.5mm"]`. |
| How should plaid and print matching be represented? | Confirmed by the maintainer. Plaid describes appearance and Print matching describes a sewing technique. |
| How should design tags be scoped? | Q1. Retain the shared field, apply the workbook's craft-specific suggestions, and keep saved selections after a craft change. |
| Which additions are crafts or methods? | Q2. Add six primary crafts. Machine Knitting is a Knitting method and Needle Felting is a Felting method. |
| How does an author record hand and machine knitting? | Confirmed in coding plan review. Knitting methods offers separate optional Hand and Machine selections. The record stores the selected method IDs in `methods`. A project with neither selected has an unspecified method. |
| Where do the new crafts appear? | Q3 and Q12. Include project pickers, profile interests, onboarding, Home craft feeds and grouped hashtag discovery. Keep the existing Home preference ordering. |
| What can ship independently? | Q4 and Q13. Release the existing-craft changes first. Release the six new crafts and their associated surfaces together in the second phase. |
| What belongs under Techniques? | Q5, Q14 and Q16. Use Techniques for Knitting, Crochet and Sewing. Use catalog selections for new tags, techniques and styles. Preserve both uses of Granny square. Amigurumi stays an item subtype. |
| Can a Knitting technique have several subtypes? | Q6. An author can choose zero or more subtypes for each selected technique. |
| Can Embroidery combine methods? | Q7 and Q15. Allow Hand, Machine or both, with separate style selections for each method. Keep single- and multi-needle equipment choices out of Styles and defer an equipment field. |
| What can people dye? | Q8 and Q11. Use **Dyeing** for projects using spun yarn or unspun fibre, including fleece and roving. |
| How do readers find these projects? | Q9. Add keyword discovery for techniques and styles and dedicated filters for Machine Knitting and Needle Felting within their parent crafts. |
| How does size compatibility work? | Q10 and Q18. Read scalar sizes as one selection and convert scalar writes from older clients to single-item lists on new publications and published size edits. Preserve existing public records and support older clients through the transition. |
| How should multiple patterns be stored? | The maintainer requested up to five per project with the same migration approach as tool sizes. Add a shared `patterns` list of existing pattern objects in Phase 1 and retain the legacy `pattern` field. |
| Does scheduling need legacy compatibility? | Scheduling is prelaunch. Support the new fields through the current scheduling flow. Existing prelaunch schedule payloads and older scheduling clients are outside the compatibility scope. |
| What is needed in the expanded craft editors? | Q17. Include the optional starter fields in §11. Each craft can publish using shared fields alone. |

## 4. Candidate approaches

| Approach | Benefits | Trade-offs |
|---|---|---|
| Shared design tags with craft-specific suggestions and typed details | Preserves existing values and cross-craft discovery. Adds fields where their meaning and units are specific to the craft. | Requires catalog parity and handling for legacy and new sizes and patterns across both clients and AppView. |
| Craft-specific design-tag fields and separate craft identities for every method | Gives every craft or method an independent set of rules and its own feed identity. | Duplicates concepts such as Floral, complicates search across crafts and separates projects such as hand- and machine-knit cardigans. |
| A single shared techniques field | Gives the editors one vocabulary structure. | Method, equipment and project form still need craft-specific rules. Shared labels can conceal different meanings, such as mosaic knitting and mosaic crochet. |

## 5. Recommended direction

Keep the shared project envelope and design-tag values. Extend the optional craft details and use the catalog for picker choices, aliases and relationships. Introduce additive size and pattern lists and craft variants so public records remain readable through the rollout. Use project type and subtype for the item made, method for how the overall craft was practised, and techniques for selections within that work.

## 6. Problem and opportunity

Current project fields cannot describe common cases such as a sweater knitted with two needle sizes, a crochet blanket assembled from granny squares, an independently spun skein, or a garment made using two patterns. Adding a small set of relevant fields makes these projects easier to share and find while keeping ordinary project posting quick.

## 7. Goals

- **G-001** Give makers relevant suggestions and enough detail to describe their work.
- **G-002** Let readers discover projects by craft and by the techniques and methods described here.
- **G-003** Preserve saved project information and public post identity through the change.

## 8. Non-goals

- **NG-001** Bulk rewriting PDS records, changing post collections or requiring authors to re-enter old information.
- **NG-002** Multiple craft-detail objects on one post, reclassifying existing posts or guessing missing methods from other fields.
- **NG-003** A complete textile taxonomy, full dye recipe management, a redesign of profile interests or onboarding, or new Quilting technique fields.
- **NG-004** A new search surface, release execution or production changes in this requirements stage.
- **NG-005** First-party free-text creation and generic Other entries for design tags, techniques or styles, Embroidery equipment fields, and Dyeing support for fabric or finished garments in this release.
- **NG-006** Compatibility with prelaunch schedule payloads or older scheduling clients.

## 9. Users and actors

| Actor | Need |
|---|---|
| Project author | Describe the project with relevant defaults, save and reopen it, and clear or change selections intentionally. |
| Reader | See the details that were recorded and find relevant projects without learning stored tokens. |
| New or returning member | Select the new crafts during onboarding or when editing interests and reach their feeds. |
| Craft reviewer | Check labels, aliases and groupings in language familiar to makers. |
| Federated reader or author | Keep existing valid records readable and preserve valid metadata outside the first-party catalog. |

## 10. Current behavior

Design suggestions use the shared vocabulary. Knitting and Crochet each store one tool size. Projects store one optional shared pattern. Existing project types and subtypes classify the item made. Quilting already has separate piecing and quilting method fields. The six proposed primary crafts have no first-party project picker entries or detail editors.

## 11. Desired behavior and starter vocabulary

### Patterns

Any project can have up to five patterns. Authors can add, edit and remove individual entries in either client. Keep their order and display each entry with its own name, designer, publisher, link, difficulty and self-drafted status where supplied. Keep rich-text annotations associated with the relevant text in each entry.

Search the existing eligible fields of every effective pattern. Continue to use visible pattern text for keyword matching and pattern tag facets for hashtag discovery. Apply the existing whole-project hashtag budget of ten distinct tags, processing each pattern's name, designer and publisher facets in list order before materials. Keep existing eligibility and deduplication for mention notifications across all entries. Preserve the existing pattern difficulty and self-drafted filters. When both filters apply, one pattern must satisfy both. [Pattern metadata facets ADR](../../adr/006-pattern-metadata-facets.md)

### Design and style tags

Use the workbook’s craft lists as the starting suggestions for the five existing crafts. Sewing keeps all 26 shared options. Knitting and Crochet each suggest 21 with different omissions. Quilting suggests 19 including **Book/Film Characters**, and Embroidery suggests 17 including **Quotes**. Stored tags retain their shared meanings and identity. Use the shared suggestions as a starting fallback for a new craft without its own reviewed list. [Craft schema review workbook](https://docs.google.com/spreadsheets/d/1k0Ymrb7FPNLhdVd53A_zy-i4kBXnGxmN/edit?gid=1470412807)

| Craft | Changes to the shared suggestion list in the workbook |
|---|---|
| Sewing | All shared tags. |
| Knitting | Omit Boho, Cottagecore, Gothic, Nautical and Sporty. |
| Crochet | Omit Boho, Gothic, Nautical, Plaid and Sporty. |
| Quilting | Omit Boho, Checkerboard, Colour block, Cottagecore, Nautical, Plaid, Sporty and Stripes. Add Book/Film Characters. |
| Embroidery | Omit Boho, Checkerboard, Colour block, Cottagecore, Maximalist, Minimalist, Nautical, Plaid, Sporty and Stripes. Add Quotes. |

### Existing craft details

| Craft | Draft launch vocabulary and behavior |
|---|---|
| Knitting | **Techniques** supports multiple selections. Start with Brioche, Cables, Colourwork, Entrelac, Lace and Modular knitting. Colourwork offers optional subtypes Corrugated ribbing, Double knitting, Fair Isle, Illusion, Intarsia, Ladderback jacquard, Mosaic and Stranded. An author can select Colourwork alone or with any combination of those subtypes. Subtypes can be added or cleared independently. Keep Stripes as a design tag and leave Stashbuster, generic Textured and Other out of the starter technique catalog. |
| Crochet | **Techniques** supports multiple selections. Start with Filet crochet, Freeform crochet, Front/back post stitch, Granny square, Irish crochet, Mosaic crochet, Tunisian crochet (alias Afghan crochet), Pineapple crochet, Slip stitch crochet, Surface crochet and Tapestry crochet. Keep the existing Granny square and Amigurumi project subtypes. A Granny square technique selection can describe construction of a different item such as a blanket. |
| Sewing | **Techniques** supports multiple selections with picker headings. **Seams and edges** includes French seams, flat-felled seams, Hong Kong seam finish, bias binding and rolled hems. **Fastenings** includes invisible zip insertion, fly-front zip insertion and bound buttonholes. **Shaping and texture** includes darts, gathering, pleating, shirring and smocking. **Construction and details** includes bagging out a lining, inseam pockets, welt pockets, topstitching and print matching. The headings organise the picker. Plaid is a design tag. |
| Embroidery | Add optional Embroidery details with **Embroidery methods** (Hand, Machine or both), each with optional **Styles**. Hand choices include Bargello, Blackwork, Crewelwork, Cross stitch, Goldwork, Sashiko, Silk shading, Stumpwork and Whitework. Machine choices include 3D puff, Computerised, Free-motion and Trapunto. Keep styles associated with their selected method. Single-needle and Multi-needle describe equipment and are outside this release. |
| Quilting | Apply the craft-specific design-tag suggestions. Retain the existing project type, subtype, piecing technique and quilting method. |

### Additional crafts and methods

All listed details are optional. Reuse shared fields where they express the same information. The existing searchable Materials field can capture fibre, yarn and thread as text. Preserve material associations that affect meaning, such as each cord's diameter or a material's warp or weft role. The examples seed the launch catalog.

| Craft or method | Draft launch fields |
|---|---|
| **Spinning** | Tool (spindle, wheel, e-spinner), fibre preparation (rolag, batt, combed top), drafting and plying techniques, ply count, and finished yarn yield (mass and length) and wraps per inch. |
| **Dyeing** | What was dyed (spun yarn, fleece, roving or other unspun fibre), base fibre, dye type (acid, fibre-reactive, natural), application method (kettle-dyed, hand-painted, dip-dyed), colour effect (semi-solid, speckled, self-striping), and recipe notes. |
| **Machine Knitting** | **Knitting methods** offers separate Hand and Machine checkboxes. Authors can select either, both or neither. Selecting Machine exposes optional machine model, machine gauge (needle spacing), and machine techniques such as tuck stitch, slip stitch and double-bed jacquard. Yarn weight, actual stitch and row gauge, project type and finished size still apply. Needle sizes are relevant to hand work. |
| **Punch Needle** | Project type (rug, wall hanging, cushion, hoop art), foundation fabric, needle model or size, yarn or thread, and loop height. One craft supports both rug-scale work and fine embroidery. |
| **Macramé** | Cord material, construction and diameter per cord where multiple cords are used, knots used (square knot, lark’s head, double half hitch), and finished dimensions. |
| **Weaving** | Loom type, weave structure (plain weave, twill, doubleweave), materials identified by warp and weft role, sett with unit (warp ends per inch or centimetre), and finished dimensions. |
| **Needle Felting** | **Felting** is the primary craft and **Needle** is a visible method. Offer project form (flat or sculptural), fibre, multiple needle gauges and optional core or armature details. Reserve room for other methods, such as wet felting, as the catalogue grows. |

### Release phases

1. **Existing crafts**. Release the scoped design suggestions, needle and hook size lists, shared pattern lists, Knitting, Crochet and Sewing techniques, Embroidery methods and styles, and Machine Knitting. Include legacy-client compatibility for sizes and patterns and corresponding discovery support when each field ships.
2. **Craft expansion**. Release Spinning, Dyeing, Punch Needle, Macramé, Weaving and Felting with their optional editors, project views, profile interests, onboarding, Home feeds and grouped hashtag discovery. Include Needle Felting within Felting.

## 12. Requirements

Priorities describe the reviewed product scope. §21 records details for schema design and rollout verification.

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Let authors describe the craft, item made and relevant methods or techniques independently. | Readers can find comparable projects and see how they were made. | Initial prompt and follow-ups | AC-004, AC-006, AC-007, AC-008 |
| BR-002 | Business | Must | Let authors add and clear multiple needle sizes in Knitting and multiple hook sizes in Crochet. | Real projects use more than one tool size. | Initial prompt and confirmed string lists | AC-002, AC-003 |
| BR-003 | Business | Must | Let authors credit up to five patterns on any project. | A project can combine patterns and readers need each pattern's details. | Maintainer follow-up | AC-019 |
| FR-001 | Functional | Must | Keep `project.common.designTags` and scope picker suggestions in both composers and craft-specific project filters to the selected primary craft. Keep saved selections visible and removable even when outside suggestions. | Distinguishes relevance from the meaning of the stored tag. | Workbook and discovery | AC-001 |
| FR-002 | Functional | Must | Use searchable multi-selects and write optional bounded `needleSizesMm` and `hookSizesMm` string lists. Reuse canonical metric strings and existing aliases and display all effective sizes. | Selection is quick and records remain portable. | Maintainer decision | AC-002 |
| FR-003 | Functional | Must | Read a valid legacy scalar as one selection when the list is absent. A present list takes precedence, including an explicit empty list. An explicit first-party size edit or clear removes the old scalar in the same saved change. | Old sizes remain readable and never reappear after an intentional clear. | Migration discovery | AC-003, AC-010 |
| FR-004 | Functional | Must | Provide optional multi-select Techniques for Knitting, Crochet and Sewing. Allow multiple optional subtypes per selected Knitting technique where defined, and provide picker group headings for Sewing. Show saved techniques and subtypes on project details. | Separates construction from the made item and its appearance. | Q5, Q6 and Q16 | AC-004, AC-005 |
| FR-005 | Functional | Must | Allow Hand, Machine or both as optional Embroidery methods and attach each style selection to its method. Display each method and its saved styles. | A single project can combine embroidery methods. | Q7 and Q15 | AC-006 |
| FR-006 | Functional | Must | Offer Spinning, Dyeing, Punch Needle, Macramé, Weaving and Felting as primary and additional craft choices in project authoring. For each primary craft, authors can record, reopen and display the optional details in §11. Reuse shared fields for equivalent details and preserve the material relationships described in §11. | Supports posting about these crafts without requiring duplicate entry. | Q2, Q11, Q17 and simplification review | AC-007, AC-011 |
| FR-007 | Functional | Must | Offer Hand and Machine as independent optional Knitting selections and Needle as a Felting method. Show the corresponding specialist fields when applicable and display saved selections. The Machine filter matches Knitting projects that include Machine, including those that also include Hand. The Needle filter matches Felting projects that use Needle. | People can record mixed-method work and find projects within their primary craft. | Craft expansion discovery and coding plan review | AC-008 |
| FR-008 | Functional | Must | Make new sizes, techniques, technique subtypes, embroidery styles and craft methods discoverable in project keyword search through the explicit search contract. Keep relevant existing design, project type and subtype search. Apply the existing pattern keyword, filter, hashtag and mention rules in §11 to the effective pattern list. | Readers can find a technique or any pattern used in a project. | Discovery and established search contract | AC-009, AC-021 |
| FR-009 | Functional | Must | Support the revised fields through immediate publishing, draft save and reopen, scheduled save, reopen and execution in the current scheduling flow, published owner edits and project detail display in Flutter and Svelte. | Metadata is useful only when it survives normal authoring paths. | Initial UI request and scheduling scope decision | AC-002, AC-010, AC-019 |
| FR-010 | Functional | Must | Legacy clients can publish, reopen, change and clear a single size. Store a valid legacy scalar as a canonical single-item list on publication or an explicit published size edit. Preserve all stored sizes on unrelated edits and reject a legacy size edit that would discard multiple values. | Older editors can represent one size during the transition. | Q18 and simplification review | AC-015 |
| FR-011 | Functional | Must | Include the six new primary crafts in profile interests, onboarding, Home feeds and grouped hashtag discovery on both clients. Preserve preference ordering and make all craft feeds reachable to people with any interests. | The new crafts are usable throughout the normal experience. | Q3 and Q12 | AC-016 |
| FR-012 | Functional | Must | Store up to five existing pattern objects in an optional ordered `project.common.patterns` list. Edit and display entries independently in both clients. A present list, including an empty list, takes precedence. When absent, read legacy `pattern` as one entry. Preserve the stored representation on an unrelated edit and remove `pattern` on an explicit replacement or clear. | Makes the pattern list additive and keeps edits from resurrecting old metadata. | Maintainer follow-up | AC-019, AC-020 |
| FR-013 | Functional | Must | Legacy clients can publish, reopen, change and clear a single pattern. Store a valid legacy write as a single-item list and expose the old field on legacy reads for an effective singleton. Preserve stored patterns on unrelated or ambiguous saves and reject an explicit legacy edit that would discard multiple entries. | The previous editor can represent one pattern. | Maintainer follow-up and legacy size approach | AC-020 |
| NFR-001 | Non-functional | Must | For equivalent picker contexts, Flutter and Svelte offer the same eligible choices, aliases and group relationships and store the same values. AppView resolves those values and their agreed aliases and parent relationships in applicable searches and filters. | Aligns classification and discovery across both clients. | Catalog discovery and simplification review | AC-012 |
| NFR-002 | Non-functional | Must | Preserve existing public records and post identities without bulk PDS rewrites. Deploy AppView support for both representations of sizes and patterns before enabling clients to publish the new fields. | Keeps existing public records readable throughout rollout. | Architecture and migration discovery | AC-010, AC-013 |
| NFR-003 | Non-functional | Must | Make the first phase in §11 independently releasable, including AppView and client support for its fields. | Existing-craft improvements can ship while the craft expansion is being built. | Q4 and Q13 | AC-017 |
| RULE-001 | Business rule | Must | The primary craft selects at most one optional details variant and controls craft-feed eligibility. Additional crafts participate in existing keyword and tag discovery. | Supports mixed-craft projects within the current public contract. | Project craft ADR | AC-011 |
| RULE-002 | Business rule | Must | Use open, bounded vocabularies and stable values. Preserve valid unknown values and extension data through reads and unrelated edits. Suggestions and method-dependent choices guide first-party authoring. | Independent clients and older projects can carry values outside current suggestions. | Project craft ADR and Lexicon evolution | AC-001, AC-003, AC-006, AC-013 |
| RULE-003 | Business rule | Must | Apply existing owner authorization, conditional publish/edit behavior and visibility rules to all new fields. | New fields must follow the same trust and publication boundaries. | Architecture and published editing ADR | AC-014 |
| RULE-004 | Business rule | Must | Limit new first-party design tags, techniques and styles to catalog choices. Match aliases in the picker and preserve valid previously saved values outside the catalog. | Makers can find familiar terms and keep their existing records. | Q14 | AC-018 |

## 13. Acceptance criteria

| ID | Requirement IDs | Acceptance criterion |
|---|---|---|
| AC-001 | FR-001, RULE-002 | Given a project with a selected tag outside the new primary craft’s suggestion list, the editor still shows it as selected and removable. A fresh Quilting picker and filter offer Book/Film Characters, a fresh Embroidery picker and filter offer Quotes, and Floral keeps its existing shared value and cross-craft search meaning. |
| AC-002 | BR-002, FR-002, FR-009 | Given a new Knitting or Crochet project, an author can search size options, select multiple distinct values, publish and reopen the same selections. Only the plural list is needed for a new size write. AppView and both clients read the canonical metric strings and show each size. |
| AC-003 | BR-002, FR-003, RULE-002 | Given a record or draft carrying only a legacy size, the editor shows one selected value, including a valid out-of-catalog string. A present list wins over the scalar. An unrelated published edit preserves the stored representation. An explicit size replacement or clear removes the scalar and reopening a cleared project shows no selected size. |
| AC-004 | BR-001, FR-004 | A Knitting project can record Cables alongside Colourwork with Intarsia and Stranded selected together. Removing Intarsia alone keeps Stranded. Clearing both subtypes keeps Colourwork selected. Removing Colourwork clears its associated subtypes. A Crochet blanket can record Granny square as a technique and remain a blanket. Existing Granny square and Amigurumi item subtypes are still readable and selectable where applicable. |
| AC-005 | FR-004 | A Sewing project can select French seams, Invisible zip insertion and Print matching across the group headings. The project can also carry Plaid as a separate design tag. Its detail view presents the selected techniques. |
| AC-006 | BR-001, FR-005, RULE-002 | An Embroidery project can save Hand with Sashiko and Machine with 3D puff. Reopening shows each style under its method. Removing Hand removes its styles and retains Machine and 3D puff. Machine Styles excludes the single- and multi-needle equipment choices. Older Embroidery projects without details can still be opened and edited. |
| AC-007 | BR-001, FR-006 | For each new primary craft, an author can publish using shared fields alone and can also save, reopen and display the details listed for that craft in §11. Equivalent material information can be entered once in the shared Materials field. Dyeing accepts spun yarn and unspun fibre. Cord diameters remain associated with their cords, and warp and weft materials remain distinguishable. |
| AC-008 | BR-001, FR-007 | A Knitting author can independently select Hand and Machine. Selecting Machine exposes machine fields. A machine-knit cardigan retains its item subtype and stores machine gauge and measured stitch gauge independently. On reopening, the editor and detail view show the saved selections. The Machine filter matches projects whose selections include Machine. With neither selected, the method remains unspecified, including on older records. A needle-felted sculpture keeps Felting as its craft, Needle as its method and sculptural as its form, and is findable with the Needle method filter. |
| AC-009 | FR-008 | Visible projects can be found by entered needle or hook sizes in either representation, by terms such as Fair Isle, Granny square and Print matching, by Embroidery styles, and by craft methods. Matching can use reviewed aliases such as Afghan crochet. Existing Granny square subtype matches still work and irrelevant text fields do not gain search eligibility automatically. |
| AC-010 | FR-003, FR-009, NFR-002 | A legacy local draft reopens with its scalar size as one selection. A new draft and a new schedule with revised metadata can each be saved, reopened and published with their entered values. Published owner edits preserve the URI and untouched data. Retrying a publication through the current flow creates at most one post. |
| AC-011 | FR-006, RULE-001 | A skein project with Spinning as primary is eligible for the Spinning feed and Spinning details. A sock project with Knitting as primary and Spinning as additional stores Knitting details and participates in Spinning keyword and tag discovery. Craft-feed eligibility follows the primary craft in both cases. |
| AC-012 | NFR-001 | For representative crafts, equivalent Flutter and Svelte pickers offer the same eligible options, recognise the same aliases and use the same group relationships and stored values. AppView keyword searches resolve the agreed aliases and parents, and applicable filters find projects with matching values. |
| AC-013 | NFR-002, RULE-002 | On the new AppView, fixtures for legacy post records, additive records and valid records with unknown option values or extension properties all retain their public contents. Inputs outside the new field bounds fail validation. A rollout check confirms that AppView can accept the new fields before the first new client is enabled to write them. |
| AC-014 | RULE-003 | An authorized author can edit the fields through the existing edit flow. A different user and a stale conflicting edit cannot overwrite the record. Existing visibility rules continue to apply to keyword and filter results. Draft and schedule metadata become public only through successful publication. |
| AC-015 | FR-010 | A legacy Flutter or web request can publish or explicitly change a single size, and the public record stores a single-item list. The legacy client can reopen that size and clear it. An unrelated save preserves multiple stored sizes. An explicit scalar replacement or clear that would discard multiple values fails through the existing save-error flow and preserves the saved list. |
| AC-016 | FR-011 | A new member can select any new primary craft during onboarding, and an existing member can add or remove it from their interests. Saved choices round-trip and are prioritised in Home. All eleven primary crafts are reachable in Home and participate in grouped hashtag discovery even when absent from the viewer's interests. Machine Knitting and Needle Felting are available through their parent crafts. |
| AC-017 | NFR-003 | With the six new primary crafts still unavailable, both clients can use all Phase 1 features in §11 through the supported AppView. When Phase 2 is available, the new crafts have the editor, profile, onboarding and discovery entry points described in §11. |
| AC-018 | RULE-004 | Searching Tunisian crochet by its alias Afghan crochet offers the canonical technique. The first-party pickers add only catalog choices and offer no generic Other item. An unfamiliar valid saved value remains visible and removable, and an unrelated edit preserves it. |
| AC-019 | BR-003, FR-009, FR-012 | In Flutter and Svelte, an author can add and publish up to five patterns. Drafts and new schedules restore the order and each pattern's metadata and facets. Project details show the saved entries in that order. Editing or removing one leaves the other entries intact. An editor at five entries prevents a sixth, and AppView rejects a six-entry request. |
| AC-020 | FR-012, FR-013 | An old record or local draft reopens with its single pattern intact. A present list wins over the legacy field, even when empty. An unrelated edit preserves the stored shape and extensions. An explicit updated-client replacement or clear removes the legacy field. A legacy client can create, reopen, change and clear an effective singleton. A partial change preserves the singleton's other fields, and valid legacy writes use the canonical list. An ambiguous save preserves multiple patterns and an explicit legacy attempt to replace or clear them fails without losing data. |
| AC-021 | FR-008, RULE-003 | Search finds visible projects by eligible text or attributes on any effective pattern and by a hashtag found only on a later pattern, within the existing budget. Raw URLs and overridden legacy entries do not contribute. Existing pattern filters match any effective entry, and a combined difficulty and self-drafted request matches only when one entry satisfies both. A project appears once per result set and existing visibility checks apply. Publication delivers eligible mentions from later patterns and deduplicates them per post and account. |

## 14. Edge cases

| ID | Case | Expected behavior | Requirements |
|---|---|---|---|
| EC-001 | Both representations of a size or pattern are present, including a list with no members. | Read the list. Clearing both is a durable first-party operation. | FR-003, FR-012 |
| EC-002 | Primary craft change leaves some design tags outside suggestions. | Saved tags are retained and removable. For new craft-specific choices, the author sees the selected craft’s fields. | FR-001, RULE-001 |
| EC-003 | An author removes a technique or one of several Embroidery methods. | Remove the dependents associated with that selection. Keep the other selected techniques or methods and their dependents. Preserve valid unknown values on unrelated edits. | FR-004, FR-005, RULE-002 |
| EC-004 | Crochet construction and item subtype have the same label. | Preserve both meanings. A keyword can find either use and a subtype filter still matches the item classification. | FR-004, FR-008 |
| EC-005 | Older records have no size, method or detail object. | The absent value is optional. The editor and display can operate with the shared fields. | FR-003, FR-005, FR-007, NFR-002 |

## 15. Data and persistence impact

- The record additions are optional size and pattern lists, techniques and applicable subtypes, method and style fields, and typed details variants for Embroidery and the new primary crafts. The existing post collection, shared design tags, and optional singular size and pattern fields remain valid.
- The existing AppView project JSON and profile craft arrays can store the new data. Neither phase requires a PostgreSQL schema migration, a backfill or a bulk PDS rewrite. Search covers legacy and plural shapes, including already-indexed legacy projects. Dedicated columns or indexes remain implementation options if performance warrants them.
- Legacy local drafts can reopen in updated clients with one size or pattern. The storage representation for unpublished drafts and new schedules is an implementation choice. Current scheduling saves, reopens and publishes the revised fields.
- During the compatibility period, expose effective singletons through the corresponding legacy fields. New publications and explicit published replacements store the canonical list. An omitted edit preserves the stored representation. Legacy editing follows FR-010 and FR-013.
- Phase 2 extends profile craft choices using existing profile identifiers for Spinning, Weaving, Felting and Macramé. Keep saved interests and use the existing profile preference rules and limits.
- The implementation handoff includes an ADR for the Lexicon change, regenerated schema artifacts, a catalog revision and an update to the searchable-field contract where matching changes.

## 16. UI, API and CLI impact

- **Flutter and Svelte** need composer, saved-project and detail changes, including repeatable pattern entries, plus craft-specific filters. Phase 2 also extends existing profile interests, onboarding, Home craft navigation and grouped hashtag discovery. Method-specific controls appear when relevant in both clients.
- **AppView** needs validation, record and edit handling, singular-field compatibility, and expanded craft and pattern support in discovery. Use existing post, draft, schedule and profile flows. Document new request or response fields where applicable.
- **CLI** has no new user-facing command in scope. **Background jobs** publish the revised fields through the current scheduling flow.

## 17. Security, privacy and permissions

Existing AppView authentication, PDS authorization and conditional owner edits apply. Drafts and pending schedules retain their private storage. First-party fields explained as project details, including any dye recipe notes, become public when published. Apply bounds to new arrays and strings and keep the existing search visibility and moderation gates. No new OAuth permissions are required. [Architecture](../../architecture.md) · [Published post editing ADR](../../adr/022-published-post-editing.md)

## 18. Observability

Use existing validation, indexing and scheduled publishing health signals when checking rollout. No new telemetry or alerts are proposed. Any diagnostic change follows the existing logging guide and avoids including private draft or schedule contents. [Logging and error reporting](../../development/logging-error-reporting.md)

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Size or pattern fields clear differently across edit paths. | Legacy values reappear or draft values disappear. | Exercise absent, present, empty and replaced lists in the current flows. Cover legacy values in immediate publishing, published edits, local drafts and search. |
| RISK-002 | A legacy client can display only one size or pattern. | A published edit could collapse several valid entries. | Deploy AppView support first and verify legacy singleton edits and list preservation on both clients. |
| RISK-003 | Options drift across clients or terms carry different meanings. | Authors choose inconsistent values and filters miss records. | Version the catalog, test parity and review trade terms and parent relationships with a craft reviewer. |
| RISK-004 | A broad schema change makes the initial taxonomy difficult to revise. | Public metadata becomes harder to evolve as crafts are reviewed. | Add optional fields and open vocabularies and publish the Lexicon through the reviewed release path. |
| RISK-005 | Project filters, Home, interests and hashtag discovery use separate craft lists. | A craft can appear in one surface and be missing from another. | Verify supported crafts across the AppView and both clients before Phase 2 ships. |

## 20. Assumptions

| ID | Assumption | Impact if wrong |
|---|---|---|
| ASM-001 | Resolved by Q3 and Q12. Include profile interests and onboarding in Phase 2. | Covered by FR-011. |
| ASM-002 | Resolved by Q6 and Q7. Knitting can capture several subtypes per technique, and Embroidery can group styles under both methods. | Covered by FR-004 and FR-005. |
| ASM-003 | Resolved by Q8 and Q11. Dyeing covers spun yarn and unspun fibre. | Covered by FR-006. |
| ASM-004 | Production record counts and installed beta versions remain unverified. Compatibility is an explicit rollout requirement. | Identify the legacy builds to exercise for rollout. Cover existing published records and local drafts. |
| ASM-005 | A legacy editor can faithfully express an effective singleton. Authors use an updated client to change projects with several sizes or patterns. | Verify compatibility with legacy Flutter and Svelte requests and use the updated editors for lists. |
| ASM-006 | A combined difficulty and self-drafted filter describes one pattern. | A project with the selected difficulty and self-drafted status on different entries is excluded. |

## 21. Open questions

Product decisions from both review rounds are recorded in §3. The remaining items are implementation and rollout checks.

- **Schema design**. Document the limits for new fields before implementation. Preserve the existing 20-byte and 20-grapheme limits for individual size strings. A pattern list allows five entries and reuses the existing limits for each pattern object.
- **Rollout check**. Exercise legacy size and pattern compatibility with older mobile and web builds before enabling plural-list writes. Review continued usage before retiring the compatibility path.

## 22. Review status

**Requirements reviewed · High risk.** This revision incorporates both product review rounds, the approved simplifications, the prelaunch scheduling scope, the optional Knitting method multi-select and the follow-up request for multiple patterns. Public schema changes, legacy editing and coordinated AppView and client support require release verification.

## 23. Handoff to test design

- The next artifact is `02-acceptance-tests.md`. All Must requirements in §12 have acceptance criteria in §13.
- Verify each craft's launch fields and discovery surfaces. Cover the publish, draft, schedule, edit and clear paths with representative payload shapes, including nested selections, material associations, size lists and pattern lists. Verify Flutter and Svelte picker agreement and AppView search behaviour in their respective tests.
- Exercise legacy single-value publishing and edits, including an explicit replacement that would discard hidden sizes or patterns. Use current payloads for schedule coverage. Cover each rollout phase independently and use §21 for compatibility fixtures and release checks.
