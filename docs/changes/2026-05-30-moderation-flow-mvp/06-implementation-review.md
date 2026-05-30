# Implementation Review: Moderation Flow MVP Without Live Ozone/PDS Report Submission

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation reviewer
Date: 2026-05-30
Risk level: High

## Summary
The implementation covers a large part of the planned AppView and Flutter surface, and the focused AppView and Flutter test commands pass. However, several Must requirements are not actually implemented or not protected by tests. The most significant gaps are that the dev synthetic moderation endpoint still returns `501 not_implemented` for valid requests, profile reports submitted by handle fail server-side even though Flutter sends handles, and several post/profile/thread list surfaces still leak hide/takedown-moderated posts or authors.

Because these gaps affect required moderation intake, required synthetic ingestion, and required server-side read enforcement, the change is not ready for merge or handoff as complete.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Critical | Behavior / Tests | `POST /v1/dev/moderation/ozone-events` is registered behind the dev/token gate, but a valid dev-token request returns `501 not_implemented` and never decodes or persists the moderation output. This leaves the required synthetic Ozone-like ingestion seam unusable. Existing tests cover config/token gating plus request/store units, but not the valid route-to-persistence path. | `FR-009` through `FR-012`, `AC-019`, `AC-023`, `AC-036`; `02-acceptance-tests.md` `AT-006`, `IT-017`; `appview/internal/api/moderation.go`; `appview/internal/routes/routes.go`; `appview/internal/app/deps.go` | Wire a `ModerationStore` and moderation request config into deps/routes; update the dev handler to validate one request, enforce trusted/default source DID, insert the output, and return `201 {"outputId":"...","status":"indexed"}`. Add a handler/route test proving a valid fully-enabled request persists exactly one output and invalid/untrusted requests do not mutate state. |
| IR-002 | Critical | Behavior / Traceability | Profile report flow does not work for handles. Flutter submits `/v1/profiles/@{handle}/reports`, but `ProfileStore.ResolveAccountReportTarget` only parses DIDs and returns `ErrProfileNotFound` for handles, and `ReportProfileHandler` is not given a handle resolver. This violates the handle-or-DID profile report contract and the primary profile report UX. | `FR-002`, `FR-006`, `AC-002`, `AC-008`, `AC-044`; `04-coding-plan.md` §5.2; `app/lib/profile/data/profile_api_client.dart`; `app/lib/profile/pages/profile_page.dart`; `appview/internal/api/profile_store.go`; `appview/internal/api/report.go`; `appview/internal/routes/routes.go` | Add handle-or-DID resolution for profile reports before persistence, canonicalize to the target DID, keep the submitted-handle snapshot when applicable, and preserve hidden-but-indexed report eligibility. Add integration tests with a real resolver/store path for reporting by handle, reporting by DID, malformed identifiers, unresolvable profiles, and hidden-but-indexed accounts. |
| IR-003 | Critical | Behavior / Risk | Hide/takedown enforcement is incomplete on required list surfaces. `ListByAuthor` and `ListCommentsByAuthor` do not apply `postVisibleModerationPredicate`, `ListRootComments` does not apply moderation filtering, `ListCommentBranchReplies` / `ListCommentBranchRepliesAround` filter after the recursive page `LIMIT`, and `ReadPostByURI` can return hidden rows for focus/parent hydration. Hidden posts or posts by hidden authors can therefore appear in profile post/comment tabs and thread/comment APIs, and pagination can skip visible rows after hidden rows. | `BR-004`, `FR-013`, `FR-014`, `FR-015`, `NFR-003`, `NFR-004`, `RULE-003`; `AC-012`, `AC-013`, `AC-014`, `AC-040`; `02-acceptance-tests.md` `AT-007`, `IT-010`, `IT-011`, `IT-019`; `appview/internal/api/post_store.go` | Apply hide/takedown predicates before cursor/ordering/limit on all profile-authored and thread/comment list queries. Ensure focus/parent URI hydration cannot reintroduce hidden/taken-down rows into responses. Add failing tests for profile posts, profile comments, root comments, branch replies, focus-on-hidden targets, account-level hide, and pagination where hidden rows precede visible rows. |
| IR-004 | Important | Tests / Traceability | The implementation plan marks all Must tests complete, but the current automated suite does not cover the gaps above: no valid synthetic route ingestion test, no profile-report-by-handle server integration test, and no moderation enforcement tests for profile authored lists or root/branch comment lists. The passing test evidence is therefore insufficient for the Must acceptance criteria. | `05-implementation-plan.md` Steps 8, 10, 17, 18; `02-acceptance-tests.md` `AT-002`, `AT-006`, `AT-007`, `IT-010`, `IT-011`, `IT-017`, `IT-019` | Extend the red phase before fixing each behavior gap. Update `05-implementation-plan.md` only if needed to accurately record the additional TDD loops and final evidence. |

## Requirement And Test Traceability
- Requirements implemented: Private report persistence, report request validation/detail normalization, placeholder forwarding metadata without PDS submission, report response privacy, report route middleware, dev moderation config/token gating, moderation output store/policy units, timeline/direct post/direct profile/notification hide filtering, warning metadata, Flutter report entry points/providers/sheet, and generic warning rendering are partially or fully implemented.
- Tests implemented: Focused AppView API/routes/app tests and focused Flutter feed/profile/moderation/notifications tests are present and passing for the covered paths.
- Unplanned behavior: None identified as an intentional product expansion; the main issue is incomplete planned behavior.
- Remaining gaps: Valid synthetic route ingestion, profile reports by handle, full read-path hide/takedown enforcement for profile-authored and thread/comment list surfaces, and related regression tests.

## Test Evidence
- Commands reviewed:
  - `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes ./internal/app` from `appview/`
  - `flutter test test/feed test/profile test/moderation test/notifications` from `app/`
  - Recorded implementation evidence in `05-implementation-plan.md`
- Passing evidence:
  - AppView focused command passed.
  - Flutter focused command passed (`All tests passed!`).
- Failing or skipped tests:
  - No current automated test fails, but coverage is missing for the blocking behavior gaps above.
  - Manual checks `MAN-001` through `MAN-004` were documented as not run; rerun or complete them after blocking fixes.

## Risk Review
- Risk level: High
- Risk notes: This change affects private safety data, dev-only mutation controls, and server-side content suppression. The incomplete synthetic endpoint prevents local/dev moderation-output ingestion, and incomplete read-path filtering can leak hidden content through existing product APIs.
- Approval notes: Do not approve until the missing Must behavior and tests are addressed. The existing privacy boundaries for report response bodies and warning metadata look directionally correct, but they need to be re-verified after the required fixes.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The implementation includes user-facing report UI and warning banners. The blocking findings are behavioral/server-side, not polish-level UI defects.
- Suggested polish notes: After the required fixes pass, a small polish pass could review the report sheet layout, accessibility semantics, in-flight state affordance, and warning banner visual treatment. Do not run polish before the blocking behavior issues are fixed unless the user explicitly chooses to do so.

## Handoff Back To TDD Builder
- Required fixes:
  1. Complete valid dev synthetic moderation ingestion through the registered route.
  2. Fix profile report handle-or-DID resolution and persistence snapshots.
  3. Complete hide/takedown filtering before pagination for profile-authored and thread/comment read surfaces, including focus/parent hydration paths.
  4. Add regression tests that fail before each fix and pass after.
- Suggested next failing test: Start with `IT-017`/`AT-006` route-level coverage proving a valid `POST /v1/dev/moderation/ozone-events` persists one output and returns `201 indexed`.
- Verification to rerun:
  - From `appview/`: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app`
  - From repo root: `just test`
  - From `app/`: `flutter test test/feed test/profile test/moderation test/notifications`
  - Complete/manual-review `MAN-001` through `MAN-004` after automated fixes pass.
