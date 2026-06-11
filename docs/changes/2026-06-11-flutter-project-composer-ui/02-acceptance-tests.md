# Acceptance Test Specification: Flutter Project Composer UI

## 1. Test Strategy

Design test-first coverage for a Flutter-only UI slice that adds reusable Craftsky FormBuilder fields, token option catalogs, a responsive post-type chooser, and a full-screen project composer. Prefer widget tests for user-visible flows, unit tests for field adapters, token catalogs, validation, payload building, and draft detection, and regression tests around the existing regular composer. No AppView, lexicon, migration, dependency or direct-PDS tests are in scope.

Risk level carried forward from requirements: **Medium**. Review is recommended before implementation because this slice touches composer navigation, media/facet reuse, FormBuilder state, validation and create payloads. The user may explicitly skip review, but document review is the recommended next workflow stage.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-007, AC-012 | AT-001, AT-003, AT-004, UT-007, IT-001, IT-004 | Acceptance / Unit / Integration | Yes |
| BR-002 | AC-002, AC-014 | AT-002, AT-010, IT-003, REG-001, REG-003 | Acceptance / Integration / Regression | Yes |
| BR-003 | AC-010, AC-011 | AT-005, UT-008, UT-009, UT-010, UT-011 | Acceptance / Unit | Yes |
| FR-001 | AC-003, AC-004 | UT-001, UT-002 | Unit / Widget | Yes |
| FR-002 | AC-003, AC-005 | UT-003, UT-004, UT-005 | Unit / Widget | Yes |
| FR-003 | AC-004, AC-006 | UT-001, UT-003, UT-004, UT-005, AT-004 | Unit / Acceptance | Yes |
| FR-004 | AC-008, AC-009 | UT-006, UT-007, REG-002 | Unit / Regression | Yes |
| FR-005 | AC-009, AC-012 | AT-004, AT-005, UT-007, UT-008, UT-009, UT-010, UT-011 | Acceptance / Unit | Yes |
| FR-006 | AC-001, AC-002 | AT-001, AT-002, IT-001, IT-003, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-007 | AC-002, AC-014 | AT-002, IT-003, REG-001 | Acceptance / Integration / Regression | Yes |
| FR-008 | AC-001, AC-007 | AT-001, AT-003, IT-001 | Acceptance / Integration | Yes |
| FR-009 | AC-006, AC-007, AC-012, AC-025 | AT-003, AT-004, AT-007, UT-007 | Acceptance / Unit | Yes |
| FR-010 | AC-010 | AT-005, UT-014 | Acceptance / Unit | Yes |
| FR-011 | AC-011, AC-012 | AT-005, UT-008 | Acceptance / Unit | Yes |
| FR-012 | AC-011, AC-012 | AT-005, AT-006, UT-009 | Acceptance / Unit | Yes |
| FR-013 | AC-011, AC-012 | AT-005, AT-006, UT-010 | Acceptance / Unit | Yes |
| FR-014 | AC-011, AC-012 | AT-005, UT-011 | Acceptance / Unit | Yes |
| FR-015 | AC-013 | AT-004, UT-012 | Acceptance / Unit | Yes |
| FR-016 | AC-012, AC-015 | AT-004, UT-015, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-015, AC-016 | AT-008, IT-004, IT-005 | Acceptance / Integration | Yes |
| FR-018 | AC-017 | AT-009, UT-014 | Acceptance / Unit | Yes |
| FR-019 | AC-018, AC-026 | AT-007, UT-004, UT-007 | Acceptance / Unit | Yes |
| FR-020 | AC-019 | AT-007, UT-007 | Acceptance / Unit | Yes |
| FR-021 | AC-023 | AT-006, UT-009, UT-010 | Acceptance / Unit | Yes |
| FR-022 | AC-024 | AT-005, UT-013 | Acceptance / Unit | Yes |
| FR-023 | AC-025, AC-026 | AT-007, UT-004, UT-007 | Acceptance / Unit | Yes |
| FR-024 | AC-027 | AT-003, UT-006, UT-007 | Acceptance / Unit | Yes |
| FR-025 | AC-028 | AT-010, IT-001, IT-002, IT-003 | Acceptance / Integration | Yes |
| RULE-001 | AC-012, AC-015 | AT-004, UT-015, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-009, AC-012 | UT-006, UT-007, REG-002 | Unit / Regression | Yes |
| RULE-003 | AC-008, AC-013 | UT-006, UT-012, REG-002 | Unit / Regression | Yes |
| RULE-004 | AC-012, AC-018, AC-019 | AT-004, AT-007, UT-007, UT-012 | Acceptance / Unit | Yes |
| RULE-005 | AC-012, AC-023 | AT-004, AT-006 | Acceptance | Yes |
| NFR-001 | AC-003, AC-005, AC-020 | UT-001, UT-003, UT-004, UT-005, AT-008, MAN-001 | Unit / Acceptance / Manual | Mixed |
| NFR-002 | AC-021 | IT-006, REG-005 | Integration / Regression | Yes |
| NFR-003 | AC-022 | UT-016, MAN-003 | Unit / Manual | Mixed |
| NFR-004 | AC-014, AC-016 | REG-001, AT-008 | Regression / Acceptance | Yes |
| NFR-005 | AC-021 | IT-006, REG-005 | Integration / Regression | Yes |

### Acceptance Criteria Coverage Index

| Acceptance Criteria | Primary Test IDs |
|---|---|
| AC-001 | AT-001, IT-001 |
| AC-002 | AT-002, IT-003 |
| AC-003 | UT-001, UT-002, UT-003, UT-004, UT-005 |
| AC-004 | UT-001, UT-002 |
| AC-005 | UT-003, UT-004, UT-005 |
| AC-006 | AT-004, UT-007 |
| AC-007 | AT-003 |
| AC-008 | UT-006 |
| AC-009 | UT-006, UT-007 |
| AC-010 | AT-005 |
| AC-011 | AT-005, UT-008, UT-009, UT-010, UT-011 |
| AC-012 | AT-004, AT-005, UT-015, IT-004 |
| AC-013 | AT-004, UT-012 |
| AC-014 | REG-001, IT-003 |
| AC-015 | AT-008, IT-004 |
| AC-016 | AT-008, IT-005 |
| AC-017 | AT-009, UT-014 |
| AC-018 | AT-007, UT-007 |
| AC-019 | AT-007, UT-007 |
| AC-020 | AT-008, UT-001, UT-003, UT-004, UT-005 |
| AC-021 | IT-006, REG-005 |
| AC-022 | UT-016, MAN-003 |
| AC-023 | AT-006, UT-009, UT-010 |
| AC-024 | AT-005, UT-013 |
| AC-025 | AT-007, UT-007 |
| AC-026 | AT-007, UT-004 |
| AC-027 | AT-003, UT-006, UT-007 |
| AC-028 | AT-010, IT-001, IT-002, IT-003 |

## 3. Acceptance Scenarios

### AT-001: Compact chooser opens the project composer
Requirement IDs: BR-001, FR-006, FR-008, FR-025  
Acceptance Criteria: AC-001, AC-028  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/widgets/post_type_chooser_test.dart`

```gherkin
Feature: Choosing a post type
  Scenario: A compact top-level New post action opens the project composer
    Given an authenticated user is on a compact-width feed screen
    When they tap "New post"
    Then a Craftsky bottom-sheet chooser appears
    And it contains "Regular post" and "Project post" with brief descriptions
    When they choose "Project post"
    Then the chooser is dismissed
    And a full-screen project composer opens on the root navigator
```

### AT-002: Regular post branch preserves the existing composer
Requirement IDs: BR-002, FR-006, FR-007, FR-025  
Acceptance Criteria: AC-002, AC-014, AC-028  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/widgets/post_type_chooser_test.dart`

```gherkin
Feature: Choosing a post type
  Scenario: A top-level user chooses a regular post
    Given an authenticated user opens the responsive post-type chooser
    When they choose "Regular post"
    Then the chooser is dismissed
    And the existing regular post composer opens
    And no project-only field or validation is shown in the regular composer
    When they create a non-project post
    Then the create call contains no project payload
    And the chooser flow resolves with the created Post
```

### AT-003: Project composer primary form renders with default status
Requirement IDs: BR-001, FR-008, FR-009, FR-024  
Acceptance Criteria: AC-007, AC-027  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/projects/widgets/project_composer_sheet_test.dart`

```gherkin
Feature: Project composer
  Scenario: The project composer opens with primary project fields
    Given a user opens the project composer
    Then the visible primary flow includes photo controls, body text, project title, craft type, status, materials, colours, design tags, an "Add pattern" affordance, a "More project details" disclosure and a submit action
    And status is initially "Finished"
    And the selected status value is backed by "social.craftsky.feed.defs#finished"
```

### AT-004: Valid common-only project submits through CreatePost
Requirement IDs: BR-001, FR-003, FR-005, FR-009, FR-015, FR-016, RULE-001, RULE-004, RULE-005  
Acceptance Criteria: AC-006, AC-012, AC-013  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/projects/widgets/project_composer_submit_test.dart`

```gherkin
Feature: Project composer submission
  Scenario: A user creates a common-only embroidery project
    Given the project composer has body text "Finished my hoop", one uploaded photo, craft type "Embroidery" and status "Finished"
    And no optional common, pattern or detail fields contain values
    When the user taps "Post"
    Then CreatePost.create is called with the body text
    And reply is null
    And the image is included
    And facets are generated from the body when applicable
    And project.common.craftType contains the embroidery token string
    And project.details is omitted
    And empty optional fields are omitted or null
```

### AT-005: Craft-specific detail sections match the active craft
Requirement IDs: BR-003, FR-010, FR-011, FR-012, FR-013, FR-014, FR-022  
Acceptance Criteria: AC-010, AC-011, AC-012, AC-024  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/projects/widgets/project_composer_details_test.dart`

```gherkin
Feature: Project details
  Scenario Outline: The more-details section shows only fields for the active craft
    Given the project composer is open
    When the user selects craft type "<craft>"
    And expands "More project details"
    Then fields for "<included fields>" are visible
    And fields belonging only to unrelated crafts are not visible
    When the user selects a project type
    Then the subtype options are filtered to that project type
    When the user changes the project type to one that does not support the selected subtype
    Then the invalid subtype is cleared before submit

    Examples:
      | craft    | included fields |
      | Sewing   | sewing project type, subtype, size made, fit notes |
      | Knitting | knitting project type, subtype, yarn weight, needle size, gauge, finished size |
      | Crochet  | crochet project type, subtype, yarn weight, hook size, gauge, finished size |
      | Quilting | quilting project type, subtype, size, piecing technique, quilting method |
```

### AT-006: UI-safe validation blocks invalid project submissions
Requirement IDs: FR-012, FR-013, FR-021, RULE-005  
Acceptance Criteria: AC-023  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/projects/widgets/project_composer_validation_test.dart`

```gherkin
Feature: Project composer validation
  Scenario Outline: Invalid input blocks submit and explains what to fix
    Given the project composer is open
    And the user has entered "<input>"
    When the user attempts to submit
    Then submit is blocked
    And a field-level or form-level validation message identifies "<problem>"

    Examples:
      | input | problem |
      | no body text | body text is required |
      | no craft type | craft type is required |
      | no photo | project posts need at least one photo |
      | text beyond the existing post limit | body text length limit |
      | gauge stitches without measurement and unit | incomplete gauge |
      | gauge measurement without unit | missing gauge unit |
      | zero, negative or decimal gauge numbers | positive whole numbers |
```

### AT-007: Optional metadata, pattern and count limits behave correctly
Requirement IDs: FR-019, FR-020, FR-023, RULE-004  
Acceptance Criteria: AC-018, AC-019, AC-025, AC-026  
Priority: Must / Should  
Level: Acceptance  
Automation Target: `app/test/projects/widgets/project_composer_metadata_test.dart`

```gherkin
Feature: Project metadata
  Scenario: Optional project metadata is visible, limited and serialized only when present
    Given the project composer is open
    Then pattern fields are hidden behind "Add pattern"
    When the user adds materials, colours and design tags
    Then selected items are visible as chips or selected rows
    And materials accept up to 20 values
    And colours accept up to 10 known values
    And design tags accept up to 10 known values
    When the user attempts to exceed a maximum
    Then the UI prevents the extra value and explains the limit
    And existing selected values can still be removed
    When the user submits valid input
    Then non-empty metadata maps to ProjectCommon strings
    And unselected metadata and empty pattern fields are omitted or null
```

### AT-008: Project composer matches regular composer feedback states
Requirement IDs: FR-017, NFR-001, NFR-004  
Acceptance Criteria: AC-015, AC-016, AC-020  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/projects/widgets/project_composer_feedback_test.dart`

```gherkin
Feature: Project composer feedback
  Scenario: Loading, success, error and image warnings mirror the regular composer
    Given the project composer has valid input with an uploaded image missing alt text
    When the user submits
    Then the missing-alt confirmation is shown instead of hard-blocking submit
    When the user confirms
    Then create loading disables text inputs, selection controls, image actions and submit affordances
    And disabled state is visible and exposed semantically where practical
    When create succeeds
    Then the composer closes, shows the success message and resets the create provider
    When create fails in a retry case
    Then the composer remains open, shows the error message and allows retry after provider reset
```

### AT-009: Discard confirmation protects project drafts
Requirement IDs: FR-018  
Acceptance Criteria: AC-017  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/projects/widgets/project_composer_discard_test.dart`

```gherkin
Feature: Project draft protection
  Scenario Outline: Closing the composer with unsaved project edits asks for confirmation
    Given the project composer is open
    And the user changes "<draft source>"
    When they close the composer
    Then a discard confirmation appears
    When they choose to keep editing
    Then the composer remains open with the entered values
    When they choose to discard
    Then the composer closes without creating a post

    Examples:
      | draft source |
      | body text |
      | selected images |
      | project metadata form fields |
```

### AT-010: Reply actions bypass the chooser and chooser returns Post results
Requirement IDs: BR-002, FR-025  
Acceptance Criteria: AC-028  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/widgets/post_type_chooser_result_test.dart`, `app/test/profile/widgets/profile_posts_tab_test.dart`

```gherkin
Feature: Composer navigation results
  Scenario: Top-level composer returns created posts while replies stay direct
    Given a caller opens the top-level post-type chooser
    When the selected composer creates a post
    Then the chooser flow resolves with that Post
    When the chooser or selected composer is dismissed without posting
    Then the chooser flow resolves null
    Given a user taps a reply action
    Then no post-type chooser appears
    And the existing regular reply composer opens directly
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-003, NFR-001 | AC-003, AC-004, AC-020 | Craftsky FormBuilder text field forwards value, error, save, validate, reset, enabled and semantic label state. | FormBuilder with named text field, validator and initial value. | FormBuilder value changes by field name; error/helper visible; reset restores initial value; disabled field is not editable. | `app/test/theme/craftsky_form_builder_text_field_test.dart` |
| UT-002 | FR-001, NFR-001 | AC-003, AC-004 | Multiline text field supports controller/focus or initial value, keyboard/action, line counts and change/submit hooks without breaking BrandTextField styling. | Multiline field with controller, focus node, min/max lines and callbacks. | Focus and controller are honoured; callbacks fire; line configuration and labels/helper/error render. | `app/test/theme/craftsky_form_builder_text_field_test.dart` |
| UT-003 | FR-002, FR-003, NFR-001 | AC-003, AC-005, AC-020 | Dropdown field integrates with FormBuilder and Material selection semantics. | Options with labels/token strings, initial value, validator. | Selected value is saved by name, reset works, errors visible, disabled state prevents changes. | `app/test/theme/craftsky_form_builder_dropdown_test.dart` |
| UT-004 | FR-002, FR-019, FR-023, NFR-001 | AC-003, AC-005, AC-018, AC-025, AC-026 | Multi-select field handles string lists, chips/selected rows, max counts and removals. | Free-text materials max 20; known colours/design tags max 10. | FormBuilder saves selected string lists; max prevents extras with helper/error; removals remain possible. | `app/test/theme/craftsky_form_builder_multi_select_test.dart` |
| UT-005 | FR-002, NFR-001 | AC-003, AC-005, AC-020 | Radio field supports option labels, selected token value, validation, reset and disabled state. | Status or craft radio group. | Selected string flows through FormBuilder; validation/error state is visible; reset/disabled behaviour works. | `app/test/theme/craftsky_form_builder_radio_test.dart` |
| UT-006 | FR-004, RULE-002, RULE-003, FR-024 | AC-008, AC-009, AC-027 | Project option catalogs expose labels, token values and representative filtering/grouping metadata without changing DTOs to enums. | Craft/status/pattern difficulty/project type/subtype/yarn/needle/hook/gauge/quilting/colour/design tag catalogs. | Representative labels map to expected known token strings; status Finished token is present; option types are UI-only and models remain string-backed. | `app/test/projects/options/project_option_catalogs_test.dart` |
| UT-007 | FR-005, FR-009, FR-019, FR-020, FR-023, FR-024, RULE-004 | AC-006, AC-009, AC-012, AC-018, AC-019, AC-025, AC-027 | Project payload builder maps FormBuilder values into `ProjectCommon`, default status, optional pattern and metadata while trimming/omitting empty values. | Valid common form data, whitespace optional fields, selected materials/colours/design tags, pattern values. | `Project` uses string tokens, default or selected status, non-empty lists only, non-empty pattern fields only, no meaningless empty strings. | `app/test/projects/project_composer_payload_test.dart` |
| UT-008 | FR-011, BR-003 | AC-011, AC-012 | Sewing detail builder creates `SewingProjectDetails` only when applicable values exist. | Sewing type/subtype/size made/fit notes and empty sewing values. | Detail variant is sewing with expected string fields; empty detail object is omitted. | `app/test/projects/project_composer_payload_test.dart` |
| UT-009 | FR-012, FR-021, BR-003 | AC-011, AC-012, AC-023 | Knitting detail and gauge validation/building accept complete positive whole-number gauge and reject partial/invalid gauge. | Knitting type/subtype/yarn weight/needle size/gauge/finished size. | Valid detail uses `KnittingProjectDetails`; rows are optional; partial, missing-unit, zero, negative or decimal gauge values fail validation. | `app/test/projects/project_composer_payload_test.dart` |
| UT-010 | FR-013, FR-021, BR-003 | AC-011, AC-012, AC-023 | Crochet detail and gauge validation/building mirror knitting with hook size. | Crochet type/subtype/yarn weight/hook size/gauge/finished size. | Valid detail uses `CrochetProjectDetails`; invalid gauge cases fail validation. | `app/test/projects/project_composer_payload_test.dart` |
| UT-011 | FR-014, BR-003 | AC-011, AC-012 | Quilting detail builder creates the quilting variant and omits empty detail values. | Quilting type/subtype/size/piecing technique/quilting method. | Detail variant is quilting with expected string fields or omitted when no values exist. | `app/test/projects/project_composer_payload_test.dart` |
| UT-012 | FR-015, RULE-003, RULE-004 | AC-013 | Common-only craft payloads submit no details for crafts without implemented detail schema. | Embroidery or future known craft token with required common fields. | Payload includes `project.common.craftType` and no `details`; data model can still carry open token strings. | `app/test/projects/project_composer_payload_test.dart` |
| UT-013 | FR-022 | AC-024 | Project subtype filtering depends on selected project type and clears invalid values on type changes. | Type/subtype catalogs for sewing, knitting, crochet and quilting. | No type disables subtype; active type filters options; changing type clears invalid subtype before submit. | `app/test/projects/options/project_subtype_filter_test.dart` |
| UT-014 | FR-018, FR-010 | AC-010, AC-017 | Project draft detection includes body text, images, primary form fields, pattern fields and active detail values. | Empty composer state; changed body; uploaded image; changed FormBuilder values. | Unchanged state closes immediately; changed state requires discard; collapsed details retain active values. | `app/test/projects/project_composer_draft_state_test.dart` |
| UT-015 | FR-016, RULE-001 | AC-012, AC-015 | Submit adapter builds `CreatePost.create` arguments with generated facets, create images, project payload and `reply == null`. | Body with mention/link/tag, uploaded image state, valid project payload. | Facets generated by existing facet pipeline, image converted to create image, project included, reply never included. | `app/test/projects/project_composer_submit_adapter_test.dart` |
| UT-016 | NFR-003 | AC-022 | New user-visible strings are localised and follow copy conventions. | ARB entries for chooser, project composer, fields, helpers, errors and actions. | Strings are present in app localisation resources, use sentence case, British English where applicable, and no emoji in app chrome. | `app/test/l10n/project_composer_l10n_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-006, FR-008, FR-025 | AC-001, AC-028 | Feed top-level New post opens chooser and project branch returns the created post to caller. | Widget harness around `FeedPage` with fake repository and compact `MediaQuery`. | Tap New post, choose Project post, submit valid project. | Bottom sheet chooser appears; project composer opens; create succeeds; caller receives created `Post`. | `app/test/feed/pages/feed_page_composer_entry_test.dart` |
| IT-002 | FR-025 | AC-028 | Own-profile Posts tab top-level entry propagates composer result without affecting non-owner profiles. | `ProfilePostsTab` with fake repo, owner and non-owner modes. | Owner taps New post and creates/dismisses; non-owner renders tab. | Owner sees chooser and gets Post/null result; non-owner has no New post entry. | `app/test/profile/widgets/profile_posts_tab_project_composer_test.dart` |
| IT-003 | BR-002, FR-007, FR-025 | AC-002, AC-014, AC-028 | Regular branch and reply branch still use regular composer and no project payload. | Fake post repository captures `project` and `reply`; existing feed/profile post card harness. | Top-level choose Regular post; reply from a post card. | Regular top-level create has `project == null`; reply action bypasses chooser and sends reply through regular composer. | `app/test/feed/widgets/post_type_chooser_regular_branch_test.dart` |
| IT-004 | FR-016, FR-017, RULE-001 | AC-012, AC-015 | Project composer provider integration handles create loading, success, error and reset. | Fake repository returns success, then error in separate tests. | Submit valid project. | Loading disables inputs; success closes and shows success; error keeps composer open and shows error; provider reset allows retry; reply null always. | `app/test/projects/widgets/project_composer_provider_test.dart` |
| IT-005 | FR-017, NFR-004 | AC-016 | Project composer uses existing image provider behaviours for missing alt text and notices. | Override `composerImagesProvider` with uploaded image missing alt and notice states. | Submit/select image scenarios. | Missing-alt confirmation appears; unsupported/limit/picker notices show same messages as regular composer. | `app/test/projects/widgets/project_composer_images_test.dart` |
| IT-006 | NFR-002, NFR-005 | AC-021 | Static and generated-code checks pass with existing dependencies. | After implementation, dependencies unchanged; build runner used only if generated Riverpod/routes/mappable code changes. | Run discovered commands. | `flutter analyze` and relevant/full Flutter tests pass; no dependency additions. | Commands: `cd app && flutter analyze`, `cd app && flutter test`, and if required `cd app && dart run build_runner build --delete-conflicting-outputs` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Existing regular composer text, photo, facet, reply, missing-alt and discard flows. | BR-002, FR-007, NFR-004 | Keep and extend `app/test/feed/widgets/post_composer_sheet_facets_test.dart`, `post_composer_sheet_discard_test.dart`, and image/provider tests so they pass after the chooser/project composer are added. |
| REG-002 | Project DTO/model fields remain string-backed and open-token compatible. | FR-004, RULE-002, RULE-003 | Keep `app/test/projects/models/project_test.dart` and `project_details_test.dart` passing, especially unknown details and constructors that do not enforce lexicon validation. |
| REG-003 | Own-profile composer entry remains present only for own profile; reply navigation from profile posts still opens a thread focused on the new comment. | BR-002 | Update existing profile posts tests to account for the chooser only on top-level own-profile New post; reply tests must not see chooser. |
| REG-004 | Existing responsive Craftsky context menu presentation remains bottom sheet on compact screens and anchored popup/menu on larger screens. | FR-006 | Add or retain focused tests around `showCraftskyContextMenu` behaviour while using it for the post-type chooser. |
| REG-005 | Dependency, localisation generation and code generation conventions remain stable. | NFR-002, NFR-005 | Verify `app/pubspec.yaml` has no new dependencies and generated files are updated only when codegen is required. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Authenticated user and fake repository harness | `ProviderScope`, `MessengerScope`, `MaterialApp` with `AppLocalizations`, fake `PostRepository`, current user context where required. | AT-001, AT-002, AT-004, IT-001, IT-002, IT-003, IT-004 |
| TD-002 | Uploaded project photo | `ComposerImageDraft(id: image-1, fileName: project.jpg, mimeType: image/jpeg, phase: ImageUploaded(...), altText: '')` and variant with alt text. | AT-004, AT-006, AT-008, IT-004, IT-005 |
| TD-003 | Facet-aware body text | `Hi @alice.craftsky.social see craftsky.social, #SockKAL` plus plain body text such as `Finished my hoop`. | AT-004, UT-015 |
| TD-004 | Common known tokens | Craft: knitting, crochet, sewing, quilting, embroidery; status: `social.craftsky.feed.defs#finished`; pattern difficulty examples; colours/design tags. | AT-003, AT-004, AT-007, UT-006, UT-007 |
| TD-005 | Craft detail values | Sewing dress/custom size/fit notes; knitting sweater/DK/4.0mm/gauge; crochet amigurumi/worsted/5.0mm/gauge; quilting throw/improv/machine quilted. | AT-005, UT-008, UT-009, UT-010, UT-011 |
| TD-006 | Invalid gauge values | Partial stitches-only, missing unit, zero, negative and decimal number strings. | AT-006, UT-009, UT-010 |
| TD-007 | Count-limit lists | 20 materials plus attempted 21st; 10 colours plus attempted 11th; 10 design tags plus attempted 11th. | AT-007, UT-004 |
| TD-008 | Repository outcomes | Fake repository success returning a `Post`; fake repository error throwing create failure; loading state controlled by delayed future. | AT-008, IT-004 |
| TD-009 | Localisation strings | New ARB keys for chooser labels/descriptions, project composer title/actions, field labels/helpers/errors, count-limit messages and discard/missing-alt reuse. | UT-016, MAN-003 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | NFR-001 | Accessibility smoke check for semantics, focus order and tap targets. | Run the implemented app or widget explorer with keyboard/screen-reader inspection on chooser, fields, chips, disclosure and disabled create state. | Labels, errors and disabled state are perceivable; focus order is logical; controls meet practical tap-target expectations. |
| MAN-002 | FR-006, FR-008 | Responsive visual placement and shell-covering route check. | Exercise compact and larger widths; open chooser and project composer from feed and own-profile Posts tab. | Compact uses bottom sheet; larger uses anchored popup/menu; project composer covers shell navigation like existing composer. |
| MAN-003 | NFR-003 | Copy review for Craftsky tone. | Review rendered strings in app chrome. | User-visible copy is sentence case, British English where applicable, and contains no emoji. |
| MAN-004 | FR-010, RISK-004 | Composer density and collapsed details usability review. | Create a minimal project, then expand details for each craft. | Primary flow remains usable when details are collapsed; expanded sections are understandable and not visually overwhelming. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Exact screen-reader output and platform-specific keyboard traversal cannot be fully guaranteed by widget tests. | NFR-001 | Flutter widget tests can verify semantics nodes, enabled state and labels, but not every assistive technology interaction. | Cover with MAN-001 before release. |
| GAP-002 | Visual polish of the responsive chooser and dense project form needs human judgement. | FR-006, FR-010, FR-008 | Automated tests can assert presentation type and key widgets, not product-level visual balance. | Cover with MAN-002 and MAN-004 during document/implementation review. |
| GAP-003 | Token catalog completeness can drift from lexicon after this slice. | FR-004, RULE-003 | Requirements explicitly defer generated token catalogs. Tests can verify representative current tokens, not future lexicon updates. | Keep UT-006 representative coverage and open a future task for generated catalog drift checks if needed. |
| GAP-004 | No backend/API contract test is added for project create payloads in this stage. | FR-016, RULE-001 | Requirements state no AppView/API changes; Flutter should use existing create plumbing. | Rely on fake repository integration for arguments now; add AppView contract tests only if API payload shape changes later. |

## 10. Out Of Scope

- Feed card rendering, Projects tab UI, project detail pages and edit-project flows.
- AppView route/API/database/firehose/lexicon tests.
- Direct PDS writes, PDS token handling or Token Mediating Backend changes.
- Project replies, quote posts, drafts/autosave and scheduled posting.
- Generated lexicon-to-UI token catalog pipeline.
- Project hashtag-to-`ProjectCommon.tags` merging.
- Advanced search/filter UI for project metadata.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-06-11-flutter-project-composer-ui/01-requirements.md`
- Test specification: `docs/changes/2026-06-11-flutter-project-composer-ui/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-06-11-flutter-project-composer-ui/`
- Risk-based review recommendation: Medium risk; document review recommended before implementation, but the user may explicitly skip.
- Recommended first failing test for implementation: `UT-006` for project option catalogs, especially the Finished status token and representative craft/detail tokens, because later composer fields and payload tests depend on stable option values.
- Suggested test order for implementation:
  1. `UT-006` option catalogs.
  2. `UT-001` through `UT-005` reusable FormBuilder field adapters.
  3. `UT-013`, `UT-007` through `UT-012`, and `UT-015` payload, subtype and validation helpers.
  4. `AT-001`, `AT-002`, `AT-010` chooser navigation and result forwarding.
  5. `AT-003` through `AT-009` project composer render, details, validation, metadata, feedback and discard.
  6. `IT-001` through `IT-005` page/provider/image integration.
  7. `REG-001` through `REG-005` existing composer/model/context-menu/profile protections.
  8. `IT-006` final analyze/test/codegen command pass.
- Commands discovered:
  - `cd app && flutter test`
  - `cd app && flutter analyze`
  - `cd app && dart run build_runner build --delete-conflicting-outputs` if generated Riverpod/routes/mappable files are added or changed.
- Blocking gaps: None. Known non-blocking gaps are listed in Section 9.
