# Implementation Review: Flutter Image Posts

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation-reviewer
Date: 2026-05-20
Risk level: High

## Summary
The remediation resolved the prior production no-op service, composer reordering, aspect-ratio payload, and several gesture/test coverage gaps. The relevant automated Flutter tests pass, and the create/read/feed/gallery plumbing is substantially improved.

The implementation is still not ready because several Must user-facing and privacy/platform requirements remain incomplete. The composer does not render local image previews or actual upload progress, iOS image-picker privacy configuration is missing, WebP uploads bypass byte-level metadata stripping, and over-cap selections can be silently ignored instead of rejected with visible feedback. There is also an uncommitted generated iOS lockfile change that must be resolved by the implementation pass.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Critical | Behavior / UI | The composer still does not show the required local image preview or per-image upload progress. `DraftImageState` stores file name/lifecycle/progress but no preview bytes/path, and `_DraftImageTile` renders only the file name, status text, errors, reorder/remove controls, and alt field; `uploadProgress` is updated by the service but never displayed. | `app/lib/feed/providers/image_draft_controller.dart`; `app/lib/feed/widgets/post_composer_sheet.dart`; FR-005, FR-003A, NFR-004; AC-008, AC-014; AT-001, AT-002, IT-005 | Carry preview data in draft state or another UI-safe model and render a real local preview for each selected image. Display meaningful in-progress upload feedback derived from `uploadProgress` or equivalent state. Add widget/provider tests that would fail if preview/progress UI is absent. |
| IR-002 | Critical | Platform / Risk | The new production picker uses `image_picker`, but required platform configuration was not added. `app/ios/Runner/Info.plist` has no `NSPhotoLibraryUsageDescription`; `image_picker` documents this key as required/App-Store-required for photo-library usage. The working tree also contains an uncommitted `app/ios/Podfile.lock` diff adding `image_picker_ios`, so the implementation artifact set is incomplete. If macOS remains a supported target for this app, the documented user-selected read entitlement is also absent from the macOS entitlements. | `app/lib/feed/providers/composer_image_service.dart`; `app/ios/Runner/Info.plist`; `app/ios/Podfile.lock` working-tree diff; `app/macos/Runner/*.entitlements`; `image_picker` setup docs; FR-001; RISK-004; MAN-006; AC-001 | Add the required iOS photo-library usage description with accurate, non-private-implying copy; resolve and commit the generated iOS Podfile lock change in the implementation pass; add macOS entitlement or explicitly document/remove macOS support for this picker path. Re-run platform picker/permission smoke checks. |
| IR-003 | Important | Privacy / Behavior | WebP upload preparation returns `originalBytes` unchanged. WebP files can carry EXIF/XMP-style metadata, so this path can upload privacy-sensitive metadata while the JPEG/PNG paths are re-encoded. That does not satisfy the supported-image metadata-stripping requirement for WebP. | `app/lib/feed/media/image_metadata_stripper.dart` lines 102-106; FR-004, FR-004A, FR-004B, FR-004C; AC-009, AC-009A; UT-004, IT-004, TD-004, MAN-002 | Strip WebP metadata by re-encoding with a supported encoder, or fail/reject WebP files whose metadata cannot be safely stripped while preserving display/format requirements. Add fixture coverage for WebP metadata or document a justified requirement/test adjustment. |
| IR-004 | Important | Behavior / Tests | Over-cap image selections are not reliably rejected with user-visible feedback. `DeviceComposerImagePicker.pickImages` truncates platform results with `files.take(maxImages)` before validation, and `DefaultComposerImageService.addImages` silently returns when no slots remain. As a result, excess selections can disappear without the rejection feedback required by AC-007, and the existing validator rejection path is not exercised by production picker behavior. | `app/lib/feed/providers/composer_image_service.dart` lines 90-94 and 206-208; FR-001, FR-002, FR-012, RULE-002; AC-007; EC-001; AT-002; UT-002, IT-005 | Do not silently drop excess selections. Either constrain the platform picker with clear UI state and disable/explain at cap, or pass all returned files through validation and surface rejected/excess selections with visible feedback. Add a default-service/widget test for over-cap selection feedback. |
| IR-005 | Important | Tests / Traceability | Automated tests pass, but they do not cover the remaining Must gaps: composer preview/progress rendering, iOS picker privacy configuration, WebP metadata stripping, or production over-cap feedback. Manual checks MAN-001..MAN-006 also remain unexecuted/documented for a candidate build. | `03-acceptance-tests.md` AT-001, AT-002, IT-004, IT-005, MAN-001..MAN-006; `04-document-review.md` DR-002; `05-implementation-plan.md` remediation notes | Add tests for the blocking gaps above and keep manual checks documented for the next review. Do not mark the implementation ready until platform permissions, metadata stripping, gestures, contrast, hero transition, and public-media wording have been manually smoke-tested or explicitly deferred with rationale. |
| IR-006 | Suggestion | Code Quality | `flutter analyze` reports 38 info-level issues, mostly directive ordering, one-member abstract lint, catch clauses without `on`, redundant/default arguments, const suggestions, line length, and cascade suggestions. These are not the approval blockers, but they should be cleaned up while touching the affected files. | Reviewer-run `flutter analyze`; affected changed files listed in command output | Address analyzer info findings during the follow-up implementation pass where practical. |

## Requirement And Test Traceability
- Requirements implemented: Partial. The remediation implements a real picker/preparer/uploader abstraction, AppView upload API use, create-post `images[]` serialization, optional aspect-ratio propagation, draft reordering, submit gating, text-only regressions, feed carousel rendering, gallery routing, hero-tag wiring, and visible/semantic alt text.
- Tests implemented: UT-001, UT-002, UT-003, UT-004, UT-005, UT-006, UT-007, UT-008, UT-009, UT-010, UT-011, UT-013, IT-001, IT-002, IT-004, IT-005, IT-006, IT-007, and REG-001 through REG-006 have automated coverage in the reviewed test set.
- Unplanned behavior: The production picker truncates returned files before validation, which can silently ignore selected excess images. A generated iOS `Podfile.lock` change is currently unstaged/uncommitted in the working tree.
- Remaining gaps: Composer preview/progress UI, platform permission/config completion, WebP byte-level metadata stripping, over-cap feedback in the production service/UI, and manual checks MAN-001 through MAN-006.

## Test Evidence
- Commands reviewed:
  - Implementation plan reports focused red/green commands for the original pass and remediation steps R1-R5.
  - Reviewer ran: `flutter test test/feed/media test/feed/providers test/feed/data/post_api_client_test.dart test/feed/widgets/post_composer_sheet_test.dart test/feed/widgets/post_card_test.dart test/feed/widgets/post_image_gallery_test.dart` from `app/`.
  - Reviewer ran: `flutter analyze` from `app/`.
  - Reviewer inspected `git status --short`, `git diff --stat`, `git log --oneline -10`, the latest commit stat, changed implementation files, iOS/macOS/Android platform config, and `image_picker` setup documentation.
- Passing evidence:
  - Reviewer-run relevant Flutter suites passed: 114 tests passed.
- Failing or skipped tests:
  - `flutter analyze` completed with 38 info-level issues.
  - No automated test currently fails, but blocking acceptance behavior remains untested/missing as described in IR-001 through IR-005.
  - Manual checks MAN-001..MAN-006 were not completed during this review because implementation blockers remain.

## Risk Review
- Risk level: High.
- Risk notes: Production media selection is now wired, but platform privacy configuration, composer feedback, WebP metadata privacy, over-cap UX, and manual device checks remain unresolved. The uncommitted `app/ios/Podfile.lock` diff also means the implementation artifact set is not clean.
- Approval notes: Not ready for merge or handoff as complete. The implementation should return to TDD for the findings above.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The current UI issues around missing preview/progress and over-cap feedback are implementation requirements, not polish. After those are fixed, a small polish pass may be useful for composer tile layout, progress/error visual states, hard-coded image copy, reorder affordance labels, indicator contrast, and accessibility labels.
- Suggested polish notes: Defer polish until the blocking behavior and platform/privacy issues are fixed.

## Handoff Back To TDD Builder
- Required fixes:
  1. Render local preview and upload progress for selected composer images.
  2. Add required iOS picker privacy configuration, resolve/commit the iOS Podfile lock update, and handle/document macOS entitlement support.
  3. Fix WebP metadata stripping or safe rejection behavior.
  4. Surface visible feedback for over-cap image selection rather than silently truncating/returning.
  5. Add tests for those gaps and update `05-implementation-plan.md` with the new remediation loop and verification.
- Suggested next failing test: Add a widget/default-service test proving that after tapping **Add image**, selected images render a local preview and an upload-progress indicator before completion, and that an over-cap selection produces visible rejection feedback.
- Verification to rerun:
  - `cd app && flutter test test/feed/media test/feed/providers test/feed/data/post_api_client_test.dart test/feed/widgets/post_composer_sheet_test.dart test/feed/widgets/post_card_test.dart test/feed/widgets/post_image_gallery_test.dart`
  - `cd app && flutter analyze`
  - Relevant platform/manual checks MAN-001 through MAN-006 before approval.
