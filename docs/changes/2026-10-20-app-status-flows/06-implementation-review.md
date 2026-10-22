# Implementation Review: App Status Flows

## Verdict

Status: Approved

Reviewer: Codex, implementation review

Date: 2026-10-20

Risk level: Medium

## Summary

Subsequent user amendment, 2026-10-21: D-014 selects plain `Image.network` for a trusted public host. The approval and custom-image transport assessment below describe the earlier reviewed implementation. Image-specific custom transport guarantees/tests are superseded; see updated `01`/`02`/`04` and the D-014 execution/verification record in `05`. D-015 subsequently selected a strict `pub_semver` wrapper with a shared 15-digit numeric-component limit, replacing custom Dart comparison. See its execution/verification record in `05`. D-016 (2026-10-22) subsequently restricts messages to resolved signed-in users while preserving universal blockers and auth-independent fetches; its tests/execution are recorded in `05`. D-017 (2026-10-22) additionally requires completed onboarding for the current active account/session; its regression evidence is recorded in `05`. These annotations do not claim an independent re-review of the subsequent amendments.

The implementation follows the approved independent schema-2 controls, priority, fail-open update trust, retained application host, anonymous media adapters and offline publication gate. IR-001 is resolved: user-action ordering is now checked inside the serialized persistence queue, preventing an older CTA from overwriting a newer successful dismissal. No new required changes were identified.

This re-review covers the correction in the current working tree against the approved `01`–`04` documents, prior review and `05` execution evidence. The broader initial review remains represented by the traceability/risk assessment below. Only this review artifact changed during re-review; no implementation edits, commit, publication or deployment occurred. Approval is for implementation readiness, with the explicitly planned physical-platform and hosted-cache verification still open before release claims.

## Findings

Active findings: None identified.

The initial verdict was Changes required. Stable finding IR-001 is retained below as resolved after the selected TDD correction pass.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 — Resolved | Important (original) | Behavior / Tests | Previously, A's delayed CTA success overwrote replacement B's completed dismissal and resurfaced B. The controller now captures order when each action starts, rejects a superseded write inside the serialized queue, and applies the durable marker/state together only after successful storage. Captured A retains session suppression when its persistence is superseded. Newer failed activation/storage does not incorrectly supersede an older successful save. The original reproduction and new regression matrix pass. | FR-008; AC-012/022; UT-006, AT-008, IT-002; `app/lib/service_status/providers/service_status_controller.dart:263`, `:292`, `:323`, `:340`; new cases in `status_flows_controller_test.dart` and `announcement_persistence_test.dart`; `04-coding-plan.md` §6; `05-implementation-plan.md` IR-001 correction pass | None. Closed on re-review. One durable revision and existing session-only fallback are retained; no historical campaign ledger was added. |

## Requirement And Test Traceability

| Requirements | Implementation and reviewed tests | Assessment |
|---|---|---|
| BR-001, RULE-001 | Independent mapped sections; controller presentation selector; `status_flows_controller_test.dart`; host/app transition suites | Priority, independent removal and non-dismissal on suppression represented. |
| BR-002, FR-002, RULE-002 | `StatusAppVersion`, runtime/platform providers; `app_version_eligibility_test.dart`; literal Dart/Python corpus | Strict precedence, metadata omission, inclusive ranges, platform thresholds and large decimal comparisons represented. |
| FR-001, FR-006, RULE-003 | Existing anonymous repository/lifecycle; epoch/deadline/freshness controller; controller/restart/estimate suites | Bounded checks, maintenance freshness, update invalidation and later recovery represented. Runtime changes notify rather than rebuild the controller. |
| FR-003, NFR-001 | Stable host child; shared blocker predicate; ready root dispatcher and loading/error PopScope; app/editor/recovery/router tests | Cover, retained editor/media/write state and normal account gates represented. Physical back remains a release check. |
| FR-004, FR-005 | Fixed official listings; injectable launcher; conditional `location.reload`; update tests and compiled A/B external runner | Local adapter/reload evidence present. Official store handoff and hosted cache behavior are unverified release checks. |
| FR-007, FR-009 | Schema-2 message model, inclusive eligibility and independent message revision; shared fixtures and persistence/operator tests | Targeting and identity follow the plan; unrelated publication does not intentionally reset dismissal. |
| FR-008 | Controller dismissal/action methods; per-revision pending state; serialized writes; CTA, persistence, native/browser storage tests | IR-001 closed. New tests combine delayed activation with completed/pending replacement dismissal, successful/failed outcomes, fresh adapters/controllers and both blocker types. |
| FR-010 | Closed destination enum, typed route mapping and external launcher; CTA tests and real router/account gate matrix | Normal router acceptance/redirect defines in-app success as approved. Loading/error hosts return retryable unavailable action. |
| FR-011, NFR-002 | Bounded optional image widget; dedicated native/web transports; image, layout, focus and semantics suites | Fail-soft media and local accessibility coverage present. Physical readers remain unverified. |
| FR-012, FR-014 | Atomic Dart decoder; formal Draft 2020-12 schema; Python structural/semantic checks and local-reference preflight; corpus/schema/publication tests | Final encoded payload is validated after root revision replacement. Missing/broken schema/validator paths fail before writes. |
| FR-013 | Hosted CLI and local helper; independent summaries/clear, unchanged message ID, runbook/examples; Python/Worker/local smoke | Complete-document publication, confirmation and verified/unverified outcomes retained. |
| NFR-003 | Dedicated anonymous requests; bounded image bodies/deadlines; diagnostic ownership and serialized sink tests | Credential isolation, protected canaries and original cause/stack handling represented. No authenticated API client used for public media. |

All 22 Must requirements have implementation/test mappings. The identified FR-008 ordering defect is corrected and covered by passing regressions. All 23 acceptance criteria have verification paths, with AC-007/008/019 still carrying the explicitly planned real-platform/hosting portions.

Unplanned behavior: none outstanding; the unintended dismissal-order deviation in IR-001 is resolved. No unrelated API, backend, lexicon, migration, app dependency/version, Xcode project or production infrastructure changes were found. The retained `ServiceStatusMode` is a presentation/test summary; it does not reintroduce legacy wire-mode support. Python validator pins and the dedicated external browser harness were planned additions.

## Test Evidence

### Checks performed during re-review

- `flutter test --no-pub /private/tmp/craftsky-status-review-race_test.dart test/service_status/status_flows_controller_test.dart test/service_status/announcement_persistence_test.dart test/service_status/announcement_dismissal_test.dart test/service_status/message_cta_test.dart` from `app/`: **passed, 21 tests**, including the unchanged original independent reproduction. Output: `/private/tmp/craftsky-status-ir001-rereview.txt`.
- Inspected order allocation, queue-time supersession, successful storage/state application, read-generation protection, session fallback and disposed-controller guards. The write-order guard runs after any newer queued write finishes, covering pending as well as already-completed storage.
- Reviewed eleven new cases: original ordering race; eight explicit/CTA × immediate/pending storage × maintenance/update cases with fresh adapter/controller reads; two newer activation/persistence failures. Tests check visible eligibility and stored revision through public interfaces. Preference storage is mocked for these new restart cases; they do not claim physical durability proof.
- Reviewed correction-stage logs: 137 relevant Flutter tests passed in `/private/tmp/craftsky-status-ir001-tests-direct.txt`; analysis reported no issues in `/private/tmp/craftsky-status-ir001-analyze.txt`. The recipe's stalled Rive download and the corrected overlong comment are disclosed in `05`; equivalent direct Flutter tests completed. These full checks were not redundantly rerun during re-review.
- `git diff --check`: passed. No app dependency/version, Xcode project, API/backend, lexicon, migration or infrastructure change was introduced by the correction.

### Initial review evidence (before the correction)

- `fvm flutter test --no-pub /private/tmp/craftsky-status-review-race_test.dart` from `app/`: **failed as expected for IR-001**, one test. The temporary test uses the production controller with existing repository/store fixtures. After A starts, B is accepted and successfully dismissed; A then completes successfully. Expected no visible message; actual message revision was `B`, and durable revision was `A`. The test disposes its container. No repository source or test was edited.
- `CRAFTSKY_STATUS_PYTHON=/private/tmp/craftsky-status-venv/bin/python just service-status-test`: passed, 17 Python tests and 5 Node tests.
- `git diff --check`: passed.
- Inspected current controller/action/host/runtime/model/version/media code, route integration, schema/operator paths, shared corpus, relevant tests and the external reload spec. Reviewed `01`–`05` for scope, traceability and evidence boundaries.

### Existing implementation-stage evidence reviewed

These results were produced during implementation, not rerun wholesale during this review:

- 126 Flutter regressions passed in `/private/tmp/craftsky-status-final-tests.txt`: status suites, observability, router redirects and composer discard regression.
- `just app-analyze` passed with no issues in `/private/tmp/craftsky-status-analyze-final.txt`.
- Final-source A/B debug builds and all four external Playwright reload/storage cases passed in `/private/tmp/craftsky-status-final-browser.txt`. This covers an actual compiled build transition, still-old gating, unavailable status and browser dismissal across document reload. The spec distinguishes actual document reloads from same-document routing updates.
- `05-implementation-plan.md` records four Chrome transport/image checks, a real iOS simulator preferences/controller journey and an actual local Worker/R2 smoke test. Their source fixtures and claimed scope were inspected; these platform/local smoke checks were not repeated during review.

The initial green suites missed the combined ordering: their delayed-activation test left replacement B undismissed, while their serialization test queued both dismissals before releasing the first write. The new cases now cover A completing after B has been dismissed, including B storage still pending and fresh-controller reads. The original failing reproduction now passes.

### Skipped or remaining evidence

- MAN-001: physical iOS/Android official listing handoff and availability.
- MAN-002: separately authorized Flutter preview hosting/CDN/bootstrap/service-worker recovery. The local server uses no-store and cannot establish deployed cache behavior. Full AC-008 verification remains open; this is the approved release boundary, not an invented implementation-stage deployment requirement.
- MAN-003: physical readers, predictive/platform back and browser history behavior. Automated semantics/layout/focus tests do not prove these.
- Physical process-kill/restart and Android storage were not exercised. The existing simulator fresh-adapter/controller and browser document-reload evidence is narrower.
- The reload adapter preserves the full URL at committed navigation; the existing app path strategy normalizes fragments during startup. The implementation records this boundary rather than claiming final routed fragment retention.

## Risk Review

- Risk level: Medium. This changes a public control contract and app-wide blocking presentation, plus asynchronous persistence and optional remote media.
- Resolved risk: IR-001 is closed. Serializing writes plus a queue-time successful-order guard establishes logical action ordering even when destination futures finish in reverse order. Only successful storage advances the durable order, and superseded CTA persistence retains captured session suppression.
- Privacy/authority: status/media adapters stay separate from AppView credentials; fixed store links and a closed route enum restrict action authority. No PDS/backend authorization changes were introduced.
- Publication: schema/meta-validation, semantic validation and exact final-payload validation precede writes. Hosted confirmation and honest unverified outcomes remain. No publication was performed by this review.
- Rollout: schema 1 is deliberately unsupported by new code; old clients cannot acquire schema-2 enforcement retroactively. The runbook preserves this limitation and separate client/publication coordination.
- Approval notes: no required code corrections remain. The explicit manual release checks remain prerequisites for claims of full platform/hosted behavior, independent of code-review approval. No hosted or physical-device checks were authorized or performed by this re-review.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The implementation uses existing themed controls, bounded scrollable content, localized action/failure text and covered-content exclusion. No additional blocking visual defect was established by this code/test review. Screenshot or physical-reader review was not performed.
- Suggested polish notes: visually inspect maximum-text dialogs and pending update actions on supported devices; consider a consistent progress indicator for mobile store launch. Preserve the approved behavior during any optional polish stage. Accessibility failures against explicit requirements belong in implementation findings, not optional polish.

## Handoff Back To TDD Builder

- Required fixes: None. IR-001 is resolved and the current implementation is approved.
- Suggested next failing test: None required for this correction. Retain the new ordering/restart cases alongside the existing dismissal and blocker regressions.
- Verification to rerun: no further automated rerun required for the unchanged current code. Rerun relevant checks if subsequent edits introduce risk. MAN-001/002/003 and the documented physical durability gaps remain release verification work, not passing results or authorization to deploy.
- Review scope: this re-review updated only `06-implementation-review.md`. No implementation, test, dependency or other workflow document was edited, and no commit was created.
