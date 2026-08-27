# Implementation Review: Link Previews

## Verdict
Status: Changes required
Reviewer: OpenCode
Date: 2026-08-26
Risk level: High

## Summary
The second correction pass resolves IR-013, IR-015, and IR-016, but final review found remaining Must-level lifecycle races and validation gaps. IR-012, IR-014, and IR-017 are only partially resolved. Scheduled editing can still restore or remove the wrong external card, new scheduled replacements can reuse an immutable media ID for changed content, malformed thumbnail blobs can surface as HTTP 500, and the required privacy evidence does not exercise the real fetch pipeline.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-018 | Critical | Behavior / Tests | Delayed scheduled thumbnail hydration unconditionally reseeds and selects the frozen card. If the member removes the source URL or selects another candidate while bytes load, hydration can resurrect the removed card or override the deliberate selection. Existing delayed-hydration coverage does not mutate text or selection before hydration completes. | FR-016; AC-018; IR-012; `app/lib/feed/widgets/post_composer_sheet.dart:303-317,1288-1304`; `app/lib/feed/composer/link_preview_controller.dart:118-150`; `app/test/scheduled_posts/scheduled_post_edit_test.dart:298-377,536-595` | Add failing delayed-hydration tests for source removal and alternate selection, then prevent hydration from inserting an absent candidate or overriding state changed after hydration began. |
| IR-019 | Critical | Behavior / Tests | A frozen scheduled card can be silently removed after reopening when its source URL was completed only by trailing whitespace. Submission persists trimmed text, reopening seeds the frozen card, and `updateText` then treats the terminal URL as incomplete and removes that candidate. Current fixtures retain trailing whitespace rather than exercising the actual persisted payload. | FR-016; AC-018; `app/lib/feed/widgets/post_composer_sheet.dart:303-317,616,793-817,1104-1117`; `app/lib/feed/composer/link_preview_controller.dart:131-150`; `app/lib/shared/rich_text/facet_syntax.dart:45-48`; `app/test/scheduled_posts/scheduled_post_edit_test.dart:1435-1452` | Add a create-save-reopen-save round-trip test using the actual trimmed payload and preserve the frozen selection while its normalized source remains in the text. |
| IR-020 | Important | Behavior / Tests | New schedules reuse one editor-lifetime replacement media ID across attach attempts. If thumbnail A is staged, schedule creation fails, and the member changes to thumbnail B, staging B under the same immutable ID returns `scheduled_media_conflict`. IR-014 coverage only protects existing-schedule updates. | FR-016; IR-014; `app/lib/feed/widgets/post_composer_sheet.dart:132,171,1248-1254`; `appview/internal/api/scheduled_media.go:201-204`; `app/test/scheduled_posts/scheduled_post_edit_test.dart:741-829`; `app/test/scheduled_posts/scheduled_post_submission_test.dart:482-540` | Add a new-schedule retry test where selected external content changes after successful staging, and retain an ID only for retries of identical content. Rotate it when replacement content changes. |
| IR-021 | Important | Behavior / Tests | An earlier snackbar's `closed` callback can expire a later dismissal's Undo window. After dismiss, Undo, and a second dismiss while the first snackbar closes, the first callback calls `expireUndo()` without identifying its dismissal generation. | FR-011; `app/lib/feed/widgets/post_composer_sheet.dart:633-654`; `app/lib/feed/composer/link_preview_controller.dart:160-175`; `app/test/scheduled_posts/scheduled_post_submission_test.dart:114-147,291-349` | Associate Undo expiry with a dismissal generation or snackbar instance and add a repeated dismiss/Undo/dismiss race test. |
| IR-022 | Important | Behavior / Tests | External-thumbnail request validation checks only for a non-empty `$link`. A malformed CID passes validation, fails while constructing the generated record, and is returned as HTTP 500 instead of the required client validation error before PDS work. Blob `$type` shape is also not validated. | FR-007; UT-012; `appview/internal/api/post_request.go:225-271`; `appview/internal/postrecord/external.go:12-24`; `appview/internal/api/post_create.go:155-160`; `appview/internal/api/link_preview_request_test.go:44-60` | Validate the generated blob shape, including CID and `$type`, at the request boundary and add handler tests proving malformed blobs return 4xx with zero PDS writes. |
| IR-023 | Important | Behavior / Tests | Scheduled external `sourceUri` is decoded but not validated or normalized. Empty, fragmented, credential-bearing, or noncanonical source identities can be persisted even though this field governs frozen-card identity and disposition. | FR-016; UT-018; `04-coding-plan.md:94-116,185-190`; `appview/internal/api/scheduled_post_request.go:163-190`; `appview/internal/scheduledposts/payload.go:27-33`; `appview/internal/api/scheduled_post_request_test.go:1-139` | Validate `sourceUri` as the normalized, fragmentless composer identity and add rejection and canonicalization tests. |
| IR-024 | Important | Risk / Tests | The IR-017 lifecycle privacy test substitutes static services, bypassing DNS, pinned transport, redirects, page parsing, image fetching, and decode failures. It therefore does not supply AC-022/IT-018 evidence that the real fetch lifecycle has no user-derived persistence or logging sink. | NFR-004; AC-022; IT-018; IR-017; `appview/internal/api/link_preview_privacy_test.go:97-145`; `appview/internal/scheduledposts/privacy_sink_boundary_test.go:10-39` | Add a scripted real `linkpreview.Service` lifecycle privacy test covering DNS, redirect, parse, image, and failure paths, or add an equivalent package-wide sink-dependency boundary guard plus lifecycle canaries. |

## Requirement And Test Traceability
- Requirements implemented: The main authoring, rendering, SSRF, timeout, body-limit, authentication, rate-limit, draft, and scheduled-post paths are implemented. IR-013 explicit disposition, IR-015 fail-closed IPv6 admission, and IR-016 acceptance coverage are resolved.
- Tests implemented: The implementation plan records all original TDD loops and correction steps through Step 74, including focused backend and Flutter suites.
- Unplanned behavior: None identified.
- Remaining gaps: IR-018 through IR-024. IR-012, IR-014, and IR-017 remain partially open through these findings.

## Test Evidence
- Commands reviewed: Deterministic `just lexgen`; `just fmt`; `go vet ./...`; race-enabled `just test`; `just app-analyze`; `just app-test`; focused Flutter review suite; Dart MCP analysis; `git diff --check`.
- Passing evidence: Backend verification passed; Flutter analysis passed; the full Flutter suite passed 1,536 tests; the focused review suite passed 93 tests; generated output matched the recorded deterministic hash; `git diff --check` passed.
- Failing or skipped tests: No existing automated test fails. Required regression tests for IR-018 through IR-024 do not yet exist. `MAN-001`, `MAN-002`, and `MAN-003` remain pending. Clean-tree `just lexgen-check` remains a pre-merge gate.

## Risk Review
- Risk level: High
- Risk notes: Remaining defects affect scheduled-post data integrity, deterministic member intent, API error semantics, immutable staged-media identity, and the required privacy proof for attacker-controlled network input.
- Approval notes: Do not merge or hand off until IR-018 through IR-024 are corrected and the full verification suite passes again.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The implementation is visually coherent enough for behavioral review, but a small preview-fidelity pass may still help after correctness is approved.
- Suggested polish notes: Consider displaying the fetched thumbnail in the composer carousel. Keep MAN-001 device accessibility and responsive-layout smoke testing as a release gate.

## Handoff Back To TDD Builder
- Required fixes: IR-018 through IR-024.
- Suggested next failing test: Start with delayed scheduled hydration after source removal and alternate selection, then add the trimmed create-save-reopen-save round trip. Continue with changed-content media-ID retry, repeated snackbar dismissal, malformed blob handler validation, `sourceUri` validation, and real-pipeline privacy evidence.
- Verification to rerun: Focused tests for each finding; race-enabled `just test`; `just app-analyze`; `just app-test`; deterministic `just lexgen`; `just fmt`; `go vet ./...`; Dart MCP analysis; `git diff --check`. Run clean-tree `just lexgen-check` before merge.
