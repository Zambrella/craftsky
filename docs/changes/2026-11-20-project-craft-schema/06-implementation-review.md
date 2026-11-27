# Implementation Review: Project craft schema expansion

## Review scope for the simplified schema

The earlier implementation review below covers the additive project schema. The new authoring inventory, app filters, and saved legacy-field behaviour are covered by SIM-AT-01 through SIM-AT-05 in the updated acceptance specification. Record verification of the revised behaviour in the implementation plan.

## Earlier implementation review

## Verdict

| Review field | Result |
|---|---|
| Status | **Approved** |
| Reviewer | Codex, with an independent Flutter code review |
| Date | 2026-11-19 |
| Risk level | High |

## Summary

IR-010 is closed. Flutter preserves the exact saved text and its annotations when the author restores a pattern field, including after promoting an edited row into the first slot or reopening the pattern panel. The scheduled save and Post now requests preserve the original hashtag span, mention span and DID. The previous reviews verified the Svelte and AppView corrections.

## Findings

None identified. The [third review record](#third-review-record) documents the original reproduction.

## Requirement And Test Traceability

| Finding | Verified outcome |
|---|---|
| IR-010 · FR-009, FR-012, RULE-002 · AC-019 · AT-008 | Both inputs compare raw text with the saved row before normalization. The editor and submit adapter use the same retained index, which survives removal of an earlier row. The regression verifies the restored first pattern name and an additional designer credit, then checks both again on Post now. The new row still generates its own facets from normalized text. [First pattern payload](../../../app/lib/projects/composer/project_composer_payload.dart#L430) · [Additional pattern editor](../../../app/lib/projects/widgets/project_pattern_editor.dart#L49) · [Scheduled facet adapter](../../../app/lib/projects/composer/project_composer_submit_adapter.dart#L98) · [Flutter AT-008](../../../app/test/scheduled_posts/scheduled_post_edit_test.dart#L1526) |
| IR-001–IR-009 · AT-004, AT-008, AT-009, UT-002, UT-005, IT-003–IT-005, IT-008, REG-002 | The prior review and correction records cover retained rows, validation, discovery, published editing, and public visibility after scheduled publication. This recheck focused on the only reopened finding, in Flutter. [Correction evidence](05-implementation-plan.md#corrections-after-implementation-review) · [Svelte and AppView rechecks](#third-review-requirement-and-test-traceability) |

## Test Evidence

| Evidence | Result |
|---|---|
| Flutter review rerun | The subscription-enabled AT-008 widget test passed on the Phase 2 catalog. It asserts the complete pattern list in the Save Schedule and Post now payloads, including order and facet bytes. |
| Flutter implementation verification | AT-008 also passed with the Phase 1 catalog. Static analysis passed. The broader Phase 2 suite passed 406 tests with ten conditional skips. The subscription-enabled schedule editor selection passed 25 tests and excluded the previously reproduced free-tier upsell timeout. No debug app was attached for hot reload. [Correction evidence](05-implementation-plan.md#flutter-pattern-text-reversion) |
| AppView and Svelte verification | Previous checks cover AppView worker recovery and public visibility on PostgreSQL 16 and the Svelte scheduled annotation case in both Preview and Worker. [Prior review evidence](#third-review-test-evidence) |

The focused rerun used `flutter test --no-pub --dart-define=SUBSCRIPTIONS_ENABLED=true test/scheduled_posts/scheduled_post_edit_test.dart --plain-name 'AT-008 scheduled pattern annotations survive editing and reopening'` from `app/`.

## Risk Review

The remaining release checks are MAN-001 for single-size and single-pattern compatibility on installed legacy builds and MAN-002 for AppView readiness before client writes in each phase. The local review supports merge. [Acceptance specification](02-acceptance-tests.md#8-manual-checks) · [Rollout evidence](05-implementation-plan.md#rollout-evidence-pending)

## UI Polish Recommendation

**Not needed.** The correction preserves saved text and its annotations using the existing controls.

## Handoff Back To TDD Builder

No required corrections. Carry MAN-001 and MAN-002 into rollout.

## Third review record

<details>
<summary>Findings and evidence before the Flutter exact-text correction</summary>

### Third review Verdict

| Review field | Result |
|---|---|
| Status | **Changes required** |
| Reviewer | Codex, with independent Flutter, Svelte and AppView rechecks |
| Date | 2026-11-19 |
| Risk level | High |

### Third review Summary

IR-001, IR-006 and IR-009 are closed. Svelte passes scheduled annotation retention, and Flutter passes the existing save, reopen and Post now regression. A Flutter input path still drops a valid saved mention when the author restores a pattern credit whose exact saved text ends with whitespace.

### Third review Findings

| ID | Severity | Area | Finding | References | Required action |
|---|---|---|---|---|---|
| IR-010 | Important | Restored pattern credit in Flutter | Save a current schedule with a second pattern designer of `@maker.test ` including the trailing space, and an authored mention with bytes 0–11 and DID `did:plc:maker`. The current Svelte editor can save this text. Open the schedule in Flutter, change the designer and then restore the exact original string. Change the pattern name and save. The input trims the credit before the adapter compares it to the saved text. With an unresolved handle, the actual schedule request replaces the original with `{"name":"New cuffs","designer":"@maker.test"}` and drops its saved mention. | FR-009, FR-012, RULE-002, AC-019, AT-008. [Svelte Designer input](../../../web-client/src/lib/features/composer/ProjectEditor.svelte#L443-L446) · [Flutter pattern inputs](../../../app/lib/projects/widgets/project_pattern_editor.dart#L47-L108) · [Flutter first pattern input](../../../app/lib/projects/widgets/project_composer_sheet.dart#L2006-L2017) · [Facet matching](../../../app/lib/projects/composer/project_composer_submit_adapter.dart#L170-L217) · [Schedule regression fixture](../../../app/test/scheduled_posts/scheduled_post_edit_test.dart#L1526) | Compare raw input to the saved text for its retained row before trimming. Keep the original text, byte ranges and DID when the author restores that value. Keep normalizing newly entered or genuinely changed text. Extend the schedule regression to cover restoring faceted text with trailing whitespace, including when it appears in the first pattern field, then assert exact wire retention through save, reopen and Post now. |

### Third review Requirement And Test Traceability

| Finding | Recheck evidence |
|---|---|
| IR-001 · IT-008 · AC-019 and AC-020 | Closed. The Svelte five-pattern browser test types and reverts a first-row name, edits a sibling and removes another. The PATCH retains the first row by index without editing its facets. Reload verifies the original hashtag and link remain usable. [Browser test](../../../web-client/tests/authenticated-project-detail-parity.spec.ts#L202) |
| IR-006 · IT-005 · AC-021 | Closed. Both published editors send the later `lace` tag with the unchanged caption omitted from the patch. The AppView PostgreSQL test indexes that tag in slot ten, excludes the link text and overflow tag from search, and verifies the stored overflow annotation. [Svelte browser test](../../../web-client/tests/authenticated-project-detail-parity.spec.ts#L309) · [Flutter owner edit](../../../app/test/feed/widgets/published_post_editor_test.dart#L51) · [AppView index test](../../../appview/internal/index/craftsky_post_test.go#L2355) |
| IR-009 · IT-003 · AC-014 | Closed. The guest direct-read and search handlers return 404 and zero matches for the scheduled-only project detail. After worker retry reconciles one PDS write and the production projector runs, the same URI and saved detail appear in both handlers. [AppView acceptance test](../../../appview/internal/scheduledposts/public_visibility_test.go#L21) · [Worker fixture](../../../appview/internal/scheduledposts/export_test.go#L21) |
| IR-010 · AT-008 and IT-004 · AC-019 | Svelte passes exact annotation and caption preservation through PUT, reopen and Publish now, including a shifted row and a new credit using the same handle. Flutter passes the current lifecycle fixture. The fixture restores a designer without trailing whitespace, which leaves the reproduced Flutter case open. [Svelte schedule test](../../../web-client/tests/authenticated-app.spec.ts#L847) · [Flutter lifecycle test](../../../app/test/scheduled_posts/scheduled_post_edit_test.dart#L1526) |

The checked changes follow the original ownership and release design. MAN-001 and MAN-002 remain rollout checks for supported installed builds and AppView readiness in both phases. [Requirements](01-requirements.md) · [Acceptance tests](02-acceptance-tests.md) · [Coding plan](04-coding-plan.md) · [Correction evidence](05-implementation-plan.md#corrections-after-the-second-review)

### Third review Test Evidence

| Evidence | Result |
|---|---|
| Svelte recheck | 32 targeted unit tests and all three focused Preview browser tests passed. The implementation run also passed the same project cases in Worker. The two unrelated Worker failures passed on exact rerun, and the remaining guest, beta and production build steps completed. |
| Flutter recheck | Three focused subscription-enabled tests passed for the owner PATCH, saved-link budget and current schedule lifecycle. The implementation run also recorded clean analysis, 406 focused passes and 117 Phase 1 passes. |
| Flutter review probe | A temporary copy of the existing AT-008 widget test added one trailing space to the saved designer and restored input. The rest of the fixture was identical. The test failed on the captured schedule payload because `designerFacets` was missing and `designer` was trimmed. |
| AppView recheck | Five focused tests passed with the race detector on isolated PostgreSQL 16, including public visibility, worker retry and hashtag discovery. AppView production files match the preceding full release-gate run. |
| Documentation | The project catalog and whitespace checks passed. `just docs-check` passed before the review update. The updated artifact made the generated change catalog stale. Refresh it in the next implementation update. |

### Third review Risk Review

Saving a restored credit can discard its valid saved attribution before the worker receives it. The failure affects a current Svelte-to-Flutter schedule, within the agreed scheduling scope. The tested public visibility and one-publication recovery continue to work. Installed legacy-client compatibility and deployment order remain rollout checks. [Pattern facets ADR](../../adr/006-pattern-metadata-facets.md) · [Project record ADR](../../adr/029-additive-project-craft-details.md)

### Third review UI Polish Recommendation

**Not needed.** The open item concerns persisted text and mention identity.

### Third review Handoff Back To TDD Builder

1. Use the existing Flutter AT-008 schedule test to reproduce a retained Designer value of `@maker.test ` after a temporary edit and exact reversion. Assert the saved state and published payload against the original facet.
2. Verify the equivalent first pattern field and the additional pattern editor. Keep byte-exact saved inputs until comparison with the retained baseline, then normalize new or edited values.
3. Rerun the Flutter pattern payload, published-edit and schedule tests. Keep the AppView and Svelte results as evidence for the closed findings.

</details>

## Second review record

<details>
<summary>Findings and evidence before the latest correction pass</summary>

### Second review Verdict

| Review field | Result |
|---|---|
| Status | **Changes required** |
| Reviewer | Codex, with independent Flutter, Svelte and AppView rechecks |
| Date | 2026-11-19 |
| Risk level | High |

### Second review Summary

The original failures involving retained pattern rows, grouped details, field clears and null-list validation have passing regressions. The recheck found additional paths that can lose valid pattern annotations or suppress a discoverable hashtag. The published-edit and scheduled-publication fixtures cover the main lifecycles. The project-detail assertion before scheduled publication is still open.

### Second review Findings

| ID | Severity | Area | Finding | References | Required action |
|---|---|---|---|---|---|
| IR-001 | Important | Published pattern annotations | In Svelte, edit a saved pattern name with an authored facet, restore the exact original text, then change a sibling pattern. The input has removed the facet from local state. Submission detects the restored text but keeps the now-empty local facet value. The owner PATCH includes `nameFacets: null` for the restored row, removing the original annotation from the published record. Designer and publisher fields use the same path. | FR-009, FR-012, RULE-002, AC-019, AC-020, IT-008. [Pattern input](../../../web-client/src/lib/features/composer/ProjectEditor.svelte#L134-L140) · [Pattern facet processing](../../../web-client/src/lib/features/composer/model.ts#L481-L504) · [Entry patch](../../../web-client/src/lib/features/composer/model.ts#L668-L691) | Restore the saved facets for the matching retained row when its text matches the saved value. Verify that typing and reverting before editing a sibling leaves the original field and facet out of the patch and preserves the authored byte range and feature. |
| IR-006 | Important | Hashtag discovery on published edits | In Svelte, open a project whose caption has nine saved hashtag facets plus the text `#credit` covered by a link facet. Change only a later pattern name to `Other #lace`. The editor budgets from raw caption text, so `#credit` takes the last slot even though the unchanged caption keeps its original link facet. The PATCH saves the new pattern text without a `lace` tag facet. The equivalent Flutter path budgets from the retained caption facets. | FR-008, AC-021, IT-005. [Svelte budget input](../../../web-client/src/lib/features/composer/ComposerWorkspace.svelte#L1003-L1015) · [Caption edit rules](../../../web-client/src/lib/features/composer/model.ts#L557-L565) · [Flutter facet budget](../../../app/lib/projects/composer/project_composer_submit_adapter.dart#L39-L92) | Budget from the eligible caption facets the record will actually keep or receive. Verify the saved link survives and the new pattern facet uses slot ten on a caption-unchanged owner edit. |
| IR-009 | Important | Prepublication acceptance evidence | IT-003 asks for an unpublished project detail to be absent from public reads and search. The search privacy fixture uses an unpublished caption marker. The new HTTP and worker project fixtures inspect schedule and published data, leaving the specific prepublication project-detail case uncovered. | FR-009, RULE-003, AC-014, IT-003. [Acceptance case](02-acceptance-tests.md#5-integration-test-cases) · [Search privacy fixture](../../../appview/internal/api/search_improvements_privacy_test.go#L28-L82) · [Project worker fixture](../../../appview/internal/scheduledposts/recovery_acceptance_test.go#L180-L242) | Use a unique marker in a searchable project detail on a pending schedule, confirm it is absent from public reads and search, then confirm it is visible through those paths after publication. |
| IR-010 | Important | Stored schedule annotations | Svelte loads a current schedule with its saved pattern facets. Both **Save schedule** and **Publish now** rebuild the entire project without a saved baseline. An unchanged credited handle that can no longer resolve loses its saved mention DID. A handle that has changed owners can give the annotation a different DID. Flutter also retokenizes saved schedule tags on an unrelated pattern edit. The probe changed a saved `wool-blue` tag at bytes 0–10 into `wool` at bytes 0–5. Both clients send a full replacement payload. | FR-009, FR-012, AC-019, AT-008, IT-004. [Schedule hydration](../../../web-client/src/lib/features/composer/ComposerWorkspace.svelte#L392-L406) · [Schedule serialization](../../../web-client/src/lib/features/composer/ComposerWorkspace.svelte#L830-L843) · [Schedule save and publish branches](../../../web-client/src/lib/features/composer/ComposerWorkspace.svelte#L962-L990) · [Flutter schedule submission](../../../app/lib/projects/widgets/project_composer_sheet.dart#L2640-L2650) · [Flutter facet adapter](../../../app/lib/projects/composer/project_composer_submit_adapter.dart#L60-L89) · [AppView manual publication](../../../appview/internal/api/scheduled_post.go#L332-L340) | Keep saved annotations on unchanged pattern fields in current schedules in both clients. Exercise a sibling edit through save, reopen and publish now with a saved mention whose handle stops resolving and a saved tag with a non-default byte span. Verify the original features, byte ranges and pattern order. |

### Second review Requirement And Test Traceability

| Initial finding | Recheck |
|---|---|
| IR-001 · Retained pattern rows | The extension-only row, untouched custom tag, and new-row mention cases from the initial review are covered. Restoring original text exposes the remaining annotation loss above. [Svelte model regression](../../../web-client/src/lib/features/composer/model.test.ts#L152) · [AppView record candidate](../../../appview/internal/api/post_edit_commands_test.go#L772) |
| IR-002 · Last detail clear | Flutter emits the specialized field clear when the last detail is removed. [Published patch](../../../app/lib/feed/composer/post_edit_changes.dart#L190-L210) |
| IR-003 · Grouped detail retention | Both clients preserve the two saved Machine groups when Hand is removed. The AppView integration verifies grouped fields and a retained extension through the PDS write and indexed read. [Flutter selector](../../../app/lib/projects/widgets/project_craft_fields.dart#L170-L205) · [Svelte owner edit](../../../web-client/tests/authenticated-composer-detail-parity.spec.ts#L1304) · [AppView integration](../../../appview/internal/index/post_edit_http_integration_test.go#L48) |
| IR-004 · Macramé cord edit | Svelte validates changed or newly entered diameters and preserves an unchanged sibling diameter. [Validation regression](../../../web-client/src/lib/features/composer/model.test.ts#L234) |
| IR-005 · Null pattern lists | Create and current scheduling reject an explicit null list, preserve omitted singleton fallback and honor an empty list. Published PATCH retains its explicit clear convention. [Request regression](../../../appview/internal/api/scheduled_post_request_test.go#L281) · [Edit compatibility regression](../../../appview/internal/api/post_edit_commands_test.go#L756) |
| IR-006 · Hashtag budget | Flutter's overlapping saved credit facet regression passes, and AppView indexes a later pattern tag in the tenth slot. Svelte has the separate retained-caption case above. [Flutter adapter](../../../app/lib/projects/composer/project_composer_submit_adapter.dart#L60) · [AppView worker test](../../../appview/internal/scheduledposts/recovery_acceptance_test.go#L180) |
| IR-007 · Browse parity | Knitting offers Machine and Felting offers Needle in the applicable filters. Weaving omits project type and Punch Needle keeps its own types. Knitting authoring retains Hand and Machine. [Svelte filter tests](../../../web-client/src/lib/features/social/filter-model.test.ts#L64) |
| IR-008 · Saved choices | Flutter displays an unfamiliar saved Weaving loom type and allows the author to clear it. [Flutter saved option](../../../app/lib/projects/widgets/project_craft_fields.dart#L144-L160) |
| IR-009 · Lifecycle tests | AppView grouped edits, schedule HTTP reopening and worker retry have coverage. Svelte browser fixtures publish five patterns, edit and remove one, and save and reopen a revised project schedule. The prepublication assertion above remains. [Schedule HTTP test](../../../appview/internal/api/scheduled_post_test.go#L215) · [Svelte pattern browser test](../../../web-client/tests/authenticated-project-detail-parity.spec.ts#L202) · [Svelte schedule browser test](../../../web-client/tests/authenticated-app.spec.ts#L711) |

The public schema and release design remain consistent with the approved plan. MAN-001 and MAN-002 remain rollout checks for the supported legacy builds and AppView readiness in each phase. [Requirements](01-requirements.md) · [Coding plan](04-coding-plan.md) · [Correction and rollout evidence](05-implementation-plan.md#corrections-after-implementation-review)

### Second review Test Evidence

| Evidence | Result |
|---|---|
| AppView implementation log | `just appview-check` passed the standard and race suites, Lexicon consistency and packaged checks. Focused compatibility, catalog and Lexicon tests passed on recheck. |
| Svelte implementation log | `just public-app-check` passed 627 unit tests. Preview and Worker each passed 584 authenticated and 152 guest browser cases, with three conditional authenticated skips per mode. |
| Svelte focused recheck | All 28 tests in the composer, Phase 1 and filter-model selection passed. An independent selection including the hashtag budget passed 29 tests. |
| Svelte review probes | In-memory runs of the production model reproduced the null annotation PATCH, missing later pattern hashtag and dropped stored schedule facets when mention resolution failed. |
| Flutter implementation evidence | Clean analysis, 546 scoped passes with nine conditional skips, and 71 Phase 1 passes. |
| Flutter focused recheck | The four original Flutter regression tests for explicit clear, grouped retention, hashtag budgeting and saved choice visibility passed. |
| Flutter review probes | Two temporary tests hydrated a current schedule, changed a sibling pattern, and ran the production submit adapter. The first saw the saved tag become `wool`. The second used eight caption tags, then `#first` and `#later` on two pattern entries. Adding `#extra` to the first pattern deleted the untouched second pattern's saved facet from the replacement payload. The original facet should survive while discovery remains capped at ten tags. |
| Documentation and catalog | Generated project catalog verification and `git diff --check` passed. `just docs-check` passed before the review update. It then flagged the generated change catalog as stale. The catalog refresh belongs in the next implementation update. |

### Second review Risk Review

The remaining defects affect persisted project attribution and discovery. A scheduled publish can discard the saved identity of a credited maker before the worker receives the record. The published edit defects can silently change unrelated metadata and are reproducible without contacting a live PDS. The rollout checks cover installed builds and deployment sequencing. [Project record ADR](../../adr/029-additive-project-craft-details.md) · [Pattern facets ADR](../../adr/006-pattern-metadata-facets.md)

### Second review UI Polish Recommendation

**Not needed.** The remaining issues concern saved data and discovery behavior.

### Second review Handoff Back To TDD Builder

1. Add a failing Svelte owner-edit test that restores a pattern's original text and changes its sibling. Verify the original byte range and feature survive on reload.
2. Check the ten-tag budget against an unchanged caption with a link on hashtag-shaped text. Verify the later pattern facet is indexed.
3. Exercise current schedule save and publish now with an unchanged saved mention in Svelte and an unchanged saved tag in Flutter. Verify both survive a sibling edit and add the prepublication project-detail visibility control from IT-003.
4. Rerun the relevant model, API and browser cases and the affected release checks.

</details>

## Initial review record

<details>
<summary>Findings and evidence before the correction pass</summary>

### Initial Verdict

| Review field | Result |
|---|---|
| Status | **Changes required** |
| Reviewer | Codex, with independent Flutter, Svelte and discovery reviews |
| Date | 2026-11-19 |
| Risk level | High |

### Initial Summary

The additive schema, shared catalog and release phases follow the approved design. The AppView and Svelte release gates passed, and the Flutter checks covered both phases. The candidate can still discard saved pattern data in Svelte or silently ignore a published clear in Flutter. The findings below also cover validation, discovery and missing lifecycle evidence.

### Initial Findings

| ID | Severity | Area | Finding | References | Required action |
|---|---|---|---|---|---|
| IR-001 | Critical | Published pattern retention | Editing the second pattern in Svelte can delete a valid first pattern whose only data is an extension field. The owner seed carries its `existingIndex`, but the editor drops the row before sending the replacement list. A sibling edit can also regenerate an unchanged pattern's saved tag facet. A reproduced `#wool-blue` facet over bytes 0–10 changed to `wool` over bytes 0–5 when only the second pattern name changed. A new row in an old position can also inherit the unchanged-credit check and emit a mention with an empty DID instead of resolving the handle. | FR-009, FR-012, RULE-002, AC-019, AC-020, IT-008, AT-009. [Owner seed](../../../appview/internal/api/post_edit_editable.go#L60-L68) · [Svelte pattern diff](../../../web-client/src/lib/features/composer/model.ts#L707-L713) · [Facet regeneration](../../../web-client/src/lib/features/composer/model.ts#L419-L499) · [List merge](../../../appview/internal/api/post_edit_merge.go#L229-L264) | Keep retained rows by identity, even when the editor sees no editable content. Keep facets on unchanged pattern fields when editing a sibling and apply hashtag limits to discovery. Verify the PDS write preserves the extension-only row, its position and the unchanged facet. Compare rows by retained identity. Verify explicit removal and mention resolution on newly added rows. |
| IR-002 | Important | Published detail clears | Removing the only Hand method and style from an Embroidery project in Flutter leaves them saved. The composer emits no detail object after the clear. The same-craft diff handles only baseline keys in that branch, so it sends no clear for `methods`. The same branch can miss the last specialized field on other crafts. | FR-005, FR-006, FR-009, AC-006, AC-010, AT-004, IT-003. [Composer payload](../../../app/lib/projects/composer/project_composer_payload.dart#L168-L175) · [Flutter detail diff](../../../app/lib/feed/composer/post_edit_changes.dart#L193-L235) · [Detail keys](../../../app/lib/feed/composer/post_edit_changes.dart#L278-L282) | Diff specialized fields against the empty same-craft state. Assert a field-level clear in the published patch and check that reopening shows the selection cleared. Preserve unrelated fields and stored extensions. |
| IR-003 | Important | Grouped detail retention | The Embroidery Lexicon allows two distinct saved groups with the same method. In Flutter, removing Hand from a project with Hand plus two indexed Machine groups rebuilds the list from one map entry per method and discards one untouched Machine group and its styles. | FR-005, RULE-002, AC-006, EC-003, IT-003. [Embroidery array schema](../../../lexicon/social/craftsky/project/embroidery.json#L10-L39) · [Flutter grouped selector](../../../app/lib/projects/widgets/project_craft_fields.dart#L162-L188) · [Svelte row selection](../../../web-client/src/lib/features/composer/ProjectCraftFields.svelte#L35-L41) | Apply a method toggle to the selected method and retain the original indexed rows for other methods. Check both clients with two Machine groups and remove only Hand. Assert that both Machine groups, their styles, and extensions stay intact. |
| IR-004 | Important | Unrelated Macramé edits | An existing cord diameter of `approx. 4` is valid public metadata. Changing a second cord's material in Svelte revalidates the first cord's unchanged diameter against the decimal input rule and blocks Save. | FR-009, RULE-002, AC-010, IT-003, REG-002. [Diameter schema](../../../lexicon/social/craftsky/project/macrame.json#L57-L62) · [Svelte validation](../../../web-client/src/lib/features/composer/model.ts#L320-L329) | Apply first-party decimal validation to newly entered or changed diameters by retained row identity. Assert that editing or removing the second cord succeeds and preserves the first diameter. |
| IR-005 | Important | Create and schedule validation | AppView accepts `common.patterns: null` with a valid legacy `pattern` and treats the explicit null as omission. The typed decoder loses field presence before Lexicon validation and enables the legacy fallback. The current schedule decoder uses the same project shape. | FR-012, AC-013, AC-020, UT-005, IT-008. [List schema](../../../lexicon/social/craftsky/project/defs.json#L90-L99) · [Project decoding](../../../appview/internal/api/post_project_json.go#L61) · [Immediate validation](../../../appview/internal/api/post_request.go#L269) · [Schedule decoding and validation](../../../appview/internal/api/scheduled_post_request.go#L143-L206) | Reject malformed arrays at create and schedule boundaries before their representation is lost. Test null, omission with legacy fallback, and an empty list with a conflicting legacy field. Retain the published-edit PATCH clear convention. |
| IR-006 | Important | Pattern hashtag discovery | In Flutter, plain text `#credit` in a designer field spends a hashtag slot, although the adapter emits no tag facet for designer or publisher. With nine caption tags and a second pattern named `Later #later`, the result drops `#later` and indexes only nine tags. Svelte emits tag facets for the credit fields, so the same input also differs across clients. The Flutter budget previously counted raw credit text for a single pattern. Multiple patterns extend the effect to later pattern names. | FR-008, NFR-001, AC-012, AC-021, IT-005, IT-008. [Hashtag budget](01-requirements.md#patterns) · [Flutter budget](../../../app/lib/projects/composer/project_composer_submit_adapter.dart#L55-L65) · [Flutter facet eligibility](../../../app/lib/projects/composer/project_composer_submit_adapter.dart#L167-L187) · [Svelte facets](../../../web-client/src/lib/features/composer/model.ts#L432-L442) | Align facet eligibility and charge the budget only for eligible tag facets in field order. Test both clients and AppView discovery with nine caption tags, a credit field and a later pattern name. Cover mention eligibility and deduplication in the same path. |
| IR-007 | Important | Browse picker parity | Flutter offers Machine in the Knitting method browse filter. Svelte offers Hand and Machine. Svelte also offers project types such as Garment on Weaving, where Flutter hides the filter and AppView has no matching Weaving project type field. | NFR-001, AC-012, UT-002, IT-005. [Flutter filter options](../../../app/lib/projects/options/project_option_catalogs.dart#L87) · [Svelte browse filters](../../../web-client/src/lib/features/social/filter-model.ts#L53-L99) · [Svelte filter expectation](../../../web-client/src/lib/features/social/filter-model.test.ts#L69-L71) · [AppView search](../../../appview/internal/api/search_project_store.go#L123) | Apply the same browse eligibility in both clients. Match the approved Machine filter while keeping Hand and Machine in authoring. Offer project-type filters only where the craft has types. Verify the matching AppView results and a Phase 2 craft without project types. |
| IR-008 | Important | Saved choice visibility | An unfamiliar but valid saved Weaving loom type appears on the Flutter public detail and opens as a placeholder in the editor, with a clear action beside the apparently empty field. Svelte exposes the selected unfamiliar value. | FR-006, FR-009, RULE-002, AC-007, AC-018, REG-002. [Flutter options](../../../app/lib/projects/widgets/project_craft_fields.dart#L104-L115) · [Flutter choice field](../../../app/lib/projects/widgets/project_craft_fields.dart#L144-L149) · [Single-select fallback](../../../app/lib/theme/select_inputs/single_select_input.dart#L390-L403) · [Svelte choice field](../../../web-client/src/lib/features/composer/ProjectCraftFields.svelte#L72-L88) | Show already-saved unfamiliar choices in Flutter while limiting new choices to the catalog. Verify the value displays, survives an unrelated save and can be explicitly cleared. |
| IR-009 | Important | Lifecycle acceptance evidence | The new AppView grouped-edit command tests stop before HTTP, PDS and indexed readback. Revised scheduling has request and record-construction coverage, while the worker acceptance fixture uses the standard-post shape. The Svelte five-pattern browser test stops at a four-pattern draft, and its current scheduling fixture explicitly checks a standard post. These tests can pass while important acceptance paths fail. | FR-009, FR-012, AC-010, AC-014, AC-019, IT-003, IT-004, AT-008, AT-009, DR-002. [Edit command test](../../../appview/internal/api/post_edit_commands_test.go#L801) · [HTTP edit integration](../../../appview/internal/index/post_edit_http_integration_test.go#L44) · [Schedule publication test](../../../appview/internal/scheduledposts/publication_test.go#L72) · [Worker acceptance fixture](../../../appview/internal/scheduledposts/worker_acceptance_test.go#L148) · [Svelte pattern draft browser test](../../../web-client/tests/authenticated-project-detail-parity.spec.ts#L155) · [Svelte schedule browser test](../../../web-client/tests/authenticated-app.spec.ts#L639) | Exercise a grouped published edit through HTTP, the captured PDS write and indexed reread. Exercise a current revised schedule through reopen, retry and readback. Extend the Svelte fixtures to publish five patterns, edit and remove one, and save and reopen a project schedule. Use representative Phase 1 and Phase 2 shapes. |

### Initial Requirement And Test Traceability

- The implementation covers the additive Lexicons and explorer checkpoint, plural sizes and patterns, shared design tags with craft-specific suggestions, structured techniques and methods, and Phase 2 craft options across profiles, onboarding and discovery. The generated catalog retains known identifiers for reads across both phases. [Requirements](01-requirements.md) · [Coding plan](04-coding-plan.md) · [ADR 029](../../adr/029-additive-project-craft-details.md)
- AppView has direct tests for singleton projections, destructive legacy write guards, preserved nested extensions, phase gating and pattern filter membership. PostgreSQL coverage exercises the same-pattern rule for combined pattern filters. Flutter and Svelte have composer, detail and discovery fixture coverage. The implementation plan records those checks at their respective layers. [Acceptance specification](02-acceptance-tests.md) · [Implementation evidence](05-implementation-plan.md#runtime-implementation-evidence)
- The unexpected behaviors in IR-001 through IR-008 contradict approved retention, validation or parity guarantees. IR-009 carries forward the HTTP and PDS coverage requested in DR-002 and identifies the remaining current scheduling and Svelte browser gaps. [Document review](03-document-review.md)
- MAN-001 and MAN-002 are rollout checks. Verify the supported installed legacy builds against the compatible server contract and confirm AppView readiness before enabling client writes in each phase. [Pending rollout evidence](05-implementation-plan.md#rollout-evidence-pending)

### Initial Test Evidence

| Evidence | Result |
|---|---|
| AppView release log from implementation | `just appview-check` passed, including standard and race suites and packaged release checks. |
| Flutter implementation logs | Static analysis found no issues. The focused combined suite passed 429 tests with two existing conditional skips. The Phase 1 selection passed 61 tests. There was no running debug client for hot reload. |
| Svelte release log from implementation | 623 unit tests passed. Preview and Worker each passed 581 authenticated and 152 guest browser cases, with three existing authenticated skips per mode. The production build completed. |
| Checks rerun during review | Focused AppView edit, scheduling, release phase and Lexicon tests passed. The shared catalog check, all seven Svelte filter-model tests and `git diff --check` passed. |
| Targeted review probes | A temporary Go test exercising `DecodePostCreate` and `ValidatePostCreate` failed because `patterns: null` passed validation. Running the current Svelte `projectFacets`, `editChanges` and `validateDraft` functions reproduced the dropped pattern row, changed sibling facet, unresolved mention on a new pattern row and blocked Macramé save. |
| Documentation check | All 19 catalog unit tests passed and the review links resolve. `just docs-check` flagged the generated change catalog as stale. The review stage is limited to this artifact, so the catalog refresh belongs in the next update. |

### Initial Risk Review

- **High**. Published edits can discard data or appear to save a clear that never reached AppView. The temporary probes covered local code paths. The rollout checks cover installed client and deployment conditions.
- The change uses the existing project JSONB, craft and interest fields, and schedule payload storage. The reviewed design requires no PostgreSQL DDL or bulk PDS rewrite. Existing owner-edit and conditional-write paths remain the right boundaries for regression verification. [ADR 029](../../adr/029-additive-project-craft-details.md) · [Published edit criteria](02-acceptance-tests.md#5-integration-test-cases)
- The maintainer approved the schema checkpoint and runtime implementation. The verdict applies to readiness for merge or handoff. [Checkpoint record](05-implementation-plan.md#completion-checklist)

### Initial UI Polish Recommendation

**Not needed.** The visible issues in this review require changes to save behavior, validation or catalog eligibility. Cover those outcomes in the corrective tests.

### Initial Handoff Back To TDD Builder

1. Start with IR-001. Seed a valid extension-only first pattern, a retained source facet and an editable sibling. Change the sibling in the Svelte editor and verify the owner patch and captured PDS record. Add the failing Flutter last-detail clear from IR-002 next.
2. Reproduce IR-003 through IR-008 in focused tests, then close the representative lifecycle gaps in IR-009. Record which checks exercise AppView, indexed reads and first-party editor flows in the implementation evidence.
3. Rerun the focused owner-edit, validation, search and schedule suites, the AppView release gate, the Flutter Phase 1 and combined checks and the Svelte release gate. Verify both release phase catalog configurations and keep MAN-001 and MAN-002 open until rollout.

</details>
