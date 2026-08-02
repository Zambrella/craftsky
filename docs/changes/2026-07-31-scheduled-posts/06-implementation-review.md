# Implementation Review: Scheduled Posts

## Verdict

Status: Approved with notes

Reviewer: Codex

Date: 2026-08-02

Risk level: High

## Summary

The seventh correction pass closes both remaining Must-level findings. AT-006
now exercises the real Settings page with account-keyed repositories, proves a
non-zero Needs attention badge, proves zero-count omission after an account
switch, and follows the typed Scheduled posts destination. IT-009 now points to
the recovery acceptance test that stops publication at four real process
boundaries, retains the Publishing lease state, reclaims the work through a new
worker, rejects stale completion, and preserves the frozen record identity and
body.

The broader implementation remains aligned with the approved durable AppView
queue, raw-pgx persistence, private S3-compatible staging, owner-authenticated
API, stable PDS publication identity, bounded retries and cleanup, full-composer
editing, account isolation, and no-notification boundary. Fresh focused review
tests passed for Settings, accessibility-adjacent UI, recovery, privacy,
operational signals, and notification isolation. No remaining Must-level
behavior, security, privacy, recovery, or traceability defect was identified.

One non-blocking Should-level note remains: AT-014's single automation target
does not itself drive every accessibility action named by its consolidated
scenario under large text, although neighboring automated tests cover the time
picker, staging announcement, and delete confirmation and MAN-001 remains the
real-device gate. The implementation is ready for handoff; production release
still requires the documented manual and deployed-environment evidence.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-036 | Suggestion | Tests / Accessibility | AT-014's target verifies large-text reachability and semantics for management Edit/Delete, Publishing status, and the Publishing lock. The scenario also names time selection, private-staging progress, editing, and deletion confirmation. Neighboring tests cover the time picker, staging live-region announcement, and delete confirmation separately, and MAN-001 is the approved real-device presentation gate, but the acceptance document does not explicitly map that combined evidence. | NFR-005; AC-024; AT-014; MAN-001; `app/test/scheduled_posts/scheduled_posts_accessibility_test.dart`; `scheduled_staging_progress_test.dart`; `schedule_time_picker_test.dart`; `scheduled_post_delete_test.dart` | Non-blocking: either extend the large-text accessibility fixture to the remaining controls or add a scenario-level mapping naming the neighboring automated tests and the MAN-001 clauses. |

## Requirement And Test Traceability

- Requirements implemented: BR-001 through BR-004, FR-001 through FR-021,
  NFR-001 through NFR-006, and RULE-001 through RULE-008 remain represented in
  the production diff and their approved automated/manual evidence. The feature
  remains limited to top-level standard/project posts, defaults to Now, enforces
  three unpublished schedules transactionally, uses owner-private AppView
  storage, and makes no lexicon or scheduled-notification change.
- Tests implemented: AT-001 through AT-014, UT-001 through UT-020, IT-001
  through IT-027, and REG-001 through REG-010 have implementation targets or
  explicit manual/environmental boundaries. IR-034 and IR-035 are resolved by
  the new Settings test and corrected IT-009 recovery mapping.
- Unplanned behavior: None identified. The AWS SDK/MinIO adapter, shared post
  assembly, raw query store, generated Flutter files, lifecycle composition,
  and observability additions are justified by `04-coding-plan.md` and
  accurately recorded in `05-implementation-plan.md`.
- Remaining gaps: IR-036 is non-blocking because NFR-005 is Should-level and
  the component behaviors plus real-device gate are retained. MAN-001 through
  MAN-006 and GAP-001 through GAP-004 remain explicit pre-release or deployed
  environment work; no automated pass is claimed for them.

## Test Evidence

- Commands reviewed:
  - Fresh Flutter review command covering `settings_page_test.dart`,
    `scheduled_posts_accessibility_test.dart`,
    `scheduled_staging_progress_test.dart`, `scheduled_post_delete_test.dart`,
    and `schedule_time_picker_test.dart`.
  - Fresh Go review command across `internal/scheduledposts` and `internal/api`
    selecting the four-boundary recovery, complete operational lifecycle,
    handler privacy capture, and notification-state isolation tests.
  - The immediately preceding seventh implementation gate: the three-target
    AT-006 Flutter suite, focused real-Postgres IT-009 recovery test,
    `dart analyze`, `just test`, repository-wide `flutter test`, and
    `git diff --check`.
- Passing evidence:
  - The fresh Flutter review command passed seven tests.
  - The fresh Go review command passed both packages against the
    checkout-specific Postgres service.
  - The AT-006 Settings/management/provider gate passed seven tests; the focused
    four-boundary recovery test passed.
  - `dart analyze` passed with `No issues found`; `just test` passed every
    AppView package; `git diff --check` passed.
- Failing or skipped tests:
  - Repository-wide `flutter test` passed 1,213 tests and retained the same
    unrelated pre-existing Instagram migration finder failure at
    `instagram_migration_page_test.dart:256`, where a nested RichText fragment
    is searched as a standalone `Text`. No scheduled-post or Settings test
    failed.
  - MAN-001 through MAN-006 and GAP-001 through GAP-004 were not run and remain
    release/deployment gates, not automated passes.

## Risk Review

- Risk level: High.
- Risk notes: Delayed authenticated PDS writes, private unpublished payloads
  and media, lease recovery, duplicate-publication avoidance, account deletion,
  and cleanup remain consequential boundaries. The implementation now has
  credible production-wired automated evidence for each in-repository boundary.
  Managed storage controls, live PDS/Tap behavior, real-device presentation,
  multi-device/session drills, and deployed alert delivery remain environmental
  risks covered by the explicit manual gates.
- Approval notes: Approved for implementation handoff with IR-036 as a
  non-blocking note. Do not interpret this approval as production-release
  approval until MAN-001 through MAN-006 and the applicable GAP-001 through
  GAP-004 evidence are completed.

## UI Polish Recommendation

- Recommendation: Optional.
- Reason: No blocking visual or accessibility defect was confirmed, and the
  focused scheduling UI tests pass. The feature nevertheless adds substantial
  user-facing surface across two composers, the time picker, staging progress,
  capacity guidance, Settings badge, and management rows.
- Suggested polish notes: Inspect the When row, five-minute/28-day picker edges,
  progress presentation, full-cap guidance, Needs attention expiry copy,
  Settings badge semantics, and large-text wrapping on real iOS and Android
  devices without changing approved behavior.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None required for approval. If the optional
  IR-036 note is selected, start with one large-text semantics test that drives
  the time picker, staging announcement, edit action, and delete confirmation,
  or update the acceptance mapping if the existing neighboring evidence is the
  intended boundary.
- Verification to rerun: No additional automated correction gate is required.
  Before production release, complete MAN-001 through MAN-006 and the applicable
  GAP-001 through GAP-004 evidence; rerun `dart analyze`, `just test`, the
  scheduled Flutter suite, and `git diff --check` after any subsequent change.
