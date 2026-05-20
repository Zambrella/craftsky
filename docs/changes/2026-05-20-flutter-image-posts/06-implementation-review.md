# Implementation Review: Flutter Image Posts

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation-reviewer
Date: 2026-05-20
Risk level: High

## Summary
The implementation adds useful model/API plumbing, draft-state tests, feed image rendering, and gallery routing, and the focused widget tests pass. However, it does not yet satisfy several Must requirements for the real user-facing image-posting flow. The default image service is a no-op, so tapping **Add image** in the production app cannot select, validate, strip, prepare, upload, or add any image. Manual composer reordering and aspect-ratio capture are also missing. These are acceptance-criteria gaps, not polish.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Critical | Behavior / Risk | Production image selection/upload is not implemented. `composerImageServiceProvider` returns `NoopComposerImageService`, whose `addImages` method is empty, so the top-level composer shows an **Add image** button but cannot actually add images in the app. Consequently the real flow does not perform selection, validation, metadata stripping, prepared-byte size validation, upload to `/v1/blobs/images`, upload progress, or upload failure handling. Tests only inject a fake service that mutates draft state directly. | `app/lib/feed/providers/composer_image_service.dart`; `app/lib/feed/widgets/post_composer_sheet.dart`; BR-001; FR-001, FR-002, FR-003, FR-003A, FR-004..FR-006, FR-012; NFR-001; AC-001, AC-007, AC-008, AC-009, AC-010; AT-001, AT-002, IT-005 | Replace the production no-op with a real service or app-wired implementation that selects supported images, validates them against config, strips/prepares bytes, validates prepared size, uploads via `PostApiClient.uploadImage`, updates draft lifecycle/progress/failure state, and is covered by focused tests through fakes. |
| IR-002 | Critical | Behavior / Tests | Composer image reordering is required but absent. There is no reorder method on `ImageDraftController`, no reorder UI in `PostComposerSheet`, and no failing/passing UT-007 coverage. The create payload can preserve a supplied list order, but the user cannot manually reorder selected images in the composer. | `app/lib/feed/providers/image_draft_controller.dart`; `app/lib/feed/widgets/post_composer_sheet.dart`; `docs/.../05-implementation-plan.md`; FR-013, FR-013A; AC-015, AC-015A; AT-001; UT-007 | Add draft reorder behavior and top-level composer UI for reordering images before submission, including tests that uploads may complete out of order while composer order controls payload order. |
| IR-003 | Important | Behavior / Data | Aspect-ratio capture is not implemented in the composer flow. `CreatePostImage` supports `aspectRatio`, but `UploadedDraftImage` stores only `cid`, `mime`, and `size`, and `_payloadImages()` never supplies `aspectRatio`. There is no local dimension extraction or UT-010-style coverage. | `app/lib/feed/providers/image_draft_controller.dart`; `app/lib/feed/widgets/post_composer_sheet.dart`; `app/lib/feed/models/create_post_image.dart`; FR-009, FR-010; AC-013; UT-010; IT-002 | Capture optional positive width/height metadata during preparation/selection, carry it through draft state, and include it in `CreatePostImage` when available; add tests for known, unknown, and invalid dimensions. |
| IR-004 | Important | Tests / Traceability | Several planned Must/partial-risk tests were skipped or replaced by narrower tests without documenting cancellation. Missing or incomplete coverage includes UT-007 reordering/deletion order, UT-010 aspect-ratio extraction, IT-004 metadata stripping before uploader receives bytes, and stronger IT-007 pinch/zoom state verification. The implementation plan completion checklist is still unchecked. | `docs/.../03-acceptance-tests.md` UT-007, UT-010, IT-004, IT-007, MAN-002, MAN-003; `docs/.../05-implementation-plan.md` lines 196-202 | Restore the missing TDD loops or explicitly document justified cancellations/gaps. At minimum, add tests for the blocking behavior in IR-001 through IR-003 and update the completion checklist accurately. |
| IR-005 | Important | Privacy / Risk | Metadata stripping is represented as a map-key policy helper, but no implemented production upload pipeline applies it to selected image bytes before upload. This leaves the privacy requirement unfulfilled despite policy-level unit tests. | `app/lib/feed/media/image_metadata_stripper.dart`; `app/lib/feed/providers/composer_image_service.dart`; FR-004, FR-004A, FR-004B, FR-004C; AC-009, AC-009A; IT-004; MAN-002; DR-001, DR-002 | Integrate metadata stripping into the real upload-preparation service and add pipeline-level coverage that the uploader receives prepared/stripped bytes. Keep MAN-002 as a real-device follow-up. |
| IR-006 | Suggestion | Code Quality | `flutter analyze` reports 30 info-level issues, mostly directive ordering, redundant/default arguments, const suggestions, line length, and cascade suggestions. These are not the main blockers, but should be cleaned up when making the required behavior fixes. | Reviewer-run `flutter analyze`; affected changed files listed in command output | Address analyzer info findings during the follow-up implementation pass where practical. |

## Requirement And Test Traceability
- Requirements implemented: Partial. API request/response models, upload API method, create-post `images[]` serialization, text-only regressions, basic draft state, submit gating, feed carousel rendering, gallery routing, hero-tag wiring, and visible/semantic alt text are represented.
- Tests implemented: UT-001, UT-002, UT-003, UT-004, UT-005, UT-006, UT-008, UT-009, UT-011, UT-013, IT-001, IT-002, portions of IT-005, portions of IT-006, portions of IT-007, and REG-001 through REG-006 have some automated coverage.
- Unplanned behavior: None identified as harmful, but the production no-op service creates a misleading UI affordance because **Add image** is visible without real functionality.
- Remaining gaps: Real image picker/preparer/uploader integration, byte-level metadata stripping pipeline, manual reordering, aspect-ratio capture, stronger pinch/gesture evidence, and the documented manual checks MAN-001 through MAN-006.

## Test Evidence
- Commands reviewed:
  - Implementation plan reported multiple focused red/green commands and final focused widget verification.
  - Reviewer ran: `cd app && flutter test test/feed/widgets/post_image_carousel_test.dart test/feed/widgets/post_card_test.dart test/feed/widgets/post_image_gallery_test.dart test/feed/widgets/post_composer_sheet_test.dart`.
  - Reviewer ran: `cd app && flutter analyze`.
- Passing evidence:
  - Reviewer-run focused widget suites passed: 37 tests passed.
- Failing or skipped tests:
  - `flutter analyze` completed with 30 info-level issues.
  - No automated failing tests remain, but critical acceptance behavior is untested/missing because current tests rely on fakes and do not exercise production image selection/preparation/upload.

## Risk Review
- Risk level: High.
- Risk notes: The highest risks called out in document review remain unresolved for production: local media selection/permissions, privacy-sensitive metadata stripping, authenticated upload lifecycle, manual reorder behavior, and platform/gesture manual checks.
- Approval notes: Not ready for merge or handoff as complete. The implementation should return to TDD for the blocking findings above.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The current UI changes are not ready for a polish-only pass because several visible issues are behavioral requirements, not copy/style refinements. After the required fixes land, a small polish pass may be useful for composer preview layout, upload/progress visual states, hard-coded image copy, indicator contrast, and accessibility labels.
- Suggested polish notes: Defer polish until **Add image** works end-to-end and reordering/progress/preview states exist in the real UI.

## Handoff Back To TDD Builder
- Required fixes:
  1. Implement production image selection/preparation/upload service instead of the no-op provider.
  2. Integrate byte-level metadata stripping and prepared-byte size validation into the upload pipeline.
  3. Add composer image reordering behavior and tests.
  4. Capture optional image aspect ratio and include it in create payloads.
  5. Fill/update missing TDD loops and completion checklist in `05-implementation-plan.md`.
- Suggested next failing test: Add an integration-style widget/provider test proving that tapping **Add image** with a fake picker/preparer/uploader adds a draft image, strips/prepares bytes, uploads via the AppView client abstraction, updates progress, and enables submit only after valid alt text.
- Verification to rerun:
  - `cd app && flutter test test/feed/media test/feed/providers test/feed/data/post_api_client_test.dart test/feed/widgets/post_composer_sheet_test.dart test/feed/widgets/post_card_test.dart test/feed/widgets/post_image_gallery_test.dart`
  - `cd app && flutter analyze`
  - Relevant manual checks MAN-001 through MAN-006 before approval.
