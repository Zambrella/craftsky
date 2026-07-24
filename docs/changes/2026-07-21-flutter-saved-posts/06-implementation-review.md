# Implementation Review: Flutter Saved Posts

## Verdict

Status: Changes required
Reviewer: Codex implementation re-review
Date: 2026-07-22
Risk level: Medium

## Summary

The correction pass resolves the prior restart-retry, cancellation/error-projection, private-DTO stringification, and most collection-test gaps. The updated implementation evidence is materially more accurate, and the recorded canonical gates are green: focused saved-post/PostCard tests passed, Flutter analysis passed, all 1,002 Flutter tests passed, formatting/vet passed, and the full race-enabled AppView suite passed against PostgreSQL.

Two Must-level collection gaps remain. Move reconciliation updates only the destination resource using the source screen's sort, so an already-mounted destination with the other sort can remain stale. The Saved overview also returns a static empty-state `Center` before constructing its `RefreshIndicator`, so a completely empty collection cannot use the required refresh behavior. The current IT-008 additions do not cover either case.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-006 | Important | Behavior / Collection reconciliation | The move correction constructs exactly one destination key with `sort: sourceKey.sort`. A folder screen and the overview are separate mounted routes and can use different sorts; for example, moving from an Oldest folder screen to Unfiled while the underlying overview has its default Newest resource mounted updates neither that active Newest destination nor its visible order. The new reconciliation test mounts source and destination with the same Newest sort, so it cannot catch this stale-back-stack case. Keep-mode folder deletion already handles both sorts and demonstrates the needed active-resource pattern. | FR-011, FR-018, RULE-005; AC-012, AC-018, AC-031; IT-008; `app/lib/saved_posts/widgets/saved_post_row_actions.dart:58-70`; `app/lib/saved_posts/providers/saved_posts_provider.dart:181-200`; `app/test/saved_posts/pages/saved_post_folder_page_test.dart:143`, `app/test/saved_posts/pages/saved_post_folder_page_test.dart:692-701` | Add a failing public-interface regression with a mounted Oldest source and mounted Newest destination. Reconcile every existing affected destination-sort resource from the exact server-confirmed folder assignment and `savedAt`, without fetching inactive resources or changing server authority. Prove Back reveals the moved item in the correctly sorted destination. |
| IR-007 | Important | Behavior / Tests | When both folders and Unfiled are empty, `_OverviewBody` returns a `Center` before the `RefreshIndicator` and always-scrollable `CustomScrollView` are built. The full empty state therefore cannot pull to refresh even though FR-010 requires refresh for the overview collection and IT-008 explicitly covers overview/folder rendering, refresh, errors, and empty states. The added refresh test exercises only a non-empty folder screen; the overview empty-state test checks copy only. | FR-010, NFR-005; AC-010, AC-027; IT-008; `app/lib/saved_posts/pages/saved_posts_page.dart:142-145`; `app/test/saved_posts/pages/saved_posts_page_test.dart:47`; `app/test/saved_posts/pages/saved_post_folder_page_test.dart:424` | Add a failing overview widget test that starts fully empty, performs pull-to-refresh, and renders newly returned folders or Unfiled posts. Keep the localized full-empty state inside an always-scrollable refresh surface so refresh remains available without weakening the established folders-before-Unfiled hierarchy. |

## Requirement And Test Traceability

- Requirements implemented: The private AppView-backed save/folder feature, account scoping, chooser and mutation flows, typed Settings routes, two-mode folder deletion, exact saved-item navigation, shared `PostSummary`, bounded diagnostics, opaque pagination, and transactional AppView behavior remain represented in code and tests.
- Prior finding status: IR-002 is resolved by explicit page-one restart retry in chooser and overview while retaining confirmed folder entities. IR-003 is resolved by routing bookmark, row, chooser, and folder-mutation presentation through `SavedPostFailure`, including silent `ApiCanceled` cases and retry-policy tests. IR-004 is resolved by disabling private DTO/page mapper stringification and adding DID/URI/content/folder/cursor sentinel coverage. IR-001 is only partially resolved because same-sort destinations and both-sort folder deletion are covered, but cross-sort active move destinations are not. IR-005 is materially improved but remains incomplete because the IT-008 matrix misses IR-006 and IR-007.
- Unplanned behavior: None identified. No dependency, lexicon, PDS-write, notification, public-count, or analytics expansion was introduced.
- Remaining gaps: IR-006 and IR-007, plus the already documented external MAN-001/MAN-002 real-device checks.

## Test Evidence

- Commands reviewed:
  - `cd app && dart run build_runner build --delete-conflicting-outputs`
  - `cd app && flutter gen-l10n`
  - focused saved-post and PostCard regression run
  - `cd app && dart analyze lib/saved_posts test/saved_posts`
  - `just app-analyze`
  - `just app-test`
  - `just fmt`
  - `just test`
  - dependency diff, private sentinel/generated-stringify audit, and `git diff --check`
- Passing evidence: 107 focused tests passed; Flutter analysis reported no issues; all 1,002 Flutter tests passed; Go formatting/vet and the full `-race` AppView suite passed against PostgreSQL without skipped database evidence; dependency and diff audits were clean.
- Failing or skipped tests: No implemented automated test failed. IR-006 and IR-007 are missing cases established by current production control flow and test fixtures. MAN-001/MAN-002 remain pending because no real assistive-technology or narrow native-device session was available.

## Risk Review

- Risk level: Medium
- Risk notes: IR-006 is a client convergence defect rather than server data loss: AppView confirms the move, but Back can expose stale destination content when route sorts differ. IR-007 can strand an initially empty screen without an in-place recovery gesture after external or cross-device changes. Private storage, account boundaries, folder deletion atomicity, PDS isolation, and diagnostic redaction otherwise remain intact in this review.
- Approval notes: Address IR-006 and IR-007 test-first, rerun the focused collection suites and canonical gates, and correct the implementation evidence before merge or handoff. No commit, push, or pull request was created by this review.

## UI Polish Recommendation

- Recommendation: Optional after required changes
- Reason: The user-facing surfaces are coherent enough for implementation review, but a small visual/accessibility pass may still be useful after collection behavior is stable.
- Suggested polish notes: Check native narrow/large-text layouts, empty/error/refresh transitions, destructive hierarchy, folder-name wrapping, focus order, and screen-reader announcements. MAN-001 and MAN-002 remain the authoritative real-platform checks.

## Handoff Back To TDD Builder

- Required fixes: IR-006 and IR-007.
- Suggested next failing test:
  1. Keep an Unfiled Newest overview mounted, open a folder using Oldest, move a row to Unfiled, pop back, and assert the exact server-confirmed item and `savedAt` appear in Newest order without an inactive fetch.
  2. Render a fully empty overview, complete a pull-to-refresh with new folder/post data, and assert the localized empty state is replaced while folder-before-Unfiled ordering remains intact.
- Verification to rerun: focused saved provider/page/widget suites; `dart analyze lib/saved_posts test/saved_posts`; `just app-analyze`; `just app-test`; `just fmt`; `just test`; dependency/privacy/generated-stringify audits; and `git diff --check`. Complete MAN-001/MAN-002 when their external prerequisites are available.
