# TDD Implementation Plan: Flutter Project Composer UI

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Preserve the non-goals: no AppView/API, lexicon, migration, dependency, DTO enum, direct-PDS, project rendering/editing, generated token-pipeline, project replies, quote, draft autosave or scheduled-post changes.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-006 | FR-004, FR-024, RULE-002, RULE-003 | AC-008, AC-009, AC-027 | Fails: missing option catalog files/classes |
| 2 | UT-013 | FR-022 | AC-024 | Fails: missing subtype filtering helper/metadata |
| 3 | UT-001 | FR-001, FR-003, NFR-001 | AC-003, AC-004, AC-020 | Fails: missing text field wrapper |
| 4 | UT-002 | FR-001, NFR-001 | AC-003, AC-004 | Fails: missing multiline/controller support |
| 5 | UT-003 | FR-002, FR-003, NFR-001 | AC-003, AC-005, AC-020 | Fails: missing dropdown wrapper |
| 6 | UT-004 | FR-002, FR-019, FR-023, NFR-001 | AC-003, AC-005, AC-018, AC-025, AC-026 | Fails: missing multi-select/count handling |
| 7 | UT-005 | FR-002, NFR-001 | AC-003, AC-005, AC-020 | Fails: missing radio wrapper |
| 8 | UT-007 | FR-005, FR-009, FR-019, FR-020, FR-023, FR-024, RULE-004 | AC-006, AC-009, AC-012, AC-018, AC-019, AC-025, AC-027 | Fails: missing payload builder |
| 9 | UT-008 | FR-011, BR-003 | AC-011, AC-012 | Fails: missing sewing detail builder |
| 10 | UT-009 | FR-012, FR-021, BR-003 | AC-011, AC-012, AC-023 | Fails: missing knitting detail/gauge validation |
| 11 | UT-010 | FR-013, FR-021, BR-003 | AC-011, AC-012, AC-023 | Fails: missing crochet detail/gauge validation |
| 12 | UT-011 | FR-014, BR-003 | AC-011, AC-012 | Fails: missing quilting detail builder |
| 13 | UT-012 | FR-015, RULE-003, RULE-004 | AC-013 | Fails: missing common-only craft handling |
| 14 | UT-014 | FR-018, FR-010 | AC-010, AC-017 | Fails: missing draft detection helper |
| 15 | UT-015 | FR-016, RULE-001 | AC-012, AC-015 | Fails: missing submit adapter/helper |
| 16 | AT-001 | BR-001, FR-006, FR-008, FR-025 | AC-001, AC-028 | Fails: missing chooser/project branch |
| 17 | AT-002 | BR-002, FR-006, FR-007, FR-025 | AC-002, AC-014, AC-028 | Fails: missing chooser regular branch/result forwarding |
| 18 | AT-010 | BR-002, FR-025 | AC-028 | Fails: chooser does not forward result/reply bypass not proven |
| 19 | AT-003 | BR-001, FR-008, FR-009, FR-024 | AC-007, AC-027 | Fails: missing project composer primary UI |
| 20 | AT-004 | BR-001, FR-003, FR-005, FR-009, FR-015, FR-016, RULE-001, RULE-004, RULE-005 | AC-006, AC-012, AC-013 | Fails: no project create call/payload |
| 21 | AT-005 | BR-003, FR-010, FR-011, FR-012, FR-013, FR-014, FR-022 | AC-010, AC-011, AC-012, AC-024 | Fails: missing details sections/filtering UI |
| 22 | AT-006 | FR-012, FR-013, FR-021, RULE-005 | AC-023 | Fails: validation errors absent/submit not blocked |
| 23 | AT-007 | FR-019, FR-020, FR-023, RULE-004 | AC-018, AC-019, AC-025, AC-026 | Fails: metadata/pattern/count-limit UI missing |
| 24 | AT-008 | FR-017, NFR-001, NFR-004 | AC-015, AC-016, AC-020 | Fails: feedback/loading/missing-alt parity missing |
| 25 | AT-009 | FR-018 | AC-017 | Fails: project draft confirmation missing |
| 26 | IT-001 | BR-001, FR-006, FR-008, FR-025 | AC-001, AC-028 | Fails: feed entry opens regular composer directly |
| 27 | IT-002 | FR-025 | AC-028 | Fails: profile entry not updated/result not propagated |
| 28 | IT-003 | BR-002, FR-007, FR-025 | AC-002, AC-014, AC-028 | Fails: regular/reply branch integration not proven |
| 29 | IT-004 | FR-016, FR-017, RULE-001 | AC-012, AC-015 | Fails: provider integration states missing |
| 30 | IT-005 | FR-017, NFR-004 | AC-016 | Fails: project composer not wired to image behaviours |
| 31 | REG-001 | BR-002, FR-007, NFR-004 | AC-014 | Existing regular composer regression tests must pass |
| 32 | REG-002 | FR-004, RULE-002, RULE-003 | AC-008, AC-009 | Existing project model tests must pass |
| 33 | REG-003 | BR-002 | AC-014 | Profile composer/reply regression tests must pass |
| 34 | REG-004 | FR-006 | AC-001, AC-002 | Context-menu presentation regression tests must pass |
| 35 | UT-016 | NFR-003 | AC-022 | Fails: missing localised project/chooser strings |
| 36 | IT-006, REG-005 | NFR-002, NFR-005 | AC-021 | Final static/full test/dependency checks must pass |

## Implementation Steps

### Step 1: UT-006
- Requirement IDs: FR-004, FR-024, RULE-002, RULE-003
- Acceptance Criteria: AC-008, AC-009, AC-027
- Write failing test: Added `app/test/projects/options/project_option_catalogs_test.dart` to assert representative craft/status/pattern/detail/color/design token labels, the Finished default token, and that `Project`/`ProjectCommon` remain string-backed.
- Run command: `flutter test test/projects/options/project_option_catalogs_test.dart`
- Confirmed failure: Meaningful compile failure because `package:craftsky_app/projects/options/project_option_catalogs.dart` and `ProjectOptionCatalogs` did not exist.
- Implement: Added UI-only `ProjectOption` plus `ProjectOptionCatalogs` with known craft/status/pattern/yarn/needle/hook/gauge/colour/design/quilt options and representative subtype metadata helpers without changing DTOs.
- Run command: `flutter test test/projects/options/project_option_catalogs_test.dart`
- Refactor: None.
- Notes: Green. The catalog constrains MVP UI selections only; project model fields remain `String`/`String?`.

### Step 2: UT-013
- Requirement IDs: FR-022
- Acceptance Criteria: AC-024
- Write failing test: Added `app/test/projects/options/project_subtype_filter_test.dart` for disabled subtype state without a project type, project-type-filtered subtype options, and clearing invalid subtype values after a type change.
- Run command: `flutter test test/projects/options/project_subtype_filter_test.dart`
- Confirmed failure: Meaningful compile failure because `ProjectOptionCatalogs.isSubtypeSelectionEnabled` did not exist.
- Implement: Added `isSubtypeSelectionEnabled` on top of catalog subtype filtering/clearing helpers.
- Run command: `flutter test test/projects/options/project_subtype_filter_test.dart`
- Refactor: None.
- Notes: Green. Subtype helpers are UI-only and return empty lists for crafts without detail schemas.

### Step 3: UT-001
- Requirement IDs: FR-001, FR-003, NFR-001
- Acceptance Criteria: AC-003, AC-004, AC-020
- Write failing test: Added text field widget tests in `app/test/theme/craftsky_form_builder_text_field_test.dart` for FormBuilder value forwarding, helper/error visibility, save/validate/reset, and disabled editing.
- Run command: `flutter test test/theme/craftsky_form_builder_text_field_test.dart`
- Confirmed failure: Meaningful compile failure because `craftsky_form_builder_text_field.dart` and `CraftskyFormBuilderTextField` did not exist.
- Implement: Added `CraftskyFormBuilderTextField` as a `FormBuilderField<String>` wrapper around `BrandTextField`, syncing field value, error text, enabled state, controller and reset behaviour.
- Run command: `flutter test test/theme/craftsky_form_builder_text_field_test.dart`
- Refactor: None.
- Notes: Green. This is a reusable FormBuilder-compatible single-line text primitive.

### Step 4: UT-002
- Requirement IDs: FR-001, NFR-001
- Acceptance Criteria: AC-003, AC-004
- Write failing test: Extended `craftsky_form_builder_text_field_test.dart` with a multiline field test covering controller/focus ownership, min/max lines, and change/submit hooks.
- Run command: `flutter test test/theme/craftsky_form_builder_text_field_test.dart`
- Confirmed failure: Meaningful compile failure because `CraftskyFormBuilderMultilineTextField` did not exist.
- Implement: Added `CraftskyFormBuilderMultilineTextField` as a convenience wrapper over `CraftskyFormBuilderTextField` with multiline keyboard/action defaults and line-count parameters.
- Run command: `flutter test test/theme/craftsky_form_builder_text_field_test.dart`
- Refactor: None.
- Notes: Green. Single and multiline text field adapters share the same FormBuilder/BrandTextField integration path.

### Step 5: UT-003
- Requirement IDs: FR-002, FR-003, NFR-001
- Acceptance Criteria: AC-003, AC-005, AC-020
- Write failing test: Added `app/test/theme/craftsky_form_builder_dropdown_test.dart` for labels/helper text, selected value saving, validation error, reset, disabled state, and change callback.
- Run command: `flutter test test/theme/craftsky_form_builder_dropdown_test.dart`
- Confirmed failure: Initial compile failure because `craftsky_form_builder_select_fields.dart`, `CraftskySelectOption`, and `CraftskyFormBuilderDropdownField` did not exist; first implementation also produced a meaningful reset failure where the nested dropdown FormField reset to `null`.
- Implement: Added `CraftskySelectOption` and `CraftskyFormBuilderDropdownField`; replaced nested `DropdownButtonFormField` with `DropdownButton` inside `InputDecorator` so FormBuilder is the single source of field state.
- Run command: `flutter test test/theme/craftsky_form_builder_dropdown_test.dart`
- Refactor: Removed the nested FormField implementation to preserve reset semantics.
- Notes: Green. Dropdown selection values flow through FormBuilder by field name.

### Step 6: UT-004
- Requirement IDs: FR-002, FR-019, FR-023, NFR-001
- Acceptance Criteria: AC-003, AC-005, AC-018, AC-025, AC-026
- Write failing test: Added `app/test/theme/craftsky_form_builder_multi_select_test.dart` for free-text materials, known colour options, FormBuilder values, visible chips, max-count limit messaging, and removals.
- Run command: `flutter test test/theme/craftsky_form_builder_multi_select_test.dart`
- Confirmed failure: Meaningful compile failure because `CraftskyFormBuilderMultiSelectField` did not exist; the first green attempt also revealed ambiguous option/selected chip labels, fixed with stable option keys.
- Implement: Added `CraftskyFormBuilderMultiSelectField` with chip rendering, optional custom value entry, known option chips, max-selected enforcement, removal controls, and FormBuilder list values.
- Run command: `flutter test test/theme/craftsky_form_builder_multi_select_test.dart`
- Refactor: Added stable keys for option chips to keep tests and composer interactions unambiguous.
- Notes: Green. Materials can be free-text; colours/design tags can use known options with the same field primitive.

### Step 7: UT-005
- Requirement IDs: FR-002, NFR-001
- Acceptance Criteria: AC-003, AC-005, AC-020
- Write failing test: Added `app/test/theme/craftsky_form_builder_radio_test.dart` for selected token values, validation error display, reset, disabled state, and change callback.
- Run command: `flutter test test/theme/craftsky_form_builder_radio_test.dart`
- Confirmed failure: Meaningful compile failure because `CraftskyFormBuilderRadioField` did not exist. A later ambiguous assertion came from test setup reusing helper/error text and was fixed before proceeding.
- Implement: Added `CraftskyFormBuilderRadioField` with Material radio list tiles inside a FormBuilder-owned `InputDecorator`.
- Run command: `flutter test test/theme/craftsky_form_builder_radio_test.dart`
- Refactor: None.
- Notes: Green. Radio selected values flow through FormBuilder by field name and reset correctly.

### Step 8: UT-007
- Requirement IDs: FR-005, FR-009, FR-019, FR-020, FR-023, FR-024, RULE-004
- Acceptance Criteria: AC-006, AC-009, AC-012, AC-018, AC-019, AC-025, AC-027
- Write failing test: Added `app/test/projects/project_composer_payload_test.dart` common payload cases for craft/status/title/materials/colours/design tags, default Finished status, optional pattern trimming/omission, and missing craft validation.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Confirmed failure: Meaningful compile failure because `ProjectComposerFields`, `buildProjectComposerPayload`, payload result, and validation errors did not exist.
- Implement: Added `project_composer_fields.dart` and `project_composer_payload.dart` with pure common payload mapping, default status, trimmed string/list handling, non-empty `ProjectPattern` creation, and missing craft validation.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Refactor: None.
- Notes: Green. Empty optional common/pattern fields are omitted as `null`; DTO fields remain strings.

### Step 9: UT-008
- Requirement IDs: FR-011, BR-003
- Acceptance Criteria: AC-011, AC-012
- Write failing test: Extended `project_composer_payload_test.dart` with sewing detail creation and empty-detail omission cases.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Confirmed failure: Meaningful assertion failure because non-empty sewing form values returned `project.details == null`.
- Implement: Added active-craft detail dispatch and `SewingProjectDetails` construction with trimmed project type/subtype/size/fit notes, omitting the detail object when all fields are empty.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Refactor: None.
- Notes: Green. Sewing details are only built for active sewing craft values.

### Step 10: UT-009
- Requirement IDs: FR-012, FR-021, BR-003
- Acceptance Criteria: AC-011, AC-012, AC-023
- Write failing test: Extended `project_composer_payload_test.dart` with valid knitting project details/gauge and invalid partial, missing-unit, non-positive and decimal gauge cases.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Confirmed failure: Meaningful assertion failures because knitting values returned no details and invalid gauges still built a project.
- Implement: Added knitting detail dispatch/construction plus gauge validation/building for positive whole-number stitches/measurement/unit with optional positive rows.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Refactor: Extracted shared gauge helpers for upcoming crochet support.
- Notes: Green. Invalid gauge returns `ProjectComposerValidationCode.invalidGauge` and no project.

### Step 11: UT-010
- Requirement IDs: FR-013, FR-021, BR-003
- Acceptance Criteria: AC-011, AC-012, AC-023
- Write failing test: Extended `project_composer_payload_test.dart` with crochet detail creation and invalid crochet gauge coverage.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Confirmed failure: Meaningful assertion failure because crochet form values returned no `CrochetProjectDetails`.
- Implement: Added crochet detail dispatch/construction and reused the gauge validation/building helpers for crochet stitches/rows/measurement/unit.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Refactor: None.
- Notes: Green. Crochet details mirror knitting with hook size instead of needle size.

### Step 12: UT-011
- Requirement IDs: FR-014, BR-003
- Acceptance Criteria: AC-011, AC-012
- Write failing test: Extended `project_composer_payload_test.dart` with quilting detail creation and empty-detail omission cases.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Confirmed failure: Meaningful assertion failure because quilting form values returned no `QuiltingProjectDetails`.
- Implement: Added quilting detail dispatch/construction for project type/subtype, size, piecing technique and quilting method, omitting all-empty quilting details.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Refactor: None.
- Notes: Green. Quilting details are active-craft scoped and omitted when empty.

### Step 13: UT-012
- Requirement IDs: FR-015, RULE-003, RULE-004
- Acceptance Criteria: AC-013
- Write failing test: Added explicit common-only coverage to `project_composer_payload_test.dart` for embroidery and a future/open craft token.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Confirmed failure: No new failure; the behavior was already implemented by the prior common/detail dispatch loops. This test was added as regression coverage for FR-015/RULE-003/RULE-004.
- Implement: No additional code required.
- Run command: `flutter test test/projects/project_composer_payload_test.dart`
- Refactor: None.
- Notes: Green. Unknown/common-only craft tokens preserve `project.common.craftType` and omit `details`.

### Step 14: UT-014
- Requirement IDs: FR-018, FR-010
- Acceptance Criteria: AC-010, AC-017
- Write failing test: Added `app/test/projects/project_composer_draft_state_test.dart` for unchanged state, body/image/metadata changes, and collapsed detail values.
- Run command: `flutter test test/projects/project_composer_draft_state_test.dart`
- Confirmed failure: Meaningful compile failure because `project_composer_draft_state.dart` did not exist.
- Implement: Added `ProjectComposerDraftState.hasDraft` with normalised body, image count and form value comparison.
- Run command: `flutter test test/projects/project_composer_draft_state_test.dart`
- Refactor: None.
- Notes: Green. Whitespace-only form values are ignored; collapsed details remain part of form values and count as draft changes.

### Step 15: UT-015
- Requirement IDs: FR-016, RULE-001
- Acceptance Criteria: AC-012, AC-015
- Write failing test: Added `app/test/projects/project_composer_submit_adapter_test.dart` with uploaded image state, generated facets, a project payload and `reply == null` expectation.
- Run command: `flutter test test/projects/project_composer_submit_adapter_test.dart`
- Confirmed failure: Meaningful compile failure because `project_composer_submit_adapter.dart` did not exist.
- Implement: Added `ProjectComposerSubmitArguments` and `buildProjectComposerSubmitArguments` to trim text, generate facets, convert composer images, include the project and force `reply: null`.
- Run command: `flutter test test/projects/project_composer_submit_adapter_test.dart`
- Refactor: None.
- Notes: Green. Project submit argument construction is pure and provider-independent.

### Step 16: AT-001
- Requirement IDs: BR-001, FR-006, FR-008, FR-025
- Acceptance Criteria: AC-001, AC-028
- Write failing test: Added `app/test/feed/widgets/post_type_chooser_test.dart` compact chooser coverage. The test opens the top-level chooser from a `New post` action, expects `Regular post` and `Project post`, selects project, and expects the full-screen project composer route with `Craft type` visible.
- Run command: `flutter test test/feed/widgets/post_type_chooser_test.dart`
- Confirmed failure: Meaningful compile failure because `app/lib/feed/widgets/post_type_chooser.dart` and `showTopLevelPostComposerChooser` did not exist.
- Implement: Added localized post-type chooser labels/descriptions, `showTopLevelPostComposerChooser`, minimal `showProjectComposerSheet`/`ProjectComposerSheet`, and async-aware context-menu callback handling so selecting a compact bottom-sheet row dismisses the chooser and opens the selected full-screen route.
- Run command: `flutter test test/feed/widgets/post_type_chooser_test.dart`; nearby regression `flutter test test/theme/craftsky_context_menu_test.dart`
- Refactor: Removed an incidental animated placeholder from the minimal project composer so widget tests can settle on the opened route.
- Notes: Green. The project branch now uses the responsive Craftsky context-menu bottom sheet on compact screens and opens a root-navigator fullscreen dialog. Result-forwarding behaviour is partially enabled by awaiting context-menu callbacks, with fuller created/dismissed result coverage planned in AT-010.

### Step 17: AT-002
- Requirement IDs: BR-002, FR-006, FR-007, FR-025
- Acceptance Criteria: AC-002, AC-014, AC-028
- Write failing test: Extended `app/test/feed/widgets/post_type_chooser_test.dart` with regular-branch coverage that opens the compact chooser, selects `Regular post`, and verifies the existing regular composer title/body prompt appears while project-only `Craft type`/`Project post` UI is absent.
- Run command: `flutter test test/feed/widgets/post_type_chooser_test.dart`
- Confirmed failure: No new failure after the test was added; the regular branch was already satisfied by the Step 16 chooser implementation that routes `Regular post` to `showPostComposerSheet`.
- Implement: No additional production code required for this focused branch test.
- Run command: `flutter test test/feed/widgets/post_type_chooser_test.dart`
- Refactor: None.
- Notes: Green. Regular top-level selection remains the existing composer and does not introduce project-only form fields.

### Step 18: AT-010
- Requirement IDs: BR-002, FR-025
- Acceptance Criteria: AC-028
- Write failing test: Added `app/test/feed/widgets/post_type_chooser_result_test.dart` to verify that the chooser resolves with the `Post?` returned by the selected project composer. The test uses a fake project composer launcher to avoid driving a full create flow in this chooser-level test.
- Run command: `flutter test test/feed/widgets/post_type_chooser_result_test.dart`
- Confirmed failure: Meaningful compile failure because `showTopLevelPostComposerChooser` had no `showProjectComposer` injection hook for verifying selected-composer result forwarding.
- Implement: Added `PostComposerLauncher` plus optional `showRegularComposer` and `showProjectComposer` parameters to `showTopLevelPostComposerChooser`, preserving production defaults while allowing focused result tests.
- Run command: `flutter test test/feed/widgets/post_type_chooser_result_test.dart`; nearby `flutter test test/feed/widgets/post_type_chooser_test.dart`; nearby regression `flutter test test/theme/craftsky_context_menu_test.dart`
- Refactor: None.
- Notes: Green. The chooser now awaits selected composer launchers and returns their `Post?`. A parallel Flutter test invocation crashed the Flutter tool while copying native assets; the same affected chooser test passed when rerun sequentially. Reply-bypass remains unchanged in production code and will be asserted when feed/profile entry integration tests are added.

### Step 19: AT-003
- Requirement IDs: BR-001, FR-008, FR-009, FR-024
- Acceptance Criteria: AC-007, AC-027
- Write failing test: Added `app/test/projects/widgets/project_composer_sheet_test.dart` to render `ProjectComposerSheet` and verify primary flow affordances: photos/add photo, facet-aware body label, project title, craft type, status, default Finished status token, materials, colours, design tags, Add pattern, More project details, and Post action.
- Run command: `flutter test test/projects/widgets/project_composer_sheet_test.dart`
- Confirmed failure: Meaningful assertion failure because the placeholder project composer did not render `Add a photo` or the rest of the primary metadata form.
- Implement: Converted `ProjectComposerSheet` to stateful UI with body `FacetAutocompleteEditor`, FormBuilder-managed project title/craft/status/materials/colours/design-tag controls using existing Craftsky field wrappers and `ProjectOptionCatalogs`, localized project copy, Add pattern affordance, More project details disclosure, and status initialized to `ProjectOptionCatalogs.finishedStatusToken`.
- Run command: `flutter test test/projects/widgets/project_composer_sheet_test.dart`; nearby `flutter test test/feed/widgets/post_type_chooser_test.dart`
- Refactor: None.
- Notes: Green. Primary render/default-status coverage is in place; submit behaviour, craft details, validation, metadata limits and feedback remain in later acceptance loops.

### Step 20: AT-004
- Requirement IDs: BR-001, FR-003, FR-005, FR-009, FR-015, FR-016, RULE-001, RULE-004, RULE-005
- Acceptance Criteria: AC-006, AC-012, AC-013
- Write failing test: Added `app/test/projects/widgets/project_composer_submit_test.dart` with a fixed uploaded-photo `composerImagesProvider` state and fake `PostRepository`. The test enters body text with a hashtag, selects Embroidery, submits, and captures body text, generated facets, image payload, `reply == null`, common-only `Project.common.craftType`, and omitted optional fields/details.
- Run command: `flutter test test/projects/widgets/project_composer_submit_test.dart`
- Confirmed failure: Meaningful compile failure because `ProjectComposerSheet` did not accept an injectable `composerId` needed to bind the test image provider state; after initial submit wiring, a test matcher was corrected because the generated tag facet was present but raw Dart map equality made `contains(map)` unsuitable.
- Implement: Converted `ProjectComposerSheet` to a `ConsumerStatefulWidget`, added optional `composerId`, watched `createPostProvider` and `composerImagesProvider`, tracked body text for submit enablement, built `Project` via `buildProjectComposerPayload`, built facets/images/reply-null arguments via `buildProjectComposerSubmitArguments`, and called `createPostProvider.notifier.create` with project payload and no reply.
- Run command: `flutter test test/projects/widgets/project_composer_submit_test.dart`; nearby `flutter test test/projects/widgets/project_composer_sheet_test.dart`; nearby `flutter test test/projects/project_composer_submit_adapter_test.dart`
- Refactor: None.
- Notes: Green. Valid common-only embroidery project creation now flows through existing create plumbing with uploaded image and generated hashtag facets. Missing-alt confirmation, create success/error UI and validation messaging remain in later loops.

### Step 21: AT-005
- Requirement IDs: BR-003, FR-010, FR-011, FR-012, FR-013, FR-014, FR-022
- Acceptance Criteria: AC-010, AC-011, AC-012, AC-024
- Write failing test: Added `app/test/projects/widgets/project_composer_details_test.dart` incrementally for Sewing, Knitting, Crochet and Quilting active-craft examples plus subtype filtering/clearing. Each render example selects a craft, expands `More project details`, verifies active-craft fields, and verifies unrelated craft-only fields are absent.
- Run command: `flutter test test/projects/widgets/project_composer_details_test.dart`
- Confirmed failure: Meaningful failures for each new craft example before implementation because the corresponding detail fields were absent (`Sewing project type`, `Knitting project type`, `Crochet project type`, then `Quilting project type`). Subtype filtering initially needed a test scroll fix because the subtype dropdown was below the widget-test viewport.
- Implement: Added active craft state in `ProjectComposerSheet`, sewing/knitting/crochet/quilting detail sections behind the existing expansion tile, project-type/subtype dropdowns using `ProjectOptionCatalogs.projectTypesForCraft` and `projectSubtypesFor`, subtype clearing when craft/project type changes, knitting/crochet yarn/gauge/size controls, sewing size/fit controls, quilting size/piecing/method controls, and localized labels for all new detail fields.
- Run command: `flutter test test/projects/widgets/project_composer_details_test.dart`; nearby `flutter test test/projects/project_composer_payload_test.dart`; nearby `flutter test test/projects/options/project_subtype_filter_test.dart`
- Refactor: Shared the details test setup with `_openDetailsForCraft`; no production refactor beyond common active-state clearing pattern.
- Notes: Green. A parallel Flutter test invocation hit the same native-assets signing/copy race seen earlier; the affected subtype helper test passed when rerun sequentially. Details currently render and map via existing payload helpers; validation error presentation remains in AT-006.

### Step 22: AT-006
- Requirement IDs: FR-012, FR-013, FR-021, RULE-005
- Acceptance Criteria: AC-023
- Write failing test: Added `app/test/projects/widgets/project_composer_validation_test.dart` required-empty submit coverage for missing body text, craft type and photo. Extended the same file with partial knitting gauge coverage using an uploaded-photo provider override and fake repository to verify create is not called.
- Run command: `flutter test test/projects/widgets/project_composer_validation_test.dart`
- Confirmed failure: Required-empty case initially failed because submitting surfaced no errors; partial gauge case initially failed because the knitting gauge input was not keyed for targeted entry and payload validation errors were silently ignored.
- Implement: Allowed the Post action to run validation when required body/photo/craft are empty, added body/photo/craft localized validation messages, tracked attempted submit state, added keyed knitting gauge stitches input, surfaced invalid-gauge payload errors as `Complete the gauge or clear it.`, and kept create blocked when required fields, text length or payload validation fail.
- Run command: `flutter test test/projects/widgets/project_composer_validation_test.dart`; nearby `flutter test test/projects/widgets/project_composer_details_test.dart`; nearby `flutter test test/projects/widgets/project_composer_submit_test.dart`
- Refactor: None.
- Notes: Green. Required body/craft/photo and partial knitting gauge now block create with visible errors. Parallel Flutter test invocations continued to hit the native-assets race; affected tests passed sequentially.

### Step 23: AT-007
- Requirement IDs: FR-019, FR-020, FR-023, RULE-004
- Acceptance Criteria: AC-018, AC-019, AC-025, AC-026
- Write failing test: Added `app/test/projects/widgets/project_composer_metadata_test.dart` for pattern disclosure and colour count-limit/removal behavior. Extended `app/test/projects/widgets/project_composer_submit_test.dart` with widget-level metadata serialization for materials, colours, design tags and pattern fields.
- Run command: `flutter test test/projects/widgets/project_composer_metadata_test.dart`; `flutter test test/projects/widgets/project_composer_submit_test.dart`
- Confirmed failure: Pattern disclosure test failed because fields remained absent after tapping `Add pattern`. The submit serialization test then failed while targeting pattern text inputs because the fields did not expose stable input keys; test setup was also adjusted to scroll/dismiss keyboard before tapping the materials add button.
- Implement: Added `_showPatternFields` state to `ProjectComposerSheet`, revealed pattern name/URL/difficulty/designer/publisher fields behind `Add pattern`, added localized pattern labels, added stable keys for pattern name/URL inputs, and reused existing metadata multi-select max limits and payload mapping for materials/colours/design tags.
- Run command: `flutter test test/projects/widgets/project_composer_metadata_test.dart`; `flutter test test/projects/widgets/project_composer_submit_test.dart`; nearby `flutter test test/projects/project_composer_payload_test.dart`
- Refactor: Ran `dart format` on touched composer and metadata/submit tests; no production refactor beyond the pattern visibility state.
- Notes: Green. Pattern fields stay hidden until activated, selected metadata chips are visible, colour max 10 blocks extra selections with `You can choose up to 10.` while removal remains possible, and non-empty metadata/pattern values submit into `ProjectCommon`/`ProjectPattern`. Parallel Flutter test invocations continued to hit the native-assets race; affected tests passed sequentially.

### Step 24: AT-008
- Requirement IDs: FR-017, NFR-001, NFR-004
- Acceptance Criteria: AC-015, AC-016, AC-020
- Write failing test: Added `app/test/projects/widgets/project_composer_feedback_test.dart` coverage for missing-alt confirmation before project create, disabled submit/body/selection controls while create is loading, success close/info/reset behaviour, and error snackbar/retry behaviour.
- Run command: `flutter test test/projects/widgets/project_composer_feedback_test.dart`
- Confirmed failure: Missing-alt initially failed because no confirmation appeared. Loading-state coverage then failed because the project body `TextField` remained enabled while create was pending. Success coverage failed because the composer stayed open after create completed. Error coverage failed because no `Couldn't post.` message was shown after a repository failure.
- Implement: Reused the regular composer missing-alt confirmation copy via `showCraftskyConfirmDialog`; propagated `createPostProvider.isLoading` disabled state through the body editor, FormBuilder metadata controls, pattern controls and craft-specific detail controls; added a `createPostProvider` listener to pop with the created `Post`, show `Posted.`, show `Couldn't post.` on error, and reset the provider after consumed success/error transitions.
- Run command: `flutter test test/projects/widgets/project_composer_feedback_test.dart`; nearby `flutter test test/projects/widgets/project_composer_feedback_test.dart test/projects/widgets/project_composer_submit_test.dart test/projects/widgets/project_composer_validation_test.dart`
- Refactor: Ran `dart format` on `lib/projects/widgets/project_composer_sheet.dart` and `test/projects/widgets/project_composer_feedback_test.dart`; no unrelated refactor.
- Notes: Green. Project composer feedback now covers missing-alt parity, create loading disabled state, success close/info/reset, and error snackbar with retry. Image provider notice parity remains planned for `IT-005`.

### Step 25: AT-009
- Requirement IDs: FR-018
- Acceptance Criteria: AC-017
- Write failing test: Added `app/test/projects/widgets/project_composer_discard_test.dart` body-text draft coverage that opens the project composer, edits the body, taps close, expects the existing discard dialog copy, keeps editing, then discards. Added follow-up coverage for unchanged close, selected image drafts, and metadata form changes.
- Run command: `flutter test test/projects/widgets/project_composer_discard_test.dart`
- Confirmed failure: Meaningful failure because tapping close after editing body text dismissed the project composer immediately and no `Discard draft?` dialog appeared.
- Implement: Imported `ProjectComposerDraftState` into `ProjectComposerSheet`, computed draft state from body text, image count and current FormBuilder instant values, wrapped the sheet in `PopScope`, and reused `showCraftskyConfirmDialog` with the regular composer discard copy before closing dirty project drafts.
- Run command: `flutter test test/projects/widgets/project_composer_discard_test.dart`; nearby `flutter test test/projects/widgets/project_composer_discard_test.dart test/projects/project_composer_draft_state_test.dart test/projects/widgets/project_composer_feedback_test.dart`
- Refactor: Ran `dart format` on the touched composer and discard test files; no unrelated refactor.
- Notes: Green. Project composer now closes immediately when unchanged, confirms for body/image/metadata draft sources, keeps the composer open when the user chooses `Keep editing`, and discards when confirmed. Create-loading close behaviour follows the regular composer by allowing pop without prompting while loading.

### Step 26: IT-001
- Requirement IDs: BR-001, FR-006, FR-008, FR-025
- Acceptance Criteria: AC-001, AC-028
- Write failing test: Added `app/test/feed/pages/feed_page_composer_entry_test.dart` to render the real `FeedPage` with a loaded empty timeline, tap top-level `New post`, expect the responsive post-type chooser, select `Project post`, and verify the project composer opens with `Craft type`.
- Run command: `flutter test test/feed/pages/feed_page_composer_entry_test.dart`
- Confirmed failure: Meaningful failure because `FeedPage` still called `showPostComposerSheet(context)` directly; `Regular post`/`Project post` chooser options were absent after tapping `New post`.
- Implement: Changed the feed top-level composer button to compute an anchor `RelativeRect` and call `showTopLevelPostComposerChooser`, while keeping reply actions on the existing direct `showPostComposerSheet(replyTarget: ...)` path.
- Run command: `flutter test test/feed/pages/feed_page_composer_entry_test.dart`; nearby `flutter test test/feed/pages/feed_page_composer_entry_test.dart test/feed/feed_page_test.dart`; nearby chooser `flutter test test/feed/widgets/post_type_chooser_test.dart test/feed/widgets/post_type_chooser_result_test.dart`
- Refactor: Updated the existing `FeedPage compose creates top-level post and prepends it` regression to choose `Regular post` before exercising the regular composer flow. Ran `dart format` on touched feed files/tests.
- Notes: Green. Feed top-level posting now uses the chooser; existing regular top-level create regression remains green after selecting the regular branch. Created-post result forwarding is covered at chooser level (`AT-010`); feed page itself does not consume the selected composer result beyond the existing create provider/cache path.

### Step 27: IT-002
- Requirement IDs: FR-025
- Acceptance Criteria: AC-028
- Write failing test: Extended `app/test/profile/widgets/profile_posts_tab_test.dart` with own-profile top-level `New post` coverage that expects the post-type chooser, selects `Project post`, and verifies the project composer opens. Existing non-owner coverage already asserts no `New post` entry.
- Run command: `flutter test test/profile/widgets/profile_posts_tab_test.dart`
- Confirmed failure: Meaningful failure because own-profile `ProfilePostsTab` still opened the regular composer directly; `Regular post`/`Project post` chooser options were absent.
- Implement: Changed the own-profile top-level composer button to compute an anchor `RelativeRect` and call `showTopLevelPostComposerChooser`, while preserving direct reply composer navigation.
- Run command: `flutter test test/profile/widgets/profile_posts_tab_test.dart`; nearby `flutter test test/profile/widgets/profile_posts_tab_test.dart test/feed/widgets/post_type_chooser_test.dart test/feed/widgets/post_type_chooser_result_test.dart`
- Refactor: Ran `dart format` on touched profile files/tests; no unrelated refactor.
- Notes: Green. Own-profile top-level post entry now uses the chooser and can open the project composer; non-owner profiles still have no `New post` entry. Result-forwarding remains covered by the chooser-level `AT-010` and profile reply result behaviour remains covered by existing profile tests.

### Step 28: IT-003
- Requirement IDs: BR-002, FR-007, FR-025
- Acceptance Criteria: AC-002, AC-014, AC-028
- Write/update test: Extended existing `app/test/feed/feed_page_test.dart` coverage so the regular top-level create path selects `Regular post` from the chooser and explicitly captures `project == null`; extended the reply test to assert `Regular post`/`Project post` chooser options do not appear when tapping a reply action.
- Run command: `flutter test test/feed/feed_page_test.dart`
- Confirmed failure: No new failure after the test updates; the behaviour was already satisfied by prior feed/profile chooser wiring and unchanged direct reply paths.
- Implement: No production code required for this focused loop.
- Run command: `flutter test test/feed/feed_page_test.dart`
- Refactor: Ran `dart format` on `test/feed/feed_page_test.dart`.
- Notes: Green. Existing regular composer create remains top-level with no project payload after choosing `Regular post`; feed reply still opens the regular reply composer directly with no chooser. Profile reply bypass remains covered by `profile_posts_tab_test.dart`.

### Step 29: IT-004
- Requirement IDs: FR-016, FR-017, RULE-001
- Acceptance Criteria: AC-012, AC-015
- Write/update test: Added `app/test/projects/widgets/project_composer_provider_test.dart` to exercise provider integration directly with a `ProviderContainer`: successful project create captures `reply == null` and non-null `project`, consumes the success message, and resets `createPostProvider`; error create consumes the error message, resets the provider, and leaves the Post action retryable.
- Run command: `flutter test test/projects/widgets/project_composer_provider_test.dart`
- Confirmed failure: No new failure; this behaviour was already satisfied by the `AT-008` create-state listener implementation.
- Implement: No production code required for this focused loop.
- Run command: `flutter test test/projects/widgets/project_composer_provider_test.dart`; nearby `flutter test test/projects/widgets/project_composer_provider_test.dart test/projects/widgets/project_composer_feedback_test.dart test/projects/widgets/project_composer_submit_test.dart`
- Refactor: Ran `dart format` on the new provider test.
- Notes: Green. Provider success/error transitions are consumed and reset for project composer creates, retries are enabled after errors, and project create continues to force `reply == null`.

### Step 30: IT-005
- Requirement IDs: FR-017, NFR-004
- Acceptance Criteria: AC-016
- Write failing test: Added `app/test/projects/widgets/project_composer_images_test.dart` to assert that project composer image notices show the same selection-limit message as the regular composer. Added follow-up notice coverage for unsupported images and image-picker failures.
- Run command: `flutter test test/projects/widgets/project_composer_images_test.dart`
- Confirmed failure: Meaningful failure because the project composer did not listen to `composerImagesProvider` notices, so no image selection-limit message was recorded.
- Implement: Added project composer image-notice consumption for `ImageSelectionLimitNotice`, `UnsupportedImagesNotice`, and `ImagePickerFailedNotice`, using the regular composer localization messages and one-shot notice id guarding. Notices are cleared through the image notifier in production; fixed-value test overrides are guarded so the one-shot display remains testable.
- Run command: `flutter test test/projects/widgets/project_composer_images_test.dart`; nearby `flutter test test/projects/widgets/project_composer_images_test.dart test/projects/widgets/project_composer_feedback_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart`
- Refactor: Ran `dart format` on the touched composer and image-notice test files.
- Notes: Green. Project composer now surfaces image selection-limit, unsupported-image and picker-failure notices consistently with the regular composer. Missing-alt confirmation parity remains covered by `AT-008`.

### Step 31: REG-001
- Requirement IDs: BR-002, FR-007, NFR-004
- Acceptance Criteria: AC-014
- Write/update test: Kept existing regular composer facet, discard, missing-alt and image provider/state tests as regression coverage after chooser/project composer changes.
- Run command: `flutter test test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart test/feed/providers/composer_images_provider_test.dart test/feed/providers/composer_image_state_test.dart`
- Confirmed failure: None.
- Implement: No production code required.
- Run command: Same as above.
- Refactor: None.
- Notes: Green. Existing regular composer text/facet, discard, missing-alt and image state/provider behaviours remain intact.

### Step 32: REG-002
- Requirement IDs: FR-004, RULE-002, RULE-003
- Acceptance Criteria: AC-008, AC-009
- Write/update test: Kept existing project model/detail tests as regression coverage for string-backed/open-token compatibility.
- Run command: `flutter test test/projects/models/project_test.dart test/projects/models/project_details_test.dart`
- Confirmed failure: None.
- Implement: No production code required.
- Run command: Same as above.
- Refactor: None.
- Notes: Green. Project DTO/model constructors and unknown detail parsing remain open-token compatible and were not changed to enums.

### Step 33: REG-003
- Requirement IDs: BR-002
- Acceptance Criteria: AC-014
- Write/update test: Kept and extended `app/test/profile/widgets/profile_posts_tab_test.dart` as own-profile/non-owner/reply regression coverage.
- Run command: `flutter test test/profile/widgets/profile_posts_tab_test.dart`
- Confirmed failure: None after the IT-002 chooser wiring was implemented.
- Implement: No additional production code required.
- Run command: Same as above.
- Refactor: None.
- Notes: Green. Own-profile top-level entry remains present and now opens the chooser; non-owner profiles still do not show `New post`; profile reply create still opens the focused thread via the regular composer path.

### Step 34: REG-004
- Requirement IDs: FR-006
- Acceptance Criteria: AC-001, AC-002
- Write/update test: Kept existing `CraftskyContextMenuButton` responsive presentation tests and chooser tests as context-menu regression coverage.
- Run command: `flutter test test/theme/craftsky_context_menu_test.dart test/feed/widgets/post_type_chooser_test.dart`
- Confirmed failure: None.
- Implement: No production code required.
- Run command: Same as above.
- Refactor: None.
- Notes: Green. Compact bottom-sheet and larger-screen anchored menu behaviour remain covered while the chooser uses the context-menu presentation.

### Step 35: UT-016
- Requirement IDs: NFR-003
- Acceptance Criteria: AC-022
- Write/update test: Added `app/test/l10n/project_composer_l10n_test.dart` to inspect `app_en.arb` for required chooser/project composer keys, non-blank localized values, no emoji in app chrome, and British English `Colours` copy.
- Run command: `flutter test test/l10n/project_composer_l10n_test.dart`
- Confirmed failure: None; the localization keys had already been added during earlier composer/chooser work.
- Implement: No production code required.
- Run command: Same as above.
- Refactor: Ran `dart format` on the new l10n test.
- Notes: Green. New user-visible chooser/project composer strings are present in localization resources and conform to the requested copy constraints covered by automation.

### Step 36: IT-006, REG-005
- Requirement IDs: NFR-002, NFR-005
- Acceptance Criteria: AC-021
- Commands: `flutter analyze`; `flutter test`
- Results: Green. `flutter analyze` reported `No issues found!`; full `flutter test` completed with `All tests passed!` after fixing one multiline field test expectation introduced while addressing analyzer lint.
- Notes: No dependency changes were made. No build-runner changes were required. Localization generation had already been run after ARB updates, and generated localization files are included with the implementation changes.

## Execution Log
- 2026-06-11: Created implementation plan from approved workflow documents. No blocking gaps in `03-document-review.md`; coding plan says start with `UT-006`.
- 2026-06-11: Implementation review returned `Changes required` with `IR-001`, `IR-002`, and `IR-003`. The review-fix TDD order is:
  1. `AT-003` / `IT-005` (`BR-001`, `FR-009`, `FR-017`, `RULE-005`; `AC-007`, `AC-016`, `AC-023`) — prove the project composer exposes an enabled provider-backed add-photo path and selected image/alt-text controls, then implement the minimum shared/equivalent image UI.
  2. `AT-008` / `IT-005` (`FR-017`, `NFR-004`; `AC-015`, `AC-016`, `AC-020`) — verify image actions and controls remain disabled/loading-aware and preserve notice/missing-alt parity after the photo UI fix.
  3. `UT-004` / `UT-016` (`FR-002`, `NFR-003`; `AC-005`, `AC-022`, `AC-026`) — localize or parameterize the remaining reusable multi-select copy and add coverage for those strings.
- 2026-06-11: Review-fix loop 1 (`AT-003` / `IT-005`, `BR-001`, `FR-009`, `FR-017`, `RULE-005`; `AC-007`, `AC-016`, `AC-023`) added a failing widget test proving the project composer needed an enabled `composerImagesProvider(...).notifier.addImages()` path and visible alt-text controls. Red failure: `composer-add-image` was absent because the project composer rendered a disabled `OutlinedButton`. Implemented `ComposerImageAttachmentSection` using the existing composer image state/provider interactions, wired it into `ProjectComposerSheet`, and verified `flutter test test/projects/widgets/project_composer_images_test.dart` passed.
- 2026-06-11: Review-fix loop 2 (`AT-008` / `IT-005`, `FR-017`, `NFR-004`; `AC-015`, `AC-016`, `AC-020`) extended project composer feedback coverage so loading state disables the provider-backed add-photo action, image alt-text field and remove control. Red/fix context: after rendering real image tiles, existing tests that targeted the first `TextField` had to use the keyed body editor because attached image alt-text controls now render before the body. Implemented a stable body-editor key and verified `flutter test test/projects/widgets/project_composer_feedback_test.dart` passed with missing-alt, disabled-state, success and error cases.
- 2026-06-11: Review-fix loop 3 (`UT-004` / `UT-016`, `FR-002`, `NFR-003`; `AC-005`, `AC-022`, `AC-026`) added failing multi-select tests for caller-supplied add-hint/add-action/disabled/max-count copy. Red failure: `CraftskyFormBuilderMultiSelectField` did not expose `customValueHintText`, `addCustomValueLabel`, `disabledText`, or `maxSelectedErrorText`. Implemented those parameters, removed the remaining hard-coded reusable-field copy from rendering paths, supplied project composer localized strings from ARB/generated `AppLocalizations`, updated `UT-016` localization coverage, and verified `flutter test test/theme/craftsky_form_builder_multi_select_test.dart`, `flutter test test/l10n/project_composer_l10n_test.dart`, and focused project composer image/submit/feedback tests passed.
- 2026-06-11: Final full-suite verification initially found one `AT-009` regression in `project_composer_discard_test.dart`: the metadata draft test tapped the first craft dropdown after the new image attachment section increased vertical layout height, so the dropdown was below the widget-test viewport and `Embroidery` was never opened. This was a test harness targeting issue rather than a behavior regression. Updated the test to `ensureVisible` the craft dropdown before tapping it, verified `flutter test test/projects/widgets/project_composer_discard_test.dart`, then reran final `flutter analyze` and full `flutter test` successfully.
- 2026-06-11: User manual UI comment after the review-fix commit requested additional project-composer polish before moving on: permanent above-field labels for all field types, removal of stray left/internal padding, dropdown height parity with text fields, status as a dropdown, searchable option multi-selects with selected chips instead of a large chip cloud, pattern as an expandable section, localized placeholder text for all text inputs, bottom safe-area padding handled as list spacing, and moving the rich-text body field directly below craft type. These map to approved requirements `FR-001`, `FR-002`, `FR-009`, `FR-020`, `FR-023`, `FR-024`, `NFR-001`, and `NFR-003`; implement as follow-up TDD loops before final stage exit.
- 2026-06-11: UI polish loop (`UT-003`, `FR-002`, `NFR-001`; `AC-003`, `AC-005`, `AC-020`) added failing field tests for permanently above-field select labels and reduced internal padding. Red failure: `CraftskyFormBuilderDropdownField` still used `InputDecoration.labelText` and default decorator padding. Implemented a shared select-field frame with above-field labels, zero decorator padding plus consistent horizontal content padding, and reused it for dropdown/radio/multi-select fields.
- 2026-06-11: UI polish loop (`AT-003`, `FR-009`, `FR-024`; `AC-007`, `AC-027`) added failing project-composer render checks for status as a dropdown and the rich-text body field directly below craft type. Red failure: status rendered as a radio field and body appeared above the FormBuilder metadata. Replaced status with `CraftskyFormBuilderDropdownField`, moved `FacetAutocompleteEditor` below craft type, and preserved the Finished default token.
- 2026-06-11: UI polish loop (`UT-004`, `AT-007`, `FR-002`, `FR-023`; `AC-005`, `AC-025`, `AC-026`) added failing tests for searchable known-option multi-selects. Red failure: colours/design tags rendered all option chips immediately. Implemented searchable option input with filtered result chips and selected chips/removal, while keeping the materials free-text add flow.
- 2026-06-11: UI polish loop (`AT-007`, `FR-020`; `AC-019`) changed pattern from a one-way “Add pattern” action to an expandable Pattern section with `maintainState`. Red failure: tests expected a Pattern disclosure and no fields until expansion. Implemented the expansion tile and updated submit/metadata tests to expand Pattern before entering optional pattern values.
- 2026-06-11: UI polish loop (`UT-016`, `NFR-003`; `AC-022`) added failing localization coverage for project field placeholders and searchable multi-select hints. Added ARB strings and regenerated Flutter localization files; project text fields now pass localized hints for title, pattern, sewing notes/sizes, gauge values and finished size.
- 2026-06-11: UI polish loop (`AT-003`, `FR-008`, `NFR-001`; `AC-007`, `AC-020`) added render coverage for `SafeArea(bottom: false)` and an explicit bottom spacing box keyed `project-composer-bottom-safe-space`. Implemented bottom safe-area handling as list spacing instead of SafeArea padding.
- 2026-06-11: User manual UI comment after the first polish commit requested a second pass: non-text inputs should match text-field height, select-field borders must not clip, searchable option fields should avoid showing selected items twice, expansion-section dividers need bottom spacing, left padding should align cleanly across fields, and searchable options should render as a single-column list instead of a chip grid. These remain within approved UI requirements `FR-002`, `FR-020`, `FR-023`, `NFR-001`, and `NFR-003`.
- 2026-06-11: Second UI polish loop (`UT-003`, `FR-002`, `NFR-001`; `AC-003`, `AC-005`, `AC-020`) added a failing dropdown-field parity check comparing the select frame height against a text field. Red failure: the select frame was 12 px taller than the text field after the first polish pass. Reduced select-frame vertical padding while keeping a minimum field height, which also stopped the clipped-border look on select/search fields.
- 2026-06-11: Second UI polish loop (`UT-004`, `AT-007`, `FR-002`, `FR-023`; `AC-005`, `AC-025`, `AC-026`) added failing search-result tests for known-option multi-selects: results should render as a single-column list and exclude values that are already selected. Red failure: matching options rendered in a chip grid and repeated already-selected values. Updated searchable option results to a one-column tappable list and filtered out selected values from the search results while keeping selected chips above the input.
- 2026-06-11: Second UI polish loop (`AT-007`, `FR-020`, `NFR-001`; `AC-019`, `AC-020`) added bottom spacing to the Pattern and More project details expansion sections via `childrenPadding`, so the divider and collapsed/expanded boundaries have breathing room.
- 2026-06-11: User manual UI comment after the second polish commit requested a third pass for searchable multi-select behaviour: the field should behave more like an inline chips-plus-search control, similar to the provided screenshot, and the active query should clear after selecting an option. This remains within approved requirements `FR-002`, `FR-023`, and `NFR-001`. The user explicitly allowed research into existing Flutter package patterns, but dependency changes are still out of scope unless separately requested.
- 2026-06-11: Research note for the third pass: reviewed current public Flutter chip/autocomplete patterns (for example, chip-input and autocomplete-field package docs) to confirm common behaviours such as inline selected chips, list-based suggestions, and `clearInputOnSelect`. No dependency was added; the implementation stayed on existing app dependencies.
- 2026-06-11: Third UI polish loop (`UT-004`, `AT-007`, `FR-002`, `FR-023`, `NFR-001`; `AC-005`, `AC-025`, `AC-026`) updated the known-option multi-select interaction to match the requested inline pattern more closely. Red failures: the field still rendered selected chips separately from the search input, and the active search query persisted after choosing an option. Implemented inline selected `InputChip`s with the search field in the same control surface, changed results to a one-column tappable list, filtered out already-selected values from the search results, and cleared the active query after a successful option selection while preserving the limit message when selection is blocked.
- 2026-06-11: User manual UI comment after the third polish commit requested one more inline chip/search adjustment: the embedded input must be centered and sized correctly inside the decorated field surface. This remains within approved requirements `FR-002`, `FR-023`, and `NFR-001`.
- 2026-06-11: Fourth UI polish loop (`UT-004`, `FR-002`, `FR-023`, `NFR-001`; `AC-003`, `AC-005`, `AC-025`) added a failing alignment/size test for the inline chip-plus-search control. Red failure: the search field used a fixed narrow width inside a wrap layout, which left it visually mis-sized and top-aligned within the decorated field surface. Replaced that section with a horizontally scrolling selected-chip strip plus an expanded search input in the same row, keeping the query-clear-on-select behaviour and one-column results.

## Verification Results
- Focused and nearby TDD commands passed throughout the loops, including project option, field, payload, chooser, project composer render/submit/details/validation/metadata/feedback/discard/provider/image tests.
- Regression commands passed for regular composer, project models, profile posts/replies and context-menu presentation.
- Final static check: `cd app && flutter analyze` — passed, `No issues found!`.
- Final full suite: `cd app && flutter test` — passed, `All tests passed!` after the third manual UI polish follow-up.
- Final focused follow-up verification: `cd app && flutter analyze` and `cd app && flutter test test/theme/craftsky_form_builder_multi_select_test.dart` — both passed after the fourth inline chip/search adjustment.
- Build runner was not required because no Riverpod routes/providers or mappable classes were added or changed.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped
