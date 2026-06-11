# Implementation Review: Flutter Project Composer UI

## Verdict
Status: Changes required
Reviewer: Implementation-review agent
Date: 2026-06-11
Risk level: High

## Summary

The implementation adds the planned Flutter-only project composer surface, option catalogs, FormBuilder field wrappers, chooser routing, payload helpers and broad test coverage. Static analysis and focused project composer tests pass, and the implementation plan records a passing full Flutter suite.

However, the project composer MVP is not ready because the user-facing photo workflow is not implemented in the project composer. The composer requires at least one photo, but its visible “Add photo” control is always disabled and the screen does not render attached image tiles, reorder controls or alt-text controls. Tests bypass this by injecting uploaded image provider state, so the critical end-user path is untested and unusable.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Critical | Behavior | Project posts require at least one photo, but users cannot add a photo in the project composer. The visible add-photo button has `onPressed: null`, there is no call to `composerImagesProvider(...).notifier.addImages()`, and the project composer does not render selected image tiles, remove/reorder controls or alt-text controls. Because submit validation requires `imagesState.images.isNotEmpty`, the real UI path cannot create a project post. | `01-requirements.md` `BR-001`, `FR-009`, `FR-016`, `FR-017`, `RULE-005`; `AC-007`, `AC-012`, `AC-016`, `AC-023`; `04-coding-plan.md` planned `ComposerImageAttachmentSection`; `app/lib/projects/widgets/project_composer_sheet.dart:177-187`, `743-805` | Implement the project composer photo attachment UI by reusing/extracting the regular composer image section or equivalent provider-backed controls. The project composer must allow adding photos, show selected image state, support alt text controls and existing image notices/missing-alt behaviour, then submit uploaded images through the existing create plumbing. |
| IR-002 | Important | Tests | The acceptance tests do not cover the real project photo-add/alt-text UI path. Project submit/feedback tests inject uploaded `composerImagesProvider` state, so they pass while the actual add-photo affordance is disabled. | `02-acceptance-tests.md` `AT-003`, `AT-004`, `AT-008`, `IT-005`; `05-implementation-plan.md` Steps 20, 24, 30; `app/test/projects/widgets/project_composer_submit_test.dart`; `app/test/projects/widgets/project_composer_feedback_test.dart`; `app/test/projects/widgets/project_composer_images_test.dart` | Add or update widget tests so the project composer exposes an enabled add-photo action and selected image UI/alt-text controls through the same public/provider-backed path users will exercise. The test should fail on the current disabled button. |
| IR-003 | Important | Localization | New user-visible copy remains hard-coded inside the reusable multi-select field (`Add item`, `Add`, `Disabled`, and the max-count message). This conflicts with the localization requirement for new user-visible strings. | `01-requirements.md` `NFR-003`, `AC-022`; `app/lib/theme/craftsky_form_builder_select_fields.dart:254-258`, `356-368`, `372-378` | Move these strings behind parameters supplied by the localized composer or otherwise localize them through app localization resources. Add test coverage for the remaining multi-select strings, not only the project composer ARB keys. |

## Requirement And Test Traceability

- Requirements implemented: UI-only option catalogs; FormBuilder-compatible text/dropdown/multi-select/radio fields; responsive top-level chooser; separate full-screen project composer; common/pattern/detail payload builders; craft-specific detail sections for sewing, knitting, crochet and quilting; common-only crafts; submit adapter with `reply == null`; discard, feedback and image notice handling; localized project/chooser strings; feed/profile top-level chooser wiring; regular/reply regression tests.
- Tests implemented: The implementation commit added the planned option, field, payload, subtype, draft, submit adapter, chooser, project composer, metadata, details, validation, feedback, image notice, provider, feed/profile and localization tests, and updated existing feed/profile regression tests.
- Unplanned behavior: None identified in backend/API/lexicon/migration/dependency areas. The implementation stayed Flutter-only and did not change project DTOs to enums.
- Remaining gaps: Real user photo attachment/alt-text UI for the project composer is missing and untested; several multi-select strings are still hard-coded instead of localized.

## Test Evidence

- Commands reviewed:
  - `git status --short` — clean before writing this review.
  - `git diff --name-status 21a09dc..e707e47` and `git diff --stat 21a09dc..e707e47` — reviewed implementation commit files.
  - `cd app && flutter analyze` — run during review.
  - `cd app && flutter test test/projects/widgets/project_composer_sheet_test.dart test/projects/widgets/project_composer_submit_test.dart test/projects/widgets/project_composer_feedback_test.dart` — run during review.
  - `05-implementation-plan.md` verification evidence for `cd app && flutter analyze` and `cd app && flutter test`.
- Passing evidence:
  - Review-run `flutter analyze`: `No issues found!`.
  - Review-run focused widget tests: `All tests passed!`.
  - Implementation plan reports final `flutter test`: `All tests passed!`.
- Failing or skipped tests:
  - No failing commands were observed during review.
  - A blocking coverage gap remains because no test fails on the disabled project add-photo affordance.

## Risk Review

- Risk level: High.
- Risk notes: The feature is user-facing and the missing photo UI blocks the central project-post requirement. Because project posts require photos, the composer is effectively non-functional for real users despite green tests.
- Approval notes: Do not finalize this stage until the project composer photo attachment UI and associated tests are corrected.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The visible issue with the disabled photo control is behavioral and must be fixed by TDD, not polish. After blocking fixes, a small polish pass could be useful for dense-form spacing/copy review, but it is not the next required step.
- Suggested polish notes: After implementation fixes pass, review the composer’s dense metadata layout, chip spacing and helper/error copy at compact and large widths.

## Handoff Back To TDD Builder

- Required fixes:
  - Add provider-backed project composer photo attachment UI equivalent to the regular composer path: enabled add-photo action, selected image display, alt text editing, removal/reorder where applicable, disabled/loading states and existing image notices.
  - Add tests that fail against the current disabled `Add photo` button and prove the real project photo/alt-text UI path works.
  - Localize or parameterize the remaining reusable multi-select strings and cover them in tests.
- Suggested next failing test:
  - Extend `app/test/projects/widgets/project_composer_images_test.dart` or `project_composer_sheet_test.dart` to open `ProjectComposerSheet`, verify the `Add a photo` control is enabled when not loading, trigger the real image-add path or injected image-provider action, and verify attached images expose alt-text controls before submit.
- Verification to rerun:
  - `cd app && flutter test test/projects/widgets/project_composer_images_test.dart test/projects/widgets/project_composer_submit_test.dart test/projects/widgets/project_composer_feedback_test.dart`
  - `cd app && flutter test test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart test/feed/providers/composer_images_provider_test.dart test/feed/providers/composer_image_state_test.dart`
  - `cd app && flutter analyze`
  - `cd app && flutter test`
