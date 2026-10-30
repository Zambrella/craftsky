# Acceptance Test Specification: Basic Image Editing and Avatar Preview

## 1. Test Strategy

Design status: Revised to match maintainer approval, 2026-10-29. Risk: Medium; document review completed and scope approved. All requirements assumptions A1–A4 were explicitly confirmed by the maintainer before this stage. The requirements artifact now records that confirmation; no requirement IDs or scope are changed here.

The maintainer approved the plan with **no new integration tests**. Trust the package's core crop/rotate implementation. Automated coverage focuses on CraftSky's own unit/provider logic and widget behaviour: invocation, selected-image identity, cancellation, output handoff, draft mapping, ownership, and avatar save semantics. Use deterministic editor results for orchestration and a small real-package widget smoke check for our restricted controls/circular configuration. Do not build a package export/conformance suite.

Retain existing test suites and existing integration tests unchanged unless another task requires changes. No new device/browser harness, driver, real-codec export matrix, file-backed cross-layer journey, or loopback/live-write test is required for this feature. A few manual checks of the finished UI supplement unit/widget coverage; they are not a substitute test-infrastructure project.

Existing conventions and targets inspected:

- `app/test/feed/providers/composer_images_provider_test.dart`: `ProviderContainer.test`, picker overrides, completers, prepared image fixtures.
- `app/test/feed/widgets/composer_image_attachment_section_test.dart`: localized theme harness, summary/details routes, stable image keys.
- `app/test/profile/providers/profile_image_picker_provider_test.dart`: gallery/camera cases and intercepted blob authorization/upload boundaries.
- `app/test/drafts/data/file_local_post_draft_repository_test.dart`: temporary directories, restarted repositories, `DraftComposerHydrator`, standard/project variants.
- `app/test/observability/profile_image_error_reporting_test.dart` and `app/test/test_support/serialized_sentry_transport.dart`: original causes/stacks, expected-outcome suppression, serialized sink evidence.
- `app/integration_test/image_compression_test.dart`: real JPEG orientation/metadata, iOS HEIC, and large-image preparation. Host tests install a codec substitute in `flutter_test_config.dart`.
- `app/integration_test/critical_journeys_test.dart`: mobile-only loopback HTTP journeys. Its `dart:io` server prevents web execution; it does not cover OS photo pickers.

Paths marked **proposed** below do not exist yet and are design targets, not required architecture. No test files, dependencies, or application code are created in this stage.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-005 | AT-001, AT-002, AT-005 | Widget | Planned |
| FR-001 | AC-001 | AT-001, REG-001 | Widget / Regression | Planned |
| FR-002 | AC-002 | AT-002, MAN-001 | Widget / Manual | Planned + manual |
| FR-003 | AC-003 | AT-003, UT-002 | Widget / Unit | Planned |
| FR-004 | AC-004 | AT-004, UT-003, REG-003 | Widget / Unit | Planned |
| FR-005 | AC-005 | AT-005, MAN-001 | Widget / Manual | Planned + manual |
| FR-006 | AC-006 | AT-006, REG-003 | Widget / Regression | Planned |
| FR-007 | AC-007 | AT-007, UT-007, MAN-002 | Widget / Unit / Manual | Planned + manual |
| RULE-001 | AC-008 | AT-008, UT-001, REG-002 | Widget / Unit | Planned |
| RULE-002 | AC-009 | UT-004 | Unit / Regression | Planned |
| RULE-003 | AC-010 | AT-009, UT-005 | Widget / Unit | Planned |
| NFR-001 | AC-011 | MAN-001 | Manual | Manual; package platform support trusted |
| NFR-002 | AC-012 | AT-010, MAN-003 | Widget / Manual | Planned + manual |
| NFR-003 | AC-013 | UT-006, UT-008, REG-004 | Unit / Regression | Planned |

All 14 Must requirements and all 13 acceptance criteria have a planned verification path. Real-package device/browser export correctness is trusted rather than independently automated, as requested. No test is claimed to have been executed.

## 3. Acceptance Scenarios

### AT-001: Open the selected image directly from the composer

Requirement IDs: BR-001, FR-001
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance (widget)
Automation Target: `app/test/feed/widgets/composer_image_attachment_section_test.dart`, existing post composer widget suites, `app/test/projects/widgets/project_composer_images_test.dart`

```gherkin
Scenario Outline: Optional per-image editing from the summary
  Given a <composer> with three ready local photos and written content
  When I select Edit over the second photo in the composer summary
  Then that photo opens in the image editor without visiting photo details
  And the other photos and written content remain in the composer
  And adding those photos did not automatically open an editor
  Examples:
    | composer |
    | post     |
    | reply    |
    | quote    |
    | project  |
```

Parameterize gallery and camera acquisition where already supported. Use distinguishable valid image bytes and assert the editor input belongs to the selected stable image ID, not its current list index.

### AT-002: Restrict the real editor to basic crop and rotation

Requirement IDs: BR-001, FR-002
Acceptance Criteria: AC-002
Priority: Must
Level: Acceptance (widget smoke check; visual behaviour also MAN-001)
Automation Target: **proposed** `app/test/shared/image/basic_image_editor_test.dart`

```gherkin
Scenario: Crop and rotate a composer photo
  Given the actual pro_image_editor screen displays the asymmetric test image
  When I pan and zoom, choose a non-square crop, and rotate by one quarter-turn
  Then the preview reflects that composition
  And Done and Cancel are available
  And filters, adjustments, paint, text, stickers, AI, flip, and straightening tools are absent
```

Verify our visible tool restrictions and action wiring in the real-package widget. Use controlled editor output for application state assertions. Core package geometry, gestures and export are trusted; visually exercise crop/rotation in MAN-001.

### AT-003: Accept an edit without losing attachment metadata

Requirement IDs: FR-003
Acceptance Criteria: AC-003
Priority: Must
Level: Acceptance (widget/provider)
Automation Target: attachment section and post/project composer suites; `app/test/feed/providers/composer_images_provider_test.dart`

```gherkin
Scenario: Edit an image after reordering
  Given three photos have distinct alt text and I have reordered them
  When I edit the middle photo and confirm a valid crop and rotation
  Then its summary and photo-details previews show the new composition and dimensions
  And its identity, list position, and alt text remain unchanged
  And other photos, text, project fields, and scheduling choices remain unchanged
```

### AT-004: Discard or recover from an unsuccessful editor session

Requirement IDs: FR-004
Acceptance Criteria: AC-004
Priority: Must
Level: Acceptance (widget/provider)
Automation Target: **proposed** editor suite, composer provider suite, existing Edit Profile/onboarding suites

```gherkin
Scenario Outline: Preserve the previous accepted selection
  Given a composer photo or avatar has an accepted selection
  When a new editing session ends with <outcome>
  Then the previous accepted pixels and form content remain available
  And the editor's unaccepted result is not uploaded or persisted
  And <feedback>
  Examples:
    | outcome             | feedback                                  |
    | Cancel              | no failure message appears                |
    | Back                | no failure message appears                |
    | editor/export error | a recoverable message allows another edit |
    | preparation error   | a recoverable message allows another edit |
    | rejected output     | a recoverable message allows another edit |
```

Also test no prior avatar selection and cancellation of the OS picker. Count upload calls specifically for the new selection, allowing an earlier accepted selection to have been uploaded already.

### AT-005: Compose avatars against a live circle

Requirement IDs: BR-001, FR-005
Acceptance Criteria: AC-005
Priority: Must
Level: Acceptance (widget smoke check; visual behaviour also MAN-001)
Automation Target: **proposed** editor suite, `app/test/onboarding/onboarding_profile_step_test.dart`, `app/test/profile/widgets/edit_profile_banner_avatar_test.dart`

```gherkin
Scenario Outline: Avatar editing precedes upload
  Given I am in <flow> and choose a new photo
  Then the editor opens before any upload of that selection
  And the crop is fixed at 1:1 with a circular composition preview
  When I pan, zoom, and rotate
  Then the live circle updates with the image
  When I confirm
  Then the result is square with real image pixels in its corners
  And the displayed CraftSky avatar uses the same circular composition
  Examples:
    | flow         |
    | onboarding   |
    | Edit Profile |
```

Cover camera/gallery using deterministic picker boundaries. In widget tests verify fixed-square/circular-preview configuration and acceptance wiring; the result/appearance steps are checked briefly in MAN-001. Do not add automated pixel-export comparisons.

### AT-006: Retain profile save and onboarding completion semantics

Requirement IDs: FR-006
Acceptance Criteria: AC-006
Priority: Must
Level: Acceptance (widget/service)
Automation Target: Edit Profile, onboarding profile step/payload, and profile picker suites

```gherkin
Scenario Outline: Confirming an avatar does not itself save the profile
  Given <flow> has a previous avatar and unsaved profile fields
  When I accept a new avatar composition
  Then the preview and upload represent the accepted image
  And no profile record write has yet occurred
  When I invoke <save>
  Then the existing profile write contains the accepted blob
  And the other profile fields are preserved
  Examples:
    | flow         | save                  |
    | Edit Profile | Save                  |
    | onboarding   | onboarding completion |
```

Pair with cancellation in AT-004 and late-account-change cases in AT-009.

### AT-007: Reopening editing uses the accepted selection

Requirement IDs: FR-007
Acceptance Criteria: AC-007
Priority: Must
Level: Acceptance (widget/provider)
Automation Target: existing composer provider and draft snapshot adapter suites

```gherkin
Scenario Outline: Use accepted pixels in subsequent operations
  Given a <kind> composer contains prepared local photos with draft metadata
  When I accept controlled editor output for a photo
  Then reopening its editor receives that accepted selection
  And the draft snapshot and submit input represent its current bytes and dimensions
  And its order and alt text remain unchanged
  Examples:
    | kind     |
    | standard |
    | project  |
```

Use existing provider/adapter test seams and fake editor output. UT-007 checks fresh-media mapping after repeated edits; the existing repository suite protects disk storage. Actual save/close/reopen is a short MAN-002 check, not a new automated cross-layer journey.

### AT-008: Offer Edit only for eligible local images

Requirement IDs: RULE-001
Acceptance Criteria: AC-008
Priority: Must
Level: Acceptance (widget)
Automation Target: attachment section and composer provider suites

```gherkin
Scenario Outline: Reference-only and incomplete images have no Edit control
  Given an attachment is <state>
  Then Edit is <visibility>
  And its existing other controls retain their current behaviour
  Examples:
    | state                                | visibility |
    | ready local selection                | visible    |
    | ready restored local draft           | visible    |
    | queued or processing                 | absent     |
    | unavailable local media              | absent     |
    | retained published reference only    | absent     |
    | stored scheduled reference only      | absent     |
```

Then replace an eligible-for-replacement reference/unavailable image through its existing flow: Edit appears only after local preparation succeeds. No download of the retained image is initiated. Do not introduce a replacement action where the existing surface lacks one.

### AT-009: Fence editor completion and submission

Requirement IDs: RULE-003
Acceptance Criteria: AC-010
Priority: Must
Level: Acceptance (widget/provider)
Automation Target: composer, onboarding state, and Edit Profile suites

```gherkin
Scenario Outline: Late output cannot affect a different operation
  Given an editor or output preparation is held pending
  When <change> occurs and the old operation completes
  Then the stale result changes no active image or profile
  And no upload or save is initiated for that stale result
  Examples:
    | change                              |
    | target image is removed             |
    | target image is replaced            |
    | initiating composer/profile closes  |
    | active account changes              |
    | account activation changes away/back|
```

Use controlled callbacks/provider actions for changes inaccessible beneath the modal. Separately attempt Save, Publish, Schedule, or onboarding completion as applicable while editing/preparing: none submits an intermediate result. Rapid double Done applies once and causes at most one upload for that accepted avatar.

### AT-010: Accessible and localized editor entry and controls

Requirement IDs: NFR-002
Acceptance Criteria: AC-012
Priority: Must
Level: Acceptance (widget)
Automation Target: **proposed** editor suite and existing composer/profile localization/widget suites

```gherkin
Scenario: Operate the editor with accessible controls
  Given localized composer/avatar UI with screen-reader semantics and reduced motion enabled
  Then Edit, crop/rotate, Done, and Cancel have meaningful localized names
  And touch targets meet the application's existing minimum
  When I use keyboard focus and activation in the existing widget test harness
  Then I can enter and leave the editor without a focus trap
  And focus returns to an appropriate initiating control
  And reduced-motion settings are respected
```

Exercise a non-English supported locale, enlarged text, and narrow viewport. Physical assistive technology and browser focus behaviour also require MAN-003.

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | RULE-001 | AC-008 | Eligibility/state matrix | TD-002, all actual non-ready phases, read-only/busy surface | Only prepared local bytes in an enabled scoped surface permit editing; reference URL alone is insufficient | Composer provider/attachment suites |
| UT-002 | FR-003 | AC-003 | Apply valid output atomically | TD-001/TD-002, edited bytes with different dimensions | Target identity/order/alt persist; content hash/dimensions/MIME/preview describe the accepted prepared output; stored origin cannot refer to old content; other items unchanged | Composer provider suite |
| UT-003 | FR-004 | AC-004 | Preserve state on cancellation/failure | TD-003; no previous avatar and previous accepted avatar variants | No partial replacement; previous state retained; busy state clears; cancellation has no error; subsequent retry works | Composer provider/profile picker suites |
| UT-004 | RULE-002 | AC-009 | Validation at editor boundaries | TD-004, existing configured limits at/below/above boundary, corrupt/empty output | Existing supported formats/limits remain authoritative; invalid output cannot replace accepted state; no unbounded decode before source validation | Existing media service suite plus proposed editor adapter suite |
| UT-005 | RULE-003 | AC-010 | Stable operation/account fencing | TD-003/TD-006, late results, replacement with same list index, reorder, duplicate completion | Reorder still targets stable image; removed/replaced/disposed/old-account results ignored; only current operation applies once | Composer/profile/onboarding provider suites |
| UT-006 | NFR-003 | AC-013 | Diagnostic classification and serialized privacy | TD-007; typed editor/export failure with stack; cancel/expected rejection | One issue owner for unexpected failure; original permitted causes/stacks retained; positive safe fields present; protected canaries absent from local and SDK serialization; cancel creates no issue | Existing observability suites and serialized Sentry helpers |
| UT-007 | FR-003, FR-007 | AC-003, AC-007 | Draft and submit mapping after edit | Prepared images with stored origins; repeated fake edits; standard/project snapshot adapters | Edited image maps to fresh prepared media with current bytes/hash/MIME/dimensions; old origin is cleared; unchanged images retain existing mapping; submit inputs use current accepted payload | Existing draft media/snapshot adapter and composer provider suites |
| UT-008 | NFR-003 | AC-013 | Local editor boundary | Fake editor, recording picker/upload/storage clients | Opening/cancelling does not request a remote editing service, extra permission, gallery write or deletion; accepted avatar output reaches only the existing upload flow | Profile picker/provider and existing privacy unit tests |

## 5. Integration Test Cases

None planned, by explicit maintainer decision. IT-001–IT-006 from the initial draft are retired and must not be implemented or reused for other cases. No new integration files, test drivers, browser harnesses or package export checks. Existing integration tests remain in the repository; this change does not remove them.

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test / Automation Target |
|---|---|---|---|---|
| REG-001 | Optional editing, existing attachment controls and form data | FR-001, FR-003 | AC-001, AC-003 | Add/remove/reorder/alt-text editing, limits, camera/gallery, photo-details navigation, and post/project submission remain usable without editing; extend shared attachment and project composer suites |
| REG-002 | Local drafts and retained published/scheduled media | FR-007, RULE-001 | AC-007, AC-008 | Old unedited drafts restore; missing-media recovery and reference previews/metadata controls still work; no new old-image download or required edit step; existing draft restoration and published/scheduled composer suites |
| REG-003 | Profile/onboarding form preservation | FR-004, FR-006 | AC-004, AC-006 | Cancel/failure/retry preserve previous avatar and unrelated edits; gallery/camera behaviour and existing Save/completion semantics remain; Edit Profile, onboarding, profile picker suites |
| REG-004 | Shared business forms and private boundaries | FR-001, FR-005, NFR-003 | AC-001, AC-005, AC-013 | Product/event image selection retains prior behaviour with no new editor; profile banner has no added editor; no gallery mutation/PDS deletion; existing product/event and profile widget suites plus recording boundaries |

Regression scope assertions protect explicit non-goals; they do not add those surfaces to editing scope.

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Distinguishable selections/output | Small valid PNG/JPEG fixtures with distinct appearance and dimensions; fake edited result | AT-001–AT-007, UT-002/007/008 |
| TD-002 | Identity and eligibility | Three stable IDs, distinct alt text/order; ready/restored/pending/unavailable/reference-only states | AT-001/003/007/008, UT-001/002/007 |
| TD-003 | Timing and failure | Controlled editor/preparation/upload futures; success/cancel/error/rejected output | AT-004/006/009, UT-003/005 |
| TD-004 | Validation | Existing codec/limit fixtures and configured byte/dimension boundaries; no new physical-image matrix | UT-004, REG-002 |
| TD-005 | Accessibility | Supported locales, enlarged text, reduced motion, semantics and keyboard focus | AT-010, MAN-003 |
| TD-006 | Ownership | Two fake accounts, activation/session generations and disposed/replaced flow | AT-009, UT-005 |
| TD-007 | Diagnostic privacy | Synthetic protected-value canaries, typed cause/stack, safe operation context | UT-006/008 |

Use existing fixture helpers and fakes; do not commit personal photos, private paths or credentials.

## 8. Manual Checks

Brief checks in the finished app using available iOS/Android/web targets; no new harnesses or formal browser/codec/performance matrix. Record which targets were checked without claiming unperformed validation.

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-002, FR-005, NFR-001 | AC-002, AC-005, AC-011 | Crop/rotate and avatar appearance | Edit a composer photo, crop/rotate and accept/cancel; choose an avatar and adjust it on available supported targets | Finished UI works, accepted preview agrees with the edit, circle matches displayed avatar; no separate package pixel/export suite |
| MAN-002 | FR-007 | AC-007 | Save/reopen | Edit a post and project photo, save draft, close and reopen; inspect image/alt/order, then inspect normal publish/scheduling behaviour in a local or already authorized test environment | Accepted edit is retained by existing flows; no production write required |
| MAN-003 | NFR-002 | AC-012 | Usability | Check labels, text scaling, reduced motion, keyboard focus and available screen reader | Essential controls remain readable, identified and operable |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | Proposed coverage is not implemented yet | BR-001, FR-001–FR-007, RULE-001–RULE-003, NFR-001–NFR-003 | AC-001–AC-013 | Planning stage | Implement focused unit/widget cases in TDD |
| GAP-002 | Package geometry/export/native codecs not independently tested | FR-002, FR-005, RULE-002, NFR-001 | AC-002, AC-005, AC-009, AC-011 | Maintainer explicitly trusts package and excludes new integration tests | Rely on package and existing preparation tests; brief MAN-001; no new harness obligation |
| GAP-003 | Cross-layer persistence/publication not newly automated | FR-006, FR-007 | AC-006, AC-007 | New integration coverage excluded | Unit/provider mapping and widget save semantics plus existing repository tests and MAN-002 |
| GAP-004 | Platform assistive technology is not fully modeled by widget tests | NFR-002 | AC-012 | Host semantics checks have practical limits | AT-010 and brief MAN-003 where available |
| GAP-005 | No live PDS/federation proof | FR-006, FR-007 | AC-006, AC-007 | No changed server contract or live-write test scope | Do not claim live proof; existing live workflows remain separate |

No blocking product decisions. Risk remains Medium; the reduced automated validation scope is a confirmed maintainer decision, not an outstanding approval request.

## 10. Out Of Scope

- Advanced tools, video, banners, business product/event editing, and remote published/scheduled image acquisition: explicit non-goals. Regression checks ensure no accidental enablement.
- Durable originals/history or cross-session undo: confirmed A3/A4 and NG-004; reopen begins with accepted pixels.
- Backend/Lexicon migrations, new API contracts, infrastructure changes, production deployment, live PDS deletion or cleanup: no such changes are proposed.
- Package pixel/geometry/export conformance checks, exact JPEG comparisons, browser matrices and memory profiling: trust the package and keep checks focused on CraftSky-owned behaviour.
- Implementing tests, running tests/builds, adding dependencies, committing, or pushing during this stage.

## 11. Handoff To Document Review

- Requirements: `01-requirements.md`; tests: `02-acceptance-tests.md`; review: `03-document-review.md`; coding plan: `04-coding-plan.md`, in this folder.
- Maintainer approved A1–A4 and the plan, with new integration tests excluded. This revision aligns all documents with that decision and preserves existing IDs; IT-001–IT-006 are retired.
- First failing test: AT-001 in the existing attachment widget suite: on-image Edit opens the correct prepared image from the composer summary.
- Suggested TDD order: AT-001/UT-001; restricted editor widget smoke checks AT-002/005; state/cancel/fencing UT-002/003/005 and AT-003/004/008/009; avatar AT-005/006; mapping AT-007/UT-007; privacy/validation UT-004/006/008; AT-010 and regressions.
- Focused commands from root: `just app-test test/feed/widgets/composer_image_attachment_section_test.dart`; corresponding existing provider/profile/onboarding/draft adapter tests as changed; `just app-analyze`; full `just app-test` after focused tests pass.
- No new integration runner, device test target, ChromeDriver, browser export harness or file-backed end-to-end journey. Brief manual checks use the finished app.
- Blocking gaps: None. Implementation is not started by this documentation revision.
