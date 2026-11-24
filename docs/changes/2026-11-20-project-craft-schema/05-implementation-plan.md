# Project craft schema TDD implementation plan

## Approved simplification implementation

| Order | Tests | Requirements | Work |
|---|---|---|---|
| 1 | SIM-AT-01 | SIM-001, SIM-002 | Build and test the schema explorer and generated contracts. |
| 2 | SIM-AT-02, SIM-AT-03 | SIM-001–SIM-004, SIM-007 | Verify shared and custom authoring, retained legacy records and pattern facets. |
| 3 | SIM-AT-04 | SIM-005, SIM-007 | Verify method, style and loom fields with current and old data. |
| 4 | SIM-AT-05 | SIM-006, SIM-008 | Verify filters, keyword matching and craft availability. |

See the [coding plan](04-coding-plan.md#plan-for-the-approved-simplification) for boundaries and component ownership.

### Simplification evidence

- The Lexicons and generated bindings retain the original fields and add the approved item suggestions and active craft details. The four built-site browser tests verify the grouped item subtypes, the Sewing inventory, the five-pattern limit, and the optional legacy view. The generated catalogue and embedded schemas pass their drift checks.
- AppView applies the new browse and keyword rules using the existing project detail storage. PostgreSQL tests cover broad Bag and exact Tote, custom subtype search, the old item projections, and method fallback. Published-edit tests cover saving and clearing custom text while preserving older fields and extensions. The scheduled worker tests exercise create, update, reopen and retry through indexed publication. The change requires no database migration or bulk rewrite.
- Flutter static analysis passed. The focused suite passed 292 cases with three feature-flag skips, and eight project card checks passed. The widget tests exercise the simplified composer, selected and custom subtypes, older values, and the absence of retired controls. The subscription-enabled schedule, draft, onboarding and profile verification passed 157 cases. The free-tier upsell settle timeout also reproduced on the unchanged baseline.
- The Svelte production build, diagnostics, lint, and 635 unit tests passed. The 179-case authenticated run covers custom subtype draft reopening, the method and style distinction, scheduled Knitting methods, five-pattern publication and editing, and filter navigation. Three focused browser cases verified that older Crochet and Felting methods can be extended and cleared and that independent Embroidery filters reopen correctly. Twenty guest browser cases passed.

The earlier evidence below applies to the original additive rollout.

## Earlier implementation plans and evidence

## Inputs

- [Requirements](01-requirements.md) · [Acceptance tests](02-acceptance-tests.md) · [Document review](03-document-review.md) · [Coding plan](04-coding-plan.md#schema-review-checkpoint)

## Implementation rules

- The maintainer approved the revised schema, including multiple patterns and the relationship between Embroidery methods and styles, and authorized implementation.
- Work through the schema checkpoint first. Draft and generate the additive public schemas for both phases and verify the local explorer.
- Add focused cases before schema and explorer changes. Run the same focused commands after the changes, then verify the complete checkpoint.
- Begin AppView behaviour and client work after the maintainer confirms the rendered schema. Keep the two release phases distinct.

## Test order

| Step | Test | Requirements and criteria | Expected initial failure |
|---|---|---|---|
| 0a | UT-005 schema checks | FR-002, FR-012, NFR-002, RULE-002 · AC-002, AC-013, AC-019 | The old schema accepts invalid values in the proposed size and pattern fields as unknown properties. |
| 0b | Project Schema browser smoke | FR-002, FR-004–FR-007, FR-012 · AC-002, AC-004, AC-006–AC-008, AC-019 | The old explorer lacks the new fields and craft variants. |
| 1 | IT-002 and REG-001 | FR-003, FR-010 · AC-003, AC-015 | Legacy edits leave old scalars and can overwrite canonical multiple sizes. |
| 2 | UT-001, UT-005, IT-001 and AT-002 | FR-002, FR-003 · AC-002, AC-003 | Legacy creates store scalars and clients only edit one size. |
| 2a | AT-009, UT-006, IT-005 and IT-008 | FR-008, FR-012, FR-013 · AC-019–AC-021 | Pattern authoring, editing and discovery handle one object. |
| 3–7 | Grouped fields, method filters, lifecycle checks and craft expansion in the [approved order](04-coding-plan.md#9-test-implementation-plan) | See the [coverage matrix](02-acceptance-tests.md#2-requirement-coverage-matrix). | Existing editors and discovery cover the original craft inventory. |

## Schema review execution

- **UT-005**. Exercise tool-size and pattern list bounds against the previous embedded schema. Draft the additive Lexicons and required ADR, regenerate Go and CBOR types and the validation catalog, then rerun the focused cases.
- **Project Schema smoke**. Exercise the built page using shared patterns, representative grouped fields and all new craft tabs. Share the generated local HTML for schema review.

## Schema review evidence

- Against the pre-change validation catalog, the new tests failed because it accepted malformed size lists and grouped craft fields. The generated candidate rejects those values and preserves existing scalar sizes and bounded unfamiliar entries.
- `go -C appview test ./internal/lexicon/schema ./internal/lexicon/craftsky ./internal/sourcevalidation -count=1` passed. The checks include list and item bounds and generated JSON and CBOR round trips for Knitting, Embroidery and Weaving.
- The generated encoders omit empty optional arrays. The AppView compatibility work needs the original field presence to distinguish explicit empty tool-size and pattern lists from absent lists.
- Generation and the regeneration diff check passed using the checked-in recipe steps and a binary built from the exact Indigo version pinned in `appview/go.mod`. The standard `go run …@version` step could not reach the Go module proxy in the sandbox.
- The explorer regression failed against the pre-change page at the missing Knitting size list. `just web-build` generated `web/dist/project-schema.html` and both Playwright schema tests passed against that build. The tests cover separate Hand and Machine options, nested Knitting and Embroidery selections, and all six new craft variants.
- The [draft ADR](../../adr/029-additive-project-craft-details.md) records the proposed bounds and measurement fields. The explorer displays shared Lexicon design tags. The [requirements](01-requirements.md#design-and-style-tags) list the planned suggestions for each craft.

### Multiple-pattern amendment

- Against the first schema draft, validation accepted a sixth entry and malformed lists, the generated JSON and CBOR round trip dropped `patterns`, and the browser smoke test could only find the legacy field.
- The revised draft validates zero to five existing pattern objects, retains the legacy shape, and rejects malformed entries. The generated round trip preserves order, an entry's metadata and its hashtag facet. The three Playwright smoke tests passed, including navigation from `patterns` to its shared object definition.
- Regeneration and the diff check passed using the same pinned local generator. The [coding plan](04-coding-plan.md) covers runtime compatibility, search, and both editors after schema confirmation.

## Runtime implementation evidence

- **IT-002**. The legacy tool-size command tests failed on the unchanged handler. It stored a legacy scalar, accepted a destructive scalar edit against multiple sizes, and rejected the plural edit field. The focused command test passed after adding presence-aware compatibility and guarded writes.
- **IT-008 and IT-003**. AppView command tests pass for retained pattern metadata, legacy singleton writes and clears, rejection of destructive edits and retained nested Embroidery and Weaving fields.
- **IT-002 and IT-008**. The isolated PostgreSQL owner-editor suite verified that public and private reads expose the effective size and pattern arrays. Only a singleton also exposes a legacy field. Existing entry indexes are available in the owner seed for retained edits.
- **IT-004 and IT-008**. Schedule request and publication tests preserve the order of arrays and lossless raw project metadata. Publication mention tests cover deduplication and a credit unique to a later pattern.
- **IT-005 and IT-006**. Focused PostgreSQL tests exercise effective pattern keyword and filter matches, Knitting and Felting method membership, and visibility. Project request validation rejects mismatched Spinning details and more than five patterns.
- **REG-004**. One catalog setting selects the release phase for AppView and both clients. The first phase offers the five existing crafts and retains the identifiers and details for known records. A temporary Phase 1 catalog generation and the Svelte Phase 1 compatibility tests exercised those offers, preferences and old filter URLs. The worktree keeps both phases enabled.
- The project-specific AppView tests passed against PostgreSQL and MinIO with the race detector. The broader PostgreSQL discovery, profile and taxonomy selection also passed.
- The full `just appview-check` gate passed on the corrected source, including embedded Lexicon regeneration, the complete standard and race suites, and the packaged release image checks.
- The Phase 1 AppView catalog overlay verified offer and filter gating against PostgreSQL and passed project write and edit regressions. Existing Phase 2 records remained readable in keyword search. The Phase 1 Flutter selection passed 61 editor, filter and preference tests before restoring the combined build catalog.
- Flutter static analysis completed without issues. The combined Flutter candidate passed 429 tests across project authoring, published editing, schedules, profile preferences, onboarding and discovery, with two existing subscription-gated skips. A regression test verifies that an unrelated edit preserves a published measurement string outside the first-party input syntax. There was no running debug app for hot reload.
- Svelte passed type and style checks, production build and 623 unit tests, including the three Phase 1 regressions. Browser fixtures verified patterns, tool sizes, grouped techniques, specialist details, filters and preferences. The first release run found two five-craft expectations and a single-pattern detail regression. After the fixes, all five affected and related cases passed when rerun.
- The full Svelte release gate passed. Preview and Worker modes each passed 581 authenticated and 152 guest scenarios, with the existing conditional skips. The final build restored the standard web configuration after the fixture runs.

## Corrections after implementation review

The maintainer authorized the fixes in the [implementation review](06-implementation-review.md). Work runs in independent AppView, Flutter and Svelte test loops, with browser lifecycle coverage against the shared authenticated fixture.

| Order | Finding and test | Requirements and criteria | Expected failure or coverage gap |
|---|---|---|---|
| 1 | IR-001 · IT-008, AT-009 | FR-009, FR-012, RULE-002 · AC-019, AC-020 | A Svelte edit drops a retained row or changes an unrelated facet. |
| 2 | IR-002, IR-003 · IT-003, AT-004 | FR-005, FR-006, FR-009, RULE-002 · AC-006, AC-010 | Clearing the last Flutter detail is ignored, or a method toggle drops an untouched group. |
| 3 | IR-004, IR-005 · REG-002, UT-005 | FR-009, FR-012, RULE-002 · AC-010, AC-013, AC-020 | Unchanged Macramé cord values block saves, or explicit null patterns are accepted. |
| 4 | IR-006–IR-008 · IT-005, UT-002, REG-002 | FR-006, FR-008, FR-009, NFR-001, RULE-002 · AC-007, AC-012, AC-018, AC-021 | Eligible pattern hashtags, browse choices or unfamiliar saved values differ across clients. |
| 5 | IR-009 · IT-003, IT-004, AT-008, AT-009 | FR-009, FR-012 · AC-010, AC-014, AC-019 | Tests stop before published readback, use a standard schedule fixture, or stop at a local draft. |

### Focused correction evidence

- **IR-001, IT-008**. Svelte model tests reproduced omission of an indexed extension-only row, regeneration of an unchanged tag facet and an empty mention DID on a newly added row. Pattern patches now keep retained identities and annotations. Added rows resolve mentions without using the removed row at the same position. The AppView edit command regression verifies that the publication candidate retains the untouched extension-only row and the original facet.
- **IR-002, IT-003**. Flutter's edit patch test reproduced the missing clear when removing the last Embroidery method. It now emits the specialized field clear and preserves unrelated extensions.
- **IR-003, AT-004 and IT-003**. Retained-group tests reproduced a missing Machine group in Flutter and two copies of the first Machine style in Svelte after removing Hand. Both editors now preserve and submit the original indexed Machine groups. The AppView HTTP integration verifies stored and indexed Knitting subtype edits, an Embroidery method removal with an untouched Machine extension, and Macramé detail creation and a cord update.
- **IR-004, REG-002**. The Svelte regression failed when editing the second Macramé cord caused an untouched saved diameter to fail decimal validation. Validation now compares diameters by retained row identity. Edited and new diameters still use the decimal rule.
- **IR-005, UT-005 and IT-008**. Immediate and scheduled request tests went red for an explicit null list. An explicit null now fails validation, an omitted list still reads a legacy singleton, and `[]` takes precedence. The published PATCH tests also cover the existing null clear convention.
- **IR-006, IT-005 and IT-008**. The Flutter regression reproduced a lost later-pattern hashtag when a saved link covered the earlier `#credit` text. The prior adapter already emitted a tag for plain credit text. The corrected budget counts eligible facets in field order and retains authored facets on unchanged text. The Svelte test checks credit and later-pattern facet eligibility, and the AppView schedule test verifies the indexed tenth tag.
- **IR-007, UT-002 and IT-005**. The Svelte browse tests exposed Hand in the Knitting method filter and generic project types for Weaving. The corrected filters offer Machine in Knitting, Needle in Felting and types for the relevant crafts, including Punch Needle. Knitting authoring continues to offer Hand and Machine.
- **IR-008, REG-002**. The Flutter owner editor initially hid the unfamiliar saved loom-type choice. The widget regression now verifies it displays, survives a caption-only edit, and can be reopened and cleared.
- **IR-009, AT-008 and AT-009**. Svelte Playwright local fixtures cover five ordered patterns through publish, edit, remove and reload, including preserved links and facets. The fake AppView now returns retained indexes in the edit seed and applies the pattern changes to the fixture record for readback. The scheduled Knitting journey verifies two sizes, two Colourwork subtypes and two patterns at save, reopen and the publish-now request.
- **IR-009, IT-003 and IT-004**. The isolated PostgreSQL 16 tests passed for grouped owner HTTP edits and current scheduling create, update and reopen. The worker test verifies Knitting and Macramé indexed readback with a simulated lost PDS response and one effective publication. It also verifies discovery of the later Knitting pattern tag in the tenth hashtag slot.

### Correction verification

- `just appview-check` passed, including regenerated Lexicon verification, the standard and race suites, and packaged startup checks. Seven focused tests covering project validation, compatibility and the published and scheduled paths also passed with the race detector against isolated PostgreSQL 16. Four AppView edit, validation and discovery tests passed using the Phase 1 catalog overlay against PostgreSQL.
- Flutter analysis completed without issues. The project, owner-edit and scheduling group passed 405 tests with nine existing skips. The onboarding, profile editor and shell group passed 141 tests. The Phase 1 catalog selection passed 71 tests. The restored Phase 2 catalog passed the generation check. There was no Dart Tooling Daemon connection available for hot reload.
- `just public-app-check` passed with 627 unit tests. Preview and Worker each passed 584 authenticated and 152 guest browser cases, plus the release-specific suites. Each authenticated run had three conditional skips. Both modes passed the new published pattern, grouped Embroidery edit and scheduled Knitting scenarios. The final production build completed.
- The change catalog includes the review and correction evidence. Catalog generation, `just docs-check` and the whitespace check passed.

## Corrections after the second review

The maintainer authorized the corrections in the [second review](06-implementation-review.md#findings). Keep the recorded annotation on an unchanged pattern field and apply the ten-tag limit to discovery in saved order. Reopened current schedules use their own saved text and annotations as the baseline.

| Finding and test | Linked requirements and criteria | Regression |
|---|---|---|
| IR-001 · IT-008 | FR-009, FR-012, RULE-002 · AC-019, AC-020 | Reverting a change to an indexed pattern field must keep its authored facet when a sibling is saved. |
| IR-006 · IT-005 | FR-008 · AC-021 | An unchanged caption's link over hashtag-shaped text leaves the tenth slot available to a later pattern. |
| IR-010 · AT-008 and IT-004 | FR-009, FR-012 · AC-019 | Saving and publishing a reopened current schedule preserves unchanged pattern facets even if a handle no longer resolves or a tag crosses the discovery budget. |
| IR-009 · IT-003 | RULE-003 · AC-014 | A searchable project detail on a pending schedule appears in public reads and search only after publication. |

### Follow-up evidence

- **IR-001, IT-008**. The Svelte five-pattern browser case initially sent `nameFacets: null` after the author restored the original text and edited a sibling. The corrected PATCH retains the row without touching the saved annotation. Reloading verifies the saved link and facet are still present.
- **IR-006, IT-005**. The Svelte browser and Flutter published editor regressions initially omitted the later tag. Both corrected saves budget from the retained caption facets, preserve the unchanged caption byte for byte, and add the tenth tag. The AppView search fixture indexes nine caption tag facets, a link over `#credit`, a `lace` tag on a later pattern and a saved over-budget tag. Only `lace` finds the post among those three queries, and the source record retains the over-budget annotation. Both index budget tests passed with the race detector against isolated PostgreSQL 16.
- **IR-010, AT-008**. Svelte schedule saves initially replaced stored sibling annotations after a handle stopped resolving. The browser regression now removes an earlier row, edits a sibling, preserves an unchanged caption and its link, saves, reopens and publishes with the original annotation intact. The saved and newly entered identical handles can resolve to different DIDs. The browser retains false-valued extension data, omits fresh blank rows and strips editor row indexes before sending the payload. The private draft snapshot carries the baseline for recovery.
- **IR-010, AT-008**. The Flutter schedule widget regression initially changed the byte range and tag of an untouched pattern and dropped a restored mention. It also reproduced trimming of an unchanged caption and pattern name. The corrected flow preserves the same saved text and annotations through row removal, save, reopen and Post now. It generates tags and mentions for newly entered text and sends the project without editor row indexes.
- **IR-009, IT-003**. The isolated fixture checks the exact future post URI and a unique Knitting `finishedSize` through the guest read and project search handlers. The pending schedule returns 404 and zero search results. After the existing lost-response recovery produces one PDS write, the project and image-safety projection exposes that same URI and detail through both public routes. The focused cases and full scheduled worker package passed against PostgreSQL 16, including the package race run.

### Follow-up verification

- Flutter analysis passed. The default project, composer, published editor and scheduling suite passed 406 tests with ten conditional skips. The Phase 1 catalog selection passed 117 applicable cases. The Weaving-specific authoring case ran in Phase 2, and the restored Phase 2 catalog passed regeneration verification. The new AT-008 lifecycle case passed in a selection of 25 subscription-enabled schedule editor tests and on the final source. A wider subscriptions-enabled run reproduced two failures on the pre-change source using the same runtime. They concern the free-tier scheduling upsell and standard-post thumbnail staging. The environment had no Dart Tooling Daemon or debug app for hot reload.
- AppView production code matches the preceding full release-gate run. The new AppView tests passed against PostgreSQL 16, including the scheduled worker race suite and the indexer hashtag race cases. `go vet` passed for both affected packages.
- The Svelte release run passed formatting, lint, diagnostics, 627 unit tests, the Preview suites and all project annotation cases in both Preview and Worker. The Worker authenticated suite passed 584 cases and had three conditional skips. Its two failures were a browser teardown timeout in the capped unread badge case and a loading skeleton style assertion in the following case. Both cases passed on an exact Worker rerun. All 152 Worker guest cases and the remaining beta case passed, and the standard production build completed.
- The project taxonomy and change catalog passed generation checks after the worktree returned to Phase 2. `just docs-check` and `git diff --check` passed.

## Flutter pattern text reversion

The [latest implementation review](06-implementation-review.md#findings) reopened IR-010 for a current schedule saved with a trailing space in a faceted pattern field. AT-008 covers exact text and facet retention in the first and additional pattern editors under FR-009, FR-012, RULE-002 and AC-019.

- **Red.** The widget test restored a saved `#wool-blue ` name after moving its pattern to first position and restored a saved `@maker.test ` designer after reopening an additional pattern editor. Saving trimmed both strings, changed the hashtag to `wool` and dropped the mention of the unresolved handle.
- **Green.** Both editors compare with the original saved text for the same pattern. The test verifies the exact text and saved annotations after save, reopen and Post now. A newly added pattern still normalizes its text and generates its own hashtag and mention.
- **Verification.** Flutter analysis passed. The Phase 2 project, composer, published editor and scheduling suite passed 406 tests with ten conditional skips. The subscription-enabled schedule editor run passed 25 tests after excluding the previously reproduced free-tier upsell timeout. The targeted AT-008 test also passed under Phase 1. No debug app was attached for hot reload.

## Rollout evidence pending

- MAN-001 uses the actual supported legacy mobile and web builds. Local contract tests and editor fixtures replay compatible requests and verify protected saves.
- MAN-002 records AppView version readiness before client writes in each release phase.

## Completion checklist

- [x] Additive schema cases and generated code pass verification.
- [x] The local Project Schema explorer opens and shows the candidate details for both phases.
- [x] Maintainer confirms the schema for behaviour implementation.
- [x] AppView, Flutter and Svelte implementation and focused verification.
- [x] Full AppView and Svelte release gates.
- [x] Implement the nine review findings and verify the corrections locally in both release phases.
- [x] Complete the four follow-up findings from the second implementation review and verify the corrected paths.
- [x] Verify exact restoration of saved Flutter pattern text and annotations after a row is promoted or an editor is reopened.
- [ ] Manual rollout checks for installed builds and deployment order.
