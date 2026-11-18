# Project craft schema coding plan

## 1. Inputs

- [Reviewed requirements](01-requirements.md), including both release phases and the exclusion of legacy schedule compatibility.
- [Acceptance test specification](02-acceptance-tests.md), including the revisions for DR-001 through DR-005.
- [Document review](03-document-review.md), approved with notes after the revisions were rechecked. This plan is for review before implementation.
- [Project craft ADR](../../adr/020-project-evolution-search-and-multiple-crafts.md) · [Published editing ADR](../../adr/022-published-post-editing.md) · [AppView API architecture](../../superpowers/specs/2026-04-21-appview-api-architecture-design.md) · [Searchable-field contract](../../development/search-fields.md)

## 2. Implementation strategy

Begin with the **schema review checkpoint** in §10. Generate the proposed Lexicons and their artifacts for both phases, then build the interactive Project Schema page from that checkout. Share the local preview and wait for the maintainer to confirm the schema before starting application behaviour changes.

After confirmation, build **Phase 1** in vertical slices. Start with legacy-compatible sizes through AppView and both clients. Add the catalog, scoped design suggestions, grouped techniques and Embroidery styles, then Machine Knitting and its filter. Exercise the current scheduling flow and close the Phase 1 regression gate before starting Phase 2 dependencies.

Build **Phase 2** on the same optional details union and catalog. Add Spinning, Dyeing, Punch Needle, Macramé, Weaving and Felting, including the Needle method. Extend authoring, details, profile interests, onboarding, Home and grouped hashtags together. Keep both phases independently testable through their release candidates.

Use the existing Go request and command boundaries, Flutter project models and Riverpod providers, and SvelteKit composer and social modules. The existing Postgres JSONB project details and profile craft arrays can hold these changes. Read the new search and filter fields from explicit JSON paths. The initial implementation needs no SQL migration or backfill.

## 3. Affected areas

| Area | Existing pattern and planned change | Requirements | Criteria | Tests |
|---|---|---|---|---|
| Schema review | Generate the full proposed schema and use the existing Project Schema explorer for maintainer review before behaviour work. | FR-002, FR-004, FR-005, FR-006, FR-007, RULE-002 | AC-002, AC-004, AC-006–AC-008, AC-013 | UT-005, existing Project Schema browser smoke test |
| Tool sizes and compatibility | Optional size fields in typed craft details. Add canonical lists, preserve old reads and accept safe legacy writes. | BR-002, FR-002, FR-003, FR-010, NFR-002 | AC-002, AC-003, AC-013, AC-015 | AT-002, UT-001, UT-005, IT-001, IT-002, REG-001, MAN-001, MAN-002 |
| Design suggestions and techniques | Shared tags with catalog labels and parent relationships. Add craft eligibility and optional grouped details. | BR-001, FR-001, FR-004, FR-005, NFR-001, RULE-002, RULE-004 | AC-001, AC-004, AC-005, AC-006, AC-012, AC-013, AC-018 | AT-001, AT-003, AT-004, UT-002, UT-003, UT-005, IT-003, REG-002 |
| Authoring lifecycle | Existing composer, draft, scheduled publication and conditional owner edit flows. Extend each with the same project data contract. | FR-003, FR-009, NFR-002, RULE-003 | AC-002, AC-003, AC-010, AC-013, AC-014 | AT-002, AT-008, IT-001, IT-003, IT-004, REG-001 |
| Keyword and method discovery | Existing explicit keyword fields and project browse filters. Add guarded detail paths and Machine and Needle predicates. | FR-007, FR-008, NFR-001, RULE-003 | AC-008, AC-009, AC-012, AC-014 | AT-006, IT-005, REG-002, REG-003 |
| New craft details | One optional recognized details variant for the primary craft. Add six authoring variants and retain additional-craft discovery. | BR-001, FR-006, FR-007, RULE-001 | AC-007, AC-008, AC-011 | AT-005, AT-006, IT-001, IT-003, IT-006 |
| Profile and discovery | Existing profile IDs and Home preference order. Expand selectable choices and discovery coverage in every applicable surface. | FR-011, RULE-001 | AC-011, AC-016 | AT-007, UT-004, IT-006, IT-007 |
| Release phases | AppView before its updated clients. Evaluate each phase with its own acceptance suite. | NFR-002, NFR-003 | AC-013, AC-017 | REG-004, MAN-002 |

## 4. Files and modules

| Path or module | Work | Requirements | Criteria and tests |
|---|---|---|---|
| `lexicon/social/craftsky/project/`, `lexicon/social/craftsky/feed/defs.json`, `appview/cmd/lexgen/` | Change the existing details and open union, add the necessary craft definitions, and regenerate the Go, CBOR and validation artifacts. Add an ADR with the implemented schema changes. | FR-002, FR-004, FR-005, FR-006, FR-007, RULE-002 | AC-002, AC-004, AC-006, AC-007, AC-008, AC-013 · UT-005, IT-001, REG-002 |
| `scripts/project-schema-diagram.py`, `scripts/project-schema/explorer.html`, `web/test/project-schema.spec.js` | Reuse the public website's Lexicon-generated explorer for a local candidate page. Change the explorer only if a new shape cannot be inspected. Extend the existing browser smoke test to check representative new fields and variants. | FR-002, FR-004, FR-005, FR-006, FR-007 | AC-002, AC-004, AC-006–AC-008 · existing Project Schema browser smoke test |
| `appview/internal/projectcatalog/catalog-v1.json`, `catalog.go`, `scripts/project-catalog` | Extend the existing source catalog and generator with scoped suggestions, craft mappings, aliases and relationships. Generate client projections. | FR-001, FR-004, FR-006, FR-011, NFR-001, RULE-004 | AC-001, AC-004, AC-007, AC-012, AC-016, AC-018 · UT-002, UT-004, IT-005, IT-007 |
| `appview/internal/api/post_project.go`, `post_request.go`, `post_create.go`, `post_response.go`, `post_edit_editable.go`, `post_edit_merge.go`, `post_edit_candidate.go`, `appview/internal/sourcevalidation/registry.go` | Extend project validation, canonical candidate construction, raw merge, public DTOs and owner editing. Add new recognized details mappings. | FR-002, FR-003, FR-004, FR-005, FR-006, FR-009, FR-010, NFR-002, RULE-001, RULE-002, RULE-003 | AC-002–AC-007, AC-010, AC-011, AC-013–AC-015 · UT-005, IT-001–IT-003, IT-006, REG-001, REG-002 |
| `appview/internal/api/scheduled_post_request.go`, `appview/internal/scheduledposts/publication_processor.go` | Verify the current request and worker use the new detail shapes. Change only seams that reject, strip or misread those shapes. | FR-009, NFR-002, RULE-003 | AC-010, AC-014 · IT-003, IT-004 |
| `appview/internal/api/{search_request,search_fields,search_project_store,search_matching_store,search_cursor,craft_type}.go`, `appview/internal/index/craftsky_post.go`, `docs/development/search-fields.md` | Use existing raw detail projections, update the search allowlist and method browse predicates, and recognize all enabled primary crafts. Keep raw project data intact. | FR-001, FR-006, FR-007, FR-008, FR-011, NFR-001, RULE-001, RULE-003 | AC-001, AC-007–AC-009, AC-011, AC-012, AC-014, AC-016 · IT-005–IT-007, REG-003 |
| `app/lib/projects/models/project.dart`, `project_extensions.dart`, `app/lib/projects/composer/`, `app/lib/feed/composer/{post_edit_hydrator,post_edit_changes}.dart` | Extend typed models and raw extensions, form and payload mappings, draft codecs, and the observed-version edit patch. Regenerate the existing mappers. | FR-002–FR-007, FR-009, RULE-002 | AC-002–AC-008, AC-010, AC-013 · UT-001, UT-003, AT-002–AT-006, AT-008, REG-001, REG-002 |
| `app/lib/projects/widgets/{project_composer_sheet,project_card}.dart`, `app/lib/projects/options/`, `app/lib/theme/select_inputs/`, `app/lib/l10n/` | Reuse the existing composer detail sections and select controls. Add field labels, alias search, grouping and display. | FR-001, FR-002, FR-004–FR-007, NFR-001, RULE-004 | AC-001, AC-002, AC-004–AC-008, AC-012, AC-018 · AT-001–AT-006, UT-002 |
| `app/lib/projects/models/project_browse_filters.dart`, `app/lib/home/pages/{project_filter_sheet,home_page}.dart`, `app/lib/projects/providers/`, `app/lib/profile/data/crafts_catalog.dart`, `app/lib/search/providers/blank_search_provider.dart` | Extend browse state, active chips, craft mappings, profile choice inventory and discovery. Reuse project repositories and onboarding/profile pickers. | FR-001, FR-007, FR-011, RULE-001 | AC-001, AC-008, AC-011, AC-016 · AT-001, AT-006, AT-007, UT-004 |
| `web-client/src/lib/features/composer/{ProjectEditor,ComposerWorkspace}.svelte`, `web-client/src/lib/features/composer/model.ts`, `web-client/src/lib/{MultiSelect,PostCard}.svelte` | Extend the existing editors, project shaping, edit patch and detail display. Preserve raw edit baselines and reuse the draft storage format. | FR-001–FR-007, FR-009, NFR-001, RULE-002, RULE-004 | AC-001–AC-008, AC-010, AC-012, AC-018 · AT-001–AT-006, AT-008, UT-001–UT-003, REG-001, REG-002 |
| `web-client/src/lib/features/social/` | Extend the existing filters, craft ordering, preference pickers, onboarding, tag discovery and optimistic published overlays. | FR-001, FR-007, FR-011, RULE-001 | AC-001, AC-008, AC-011, AC-016 · AT-001, AT-006, AT-007, UT-004 |

## 5. Services, interfaces and data flow

### Public record and vocabulary

Keep the current `social.craftsky.feed.post` collection, shared project envelope and open details union. Keep existing field types and values. Use simple string lists for tool sizes and Knitting method IDs. Use optional object arrays for grouped selections and material associations. Only fields that identify an entry, such as its technique or method token, are required inside an otherwise optional group.

| Scope | Planned public shape |
|---|---|
| Knitting and Crochet sizes | `needleSizesMm` and `hookSizesMm` are bounded arrays of canonical metric strings. Retain the scalar fields for existing records. Preserve 20 UTF-8 bytes and 20 graphemes per list item and reuse catalog size values and aliases. |
| Knitting, Crochet and Sewing techniques | `techniques` holds keyed technique entries. Knitting entries can hold optional `subtypes` keyed by subtype token, associated with that entry only. Crochet and Sewing start with flat technique choices. Sewing headings are presentation metadata. Item type and subtype retain their established keys. |
| Machine Knitting | Add an optional Knitting `methods` list of method IDs with Hand and Machine as the two suggestions. Store the two IDs separately for mixed projects. Add optional machine model and `machineGaugeMm` alongside existing measured `gauge`. Include machine techniques through the Knitting technique inventory and scope suggestions to projects with Machine selected. An absent or empty list means the method is unspecified. |
| Embroidery | Add `social.craftsky.project.embroidery#details` with optional `methods`. Each method entry owns its own optional `styles`. Hand and Machine can each appear once in first-party authoring. Keep the approved style choices and exclude equipment choices. |
| Spinning and Dyeing | Add separate details variants. Spinning captures tool, fibre preparation, drafting and plying, ply count and finished yarn measurements. Dyeing captures material form, dye type, application method, colour effect and notes. Reuse shared Materials for base fibre when text suffices. |
| Punch Needle and Macramé | Punch Needle uses `projectType` for the item made and records the foundation, tool and loop height. Reuse shared Materials for its thread or yarn. Macramé keeps each cord's material, construction and diameter together in a cord entry, with knots and finished dimensions on the project. |
| Weaving and Felting | Weaving captures loom, structure, warp and weft material roles, sett with unit and finished dimensions. Felting captures `method`, form, needle gauges and optional core or armature. Reuse shared Materials for fibre where practical. |

Use the reviewed field inventory in requirements §11. For arbitrary length, mass or density with named units, use a bounded decimal string and an explicit allowed unit rather than an unqualified number. Keep finished dimensions consistent with existing editable size strings where their free-text semantics suffice. Use integer counts for discrete measures such as ply count. Include the proposed bounds and measurement wire keys for both phases in the review candidate and record the confirmed choices in the ADR. CPQ-001 records the remaining choices.

Keep catalog values open on the wire and retain valid unfamiliar values during unrelated edits. Picker catalogs provide eligible first-party options, shared tag values, allowed parent relationships and aliases. Extend the existing generator to emit a small web projection for project taxonomy and keep Flutter generation from the same source. Preserve unrelated web catalog data and any existing variant-specific option ordering. Use the existing catalog version and drift check. Use one ordered primary-craft inventory for AppView defaults, validation and client output, with explicit mappings between profile IDs, project craft tokens and details identifiers. Preserve `spinning`, `weaving`, `felting` and `macrame` in profiles. Use `dyeing` and `punch-needle` for the new profile choices. Derive the corresponding project tokens explicitly, including `#punchNeedle`.

### AppView size compatibility at the boundaries

Use a size helper next to the existing project DTO and edit merge. Resolve size presence from raw detail keys before generated JSON or generic patch handling can drop an explicit empty array or null edit. Apply it only within the recognized Knitting and Crochet variants. Use patch key presence to distinguish current and legacy size edits. Preserve existing `baseCid`, prepared command identity, validation and publish retry boundaries.

| Boundary | Behaviour |
|---|---|
| Immediate create | Validate the incoming request using the existing create flow. Normalize a copy of the project when building the PDS candidate. A valid scalar-only request becomes a singleton list. A valid explicit list takes precedence. Canonical public writes omit the scalar. Keep the original command intent and frozen retries. Build the accepted create response from the accepted project record so it matches the PDS and indexed read. |
| Public project and quote reads | Apply the compatibility projection to response copies in `post_response.go`. When the effective size list has one value, expose the legacy scalar. For an empty or multi-value list, omit the scalar even if the source still has one. Preserve canonical list presence and all unrelated data. |
| Owner edit seed and conflict rebase | Apply equivalent size projection in `postEditEditable`, which is used for the seed and conflict response. Compute the effective size from the authoritative raw details. Preserve an explicit empty list in the editor response. Leave the authority and observed version unchanged. |
| Published edit | Inspect scalar and plural patch key presence before the generic merge. With neither key, retain both stored fields. An explicit plural edit wins, including a clear, and removes the old scalar. A scalar-only set or clear is accepted only when the current effective list has at most one value. Reject an attempt to overwrite or clear a multi-size list through the existing `FieldError` validation envelope. Updated clients submit explicit plural intent, including on clear. |
| Current scheduling and drafts | Current clients shape canonical size payloads for new publications and schedules. Reuse the scheduled request's project validation and the worker's retained payload. Restore old local draft scalars as singleton selections. The private snapshot schema can retain either shape where appropriate. |

For grouped owner edits, extend the existing retained-entry pattern used by materials and additional crafts. Expose edit-only identity for supported entries in the seed, scoped to the observed version. Merge the changed entry against the corresponding authoritative raw object. Keep untouched groups and their unknown fields. Drop an entry only on explicit removal. Strip `existingIndex` and other editor-only data at the merge boundary. Apply the same rule to nested children and per-cord material rows where separate identity is needed. Keep the existing owner authorization, expected CID check, candidate validation and idempotent projection path.

### Search and discovery

- Keep project keyword search on `/v1/search/projects` and browse filters on `/v1/projects`. Add optional `knittingMethod` and `feltingMethod` browse params using the existing repeated-filter conventions. Emit method choices only in their craft context. For Knitting, match Machine by membership in `methods`. For Felting, match Needle. Keep both browse sort paths, existing cursor handling and visibility predicates.
- Read approved fields through guarded JSONB expressions on the existing `raw_details`, using the corresponding details discriminator and matching primary craft. Extract Knitting method IDs from the guarded `methods` array. Resolve size lists by field presence, including empty lists. Limit extraction to the contracted size, technique, subtype, style and method paths and existing type fields. Verify unknown extension data does not become searchable. Update the search-field documentation and relevance cursor revision with keyword changes.
- Extend post authoring and source validation together so each recognized variant belongs to its primary craft. Keep valid unknown variants round-trippable. Keep additional crafts participating in search and grouped hashtags. Home eligibility continues to use the primary craft.
- Extend `CanonicalCraftTypes` and the defaults used by grouped hashtags using the enabled inventory for each phase. Keep all eleven Phase 2 craft choices selectable in profile and onboarding while preserving the existing maximum of ten saved interests and the existing preference ordering. The profile Lexicon already accepts bounded string IDs. Verify the existing onboarding and profile update commands round-trip the new IDs.

## 6. State, providers, controllers and dependency injection

```text
Flutter ProjectComposerSheet ConsumerState + FormBuilder
  -> project composer hydrate and draft snapshot adapters
  -> payload/submit adapters for create and current schedule
  -> post edit hydrator and leaf diff for owner edits

Flutter Home and Projects filters
  -> ProjectBrowseQuery and ProjectBrowseFilters
  -> projectFeedProvider(query, audience)
  -> projectRepositoryProvider -> projectApiClientProvider
  -> existing AppView browse endpoint

SvelteKit ComposerWorkspace and existing draft store
  -> ProjectEditor + composer model helpers
  -> current create/schedule payload or observed-version edit diff
  -> existing server proxy -> AppView
```

- **Flutter editor**. Extend active craft field names and the existing detail form sync and reset paths. Add typed codecs for nested selections and material entries in `project_draft_snapshot_adapter.dart`. Keep nullable plural properties so absence, empty and populated remain distinct. Hydrate effective values for the form while retaining an edit baseline that can distinguish a user edit from compatibility display. Compare unchanged fields against that baseline so a text edit leaves legacy sizes and nested data untouched.
- **Flutter discovery**. Reuse `preferredProjectCraftsProvider`, `preferredCraftLinksProvider` and `blankSearchProvider`. Resolve explicit profile-ID mappings at the catalog boundary. Extend `ProjectBrowseFilters` count, clear, pill and query serialization with scoped methods. Reuse the existing feed provider, repositories and Riverpod families.
- **SvelteKit**. Keep `Draft.project` as JSON in the existing draft store. Use a shared project write helper for new immediate and current scheduled publication. Keep the original `editable.project` for diffs. For an explicit size edit emit the plural key and scalar removal, including an empty plural array for a clear. Extend retained-entry patch handling for nested details. Keep reads and writes through the existing proxy and session boundary.
- **AppView**. Keep route registration and dependency injection unchanged. Extend the existing post, scheduled publication and search handlers and preserve the private editor response boundary.

## 7. UI, widgets and routes

| Surface | Planned changes |
|---|---|
| Project composer and detail view | Use existing searchable multi-selects for both tools, techniques and styles. Render Sewing headings as picker metadata and render Knitting subtypes under their technique. In Embroidery, show each method's styles in its own selectable group. Support saved unfamiliar selections as removable chips without adding them to fresh suggestions. Show all saved tool sizes and associated values in `ProjectCard` and `PostCard.svelte`. |
| Machine Knitting and Felting | In both editors, show separate optional Hand and Machine checkboxes. Show conditional machine fields whenever Machine is checked and preserve the existing measured gauge and item subtype. Keep hand needle sizes available for hand work and preserve saved sizes on unrelated edits. Show Needle as a Felting method and its applicable detail controls. In both clients, show the new method filters in the existing project filter sheet or page and track them in active chips and clear behaviour. |
| Phase 2 craft details | Extend the existing `ProjectComposerSheet` detail page and web `ProjectEditor` sections. Use shared Materials for equivalent information and preserve associations in the Macramé and Weaving controls. Use existing localized labels, formatters and fallback icons. |
| Home, profile and search | Add the six entries to the existing project and profile pickers, onboarding, Home craft feeds and grouped hashtags. Preserve preferences-first Home ordering and preferred-only behaviour in the existing craft sidebar. On web, update `Filters.svelte`, `filter-model.ts` and `publications.svelte.ts` together so the selected filter and optimistic overlays agree with AppView membership. |

Use the existing slots and routes for creation, editing and scheduled updates. No new screen-level state or feature route is needed. Regenerate Flutter localization outputs when adding labels and retain the existing searchable control semantics.

## 8. Error, loading, empty and edge states

| Case | Planned handling | Requirements | Criteria and tests |
|---|---|---|---|
| Legacy and plural fields conflict | Presence of the plural field controls tool-size reads. Empty stays empty. Keep stored fields for unrelated edits and remove the old scalar on explicit size replacement or clear. | BR-002, FR-003, FR-010 | AC-003, AC-015 · UT-001, IT-002, REG-001 |
| Old editor attempts to discard several sizes | Reject the destructive scalar-only patch through normal validation. Keep current-client explicit plural changes and existing conflict rebase intact. | FR-010, RULE-003 | AC-014, AC-015 · IT-002, IT-003, REG-001 |
| Optional details are absent or unfamiliar | Show shared fields. Keep valid unseen selections and extensions on unrelated edits and preserve other groups when removing one parent. | FR-004, FR-005, FR-006, RULE-002, RULE-004 | AC-004, AC-006, AC-007, AC-013, AC-018 · UT-003, IT-003, REG-002 |
| Primary craft changes or recognized variant mismatches | Keep saved shared design tags. Clear or switch detail editing as the existing form does for the selected primary craft. AppView rejects a recognized mismatched variant before a PDS write. | FR-001, FR-006, RULE-001 | AC-001, AC-011 · AT-001, IT-006 |
| Draft, schedule or publish retry | Preserve the existing form save, reopen, pending and error states. Current scheduled jobs retain their existing privacy and retry contracts. Replaying a prepared public write preserves its recorded intent and command result. | FR-009, NFR-002, RULE-003 | AC-010, AC-013, AC-014 · AT-008, IT-003, IT-004 |

## 9. Test implementation plan

| Order | Tests | First failing assertions and targets |
|---|---|---|
| 0 | UT-005 schema checks, Project Schema browser smoke test | Generate the draft Lexicons, check the shapes and bounds that can be validated independently, and open the generated page. Verify the new size lists, grouped fields and six new details variants are reachable from the post schema. Share the preview and wait for schema confirmation. |
| 1 | IT-002, REG-001 | After schema confirmation, exercise a legacy scalar edit and clear using the public response and owner edit seed in `post_edit_commands_test.go` and `post_edit_http_integration_test.go`. A canonical singleton currently has no compatibility scalar, and legacy input has no canonical conversion. Add the rejected multi-size scalar edit and the accepted current-client control. |
| 2 | UT-001, UT-005, IT-001, AT-002 | Verify the approved size shapes and presence cases before changing AppView create, Flutter hydration and Svelte form helpers. Assert plural PDS writes, public and editor readback, and explicit clear. |
| 3 | AT-001, AT-003, AT-004, UT-002, UT-003, IT-003, REG-002 | Add catalog eligibility and grouped selection tests. Exercise the actual owner edit command against representative groups and assert retained raw sibling data. Use fixture-backed UI tests for picker behaviour. |
| 4 | AT-006, IT-005, REG-003 | Add independent Hand and Machine authoring, clearing and Machine membership filter assertions in Phase 1, then extend the filter and eligibility tests for Needle in Phase 2. Verify aliases, parents, legacy sizes and existing item classification. |
| 5 | AT-008, IT-004, REG-004 | Verify size and grouped payloads through local drafts and the current schedule worker. Inject a retry through the existing harness. Run the complete Phase 1 candidate before Phase 2 work depends on it. |
| 6 | AT-005, AT-007, UT-004, IT-001, IT-003, IT-006, IT-007 | Exercise the six new detail variants, representative associated materials, rejection of mismatched recognized variants, profile updates, onboarding, Home and grouped hashtags. Repeat Phase 1 coverage under REG-004. |
| 7 | MAN-001, MAN-002 | Exercise real supported legacy builds and verify AppView availability and old/new reads before updated clients write the fields. This is rollout evidence. |

Use the existing fake PDS command capture, real PostgreSQL integration harness and current scheduled worker fixtures for server assertions. Flutter widgets and Svelte Playwright use existing fixture-backed project editor and discovery suites. Use the [test specification](02-acceptance-tests.md) for test data and full target inventory.

## 10. Sequencing and guardrails

### Schema review checkpoint

1. **Generate the candidate.** Draft the required ADR and the additive Lexicons for both phases. Identify which definitions ship in each phase. Add new variant and nested object encoders to `appview/cmd/lexgen/cborgen/main.go` and regenerate the Go, CBOR and validation catalog. Run the schema-only parts of UT-005 against the draft.
2. **Build and verify the Project Schema page.** Run the existing website build, which invokes the [schema explorer generator](../../../scripts/project-schema-diagram.py) and produces the standalone `web/dist/project-schema.html`. Use the existing [browser smoke test](../../../web/test/project-schema.spec.js) to exercise the page and inspect representative new fields, selectable vocabulary, bounds, nested groups and all new detail variants. Use the Lexicons as the source of the page data. Record the application catalog's craft-scoped design suggestions alongside the link, since the explorer displays declared Lexicon values.
3. **Show the local artifact and pause.** Present the interactive page and identify the fields and craft tabs to review. Regenerate the page and repeat the checks after any requested revision. Obtain explicit confirmation of the reviewed schema before starting AppView behaviour, client changes, scheduling integration or discovery changes. Generate the preview locally. Lexicon publication, website deployment and phase enablement belong to the later release steps.

```sh
just lexgen
just lexgen-check
just web-build
WEB_TEST_ROOT=dist npm test --prefix web -- test/project-schema.spec.js
```

### After schema confirmation

- **First behaviour TDD step**. Add the IT-002 owner edit seed and scalar-to-list regression cases using JSON fixtures so they can fail against the current AppView behaviour. Follow with the guarded multi-size cases and current-client REG-001. Keep subsequent slices small enough to verify red and green independently.
- Carry the confirmed definitions and generated artifacts into their respective release phases. Keep the preview's full candidate separate from any phased publication. If a later implementation choice changes a reviewed public shape, regenerate the explorer and review the difference before continuing with that change.
- Keep project catalog changes and their generated client projections together. Use `python3 scripts/project-catalog --check`. Run AppView query and projection integration tests after changing filter arguments, search paths or result hydration.
- Release AppView support before enabling each phase's new client writes. Reuse the current release process and document the AppView compatibility version. A rollback target for either client phase must be able to read and accept records already written by that phase.
- Preserve one post collection, existing link and URI identity, owner and media boundaries, current filter and visibility contracts and conditional write behavior. Keep old schedule payload conversion, data rewrites, deployment and production operations outside this implementation scope.

## 11. Risks and open questions

| ID | Type | Risk or decision | Resolution |
|---|---|---|---|
| CPQ-001 | Schema checkpoint | New group, field and list bounds and optional measurement wire keys are still to be finalized. | Use existing bounds and shapes where applicable, expose new limits in the local explorer and ADR, and exercise normal values and boundaries in UT-005. Preserve existing item bounds for both tool-size lists. |
| CPQ-002 | Rollout check | The installed beta builds to exercise remain unverified. | Identify the supported legacy mobile and web artifacts and run MAN-001. Use MAN-002 for the version ordering evidence. |
| CPQ-003 | Non-blocking | Additional detail paths could affect search or filter cost. | Verify query plans with representative data. Keep the initial JSONB approach. Propose a separate stored projection or expression index if evidence warrants it. |
| CPQ-004 | Implementation guardrail | Generated editor data and client diffs can omit unknown group fields or empty lists. | Keep raw records authoritative, preserve retained group references, strip editor-only keys, and prove both shapes and edits with IT-002, IT-003, REG-001 and REG-002. |
| CPQ-005 | Blocking before behaviour work | The maintainer needs to confirm the generated schema in the Project Schema explorer. | Complete the §10 checkpoint and record explicit confirmation before starting IT-002 or the rest of implementation. |

## 12. Handoff to TDD implementation

- This plan and the revised acceptance specification are the contract for `05-implementation-plan.md`. Start with the §10 schema checkpoint once work is authorized. Hand back the generated Project Schema page for confirmation.
- After confirmation, start **IT-002** in the existing AppView command and HTTP integration suites. Use `TestEditPostLegacyToolSizes` for candidate assertions and extend `TestPostEditHTTPCommandAndProjectionConverge` for the readback contract. Inspect the returned editor seed, captured PDS candidate and indexed read independently.
- Run the focused AppView command after the checkpoint, from the repository root with the development PostgreSQL and MinIO services available.

```sh
./scripts/appview-test focused ./internal/api ./internal/index -run 'TestEditPost|TestPostEditHTTP'
```

- Follow with the full `just appview-check`, focused `just app-test` suites for Flutter editor, draft, schedule and preference behaviour, and SvelteKit `npm run check`, `npm run test` and `npm run test:release` from `web-client/` as the relevant slices land. Run generated mappers and localization through the project's existing `build_runner` and Flutter localization commands.
