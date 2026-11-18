# Project craft schema requirements

## 1. Initial request

Make project options more relevant to each craft, allow multiple knitting needle and crochet hook sizes, and capture techniques and subtypes from the craft review workbook. Follow-up discussion added sewing techniques and coverage for Spinning, Dyeing, Machine Knitting, Punch Needle, Macramé, Weaving and Needle Felting. Include the new primary crafts in profile interests and onboarding. The experience applies to Flutter on iOS and Android and to the Svelte web client. [Craft schema review workbook](https://docs.google.com/spreadsheets/d/1k0Ymrb7FPNLhdVd53A_zy-i4kBXnGxmN/edit?gid=1470412807)

## 2. Current codebase findings

- The composer offers five primary crafts. Sewing, Knitting, Crochet and Quilting have typed details. Embroidery uses shared fields today. Spinning, Weaving, Felting and Macramé have existing profile compatibility IDs. [Project lexicon](../../../lexicon/social/craftsky/project/defs.json) · [Profile craft catalog](../../../app/lib/profile/data/crafts_catalog.dart)
- The primary craft owns the optional details object and determines craft-feed eligibility. Additional crafts participate in keyword and tag discovery. AppView indexes an explicit set of public fields and retains the raw project data. [Project craft ADR](../../adr/020-project-evolution-search-and-multiple-crafts.md) · [Searchable-field contract](../../development/search-fields.md)
- Both composers have searchable multi-selects. The size fields are currently single strings. Draft restoration, scheduling and published editing have separate entry points that need coverage. [Flutter draft adapter](../../../app/lib/projects/composer/project_draft_snapshot_adapter.dart) · [Web project editor](../../../web-client/src/lib/features/composer/ProjectEditor.svelte) · [AppView edit merge](../../../appview/internal/api/post_edit_merge.go)
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
| Where do the new crafts appear? | Q3 and Q12. Include project pickers, profile interests, onboarding, Home craft feeds and grouped hashtag discovery. Keep the existing Home preference ordering. |
| What can ship independently? | Q4 and Q13. Release the existing-craft changes first. Release the six new crafts and their associated surfaces together in the second phase. |
| What belongs under Techniques? | Q5, Q14 and Q16. Use Techniques for Knitting, Crochet and Sewing. Use catalog selections for new tags, techniques and styles. Preserve both uses of Granny square. Amigurumi stays an item subtype. |
| Can a Knitting technique have several subtypes? | Q6. An author can choose zero or more subtypes for each selected technique. |
| Can Embroidery combine methods? | Q7 and Q15. Allow Hand, Machine or both, with separate style selections for each method. Keep single- and multi-needle equipment choices out of Styles and defer an equipment field. |
| What can people dye? | Q8 and Q11. Use **Dyeing** for projects using spun yarn or unspun fibre, including fleece and roving. |
| How do readers find these projects? | Q9. Add keyword discovery for techniques and styles and dedicated filters for Machine Knitting and Needle Felting within their parent crafts. |
| How does size compatibility work? | Q10 and Q18. Read scalar sizes as one selection and convert scalar writes from older clients to single-item lists on new publications and published size edits. Preserve existing public records and support older clients through the transition. |
| Does scheduling need legacy compatibility? | Scheduling is prelaunch. Support the new fields through the current scheduling flow. Existing prelaunch schedule payloads and older scheduling clients are outside the compatibility scope. |
| What is needed in the expanded craft editors? | Q17. Include the optional starter fields in §11. Each craft can publish using shared fields alone. |

## 4. Candidate approaches

| Approach | Benefits | Trade-offs |
|---|---|---|
| Shared design tags with craft-specific suggestions and typed details | Preserves existing values and cross-craft discovery. Adds fields where their meaning and units are specific to the craft. | Requires catalog parity and handling for legacy and new sizes across both clients and AppView. |
| Craft-specific design-tag fields and separate craft identities for every method | Gives every craft or method an independent set of rules and its own feed identity. | Duplicates concepts such as Floral, complicates search across crafts and separates projects such as hand- and machine-knit cardigans. |
| A single shared techniques field | Gives the editors one vocabulary structure. | Method, equipment and project form still need craft-specific rules. Shared labels can conceal different meanings, such as mosaic knitting and mosaic crochet. |

## 5. Recommended direction

Keep the shared project envelope and design-tag values. Extend the optional craft details and use the catalog for picker choices, aliases and relationships. Introduce additive size lists and craft variants so public records remain readable through the rollout. Use project type and subtype for the item made, method for how the overall craft was practised, and techniques for selections within that work.

## 6. Problem and opportunity

Current project fields cannot describe common cases such as a sweater knitted with two needle sizes, a crochet blanket assembled from granny squares, or an independently spun skein. Adding a small set of relevant fields makes these projects easier to share and find while keeping ordinary project posting quick.

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

Design suggestions use the shared vocabulary. Knitting and Crochet each store one tool size. Existing project types and subtypes classify the item made. Quilting already has separate piecing and quilting method fields. The six proposed primary crafts have no first-party project picker entries or detail editors.

## 11. Desired behavior and starter vocabulary

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
| **Machine Knitting** | **Knitting method** offers Hand, Machine or Both. Machine and Both expose optional machine model, machine gauge (needle spacing), and machine techniques such as tuck stitch, slip stitch and double-bed jacquard. Yarn weight, actual stitch and row gauge, project type and finished size still apply. Needle sizes are relevant to hand work. |
| **Punch Needle** | Project type (rug, wall hanging, cushion, hoop art), foundation fabric, needle model or size, yarn or thread, and loop height. One craft supports both rug-scale work and fine embroidery. |
| **Macramé** | Cord material, construction and diameter per cord where multiple cords are used, knots used (square knot, lark’s head, double half hitch), and finished dimensions. |
| **Weaving** | Loom type, weave structure (plain weave, twill, doubleweave), materials identified by warp and weft role, sett with unit (warp ends per inch or centimetre), and finished dimensions. |
| **Needle Felting** | **Felting** is the primary craft and **Needle** is a visible method. Offer project form (flat or sculptural), fibre, multiple needle gauges and optional core or armature details. Reserve room for other methods, such as wet felting, as the catalogue grows. |

### Release phases

1. **Existing crafts**. Release the scoped design suggestions, needle and hook size lists with legacy-client compatibility, Knitting, Crochet and Sewing techniques, Embroidery methods and styles, and Machine Knitting. Include corresponding keyword and filter support when each field ships.
2. **Craft expansion**. Release Spinning, Dyeing, Punch Needle, Macramé, Weaving and Felting with their optional editors, project views, profile interests, onboarding, Home feeds and grouped hashtag discovery. Include Needle Felting within Felting.

## 12. Requirements

Priorities describe the reviewed product scope. §21 records details for schema design and rollout verification.

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Let authors describe the craft, item made and relevant methods or techniques independently. | Readers can find comparable projects and see how they were made. | Initial prompt and follow-ups | AC-004, AC-006, AC-007, AC-008 |
| BR-002 | Business | Must | Let authors add and clear multiple needle sizes in Knitting and multiple hook sizes in Crochet. | Real projects use more than one tool size. | Initial prompt and confirmed string lists | AC-002, AC-003 |
| FR-001 | Functional | Must | Keep `project.common.designTags` and scope picker suggestions in both composers and craft-specific project filters to the selected primary craft. Keep saved selections visible and removable even when outside suggestions. | Distinguishes relevance from the meaning of the stored tag. | Workbook and discovery | AC-001 |
| FR-002 | Functional | Must | Use searchable multi-selects and write optional bounded `needleSizesMm` and `hookSizesMm` string lists. Reuse canonical metric strings and existing aliases and display all effective sizes. | Selection is quick and records remain portable. | Maintainer decision | AC-002 |
| FR-003 | Functional | Must | Read a valid legacy scalar as one selection when the list is absent. A present list takes precedence, including an explicit empty list. An explicit first-party size edit or clear removes the old scalar in the same saved change. | Old sizes remain readable and never reappear after an intentional clear. | Migration discovery | AC-003, AC-010 |
| FR-004 | Functional | Must | Provide optional multi-select Techniques for Knitting, Crochet and Sewing. Allow multiple optional subtypes per selected Knitting technique where defined, and provide picker group headings for Sewing. Show saved techniques and subtypes on project details. | Separates construction from the made item and its appearance. | Q5, Q6 and Q16 | AC-004, AC-005 |
| FR-005 | Functional | Must | Allow Hand, Machine or both as optional Embroidery methods and attach each style selection to its method. Display each method and its saved styles. | A single project can combine embroidery methods. | Q7 and Q15 | AC-006 |
| FR-006 | Functional | Must | Offer Spinning, Dyeing, Punch Needle, Macramé, Weaving and Felting as primary and additional craft choices in project authoring. For each primary craft, authors can record, reopen and display the optional details in §11. Reuse shared fields for equivalent details and preserve the material relationships described in §11. | Supports posting about these crafts without requiring duplicate entry. | Q2, Q11, Q17 and simplification review | AC-007, AC-011 |
| FR-007 | Functional | Must | Offer Machine as a Knitting method and Needle as a Felting method. Show and display their applicable specialist fields and provide method filters within the relevant primary craft. | People can find these projects and classify their finished items. | Craft expansion discovery | AC-008 |
| FR-008 | Functional | Must | Make new sizes, techniques, technique subtypes, embroidery styles and craft methods discoverable in project keyword search through the explicit search contract. Keep relevant existing design, project type and subtype search. | Readers can find a technique whether or not it describes the object made. | Discovery and established search contract | AC-009 |
| FR-009 | Functional | Must | Support the revised fields through immediate publishing, draft save and reopen, scheduled save, reopen and execution in the current scheduling flow, published owner edits and project detail display in Flutter and Svelte. | Metadata is useful only when it survives normal authoring paths. | Initial UI request and scheduling scope decision | AC-002, AC-010 |
| FR-010 | Functional | Must | Legacy clients can publish, reopen, change and clear a single size. Store a valid legacy scalar as a canonical single-item list on publication or an explicit published size edit. Preserve all stored sizes on unrelated edits and reject a legacy size edit that would discard multiple values. | Older editors can represent one size during the transition. | Q18 and simplification review | AC-015 |
| FR-011 | Functional | Must | Include the six new primary crafts in profile interests, onboarding, Home feeds and grouped hashtag discovery on both clients. Preserve preference ordering and make all craft feeds reachable to people with any interests. | The new crafts are usable throughout the normal experience. | Q3 and Q12 | AC-016 |
| NFR-001 | Non-functional | Must | For equivalent picker contexts, Flutter and Svelte offer the same eligible choices, aliases and group relationships and store the same values. AppView resolves those values and their agreed aliases and parent relationships in applicable searches and filters. | Aligns classification and discovery across both clients. | Catalog discovery and simplification review | AC-012 |
| NFR-002 | Non-functional | Must | Preserve existing public records and post identities without bulk PDS rewrites. Deploy AppView support for both size representations before enabling clients to publish the new fields. | Keeps existing public records readable throughout rollout. | Architecture and migration discovery | AC-010, AC-013 |
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
| AC-008 | BR-001, FR-007 | A machine-knit cardigan retains its item subtype and can store machine gauge and measured stitch gauge independently. Selecting Machine within the Knitting filter finds machine-knit projects, including Both where used. A needle-felted sculpture keeps Felting as its craft, Needle as its method and sculptural as its form, and is findable with the Needle method filter. An older Knitting record without a method does not acquire an inferred method. |
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

## 14. Edge cases

| ID | Case | Expected behavior | Requirements |
|---|---|---|---|
| EC-001 | List and scalar both exist, including a list with no members. | Read the list. Clearing both is a durable first-party operation. | FR-003 |
| EC-002 | Primary craft change leaves some design tags outside suggestions. | Saved tags are retained and removable. For new craft-specific choices, the author sees the selected craft’s fields. | FR-001, RULE-001 |
| EC-003 | An author removes a technique or one of several Embroidery methods. | Remove the dependents associated with that selection. Keep the other selected techniques or methods and their dependents. Preserve valid unknown values on unrelated edits. | FR-004, FR-005, RULE-002 |
| EC-004 | Crochet construction and item subtype have the same label. | Preserve both meanings. A keyword can find either use and a subtype filter still matches the item classification. | FR-004, FR-008 |
| EC-005 | Older records have no size, method or detail object. | The absent value is optional. The editor and display can operate with the shared fields. | FR-003, FR-005, FR-007, NFR-002 |

## 15. Data and persistence impact

- The record additions are optional string lists, techniques and applicable subtypes, method and style fields, and typed details variants for Embroidery and the new primary crafts. The existing post collection, shared design tags, and optional scalar size fields remain valid.
- The existing AppView project JSON and profile craft arrays can store the new data. Neither phase requires a PostgreSQL schema migration, a backfill or a bulk PDS rewrite. Search covers both size shapes in already-indexed projects. Dedicated columns or indexes remain implementation options if performance warrants them.
- Legacy local drafts can reopen in updated clients as one selected size. The storage representation for unpublished drafts and new schedules is an implementation choice. Current scheduling saves, reopens and publishes the revised fields.
- During the compatibility period, expose an effective singleton as a scalar on legacy client reads. New publications and explicit published size edits store the canonical list. An omitted size edit preserves the stored sizes. Legacy editing follows FR-010.
- Phase 2 extends profile craft choices using existing profile identifiers for Spinning, Weaving, Felting and Macramé. Keep saved interests and use the existing profile preference rules and limits.
- The implementation handoff includes an ADR for the Lexicon change, regenerated schema artifacts, a catalog revision and an update to the searchable-field contract where matching changes.

## 16. UI, API and CLI impact

- **Flutter and Svelte** need composer, saved-project and detail changes, plus craft-specific filters. Phase 2 also extends existing profile interests, onboarding, Home craft navigation and grouped hashtag discovery. Method-specific controls appear when relevant in both clients.
- **AppView** needs validation, record and edit handling, scalar compatibility, and expanded craft support in profile writes, search and filters. Use existing post, draft, schedule and profile flows. Document new request or response fields where applicable.
- **CLI** has no new user-facing command in scope. **Background jobs** publish the revised fields through the current scheduling flow.

## 17. Security, privacy and permissions

Existing AppView authentication, PDS authorization and conditional owner edits apply. Drafts and pending schedules retain their private storage. First-party fields explained as project details, including any dye recipe notes, become public when published. Apply bounds to new arrays and strings and keep the existing search visibility and moderation gates. No new OAuth permissions are required. [Architecture](../../architecture.md) · [Published post editing ADR](../../adr/022-published-post-editing.md)

## 18. Observability

Use existing validation, indexing and scheduled publishing health signals when checking rollout. No new telemetry or alerts are proposed. Any diagnostic change follows the existing logging guide and avoids including private draft or schedule contents. [Logging and error reporting](../../development/logging-error-reporting.md)

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Size fields clear differently across edit paths. | Legacy values reappear or drafted values disappear. | Exercise absent, present, empty and replaced sizes in the current flows. Cover legacy values in immediate publishing, published edits, local drafts and search. |
| RISK-002 | A legacy client can display only one size. | A published edit could collapse several valid sizes. | Deploy AppView support first and verify legacy singleton edits and multi-size preservation on both clients. |
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
| ASM-005 | An effective singleton is the only list that a legacy picker can express faithfully. Legacy authors use an updated client to change projects with several sizes. | Full multi-size editing requires the updated picker. Verify compatibility with legacy Flutter and Svelte requests. |

## 21. Open questions

Product decisions from both review rounds are recorded in §3. The remaining items are implementation and rollout checks.

- **Schema design**. Document the limits for new fields before implementation. Preserve the existing 20-byte and 20-grapheme limits for individual size strings so a valid legacy value fits as one list entry.
- **Rollout check**. Exercise scalar compatibility with older mobile and web builds before enabling plural-list writes. Review continued usage before retiring the compatibility path.

## 22. Review status

**Requirements reviewed · High risk.** This revision incorporates both product review rounds, the approved simplifications and the prelaunch scheduling scope. Public schema changes, legacy editing and coordinated AppView and client support require release verification.

## 23. Handoff to test design

- The next artifact is `02-acceptance-tests.md`. All Must requirements in §12 have acceptance criteria in §13.
- Verify each craft's launch fields and discovery surfaces. Cover the publish, draft, schedule, edit and clear paths with representative payload shapes, including nested selections, material associations and size lists. Verify Flutter and Svelte picker agreement and AppView search behaviour in their respective tests.
- Exercise legacy scalar publishing and edits, including a scalar replacement that would discard hidden sizes. Use current payloads for schedule coverage. Cover each rollout phase independently and use §21 for compatibility fixtures and release checks.
