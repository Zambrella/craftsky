# Coding Plan: Flutter Project Composer UI

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Review verdict: approved with notes; no blocking gaps.
- Carry-forward review notes:
  - Preserve acceptance-criteria-based traceability where test metadata is broader than individual scenarios (`DR-001`).
  - Split broad acceptance scenarios into focused TDD increments where useful (`DR-002`).
  - Resolve text-length assertions from existing constants/lexicon limits, not invented values (`DR-003`).
  - Keep manual accessibility/responsive/density checks in the release checklist (`DR-004`).

## 2. Implementation Strategy

Build the slice as a Flutter-only UI addition that reuses the existing create, image, facet and cache plumbing. Keep the regular composer and reply flows intact, and add a separate full-screen project composer opened from a responsive post-type chooser at top-level entry points.

The strategy is:

1. Add UI-only option catalogs and pure payload/validation helpers under `app/lib/projects/` before any widget work, because composer controls and submit tests depend on stable token values.
2. Add reusable Craftsky FormBuilder field wrappers under `app/lib/theme/`, wrapping existing `BrandTextField` and Material selection controls so values flow through `FormBuilderState` by field name.
3. Extract only the regular composer image attachment UI that must be shared with the project composer, preserving the existing `composerImagesProvider` and regular composer behaviour.
4. Add `ProjectComposerSheet` as a separate full-screen route that uses the existing facet-aware body editor, existing composer image state, a `FormBuilder` metadata form, and `CreatePost.create(project: ..., reply: null)`.
5. Add a responsive top-level post-type chooser that uses the existing Craftsky context-menu presentation and forwards the chosen composer result. Reply actions continue to call `showPostComposerSheet(..., replyTarget: post)` directly.

No AppView API, lexicon, migration, dependency, DTO enum, direct-PDS, project-rendering, project-editing or generated token-pipeline work is planned.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Project token representation | `Project` DTOs are string-backed and open-token compatible | Add UI-only option objects/catalogs mapping labels to known string tokens; do not change DTO constructors | FR-004, FR-005, RULE-002, RULE-003, FR-024 | UT-006, REG-002 |
| Form fields | `BrandTextField`; ad hoc `FormBuilderField`; `FormBuilderRadioGroup` in report UI | Add reusable Craftsky FormBuilder text, multiline, dropdown, multi-select and radio fields | FR-001, FR-002, FR-003, NFR-001 | UT-001..UT-005 |
| Project payload creation | Existing project models and `toCreateMap()` omit empty arrays | Add pure composer payload builder/validators for common, pattern and detail variants | FR-005, FR-011..FR-015, FR-019..FR-024, RULE-004 | UT-007..UT-013 |
| Composer media/text | `PostComposerSheet` owns private image UI and facet-aware body editor | Extract reusable image attachment section; project composer uses `FacetAutocompleteEditor` directly | FR-009, FR-016, FR-017, NFR-004 | AT-004, AT-008, IT-005, REG-001 |
| Project composer route | No project composer UI | Add full-screen project composer route/sheet on root navigator | BR-001, BR-003, FR-008..FR-018, FR-021 | AT-003..AT-009, IT-004 |
| Top-level composer entry | Feed/profile top-level buttons call `showPostComposerSheet` directly | Replace top-level buttons with result-forwarding post-type chooser; keep replies direct | BR-002, FR-006, FR-007, FR-025 | AT-001, AT-002, AT-010, IT-001..IT-003, REG-003, REG-004 |
| Localisation/copy | User-visible strings in `app/lib/l10n/app_en.arb` and generated accessors | Add chooser, project composer, field, validation and helper strings; reuse regular composer discard/missing-alt strings where required | NFR-003 | UT-016, MAN-003 |
| Static/build checks | `flutter analyze`, `flutter test`, build runner when generated code changes | Keep dependencies unchanged; run l10n/codegen/build-runner only when implementation changes require it | NFR-002, NFR-005 | IT-006, REG-005 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/projects/options/project_option.dart` | Create | UI-only option model(s): label, value token, optional description/group/parent metadata | FR-004, RULE-002, RULE-003 | UT-006 |
| `app/lib/projects/options/project_option_catalogs.dart` | Create | Hand-maintained catalogs from current lexicons: craft, status, difficulty, project type, subtype, yarn, needle, hook, gauge units, quilting, colours, design tags | FR-004, FR-022, FR-024 | UT-006, UT-013 |
| `app/lib/projects/composer/project_composer_fields.dart` | Create | Central field-name constants used by widget, payload builder and tests | FR-003, FR-009 | UT-007, AT-004 |
| `app/lib/projects/composer/project_composer_payload.dart` | Create | Pure helpers to trim form values, enforce UI-safe gauge/detail rules, and build `Project` | FR-005, FR-011..FR-015, FR-019..FR-024, RULE-004 | UT-007..UT-013 |
| `app/lib/projects/composer/project_composer_draft_state.dart` | Create | Pure draft-detection helper for body/images/form values and collapsed detail values | FR-018, FR-010 | UT-014, AT-009 |
| `app/lib/projects/composer/project_composer_submit_adapter.dart` | Create | Small adapter/helper to build submit arguments: facets, images, project, `reply == null` | FR-016, RULE-001 | UT-015, IT-004 |
| `app/lib/projects/widgets/project_composer_sheet.dart` | Create | Full-screen project composer route and widget | BR-001, BR-003, FR-008..FR-018, FR-021, FR-025 | AT-003..AT-009, IT-004, IT-005 |
| `app/lib/projects/widgets/project_composer_sections.dart` | Create | Stateless primary/common, pattern and craft-specific detail sections used by the sheet | FR-009..FR-014, FR-020, FR-022, FR-023 | AT-003, AT-005, AT-007 |
| `app/lib/theme/craftsky_form_builder_text_field.dart` | Create | `FormBuilderField<String>` wrapper over `BrandTextField`; supports single/multiline via parameters | FR-001, FR-003, NFR-001 | UT-001, UT-002 |
| `app/lib/theme/craftsky_form_builder_select_fields.dart` | Create | Dropdown, multi-select and radio FormBuilder fields using Material controls and Craftsky decoration | FR-002, FR-003, NFR-001 | UT-003..UT-005 |
| `app/lib/feed/widgets/composer_image_attachment_section.dart` | Create | Extract existing photo header, add-card, reorderable list, image tile, alt text and notice-compatible UI from `PostComposerSheet` | FR-009, FR-017, NFR-004 | AT-003, AT-008, IT-005, REG-001 |
| `app/lib/feed/widgets/post_composer_sheet.dart` | Change | Use extracted image section; optionally use shared text limit constant; preserve public `showPostComposerSheet` API | BR-002, FR-007, NFR-004 | REG-001, AT-002, IT-003 |
| `app/lib/feed/widgets/post_type_chooser.dart` | Create | Result-forwarding responsive chooser with “Regular post” and “Project post” menu items | FR-006, FR-007, FR-008, FR-025 | AT-001, AT-002, AT-010, IT-001, IT-003 |
| `app/lib/feed/widgets/post_composer_entry_button.dart` | Create | Reusable `ChunkyButton` wrapper that computes anchor `RelativeRect` and opens the chooser | FR-006, FR-025 | IT-001, IT-002 |
| `app/lib/theme/craftsky_context_menu.dart` | Change | Allow async item actions so `showCraftskyContextMenu` can await selected composer result without breaking existing sync callers | FR-006, FR-025 | REG-004, AT-010 |
| `app/lib/feed/pages/feed_page.dart` | Change | Replace top-level `showPostComposerSheet(context)` with chooser entry button; keep reply branch direct | FR-006, FR-025, BR-002 | IT-001, IT-003 |
| `app/lib/profile/widgets/profile_tabs/profile_posts_tab.dart` | Change | Replace owner top-level post button with chooser entry; keep non-owner and reply behaviour unchanged | FR-006, FR-025, BR-002 | IT-002, IT-003, REG-003 |
| `app/lib/l10n/app_en.arb` and generated localisation files | Change | Add user-visible strings; use sentence case, British English, no emoji | NFR-003 | UT-016 |
| `app/test/projects/options/project_option_catalogs_test.dart` | Create | First failing test: representative catalog labels/tokens, Finished default token, DTOs remain string-backed | FR-004, FR-024, RULE-002, RULE-003 | UT-006 |
| `app/test/theme/craftsky_form_builder_*_test.dart` | Create | Field adapter/FormBuilder integration tests | FR-001..FR-003, NFR-001 | UT-001..UT-005 |
| `app/test/projects/project_composer_*_test.dart` | Create | Pure payload, subtype, validation, draft and submit adapter tests | FR-005, FR-011..FR-024, RULE-001, RULE-004 | UT-007..UT-015 |
| `app/test/projects/widgets/project_composer_*_test.dart` | Create | Project composer widget tests for render, submit, details, validation, metadata, feedback, discard/images/provider | BR-001, BR-003, FR-008..FR-018, FR-021 | AT-003..AT-009, IT-004, IT-005 |
| `app/test/feed/widgets/post_type_chooser*_test.dart` | Create | Chooser compact/popup branches, regular/project selection, result forwarding, reply bypass regression | FR-006, FR-007, FR-025 | AT-001, AT-002, AT-010, IT-003 |
| Existing regular composer/profile tests | Change/keep passing | Protect facets/photos/replies/discard and profile top-level/reply behaviour | BR-002, NFR-004 | REG-001, REG-003 |

## 5. Services, Interfaces, And Data Flow

### No backend service changes

The submit boundary remains `CreatePost.create` in `app/lib/feed/providers/create_post_provider.dart`:

```text
CreatePost.create({
  required String text,
  PostReply? reply,      // must be null for project posts
  Project? project,      // populated by project composer
  List<CreatePostImage>? images,
  List<Map<String, dynamic>>? facets,
})
```

The API client and repository already accept `project?.toCreateMap()` and enforce top-level project create. Do not change AppView routes or API payload shape.

### UI option catalog shape

Use UI-only option classes. Keep project models as strings.

```text
class ProjectOption {
  const ProjectOption({
    required this.value,       // e.g. social.craftsky.feed.defs#finished
    required this.label,       // e.g. Finished
    this.description,
    this.group,
    this.parentValue,          // for subtype filtering
  });

  final String value;
  final String label;
  final String? description;
  final String? group;
  final String? parentValue;
}

abstract final class ProjectOptionCatalogs {
  static const finishedStatusToken = 'social.craftsky.feed.defs#finished';
  static const craftTypes = <ProjectOption>[...];
  static const statuses = <ProjectOption>[...];
  static List<ProjectOption> projectTypesForCraft(String craftToken);
  static List<ProjectOption> projectSubtypesFor({
    required String craftToken,
    required String projectTypeToken,
  });
}
```

Catalogs must include all current known values from:

- `lexicon/social/craftsky/feed/defs.json`
- `lexicon/social/craftsky/project/defs.json`
- `lexicon/social/craftsky/project/{knitting,crochet,sewing,quilting}.json`
- `lexicon/social/craftsky/project/{knitting,crochet,sewing,quilting}.defs.json`

Tests may verify representative values rather than every entry, but implementation should centralize all currently known options used by the UI. Do not add a generated token pipeline in this slice.

### Payload builder and validation helpers

Keep form parsing pure and testable. The widget should not hand-build nested DTOs inline.

```text
class ProjectComposerValidationError {
  const ProjectComposerValidationError(this.fieldName, this.messageKey);
  final String fieldName;
  final String messageKey; // or enum consumed by widget for l10n
}

class ProjectComposerPayloadResult {
  const ProjectComposerPayloadResult.project(this.project);
  const ProjectComposerPayloadResult.errors(this.errors);
  final Project? project;
  final List<ProjectComposerValidationError> errors;
}

ProjectComposerPayloadResult buildProjectComposerPayload({
  required Map<String, dynamic> formValues,
  required String? activeCraftType,
});
```

Builder rules:

- Trim strings and treat whitespace-only optional values as absent.
- Required metadata: `craftType`.
- Default status to `social.craftsky.feed.defs#finished` if not changed.
- Materials: free-text list, max 20, omit empty.
- Colours/design tags: known string lists, max 10 each, omit empty.
- Pattern: only create `ProjectPattern` when at least one pattern field is non-empty.
- Details:
  - `sewing` -> `SewingProjectDetails` only if project type/subtype/size made/fit notes contains a value.
  - `knitting` -> `KnittingProjectDetails` only if any knitting detail value or valid gauge is present.
  - `crochet` -> `CrochetProjectDetails` only if any crochet detail value or valid gauge is present.
  - `quilting` -> `QuiltingProjectDetails` only if any quilting detail value is present.
  - `embroidery` and future/common-only craft tokens -> no `details`.
- Gauge:
  - Entirely empty gauge => omitted.
  - If any gauge field is present, require positive whole-number stitches, positive whole-number measurement and selected unit.
  - Rows are optional but, when present, must be a positive whole number.
  - Unit known values: `cm`, `in`.

### Submit data flow

```text
User taps Post
  -> ProjectComposerSheet validates body length/body required/photo required/FormBuilder metadata
  -> payload builder returns Project or field errors
  -> if missing alt text: reuse showCraftskyConfirmDialog copy from regular composer
  -> facetGeneratorProvider.generate(trimmedBody)
  -> composerImagesState.toCreatePostImages()
  -> createPostProvider.notifier.create(
       text: trimmedBody,
       reply: null,
       project: project,
       images: createImages,
       facets: facets.isEmpty ? null : facets,
     )
  -> existing CreatePost cache path prepends timeline/user project caches
```

## 6. State, Providers, Controllers, Or DI

### Provider graph

No new Riverpod provider is required unless implementation discovers a clear need. Prefer plain classes/constants and widget state for this slice.

```text
ProjectComposerSheet(composerId)
  watches createPostProvider
  watches composerImagesProvider(composerId)
  reads composerImagesProvider(composerId).notifier for image actions
  reads facetGeneratorProvider during submit
  reads createPostProvider.notifier during submit

PostTypeChooser
  calls showPostComposerSheet(context) for regular post
  calls showProjectComposerSheet(context) for project post
```

### Widget-owned state

`ProjectComposerSheet` should own:

- `GlobalKey<FormBuilderState>` for metadata.
- `FacetTextEditingController` and `FocusNode` for body text.
- Stable `composerId` (`Uuid` default, injectable for tests) for image provider family.
- Active craft token for conditional detail rendering and subtype filtering.
- Booleans for pattern section visibility and more-details expansion.
- Current body text string for length/dirty state.
- Optional form-level validation message for required photos/body/craft if field-level placement is not possible.

Use `FormBuilder.onChanged` plus small `setState` calls to keep submit enabled/disabled and clear stale subtype values.

### Context-menu async action adjustment

`CraftskyContextMenuItem.onPressed` currently takes `VoidCallback?`. To forward `Post?` results from a selected composer, change the type to `FutureOr<void> Function()?` and await selected actions in `showCraftskyContextMenu`. Existing synchronous callbacks remain source-compatible.

Guardrail: compact bottom-sheet menu rows must dismiss the sheet before awaiting the selected action, so the composer route is visible and not stacked behind the sheet.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Reusable FormBuilder field kit

Create reusable controls in `app/lib/theme/`:

```text
CraftskyFormBuilderTextField(
  name, label, hintText, helperText, validator,
  controller?, focusNode?, initialValue?, enabled,
  minLines, maxLines, keyboardType, textInputAction,
  onChanged, onSubmitted,
)

CraftskyFormBuilderDropdownField<String>(
  name, label, options: List<CraftskySelectOption<String>>,
  initialValue, enabled, validator, onChanged,
)

CraftskyFormBuilderMultiSelectField<String>(
  name, label, options?, allowCustomValues,
  maxSelected, selectedValues, helperText, enabled, validator,
)

CraftskyFormBuilderRadioField<String>(
  name, label, options, initialValue, enabled, validator, onChanged,
)
```

The text wrapper should use `FormBuilderField<String>` + `BrandTextField`, sharing focus nodes/controllers exactly like `EditProfileDialog` to avoid validation focus stealing.

Selection controls should preserve Material semantics and visible error/helper text. A practical implementation can use `DropdownButtonFormField`/`DropdownMenu` for single-select, `FormBuilderRadioGroup` where radio semantics are best, and `FormBuilderField<List<String>>` plus `InputDecorator`, chips and a Material menu/sheet for multi-select/free-text values.

### Project composer composition

```text
Scaffold(fullscreenDialog route)
  AppBar(title: Project post, action: Post / StitchProgressIndicator)
  SafeArea
    SingleChildScrollView
      ComposerImageAttachmentSection(requiredForProject: true)
      FacetAutocompleteEditor(label: Body, max 2000, required)
      FormBuilder
        Project title text field
        Craft type dropdown/radio
        Status radio/dropdown default Finished
        Materials multi-value free-text chips
        Colours known multi-select chips
        Design tags known multi-select chips
        Add pattern button / expandable pattern fields
        More project details ExpansionTile
          switch(activeCraftType)
            sewing fields
            knitting fields + gauge group
            crochet fields + gauge group
            quilting fields
            other/common-only helper
```

Primary fields remain visible and usable when “More project details” is collapsed. Collapsed active-craft details remain in FormBuilder state and may be submitted unless the user clears them.

### Top-level chooser

Add `showPostTypeChooser` or equivalently named helper in `feed/widgets/post_type_chooser.dart`:

```text
Future<Post?> showTopLevelPostComposerChooser(
  BuildContext context, {
  required RelativeRect position,
}) async
```

Menu rows:

- “Regular post” — brief description; opens existing `showPostComposerSheet(context)`.
- “Project post” — brief description; opens `showProjectComposerSheet(context)`.

Presentation comes from `showCraftskyContextMenu`: bottom sheet at compact widths (`<= 900`) and anchored popup/menu on larger widths. The helper must resolve with the created `Post?`, or `null` if the chooser/composer is dismissed.

`FeedPage` and own-profile `ProfilePostsTab` should use a small reusable entry button that computes the anchor position from its render box. Reply actions in feed/thread/profile comments/profile posts must remain direct calls to the regular composer.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Empty chooser dismiss | `showTopLevelPostComposerChooser` resolves `null`; no providers touched | FR-006, FR-025 | AT-010, EC-001 |
| Regular branch | Existing `PostComposerSheet` opens; no project fields/validation | BR-002, FR-007 | AT-002, IT-003, REG-001 |
| Reply action | Bypass chooser and call `showPostComposerSheet(replyTarget: post)` | FR-025, BR-002 | AT-010, IT-003 |
| Project composer initial state | Status defaults to Finished token; pattern hidden; details collapsed; no images/body | FR-020, FR-024 | AT-003, AT-007, UT-007 |
| Missing body | Submit blocked; body error near facet editor/form-level message | FR-021 | AT-006 |
| Body too long | Use existing 2000 character composer limit or shared constant extracted from it | FR-021 | AT-006, DR-003 |
| Missing craft type | FormBuilder validator shows craft field error and submit blocked | FR-021 | AT-006 |
| No project photo | Submit blocked with project-photo validation message; regular text-only posts unaffected | FR-021, RULE-005 | AT-006 |
| Image upload in progress/failed | Submit disabled through `imagesState.canSubmitImages()`; image notices mirror regular composer | FR-017 | AT-008, IT-005 |
| Missing alt text | Reuse regular missing-alt confirmation; not hard validation | FR-017 | AT-008, IT-005 |
| Create loading | Disable text, field controls, image actions and submit; show spinner in app-bar action | FR-017, NFR-001 | AT-008, IT-004 |
| Create success | Pop composer with created `Post`, show existing success snackbar, reset provider | FR-017, FR-025 | AT-008, IT-004 |
| Create error | Keep composer open, show existing error snackbar, reset provider for retry | FR-017 | AT-008, IT-004 |
| Draft close with changes | Use `PopScope`; confirm before discarding body/images/form changes | FR-018 | AT-009, UT-014 |
| Craft switch after details | Submit only active-craft details; clear invalid subtype when type/craft changes | FR-010..FR-014, FR-022 | AT-005, UT-013 |
| Gauge partial/invalid | Field-level errors; omit only when entirely empty | FR-021 | AT-006, UT-009, UT-010 |
| Materials max | Prevent 21st material, explain limit, allow removals | FR-019 | AT-007, UT-004 |
| Colours/design tags max | Prevent 11th selection, explain limit, allow removals | FR-023 | AT-007, UT-004 |
| Optional empty fields | Trim and omit/null empty values; no empty detail object | RULE-004 | UT-007..UT-012 |
| Common-only craft | Embroidery/future craft submits `common.craftType` and no details | FR-015, RULE-003 | AT-004, UT-012 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-006 | `app/test/projects/options/project_option_catalogs_test.dart` | Import new catalog; assert craft/status/difficulty/detail representative tokens and Finished default | Missing catalog files/classes |
| 2 | UT-013 | `app/test/projects/options/project_subtype_filter_test.dart` | Representative type/subtype pairs for sewing/knitting/crochet/quilting | No filtering helper/metadata |
| 3 | UT-001, UT-002 | `app/test/theme/craftsky_form_builder_text_field_test.dart` | `FormBuilder` harness with named fields, validators, reset, disabled state | Missing field wrapper |
| 4 | UT-003 | `app/test/theme/craftsky_form_builder_dropdown_test.dart` | Options with string tokens and validator | Missing dropdown wrapper |
| 5 | UT-004 | `app/test/theme/craftsky_form_builder_multi_select_test.dart` | Free-text materials max 20 and known colours/design tags max 10 | Missing multi-select wrapper/count handling |
| 6 | UT-005 | `app/test/theme/craftsky_form_builder_radio_test.dart` | Status/craft radio options in FormBuilder | Missing radio wrapper |
| 7 | UT-007 | `app/test/projects/project_composer_payload_test.dart` | Common form values, status default, empty optionals, materials/colours/design tags, pattern | No payload builder |
| 8 | UT-008..UT-012 | `app/test/projects/project_composer_payload_test.dart` | Sewing/knitting/crochet/quilting/common-only inputs | No detail builders/gauge validation |
| 9 | UT-014 | `app/test/projects/project_composer_draft_state_test.dart` | Empty vs changed body/images/form/detail values | No draft helper |
| 10 | UT-015 | `app/test/projects/project_composer_submit_adapter_test.dart` | Body with facets, uploaded image state, valid `Project` | No submit adapter/helper |
| 11 | AT-001, AT-002 | `app/test/feed/widgets/post_type_chooser_test.dart` | Compact and large `MediaQuery`; fake repository for selected branch | No chooser; top-level button opens regular composer directly |
| 12 | AT-010 | `app/test/feed/widgets/post_type_chooser_result_test.dart` and profile tests | Created/dismissed composers; reply branch harness | Chooser does not forward result; replies may incorrectly show chooser |
| 13 | AT-003 | `app/test/projects/widgets/project_composer_sheet_test.dart` | Open project composer with localization/theme | Missing project composer primary fields/default status |
| 14 | AT-004 | `app/test/projects/widgets/project_composer_submit_test.dart` | Fake `PostRepository`, uploaded photo override, embroidery common-only input | No create call/project payload |
| 15 | AT-005 | `app/test/projects/widgets/project_composer_details_test.dart` | Select each craft; expand details; project type/subtype changes | No craft-specific detail sections/filtering |
| 16 | AT-006 | `app/test/projects/widgets/project_composer_validation_test.dart` | Missing body/craft/photo, over-limit body, invalid gauge cases | Submit incorrectly enabled/no errors |
| 17 | AT-007 | `app/test/projects/widgets/project_composer_metadata_test.dart` | Add materials/colours/design tags, exceed limits, add pattern | No metadata UI/serialization |
| 18 | AT-008 | `app/test/projects/widgets/project_composer_feedback_test.dart` | Missing alt image, delayed success/error repository | No missing-alt/loading/success/error parity |
| 19 | AT-009 | `app/test/projects/widgets/project_composer_discard_test.dart` | Body/image/form changed cases | No project draft confirmation |
| 20 | IT-001 | `app/test/feed/pages/feed_page_composer_entry_test.dart` | `FeedPage`, compact width, fake timeline/repo | Feed entry still opens regular composer directly |
| 21 | IT-002 | `app/test/profile/widgets/profile_posts_tab_project_composer_test.dart` | Owner/non-owner profile tab harness | Profile entry not updated/result not propagated |
| 22 | IT-003 | `app/test/feed/widgets/post_type_chooser_regular_branch_test.dart` | Regular top-level and reply fake create captures | Project payload leaks into regular/reply flows |
| 23 | IT-004 | `app/test/projects/widgets/project_composer_provider_test.dart` | Success/error create provider integration | Loading/reset/snackbar/popup behaviour missing |
| 24 | IT-005 | `app/test/projects/widgets/project_composer_images_test.dart` | Override `composerImagesProvider` with notices/missing alt | Project composer not wired to image provider notices |
| 25 | REG-001..REG-004 | Existing/new regression suites | Run existing composer/profile/context-menu tests | Regressions from extraction/chooser |
| 26 | UT-016 | `app/test/l10n/project_composer_l10n_test.dart` | ARB/localization resource inspection | Missing/nonconforming copy |
| 27 | IT-006, REG-005 | Commands | `cd app && flutter analyze`; `cd app && flutter test`; codegen if needed | Static/test failures or unintended dependency changes |

Focused command examples during TDD:

```text
cd app && flutter test test/projects/options/project_option_catalogs_test.dart
cd app && flutter test test/theme/craftsky_form_builder_text_field_test.dart
cd app && flutter test test/projects/project_composer_payload_test.dart
cd app && flutter test test/projects/widgets/project_composer_submit_test.dart
cd app && flutter analyze
```

## 10. Sequencing And Guardrails

- First TDD step: write `UT-006` for `ProjectOptionCatalogs`, especially `social.craftsky.feed.defs#finished`, representative craft tokens, and representative detail tokens.
- Dependencies between work items:
  1. Option catalogs before fields consuming token options.
  2. Field wrappers before project composer metadata UI.
  3. Payload/subtype/gauge helpers before submit widget tests.
  4. Image UI extraction before project composer image parity tests.
  5. Project composer route before chooser project branch tests can pass.
  6. Async context-menu/chooser changes before result-forwarding tests.
- Guardrails:
  - Do not change `Project`, `ProjectCommon`, `ProjectPattern`, `ProjectDetails` fields to enums.
  - Do not change AppView API, lexicon JSON, migrations, dependencies or direct-PDS behaviour.
  - Do not support project replies; pass `reply: null` for every project create.
  - Keep regular/reply `showPostComposerSheet` path valid and light.
  - Extract shared composer image UI only as needed; avoid a broad composer refactor.
  - Use existing post body limit (`2000`) by extracting or reusing a shared constant; do not invent a new text limit.
  - Reuse existing missing-alt confirmation and image notice messages where applicable.
  - Localize new copy in ARB resources; no hard-coded user-facing copy in widgets except tests.
  - Respect open lexicon known-values: catalogs constrain MVP UI choices, not DTO/data-layer acceptance.
  - Treat collapsed active-craft details as form state that can still submit unless cleared.
- Out of scope:
  - Project post rendering/cards/detail pages/editing.
  - Generated lexicon-to-UI catalog pipeline.
  - Project hashtag-to-`ProjectCommon.tags` merging.
  - Backend/API/database/firehose/lexicon changes.
  - New dependencies.
  - Draft autosave, scheduled posts, quote posts or project replies.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact widget implementation for multi-select dropdown/chips is not prescribed by existing code | Could affect test ergonomics and accessibility polish | Use Material controls (`InputDecorator`, chips, list/check controls, menu/sheet) and verify FormBuilder value/error/disabled semantics in UT-004/MAN-001 |
| CPQ-002 | Non-blocking | Token catalog completeness is hand-maintained and can drift | Future lexicon additions may not appear in UI | Centralize catalogs, add representative tests, keep generated pipeline explicitly out of scope/future work |
| CPQ-003 | Non-blocking | Context menu currently has sync `VoidCallback` actions | Result forwarding can race unless async actions are awaited | Change item action type to `FutureOr<void> Function()?` and add REG-004/AT-010 coverage |
| CPQ-004 | Non-blocking | Existing regular composer image UI is private inside `PostComposerSheet` | Duplication or risky extraction could regress regular composer | Extract the smallest reusable `ComposerImageAttachmentSection`, run existing composer image/discard/facet tests immediately after extraction |
| CPQ-005 | Non-blocking | Text-length requirement lacks a newly named project constant | Tests could assert the wrong value | Extract/reuse existing 2000-character composer limit and reference it in tests (`DR-003`) |
| CPQ-006 | Non-blocking | Dense detail catalogs may produce a long composer | UX may feel heavy | Keep details collapsed, split sections, cover with MAN-004 |
| CPQ-007 | Non-blocking | Generated l10n files may need updating depending on repo tooling | Analysis/tests may fail if generated accessors are stale | Run Flutter localization generation as part of implementation if new ARB keys are added; include generated localization files only when required by repo conventions |

Blocking open questions: none.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-006` in `app/test/projects/options/project_option_catalogs_test.dart`
- First focused command: `cd app && flutter test test/projects/options/project_option_catalogs_test.dart`
- Source of truth: `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this coding plan.
- Notes:
  - Build in small failing-test increments even when an acceptance scenario groups multiple behaviours.
  - Keep regular composer regressions close to every extraction/chooser change.
  - Final verification should include `cd app && flutter analyze`, `cd app && flutter test`, and build runner/localization generation if implementation changes generated files.
