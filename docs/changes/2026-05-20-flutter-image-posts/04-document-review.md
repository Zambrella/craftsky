# Document Review: Flutter Image Posts

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 document-reviewer
Date: 2026-05-20
Risk level: High

## Summary
The workflow documents are ready for TDD implementation. Discovery, requirements, and test design now align on the core slice: top-level text posts may attach images; replies remain text-only; images are prepared locally, stripped of privacy-sensitive metadata, uploaded through AppView, included in top-level `images[]`, rendered in feed cards, and opened in a full-screen gallery. The test plan covers every Must requirement with automated targets or justified manual checks for platform/gesture/privacy behavior.

Implementation should proceed carefully because the work is broad and high-risk: media picker permissions, metadata stripping, upload lifecycle state, AppView API expansion in Flutter, generated model/provider code, carousel/gallery gestures, accessibility, and regression preservation are all in scope.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Discovery / Traceability | Discovery still contains earlier wording such as “originals only” and “validate-only originals,” but it also records that the decision was superseded for privacy. Requirements and tests consistently use the newer metadata-stripping policy. | `01-discovery-notes.md` Q1 update, recommendation text, scope/decision summary; `02-requirements.md` FR-004..FR-004C; `03-acceptance-tests.md` UT-004, IT-004, MAN-002, GAP-001 | No implementation block. Treat `02-requirements.md`, `03-acceptance-tests.md`, and this review as source of truth. Do not implement upload-originals behavior; implement metadata stripping as specified. |
| DR-002 | Suggestion | Tests / Risk | Metadata stripping and gesture quality necessarily rely partly on manual checks. The test design documents automated fixture tests plus real-device/manual checks, which is appropriate, but implementation review must not skip those manual checks. | `02-requirements.md` FR-004..FR-004C, FR-020, FR-021; `03-acceptance-tests.md` MAN-002, MAN-003, GAP-001, GAP-002 | Keep MAN-002 and MAN-003 in the implementation review checklist. Document any library/platform metadata limitations discovered during implementation. |
| DR-003 | Suggestion | Implementation readiness | The selected image-picker and metadata-stripping libraries are intentionally not chosen yet. This is acceptable for test design, but the first implementation steps should isolate picker/preparer interfaces so tests can drive behavior without depending on platform plugins. | `01-discovery-notes.md` dependency gap and open questions; `02-requirements.md` RISK-004/RISK-006; `03-acceptance-tests.md` UT-004, IT-004, MAN-006 | Start with config and pure validation/preparer abstractions before wiring a concrete picker/plugin. Keep platform behavior behind fakes for widget/provider tests. |

## Traceability Review
- Discovery to requirements: Confirmed. Requirements preserve the confirmed app integration scope and later review decisions: top-level only, replies text-only, text remains required, backend-mediated upload, configurable app media limits, 300-character alt text cap, metadata stripping, current composer order, feed carousel, full-screen gallery, inline/fullscreen zoom, and no orphan cleanup.
- Requirements to acceptance criteria: Confirmed. Every Must `BR`, `FR`, `NFR`, and `RULE` in `02-requirements.md` links to at least one acceptance criterion. Lettered requirements (`FR-003A`, `FR-004A`..`FR-004C`, `FR-008A`/`FR-008B`, `FR-013A`, `FR-016A`, `FR-019A`) are covered explicitly.
- Acceptance criteria to tests: Confirmed. `03-acceptance-tests.md` maps each acceptance criterion to acceptance, unit, integration, regression, and/or manual tests. Manual/partial automation cases are documented as risks rather than hidden.

## Coverage Review
- Must requirements covered: Yes. All Must business, functional, non-functional, and business-rule requirements have test IDs in the coverage matrix.
- Missing or weak coverage: None blocking. Metadata stripping, pinch zoom, and hero transition have partial/manual coverage because full automation is impractical; the plan pairs automated state/fixture/widget tests with manual real-device checks.
- Manual-only coverage: `MAN-001` for public-media wording; `MAN-002` for real-device metadata stripping; `MAN-003` for physical gesture feel; `MAN-004` for visual contrast; `MAN-005` for hero polish; `MAN-006` for platform picker/permission smoke testing. These are justified by platform, visual, and human-gesture constraints.

## Risk And Approval Review
- Risk level: High.
- Review requirement: Required before implementation due to local media access, privacy-sensitive metadata handling, authenticated uploads, public media implications, broad UI changes, generated Flutter code, and complex gestures.
- Approval notes: Approved with notes. The notes are implementation cautions, not blockers. The TDD builder should follow the documented test order and keep manual checks visible for implementation review.

## Implementation Readiness
- Ready for TDD implementation: Yes.
- Recommended first step: Start with `UT-001` for local app media configuration constants in a future `app/test/feed/media/media_config_test.dart`, as recommended by `03-acceptance-tests.md`. This gives later selection, upload-size, and alt-text tests a single source of truth.
- Blocking issues: None.

## Notes For Next Stage
- Treat `02-requirements.md`, `03-acceptance-tests.md`, and this review as source of truth. Earlier discovery wording about “originals only” is superseded by the metadata-stripping requirements.
- Keep post text required; image-only posts are explicitly out of scope for this slice.
- Do not change backend/API contracts unless implementation proves the existing backend contract cannot be consumed as documented.
- Isolate image picker, metadata stripping, upload, and draft state behind testable abstractions before wiring concrete platform plugins.
- Use the existing Flutter test conventions under `app/test/feed/**`, `http_mock_adapter`, Riverpod provider tests, and fake cache managers.
- Run generated-code tooling after model/provider/router changes: `cd app && dart run build_runner build --delete-conflicting-outputs`.
- Final implementation review should include manual checks for real-device metadata stripping, gesture feel, indicator contrast, hero transition polish, permission handling, and public-media wording.
