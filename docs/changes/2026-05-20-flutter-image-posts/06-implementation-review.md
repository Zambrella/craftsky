# Implementation Review: Flutter Image Posts

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation-reviewer
Date: 2026-05-20
Risk level: High

## Summary
The latest remediation resolves several previous blockers: the composer now renders local previews and upload progress, the iOS/macOS picker configuration artifacts are present, the generated iOS `Podfile.lock` change is committed, the production picker no longer truncates returned selections before validation, and the broad relevant Flutter test suite passes.

The implementation is still not ready for final approval. Two Must behaviors remain incomplete: WebP files from the production picker can still upload original bytes without embedded metadata removal, and partial over-cap selections can still put excess failed image tiles into the draft, causing the draft to exceed the configured maximum. The required manual checks for this high-risk feature also remain unexecuted/documented.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Critical | Privacy / Behavior | WebP metadata handling is still unsafe in the production path. `prepareImageForUpload` only rejects WebP when the caller supplies removable metadata keys, but `DeviceComposerImagePicker` always sets `metadata: const {}`. Therefore a real selected WebP file with embedded EXIF/XMP-style metadata will take the `SupportedImageFormat.webp => originalBytes` branch and upload the original bytes unchanged. The new test only covers fake caller-supplied metadata, not embedded metadata from real picker bytes. | `app/lib/feed/media/image_metadata_stripper.dart` lines 102-113; `app/lib/feed/providers/composer_image_service.dart` lines 208-219; FR-004, FR-004A, FR-004B, FR-004C; AC-009, AC-009A; IT-004; MAN-002; previous IR-003 | Do not pass through WebP original bytes unless the client can prove/removes embedded metadata. Either implement byte-level WebP re-encode/metadata stripping, convert to another supported format while preserving visible content as allowed by FR-004C, or conservatively fail WebP preparation when embedded metadata cannot be inspected/removed. Add a test that would fail for WebP original-byte passthrough in the production preparer path. |
| IR-002 | Important | Behavior / Tests | Partial over-cap selections can still exceed the draft's configured maximum. The composer now blocks taps when already at the cap, and the picker returns the full selection, but `DefaultComposerImageService` handles excess selections by adding rejected images to the draft as failed tiles. For example, with 3 existing images and 3 selected images, the draft can end with 6 tiles (1 accepted plus 2 failed excess tiles). This conflicts with RULE-002 (“A draft may contain at most the configured maximum number of selected images”) and AC-007's “excess addition is rejected” wording. Current remediation tests cover “already at cap” and picker non-truncation, but not this partial over-cap service/UI path. | `app/lib/feed/providers/composer_image_service.dart` lines 114-131; `app/lib/feed/widgets/post_composer_sheet.dart` lines 176-186; RULE-002; FR-001, FR-012; AC-007; EC-001; IT-005; `app/test/feed/widgets/post_composer_sheet_test.dart`; `app/test/feed/providers/composer_image_service_test.dart` | Add a failing test for selecting more images than remaining slots when the draft is below cap. Surface user-visible feedback for rejected excess selections without adding those excess images as draft items, or otherwise ensure `ImageDraftController.images.length` never exceeds `mediaConfig.maxImages`. |
| IR-003 | Important | Tests / Risk | Required manual checks remain unexecuted/documented for the candidate implementation. The test plan and document review explicitly call out MAN-001 through MAN-006 for public-media wording, real-device metadata stripping, physical gestures, indicator contrast, hero transition polish, and platform picker/permission behavior. The latest plan records them as remaining follow-ups, but final implementation review is expected to include them or an explicit approved deferral. | `03-acceptance-tests.md` MAN-001..MAN-006, GAP-001..GAP-003; `04-document-review.md` DR-002 and Notes For Next Stage; `05-implementation-plan.md` Remediation Pass 2 Verification; MAN-001..MAN-006 | Before approval, run and document the manual checks, or explicitly record an approved deferral/risk acceptance in the workflow docs. Pay special attention to MAN-002 because IR-001 is still unresolved for WebP. |
| IR-004 | Suggestion | Code Quality | `flutter analyze` still reports 42 info-level findings. These are not the approval blockers, but several are in changed files (`image_metadata_stripper.dart`, `composer_image_service.dart`, `image_draft_controller.dart`, `post_composer_sheet.dart`, and tests). | Reviewer-run `flutter analyze`; IR-006 from prior review | Clean up analyzer info findings opportunistically while addressing the required behavior fixes. |

## Requirement And Test Traceability
- Requirements implemented: Partial. The implementation covers model/API image payloads, AppView upload API calls, real picker/preparer/uploader abstractions, composer preview/progress, draft reordering, submit gating, aspect-ratio propagation, feed carousel/gallery rendering, hero tags, and platform config.
- Tests implemented: Automated coverage exists for UT-001..UT-011/UT-013, IT-001/IT-002/IT-004/IT-005/IT-006/IT-007, and regression coverage around text-only posts/replies and privacy copy. The latest remediation adds composer preview/progress, at-cap feedback, picker non-truncation, and WebP fake-metadata safe rejection tests.
- Unplanned behavior: Excess partial selections can remain visible as failed draft image tiles beyond the configured cap. WebP original-byte passthrough remains possible in production because metadata is not extracted from picker bytes.
- Remaining gaps: WebP embedded metadata privacy, partial over-cap draft behavior, required manual checks MAN-001 through MAN-006, and analyzer info cleanup.

## Test Evidence
- Commands reviewed:
  - Implementation plan reports focused red/green commands for R6 through R9.
  - Reviewer ran: `flutter test test/feed/media test/feed/providers test/feed/data/post_api_client_test.dart test/feed/widgets/post_composer_sheet_test.dart test/feed/widgets/post_card_test.dart test/feed/widgets/post_image_gallery_test.dart` from `app/`.
  - Reviewer ran: `flutter analyze` from `app/`.
  - Reviewer inspected `git status --short`, `git diff --stat HEAD~1..HEAD`, `git log --oneline -10`, changed implementation files, tests, and platform config files.
- Passing evidence:
  - Reviewer-run relevant Flutter suites passed: 118 tests passed.
  - Working tree was clean before writing this review.
- Failing or skipped tests:
  - `flutter analyze` completed with 42 info-level findings and no analyzer errors.
  - No automated test currently fails, but missing tests/behavior remain for WebP embedded metadata passthrough and partial over-cap selection.
  - Manual checks MAN-001..MAN-006 were not completed during this review.

## Risk Review
- Risk level: High.
- Risk notes: The highest remaining risk is privacy-sensitive metadata handling for WebP and real-device/platform behavior that cannot be fully proven by unit/widget tests. Over-cap selection handling is also still inconsistent with the configured draft maximum.
- Approval notes: Not ready for final approval. Return to TDD for IR-001 and IR-002, and complete or explicitly defer manual checks before asking for approval again.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The new preview/progress UI is functional enough for behavior review, but a polish pass could improve tile layout, hard-coded copy, progress/error visual states, icon affordance labels, and spacing after behavioral blockers are fixed.
- Suggested polish notes: Defer polish until WebP metadata handling, partial over-cap behavior, and manual checks are resolved or explicitly accepted.

## Handoff Back To TDD Builder
- Required fixes:
  1. Fix WebP embedded metadata handling so production WebP uploads do not pass original bytes through unsafely.
  2. Fix partial over-cap selection handling so rejected excess images do not make the draft exceed `mediaConfig.maxImages`.
  3. Add tests for both behaviors and update `05-implementation-plan.md` with the new red/green evidence.
  4. Run or explicitly document approved deferral of MAN-001 through MAN-006.
- Suggested next failing test: Add an IT-004 test proving that `DefaultComposerImagePreparer` does not return `originalBytes` for WebP input, or fails WebP preparation conservatively when embedded metadata cannot be stripped/inspected.
- Verification to rerun:
  - `cd app && flutter test test/feed/media test/feed/providers test/feed/data/post_api_client_test.dart test/feed/widgets/post_composer_sheet_test.dart test/feed/widgets/post_card_test.dart test/feed/widgets/post_image_gallery_test.dart`
  - `cd app && flutter analyze`
  - MAN-001 through MAN-006, or documented approved deferral.
