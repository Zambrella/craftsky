# Requirements: Flutter Project Composer UI

## 1. Initial Request

After completing the Flutter non-UI project-post plumbing, add the first project-post UI slice. The slice should introduce Craftsky-styled input field types built on Material/FormBuilder where possible, then use them in a full project composer MVP. The composer must handle the richer UX needed for project posts, including photos, title, body, craft type, status, pattern, materials and expandable craft-specific details. The plan must also resolve how Flutter should represent AppView/lexicon known-value tokens and how users choose between regular posts and project posts.

## 2. Current Codebase Findings

- Relevant files:
  - Existing regular composer: `app/lib/feed/widgets/post_composer_sheet.dart`.
  - Existing create plumbing: `app/lib/feed/providers/create_post_provider.dart`, `app/lib/feed/data/post_api_client.dart`, `app/lib/feed/data/post_repository.dart`.
  - Project models/providers from the previous slice: `app/lib/projects/models/project.dart`, `app/lib/projects/providers/user_projects_provider.dart`.
  - Existing Craftsky input styling: `app/lib/theme/brand_text_field.dart`, `app/lib/theme/app_theme.dart`, `app/lib/theme/theme_extensions.dart`.
  - Existing FormBuilder patterns: `app/lib/profile/pages/edit_profile_dialog.dart`, `app/lib/moderation/widgets/report_subject_sheet.dart`.
  - Existing composer tests: `app/test/feed/widgets/post_composer_sheet_facets_test.dart`, `post_composer_sheet_discard_test.dart`, composer image provider/media tests.
  - Project lexicons and token sources: `lexicon/social/craftsky/feed/defs.json`, `lexicon/social/craftsky/project/defs.json`, `lexicon/social/craftsky/project/{knitting,crochet,sewing,quilting}.json`.
- Existing patterns:
  - `BrandTextField` wraps Material `TextField` with Craftsky labels, helper/error text and focus-lift styling while relying on the app `InputDecorationTheme`.
  - Custom Craftsky fields can be integrated with `FormBuilderField` by forwarding `field.didChange`, `field.errorText`, `initialValue` and shared focus nodes/controllers.
  - Built-in `FormBuilderRadioGroup` is already used for report forms when Material semantics are sufficient.
  - Regular post composer is a full-screen `MaterialPageRoute` pushed on the root navigator; it handles text facets, image upload/reorder/alt text, discard confirmation, submit state and success/error snackbars.
  - `CreatePost.create` already accepts optional `Project? project` and prevents project replies.
- Current behavior:
  - Users can create regular posts and replies, but there is no project composer UI.
  - Feed and own-profile Posts tab expose a single “New post” entry that opens `PostComposerSheet` directly.
  - Project model fields are string-backed wire/data objects; there are no UI-facing option classes/catalogs for known lexicon tokens.
  - There are no reusable Craftsky dropdown, multi-select dropdown or radio field components beyond ad hoc usage of existing Material/FormBuilder widgets.
- Constraints discovered:
  - Flutter app must keep public writes through the AppView; no direct PDS writes or PDS token handling.
  - No AppView, lexicon, migration or dependency changes are needed for this slice.
  - Lexicon `knownValues` are open; Flutter must not make the data model unable to carry future/open token strings.
  - Existing DTO/model constructors intentionally do not enforce lexicon validation; composer validation should be UI-level only.
  - Project posts are standalone top-level posts; project replies remain disallowed.
  - App copy should use sentence case, British English and no emoji in app chrome.
- Test/build commands discovered:
  - Flutter tests: `cd app && flutter test`.
  - Flutter analysis: `cd app && flutter analyze`.
  - Code generation if providers/routes/models are added or changed: `cd app && dart run build_runner build --delete-conflicting-outputs`.

## 3. Clarifying Questions And Decisions

### Q1: Which scope should `01-requirements.md` cover for the first project-post UI slice?

Answer: Full composer MVP.

Decision / implication: Requirements include reusable Craftsky FormBuilder input fields, token option catalogs, project composer screen/form, entry-point choice between regular and project post, and submit via existing create plumbing. Project viewing/card/detail UI remains out of scope.

### Q2: How should Flutter represent lexicon known-values/tokens for project composer fields?

Answer: UI option classes/catalogs.

Decision / implication: Project DTO/model fields stay string-backed. New UI-facing option objects/catalogs expose labels, token strings and filtering/grouping metadata, then map selected values to strings when constructing `Project` and `ProjectDetails` at submit time.

### Q3: How should users choose between a regular post and a project post?

Answer: Entry picker plus separate screens.

Decision / implication: Feed/Profile composer entry points should open a lightweight choice surface. Regular post keeps the existing composer flow; project post opens a separate project composer. Shared media/text components may be extracted where practical, but the regular composer must not become a heavier combined project form.

### Q4: Which craft-specific project detail sections should the project composer MVP support?

Answer: All known variants.

Decision / implication: The composer must support common project fields for all supported craft tokens, craft-specific detail sections for knitting, crochet, sewing and quilting, and common-only project posts for crafts with no detail schema such as embroidery.

### Q5: What validation level should the project composer MVP enforce before submit?

Answer: UI-safe validation.

Decision / implication: Require post body text and craft type, enforce practical/max-length limits already exposed by lexicon or existing composer, validate positive integer gauge fields when present, and omit empty optional fields. Leave open-token handling and deeper lexicon/PDS validation to AppView.

### Q6: Should the recommended direction be finalized?

Answer: Confirm Option A.

Decision / implication: Requirements use the separate project composer MVP approach with shared FormBuilder field kit, token option catalogs and entry picker.

### Q7: What decisions came out of the requirements review annotations?

Answer: The user confirmed that generated token catalogs and project hashtag-to-tag merging should be deferred. The user also confirmed the composer choice surface should use the same presentation style as the existing post composer.

Decision / implication: Requirements keep token generation and hashtag/tag normalization out of this MVP. The entry picker is no longer an open visual-treatment question; it should use the existing full-screen composer/task presentation pattern rather than a bottom sheet, dialog or separate permanent page.

## 4. Candidate Approaches

### Option A: Separate Project Composer Plus Shared FormBuilder Field Kit

Summary: Build reusable Craftsky-styled FormBuilder-compatible field widgets, add UI token option catalogs, preserve the existing regular composer, and add a separate project composer opened through an entry picker.

Pros:
- Keeps regular posting and replies simple.
- Gives project posts a purpose-built layout for richer structured metadata.
- Creates reusable fields for future project/edit/profile forms.
- Preserves open-token compatibility by keeping DTOs string-backed.
- Aligns with existing full-screen composer and FormBuilder patterns.

Cons:
- Larger first UI slice than a field-kit-only pass.
- Requires careful tests so existing regular composer behavior does not regress.
- Some media/text composer code may need extraction or duplication decisions.

Risks:
- All known craft detail sections increase UX/test surface area.
- Token catalogs can drift from lexicon values if not centralized.

### Option B: Single Combined Composer With Post-Type Toggle

Summary: Extend `PostComposerSheet` so one screen can switch between regular and project modes.

Pros:
- One composer route/surface to maintain.
- User chooses post type without an extra entry surface.

Cons:
- Makes the regular post flow heavier.
- Mixes project validation with reply/general-post behavior.
- Project replies require more in-screen special cases.

Risks:
- Higher regression risk to existing post/reply composer tests and UX.

### Option C: Field Kit First, Composer Later

Summary: Only create reusable FormBuilder fields and token option catalogs in this slice; defer composer screen and submit integration.

Pros:
- Lowest implementation risk.
- Focused component tests are straightforward.

Cons:
- Does not deliver a postable project composer MVP.
- Leaves entry UX and project form behavior unresolved.
- Later composer work may force field API rework.

Risks:
- Components may be over- or under-designed without a real composer consumer.

## 5. Recommended Direction

Recommended approach: Option A — separate project composer plus shared FormBuilder field kit.

Why: It delivers the requested user-facing project composer MVP while protecting the existing regular composer, follows current full-screen composer/FormBuilder patterns, enables reusable Craftsky inputs, and handles open lexicon tokens without weakening existing project DTO compatibility.

## 6. Problem / Opportunity

Craftsky project posts can now be represented and created by Flutter plumbing, but users have no way to compose them. Project posts carry more structured metadata than regular posts, so the UI needs reusable accessible input primitives, a clear post-type choice, and a project-specific composer that can collect common and craft-specific fields without overwhelming regular post creation.

## 7. Goals

- G-001: Let an authenticated user create a standalone project post from the Flutter UI.
- G-002: Provide Craftsky-styled, FormBuilder-compatible input primitives for project and future form work.
- G-003: Keep AppView project DTOs string-backed while making known token selection user-friendly.
- G-004: Support all currently modeled craft-specific project detail variants in the composer MVP.
- G-005: Preserve existing regular post and reply composer behavior.
- G-006: Keep project composer validation helpful without making the app a closed lexicon validator.

## 8. Non-Goals

- NG-001: Do not implement project post rendering in feed cards, profile Projects tab UI, project detail pages or edit-project flows.
- NG-002: Do not change AppView routes, APIs, database schema, migrations, firehose indexing or lexicon files.
- NG-003: Do not change project DTO/model fields from strings to enums.
- NG-004: Do not add dependencies.
- NG-005: Do not support project replies, quote posts, drafts, autosave or scheduled posting.
- NG-006: Do not implement a full lexicon codegen-to-UI-token pipeline; token catalogs may be hand-maintained for this slice and generated catalogs can be considered later.
- NG-007: Do not implement advanced search/filter UI for project metadata.
- NG-008: Do not merge project body hashtags into `project.common.tags` in this MVP; preserve existing facet generation only.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Craftsky member | Authenticated app user creating posts | Choose project post creation, add photos/body/project metadata, and post successfully. |
| Regular poster | User creating a non-project post or reply | Existing post/reply flow must stay simple and unchanged except for the new entry choice. |
| Flutter UI implementer | Developer building/testing the composer | Stable component, option-catalog and form requirements. |
| Test designer / implementer | Next workflow stages | Traceable requirements and acceptance criteria for component, widget, provider and regression tests. |
| AppView API | Existing backend accepting `/v1/posts` create payloads | Receives project payloads through existing Flutter create plumbing as camelCase JSON strings. |

## 10. Current Behavior

Flutter shows “New post” actions in the feed and own-profile Posts tab. Activating the action opens the regular full-screen composer, which supports text, facets, photos, alt text, discard confirmation and submit. Replies also use the same composer. There is no user-facing way to choose a project post, no project-specific form, no UI token catalogs, and no shared Craftsky dropdown/multi-select/radio field kit.

## 11. Desired Behavior

Activating a top-level post entry point shall offer a choice between a regular post and a project post. Choosing regular post shall preserve the existing regular composer. Choosing project post shall open a full-screen project composer wrapped in `FormBuilder`. The project composer shall use Craftsky-styled, Material-based accessible inputs; collect photos, body text, title, craft type, status, pattern and materials; expose a collapsible “More project details” section; render appropriate craft-specific detail fields for knitting, crochet, sewing and quilting; and submit a top-level project post through the existing `CreatePost` provider. UI token option catalogs shall provide labels and values for known lexicon tokens while producing strings for the existing DTOs.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Flutter shall provide a user-facing MVP for creating project posts. | Project posts are a core Craftsky content type and the data plumbing is now available. | Prompt, Q1 | AC-001, AC-007, AC-012 |
| BR-002 | Business | Must | Regular post and reply creation shall remain available and shall not gain project-specific friction inside their composer screen. | Protects the simple social-post flow while adding project posts. | Q3, discovery | AC-002, AC-014 |
| BR-003 | Business | Must | The project composer shall support all currently modeled known project detail variants: knitting, crochet, sewing and quilting. | Users across supported crafts need useful structured project fields from the first MVP. | Q4, lexicon findings | AC-010, AC-011 |
| FR-001 | Functional | Must | The system shall add Craftsky-styled FormBuilder-compatible plain text and multiline plain text fields that expose label, hint, helper, error, enabled, focus/controller or initial-value, keyboard/action, line-count and change/submit hooks needed by composer forms. | Project forms need reusable text fields while retaining BrandTextField styling and FormBuilder validation. | Prompt, BrandTextField discovery | AC-003, AC-004 |
| FR-002 | Functional | Must | The system shall add Craftsky-styled FormBuilder-compatible dropdown, multi-select dropdown and radio field components that use Material input/selection widgets where possible and expose option lists, selected values, labels, helper/error text, enabled state and change callbacks. | Required input types must be reusable, accessible and consistent. | Prompt, Q1 | AC-003, AC-005 |
| FR-003 | Functional | Must | The reusable field components shall integrate with `FormBuilder` so form values can be saved, validated, reset and read by field name. | The project composer must extract values and validation state through FormBuilder. | Prompt | AC-004, AC-006 |
| FR-004 | Functional | Must | The system shall add UI-facing project option catalogs/classes for craft type, status, pattern difficulty, project type, project subtype, yarn weight, needle size, hook size, gauge unit, quilting piecing technique, quilting method, colours and design tags where applicable. | Labels, filtering and token values should be centralized without changing DTO wire fields. | Q2, lexicon findings | AC-008, AC-009 |
| FR-005 | Functional | Must | Option catalog selections shall map to the existing string-backed `Project`, `ProjectCommon`, `ProjectPattern` and `ProjectDetails` model fields at submit time. | Preserves AppView DTO compatibility and open-token behavior. | Q2, previous requirements | AC-009, AC-012 |
| FR-006 | Functional | Must | The top-level post composer entry points in feed and own-profile Posts tab shall open a post-type choice surface, presented with the same full-screen root-navigator composer/task pattern as the existing post composer, with regular post and project post options. | Users need a clear way to choose post type without introducing a different modal pattern. | Q3, Q7, current entry discovery | AC-001, AC-002 |
| FR-007 | Functional | Must | Choosing regular post from the entry picker shall open the existing regular composer and preserve existing text, photo, facet, reply and submit behavior. | Prevents regressions to existing posting. | Q3, current composer discovery | AC-002, AC-014 |
| FR-008 | Functional | Must | Choosing project post from the entry picker shall open a full-screen project composer route/sheet on the root navigator, visually consistent with the existing composer and shell-covering modal task screens. | Project composition is a temporary task and should cover shell navigation like existing composers. | Current composer/router discovery | AC-001, AC-007 |
| FR-009 | Functional | Must | The project composer shall be wrapped in `FormBuilder` and shall collect body text, photos with alt text, project title, craft type, status, pattern information and materials. | These are the top-level project-post UX fields requested. | Prompt, Q1 | AC-006, AC-007, AC-012 |
| FR-010 | Functional | Must | The project composer shall include an expandable/collapsible “More project details” section for craft-specific fields and keep the primary composer fields usable when details are collapsed. | Rich metadata should not overwhelm the main compose flow. | Prompt examples | AC-010 |
| FR-011 | Functional | Must | When craft type is sewing, the project details section shall collect sewing project type, project subtype, size made and fit notes, and construct `SewingProjectDetails` only when applicable detail values are present. | Matches the requested sewing UX and existing model. | Prompt, lexicon findings | AC-011, AC-012 |
| FR-012 | Functional | Must | When craft type is knitting, the project details section shall collect knitting project type, project subtype, yarn weight, needle size, gauge stitches/rows/measurement/unit and finished size, and construct `KnittingProjectDetails` only when applicable detail values are present. | Matches the requested knitting UX and existing model. | Prompt, lexicon findings | AC-011, AC-012 |
| FR-013 | Functional | Must | When craft type is crochet, the project details section shall collect crochet project type, project subtype, yarn weight, hook size, gauge stitches/rows/measurement/unit and finished size, and construct `CrochetProjectDetails` only when applicable detail values are present. | Completes all known craft variants. | Q4, lexicon findings | AC-011, AC-012 |
| FR-014 | Functional | Must | When craft type is quilting, the project details section shall collect quilting project type, project subtype, size, piecing technique and quilting method, and construct `QuiltingProjectDetails` only when applicable detail values are present. | Completes all known craft variants. | Q4, lexicon findings | AC-011, AC-012 |
| FR-015 | Functional | Must | When craft type has no implemented detail schema, such as embroidery, the project composer shall allow a common-only project post and shall not require detail fields. | Existing models and AppView support common-only projects. | Q4, previous slice | AC-013 |
| FR-016 | Functional | Must | On submit, the project composer shall generate rich-text facets from body text using the existing facet generator, convert image state to create images, build a typed `Project` payload, and call `CreatePost.create` with `project` and no `reply`. | Reuses existing data plumbing and keeps writes through AppView. | Current composer/create discovery | AC-012, AC-015 |
| FR-017 | Functional | Must | The project composer shall handle create loading, success, error, image notices and missing-alt confirmation consistently with the regular composer. | Provides coherent feedback and preserves accessibility expectations. | Current composer discovery | AC-015, AC-016 |
| FR-018 | Functional | Must | The project composer shall detect unsaved project drafts, including text, images and form-field changes, and ask for discard confirmation before closing when changes would be lost. | Prevents accidental data loss. | Current composer discovery | AC-017 |
| FR-019 | Functional | Should | Materials should be collected as an optional multi-value free-text field with visible selected chips/items and saved as `ProjectCommon.materials`, omitting the field when empty. | Materials are free-form in the lexicon and prominent in the requested UX. | Prompt example, lexicon findings | AC-018 |
| FR-020 | Functional | Should | Pattern input should be grouped behind an “Add pattern” action or compact expandable area and save only non-empty `ProjectPattern` fields. | Pattern fields are optional and should not dominate the primary compose flow. | Prompt example, lexicon findings | AC-019 |
| FR-021 | Functional | Must | The project composer shall enforce UI-safe validation before submit: body text and craft type are required, existing/practical text length limits are enforced, gauge numeric fields are positive whole numbers when present, and invalid fields show field-level errors. | Gives users useful feedback without turning DTO constructors into strict lexicon validators. | Q5 | AC-023 |
| RULE-001 | Business rule | Must | Project posts created by this UI shall always be top-level posts and shall never include `reply`. | AppView and existing Flutter create plumbing disallow project replies. | Previous slice, Q3 | AC-012, AC-015 |
| RULE-002 | Business rule | Must | DTO/project model fields shall remain string-backed; UI option catalogs shall not require changing project models to enums. | Preserves open lexicon known-values and prior data-layer requirements. | Q2 | AC-009, AC-012 |
| RULE-003 | Business rule | Must | Lexicon known-values are open suggestions; the UI may constrain selectable known options for MVP fields but shall not make the data layer reject future/open token strings. | atproto lexicon evolution requires open compatibility. | Lexicon findings, Q2 | AC-008, AC-013 |
| RULE-004 | Business rule | Must | Empty optional project fields shall be omitted or left null in the created `Project` rather than serialized as meaningless empty strings or empty detail objects. | Keeps AppView payloads clean and compatible with previous `toCreateMap` behavior. | Q5, previous slice | AC-012, AC-018, AC-019 |
| NFR-001 | Non-functional | Must | New input components shall preserve Material semantics/accessibility for labels, focus, enabled/disabled state, validation errors, keyboard navigation and tap targets wherever possible. | The prompt specifically asks to use Material input styles for accessibility benefits. | Prompt | AC-003, AC-005, AC-020 |
| NFR-002 | Non-functional | Must | New UI shall use existing dependencies, theme extensions, localization patterns and code generation tools without adding packages. | Keeps the slice constrained and consistent with the app. | Discovery | AC-021 |
| NFR-003 | Non-functional | Must | New user-visible copy shall use app localization resources and follow Craftsky voice conventions. | Maintains localization/test patterns and product tone. | Design docs, discovery | AC-022 |
| NFR-004 | Non-functional | Should | The implementation should share or extract composer media/text behavior only where it reduces duplication without destabilizing the existing regular composer. | Reduces maintenance risk while avoiding risky refactors. | Q3, current composer discovery | AC-014, AC-016 |
| NFR-005 | Non-functional | Must | The UI shall remain compatible with Flutter analysis and tests, including generated files if new Riverpod routes/providers or mappable classes are introduced. | Required for maintainable Flutter code. | Discovery | AC-021 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-006, FR-008 | Given an authenticated user taps a top-level “New post” action in feed or own-profile Posts tab, when the full-screen composer choice surface is shown and the user chooses project post, then a full-screen project composer opens. |
| AC-002 | BR-002, FR-006, FR-007 | Given the same full-screen composer choice surface, when the user chooses regular post, then the existing regular post composer opens and can create a non-project post as before. |
| AC-003 | FR-001, FR-002, NFR-001 | Given the reusable Craftsky text, multiline, dropdown, multi-select and radio fields are rendered, when inspected in widget tests, then they expose labels, values, helper/error text, enabled state and use Material focus/semantics/tap behavior appropriate to the control. |
| AC-004 | FR-001, FR-003 | Given a Craftsky text or multiline field is used inside `FormBuilder`, when the user edits, saves, validates and resets the form, then the field value and error state flow through the named FormBuilder field. |
| AC-005 | FR-002, NFR-001 | Given Craftsky dropdown, multi-select and radio fields are used inside `FormBuilder`, when the user changes selection, validates and resets the form, then selected string values or value lists are reflected by FormBuilder and validation errors are visible/accessibility-exposed. |
| AC-006 | FR-003, FR-009 | Given the project composer form contains required and optional fields, when the form is saved, then body/project form values can be extracted by field name and converted into a create request without reading widget internals. |
| AC-007 | BR-001, FR-008, FR-009 | Given the project composer opens, then the primary visible flow includes add photos, body text, project title, craft type, status, pattern, materials, more-details disclosure and a submit action consistent with the requested UX. |
| AC-008 | FR-004, RULE-003 | Given option catalogs are used by composer controls, then options expose user-facing labels and underlying token strings for known craft/status/detail values, and catalog code is centralized enough for tests to verify representative token values. |
| AC-009 | FR-004, FR-005, RULE-002 | Given a user selects known token options, when the project payload is built, then the resulting `Project`/`ProjectDetails` fields contain the expected AppView string token values and no DTO enum conversion is required. |
| AC-010 | BR-003, FR-010 | Given the user expands and collapses “More project details”, then craft-specific fields are hidden or shown without losing existing primary field values or entered detail values for the active craft. |
| AC-011 | BR-003, FR-011, FR-012, FR-013, FR-014 | Given the user selects sewing, knitting, crochet or quilting, then the more-details section displays fields appropriate to that craft and not fields belonging only to unrelated crafts. |
| AC-012 | BR-001, FR-005, FR-009, FR-011, FR-012, FR-013, FR-014, FR-016, RULE-001, RULE-004 | Given valid project composer input, when the user submits, then `CreatePost.create` is called with non-empty body text, `reply == null`, images/facets as applicable, and a typed `Project` containing common fields and the matching detail variant only when detail values exist. |
| AC-013 | FR-015, RULE-003 | Given the user selects a craft with no implemented detail schema such as embroidery, when they submit valid common fields, then the create call includes `project.common.craftType` and no required `details` payload. |
| AC-014 | BR-002, FR-007, NFR-004 | Given existing regular composer widget tests run after this change, then regular text, photo, facet, reply and discard behavior still passes without requiring project fields. |
| AC-015 | FR-016, FR-017, RULE-001 | Given project submission is in progress, succeeds or fails, then submit loading state, successful close/snackbar, error snackbar and provider reset behavior are consistent with the regular composer, and project create never sends a reply. |
| AC-016 | FR-017, NFR-004 | Given project images include missing alt text or image provider notices, when the user submits or selects images, then the same missing-alt confirmation and image notice behavior as the regular composer is available. |
| AC-017 | FR-018 | Given the user has changed body text, selected images or changed project form fields, when they attempt to close the project composer, then a discard confirmation appears; given there are no changes or create is loading, close behavior matches the existing composer rules. |
| AC-018 | FR-019, RULE-004 | Given the user adds materials, when the project payload is built, then materials are serialized as a non-empty list of strings; given no materials are added, then `materials` is omitted/null. |
| AC-019 | FR-020, RULE-004 | Given the user adds pattern data, when the project payload is built, then only non-empty pattern fields are included; given all pattern fields are empty, then `pattern` is omitted/null. |
| AC-020 | NFR-001 | Given controls are disabled during create loading, then text inputs, selection controls, image actions and submit affordances cannot be interacted with and expose disabled visual/semantic state. |
| AC-021 | NFR-002, NFR-005 | Given code generation has run if required, when `flutter analyze` and relevant/full Flutter tests run, then they pass without dependency changes. |
| AC-022 | NFR-003 | Given new composer/field strings are user-visible, then they come from app localization resources, use sentence case and avoid emoji in app chrome. |
| AC-023 | FR-021 | Given the project composer has missing required body text or craft type, over-limit text, partial gauge input, or non-positive/non-integer gauge numbers, when the user attempts to submit or validation runs, then submit is blocked and field-level validation errors identify what to fix. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | User opens the full-screen entry picker and dismisses it | No composer opens and no draft/provider state is changed. | FR-006 |
| EC-002 | User switches craft type after entering detail fields | The composer prevents stale detail fields from being submitted for the newly selected craft; preserving or clearing per-craft drafts may be implementation-defined but the submitted payload must match the active craft. | FR-010, FR-011, FR-012, FR-013, FR-014 |
| EC-003 | Detail section is collapsed while detail fields contain values | Entered active-craft detail values remain part of the form and may be submitted unless the user clears them. | FR-010 |
| EC-004 | Gauge rows are omitted but stitches/measurement/unit are present | Gauge can be submitted without rows. | FR-012, FR-013 |
| EC-005 | Partial gauge is entered | UI-safe validation shows a field-level error and blocks submit until required gauge parts are completed or all gauge fields are cleared. | Q5, FR-012, FR-013 |
| EC-006 | Numeric gauge fields contain zero, negative or non-integer values | UI-safe validation blocks submit and asks for positive whole numbers. | Q5, FR-012, FR-013 |
| EC-007 | Optional text fields contain only whitespace | Values are treated as empty and omitted/null in the payload. | RULE-004 |
| EC-008 | Image upload is still processing | Submit remains disabled until images can be submitted, consistent with the existing composer. | FR-017 |
| EC-009 | AppView returns an error from create | Composer remains open, shows an error message and allows the user to retry after provider reset. | FR-017 |
| EC-010 | User creates a project with body text and craft type only | Submit succeeds with a common-only project if validation passes. | FR-015, RULE-004 |
| EC-011 | User chooses regular post after the new entry picker is added | No project defaults or form validation affect regular composer behavior. | BR-002, FR-007 |

## 15. Data / Persistence Impact

- New fields: None in persistent storage or AppView API. UI may introduce form state objects and option catalog classes.
- Changed fields: None. Existing `Project`/`ProjectCommon`/`ProjectDetails` model fields remain string-backed.
- Migration required: No.
- Backwards compatibility: Existing regular posts, project models, create API and project cache behavior remain compatible. Project composer emits payloads through existing `POST /v1/posts` create plumbing.

## 16. UI / API / CLI Impact

- UI:
  - Add reusable Craftsky FormBuilder field components.
  - Add post-type choice surface at top-level composer entry points.
  - Add project composer full-screen UI with primary fields and expandable details.
  - Add localized strings for new labels, helpers, errors and actions.
- API:
  - No AppView API changes.
  - Flutter continues to call existing `CreatePost.create` / `/v1/posts` with optional `Project`.
- CLI: None.
- Background jobs: None.

## 17. Security / Privacy / Permissions

- Authentication: Only authenticated composer entry points are in scope, matching existing feed/profile composer behavior.
- Authorization: Users can only create posts as the current session user through existing AppView session auth.
- Sensitive data: Project metadata is public-by-design because project posts are public PDS records; UI should not imply that fields are private.
- Abuse cases: No new moderation/reporting behavior. Project create failures should use existing AppView error handling and user messaging.

## 18. Observability

- Events: None required for MVP.
- Logs: Use existing provider/API logging patterns only if already present; do not add noisy UI logs.
- Metrics: None required.
- Alerts: None required.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | The slice touches a broad UI surface: reusable fields, composer entry, media reuse, validation and submit. | Regressions to regular composer or incomplete project flow. | Keep regular/project composers separate, add regression tests for regular composer and focused widget tests for project flow. |
| RISK-002 | Token catalogs can drift from lexicon known-values. | Users may submit unexpected or missing token values for known options. | Centralize option catalogs and test representative labels/tokens for each supported craft. |
| RISK-003 | Making DTOs enums would conflict with open lexicon known-values. | Future/open token strings could become unreadable/unwritable. | Keep DTOs string-backed and map UI options to strings only at form boundaries. |
| RISK-004 | All craft detail variants may make the composer too dense. | Users may feel overwhelmed. | Keep details behind a “More project details” disclosure and keep primary fields usable without details. |
| RISK-005 | Shared composer extraction could destabilize existing post/photo/facet behavior. | Existing regular post creation could break. | Extract only low-risk shared pieces; rely on existing regular composer tests and add regression coverage. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Existing `CreatePost.create` project plumbing is the correct submit boundary for project composer UI. | Requirements would need API/repository updates beyond UI. |
| ASM-002 | The MVP can hand-maintain UI option catalogs from current lexicon known-values. | A generated token pipeline may become a separate future feature. |
| ASM-003 | Body text remains required for project posts, matching current composer behavior and feed post lexicon requirements. | Requirements would need to redefine submit enabling for title/photo-only projects. |
| ASM-004 | Materials are free-form user-entered strings rather than a closed known-value catalog. | The materials field might need to become a searchable picker or catalog-backed multi-select later. |
| ASM-005 | The first UI slice does not need to render the created project differently in feed/profile after submission. | A project may initially appear as an ordinary post card until a later rendering slice. |

## 21. Open Questions

- None.

## 22. Review Status

Status: Reviewed  
Risk level: Medium  
Review recommended: Yes  
Reviewer: User annotation review  
Date: 2026-06-11  
Notes: User annotations addressed. Generated token catalogs and hashtag-to-project-tag merging are deferred. The entry picker should use the same full-screen composer/task presentation pattern as the existing post composer. Medium risk remains because this is user-visible UI touching the composer, validation, image/facet reuse and create submission.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-11-flutter-project-composer-ui/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: `BR-001`, `BR-002`, `BR-003`
  - Functional: `FR-001` through `FR-018`, `FR-021`
  - Rules: `RULE-001` through `RULE-004`
  - Non-functional: `NFR-001`, `NFR-002`, `NFR-003`, `NFR-005`
- Suggested test levels:
  - Widget tests for field components and FormBuilder integration.
  - Widget tests for entry picker regular/project branches.
  - Widget tests for project composer craft switching, validation, discard and submit payloads.
  - Provider/repository fake integration tests for `CreatePost.create` arguments from project composer.
  - Regression tests for existing regular composer facets/photos/replies/discard behavior.
  - Static checks for localization, no dependency changes and analysis.
- Blocking open questions: None.
