# Implementation Review: Mutes And Blocks

## Verdict

Status: Changes required
Reviewer: Codex
Date: 2026-07-19
Risk level: High

## Summary

The correction pass closes IR-001 through IR-006 from the first implementation review. Third-party reply, mention, quote, and notification references now enforce blocked-participant policy; unblock reconciles indexed and PDS-only duplicate records; mute persistence and delivery cancellation are atomic; post, notification, and repost actors seed account-owned Flutter state with bounded reconciliation; migration 000023 upgrades the version-22 public-record foreign keys; and production paths emit bounded relationship telemetry. The implementation record reports a passing race-enabled Go suite, 932 passing Flutter tests, clean Flutter analysis, and a clean diff check.

One Must-level thread behavior remains incorrect. Muted-branch ancestry is applied only after each reply page is selected and only within that page. A muted parent at a page boundary, or outside a bounded focused-reply window, does not protect an unmuted descendant returned by a later request. That descendant can therefore appear without the explicit temporary reveal required by the approved contract. This also means the current IT-029 completion claim does not cover the failing thread pagination shape.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-007 | Important | Thread behavior / Pagination | The recursive reply queries propagate block/reference protection before `LIMIT`, but they do not propagate the viewer's mute relationship. Mute collapse is instead applied by `shapeReplyItems` using a fresh in-memory `protected` map for each response page. If a muted parent is the final item on page N, its unmuted descendant can be selected and rendered on page N+1 because the parent is absent from that response. `ListCommentBranchRepliesAround` has the same boundary when a focused descendant is returned without its older muted ancestor. This bypasses the explicit branch reveal and makes protected pagination depend on page shape. The added dense-pagination test exercises only the timeline, so it cannot catch this thread case. | FR-010, NFR-003, AC-018, AC-041, UT-005, IT-029; `appview/internal/api/post_store.go:1258`; `appview/internal/api/post_store.go:1324`; `appview/internal/api/post.go:1132`; `appview/internal/api/relationship_pagination_test.go:15`; `05-implementation-plan.md:376` | Add failing real-Postgres store/handler regressions with a muted parent at the page boundary and with a focused descendant whose muted ancestor falls outside the bounded window. Carry viewer-mute ancestry through the recursive query before pagination: return the muted parent as one revealable placeholder, exclude its descendants from ordinary pages/focus, and preserve the existing explicit direct branch-reveal path. Assert full eligible pages, stable opaque cursors, no descendant leak, and reset/account isolation in Flutter. |

## Requirement And Test Traceability

- Requirements implemented: The schema, routes, canonical PDS block lifecycle, Tap-only projection, membership boundary, read/write policy, third-party reference protection, notification and push suppression, account-keyed Flutter state, safety controls, settings lists, localization, and bounded observability are represented in code.
- Previous findings closed: IR-001 through IR-006 have source and regression evidence matching their required actions.
- Remaining requirement gap: IR-007 leaves FR-010 and NFR-003 incomplete for page-spanning and focused muted reply branches.
- Test traceability gap: `TestDenseRelationshipFilteredTimelineFillsThreeOpaquePages` is good timeline evidence, but the approved IT-029 scenario also needs the affected thread shape. Existing same-page handler shaping does not prove cross-page ancestry.
- Non-blocking evidence note: IT-030's current `EXPLAIN` test proves both block-direction indexes and the mute index for a representative set-based predicate. It is narrower than the approved representative feed/search/thread/list and bounded-call-count matrix, but NFR-004 is a Should and the inspected production relationship reads are set-based rather than per-item.
- Manual gaps: MAN-001 and MAN-002 remain explicitly unavailable; no assistive-technology/device/non-default-locale smoke or compatible-client/local-PDS interoperability smoke was claimed.

## Test Evidence

- Canonical evidence reviewed from the correction pass:
  - repository-root `just test` passed the race-enabled Go suite against the compose PostgreSQL database.
  - repository-root `just app-test` passed 932 Flutter tests.
  - repository-root `just app-analyze` reported no issues.
  - `git diff --check` passed after the correction pass and was rerun successfully during this review.
- Review inspection covered the approved workflow documents, current worktree inventory, all six prior findings and their regressions, migration up/down/up behavior, relationship mutation/index/notification paths, post/thread/reference selection and shaping, Flutter relationship propagation, dense pagination evidence, accessibility/localization evidence, and observability labels.
- No existing automated test failed. IR-007 is an absent cross-page/focused-window case whose behavior follows directly from the recursive SQL omitting mute ancestry and the handler rebuilding page-local protection state.
- The full canonical suites were not rerun a second time during this document-only review because no source changed after the recorded successful gates; only this review artifact was written.

## Risk Review

- Risk level: High
- Risk notes: Mutes and blocks span private state, public PDS records, eventual Tap projection, pagination, thread structure, notifications, push delivery, and multi-account Flutter state. The corrected implementation now protects the major storage, interoperability, migration, notification, and third-party-reference boundaries. IR-007 still lets ordinary pagination or a focused deep link route around a mute branch's explicit-reveal boundary.
- Approval notes: Do not treat implementation as complete until IR-007 is corrected test-first and the focused plus canonical gates pass. No commit, push, or pull request was created by this review.

## UI Polish Recommendation

- Recommendation: Recommended after IR-007 is fixed
- Reason: The feature adds substantial profile, post-menu, confirmation, placeholder, annotation, and Settings UI. A polish pass is worthwhile once muted-branch correctness is stable.
- Suggested polish notes: Check compact and large layouts, destructive-action hierarchy, muted/blocked annotation prominence, placeholder spacing, list busy/error states, text scaling, keyboard focus, and non-default-locale wrapping. MAN-001 remains the final real-device accessibility gate.

## Handoff Back To TDD Builder

- Required fix: IR-007.
- Suggested next failing test: Seed a top-level comment with page-size-one replies where muted Bob is page 1 and unmuted Carol replies to Bob on page 2. Load as Alice and assert page 1 contains one muted placeholder, page 2 does not expose Carol or Bob attribution, and the cursor terminates without a hidden-row leak. Repeat through the focused-reply path with Bob outside the bounded window.
- Verification to rerun: Focused real-Postgres post store/handler tests; Flutter muted-branch reveal/reset/account-isolation tests; repository-root `just test`, `just app-test`, `just app-analyze`, and `git diff --check`. Complete MAN-001 and MAN-002 when their external prerequisites are available.
