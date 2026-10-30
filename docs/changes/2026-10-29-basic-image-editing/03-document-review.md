# Document Review: Basic Image Editing and Avatar Preview

## Verdict

Status: Approved with notes
Reviewer: Codex; maintainer approved the plan with new integration tests excluded
Date: 2026-10-29
Risk level: Medium

## Summary

Requirements, acceptance specification and coding plan are aligned with the confirmed feature scope and the maintainer's latest direction: **no new integration tests**. Trust the package's core editing/platform implementation. Use focused unit/provider and widget tests for CraftSky-owned wiring, state, validation, draft mapping, ownership, and save semantics, with brief finished-UI manual checks.

The product requirements and acceptance criteria remain intact. All A1–A4 are confirmed. The plan is approved; this revision updates the validation design, without starting implementation or creating commits. Existing integration tests remain untouched.

## Findings

The prior review's expanded integration/export obligations are superseded by the maintainer's decision. No blocking findings remain.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Traceability | Previously pending assumption/status wording has been reconciled with maintainer confirmation. | Requirements sections 3/21/22; coding plan section 1 | Resolved; retain stable IDs and confirmed decisions. |
| DR-002 | Suggestion | Risk | Compatible API and callback/pop ownership still matter to our wrapper; a separate package export suite is unnecessary under the approved test scope. | FR-002/005; AT-002/005; coding plan section 5 | Inspect resolved API and test visible controls/callback wiring with unit/widget tests. No real-codec export or conformance gate. |
| DR-003 | Suggestion | Tests | Proposed web/device harnesses and browser driver work are no longer in scope. | NFR-001; test specification section 5; coding plan section 9 | Superseded. Add no integration targets, test driver or formal browser matrix; brief manual checks only. |
| DR-004 | Suggestion | Tests | Proposed pixel oracle, JPEG tolerances and repeated large-photo profiling are no longer needed. | Test specification sections 7–9; coding plan section 9 | Superseded. Use simple distinct fixtures for application state/wiring and trust package geometry/export. |

## Traceability Review

- Planning to requirements: Selected package, basic tools, on-image composer entry, avatar circle preview, square output and confirmed local-only eligibility remain unchanged.
- Requirements to acceptance criteria: All 14 Must requirements retain links to AC-001–AC-013.
- Acceptance criteria to tests: Each criterion maps to unit/widget checks, existing regression coverage or brief manual checks. IT-001–IT-006 are retired without reusing IDs. UT-007 and UT-008 cover bounded application-owned mapping/local boundary logic; they are not renamed cross-layer journeys.

## Coverage Review

- Must requirements covered: BR-001, FR-001–FR-007, RULE-001–RULE-003, NFR-001–NFR-003 have planned verification paths.
- Missing or weak coverage: Package export/native codec/platform behaviour will not be independently automated; the maintainer explicitly accepts relying on the package and existing preparation logic/tests.
- Persistence: Focused provider/snapshot/media adapter tests verify accepted bytes/hash/dimensions and clearing stale origin; existing repository tests protect storage; a brief manual save/reopen covers the finished flow. No new automated disk-to-publish journey.
- Manual-only coverage: Platform interaction/visual appearance and practical assistive-technology behaviour beyond widget semantics. No new harness or formal compatibility/performance matrix.
- Evidence: No tests have been run; no physical-device, browser-export or live-PDS proof is claimed.

## Risk And Approval Review

- Risk level: Medium, unchanged.
- Review requirement: Completed. The maintainer approved the plan and explicitly reduced test scope.
- Approval notes: Do not reintroduce integration suites, package conformance tests, pixel export comparisons, or device/browser infrastructure as prerequisites. Preserve normal validation/resource/account guards in implementation.
- No implementation, dependency changes, test execution, commits or production actions performed.

## Coding Plan Readiness

- Ready for coding planning: Yes; the existing coding plan is revised and approved in scope.
- Recommended first step: AT-001, a focused widget test for opening the correct image from an on-image Edit button.
- Blocking issues: None.

## Notes For Next Stage

- Follow revised `04-coding-plan.md` with unit/widget-first TDD when implementation is requested.
- Trust the package core; test our own adapter/UI and state boundaries.
- Keep existing integration tests unchanged. Do not create integration files, test drivers, browser harnesses, or file-backed cross-layer journeys for this feature.
- Preserve all functional requirements, confirmed A1–A4, stable IDs and scope exclusions.
