# TDD Implementation Plan: Service Status and Maintenance Messaging

## Inputs

- Requirements: `01-requirements.md`.
- Acceptance tests: `02-acceptance-tests.md`.
- Document review: `03-document-review.md`, Approved with notes, Medium risk.
- Coding plan: `04-coding-plan.md`, including DR-001–DR-003 resolutions.
- User selected `implement-tdd` on 2026-10-09. Current worktree is retained.

## Implementation Rules

- Implement only behavior linked to approved requirement IDs.
- Write one focused failing behavior test, confirm a meaningful failure, implement the minimum, then rerun that test.
- Refactor only while green. Keep existing account, router, editor and request behavior intact.
- Record red/green commands, outcomes and deviations below before advancing.
- No commit, push, provision, live publication or deployment is enabled.
- Hosted/device/manual evidence must be recorded separately from mocked/local evidence.
- Follow the invoked skill's explicit security/privacy approval gate before source changes in those areas.

## Live Progress

| Item | Status | Notes |
|---|---|---|
| Read workflow documents | completed | Loaded requirements, tests, review and coding plan from this folder. |
| Create implementation plan | completed | Mirrors all 28 case families from coding-plan section 9. |
| Inspect relevant code and tests | completed | Inspected root initialization, account/router/editor boundaries, anonymous transports, observability sinks and operator/website tooling. |
| Security/privacy approval gate | completed | User explicitly approved the described local scope on 2026-10-09. |
| Final verification | completed | Full suite, final focused suite, analysis, browser adapter, mobile journeys, Python/Node, website and workflow checks passed; hosted/manual release gaps remain explicit. |
| Update/read back implementation notes | completed | Execution outcomes and remaining release gates recorded and read back. |

## Test Order

The table records completed local loops and explicit release-evidence gaps. Detailed red/green outcomes are in the execution evidence below. Existing protected-behavior regression cases may pass without product changes.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Status | Target |
|---|---|---|---|---|---|
| 1 | UT-001 | FR-002, FR-003 | AC-003, AC-004, AC-005 | completed | `app/test/service_status/service_status_document_test.dart` |
| 2 | UT-002 | FR-002, FR-003, NFR-001, RULE-003 | AC-005, AC-019 | completed | Same Dart suite and `scripts/test_service_status_publish.py` validator group |
| 3 | UT-003 | FR-004, NFR-001 | AC-006 | completed | `app/test/service_status/service_status_controller_test.dart`; startup wiring in `app_test.dart` |
| 4 | UT-004 | FR-003, FR-004, FR-008, NFR-001, RULE-001 | AC-005, AC-007, AC-011 | completed | Same controller suite |
| 5 | IT-001 | FR-001, RULE-002, RULE-003 | AC-001, AC-019 | completed | `app/test/service_status/service_status_http_test.dart`; mobile socket smoke in `critical_journeys_test.dart` |
| 6 | AT-001 | BR-001, FR-001, FR-005 | AC-001 | completed | `app/test/service_status/service_status_app_test.dart` |
| 7 | AT-002 | BR-001, FR-002, FR-005, NFR-003, RULE-001, RULE-003 | AC-003, AC-017, AC-019 | completed | `app/test/service_status/service_status_widgets_test.dart` |
| 8 | AT-005 | FR-007, NFR-001, RULE-001 | AC-010 | completed | Same app suite and existing error fixtures |
| 9 | UT-005 | FR-006 | AC-004, AC-009 | completed | `app/test/service_status/announcement_dismissal_test.dart` |
| 10 | IT-002 | FR-006, FR-008 | AC-009, AC-011 | completed | `app/test/service_status/announcement_persistence_test.dart`; `critical_journeys_test.dart` |
| 11 | AT-003 | BR-001, FR-002, FR-006 | AC-004 | completed | `service_status_app_test.dart` |
| 12 | AT-007 | FR-006, FR-007, FR-008, NFR-001 | AC-009, AC-010, AC-011 | completed | `app/test/service_status/service_status_restart_test.dart` |
| 13 | AT-004 | FR-005, FR-009, RULE-004 | AC-008, AC-012 | completed | `app/test/service_status/service_status_editor_test.dart` |
| 14 | AT-006 | FR-004, FR-008, FR-009, RULE-004 | AC-007, AC-011, AC-012 | completed | `app/test/service_status/service_status_recovery_test.dart` |
| 15 | REG-001 | FR-007, RULE-004 | AC-010, AC-012 | completed | Existing sign-out interceptor/error mapper tests |
| 16 | REG-002 | FR-009, RULE-004 | AC-012 | completed | Existing account-switch/OAuth handoff/restricted routing/gate suites |
| 17 | REG-003 | FR-009, RULE-004 | AC-008, AC-012 | completed | Existing composer/video/draft suites and mobile critical journey |
| 18 | UT-006 | FR-002, NFR-003, RULE-001 | AC-003, AC-017 | completed | `app/test/service_status/service_status_estimate_test.dart` |
| 19 | UT-007 | FR-002, RULE-003 | AC-005, AC-019 | completed | Document/widgets suites |
| 20 | AT-009 | NFR-003 | AC-017 | completed | `app/test/service_status/service_status_accessibility_test.dart` |
| 21 | IT-006 | NFR-004, RULE-002 | AC-018, AC-019 | completed | `app/test/observability/service_status_diagnostics_test.dart`; Python/Node output checks |
| 22 | AT-008 | BR-002, FR-001, FR-012 | AC-002, AC-015 | completed | `scripts/test_service_status_publish.py` command workflow |
| 23 | IT-003 | BR-002, FR-003, FR-012, RULE-002, FR-006 | AC-002, AC-005, AC-015, AC-019, AC-009 | completed | Same Python suite |
| 24 | IT-004 | BR-002, FR-012, NFR-002 | AC-002, AC-015, AC-016 | completed | Same Python suite |
| 25 | IT-005 | BR-002, FR-001, FR-012 | AC-002, AC-015 | completed | Python command/website suites and Node Worker tests |
| 26 | REG-004 | FR-001, FR-012 | AC-015 | completed | Existing `test_web_deploy.py`, `test_cloudflare_pages_build.py` |
| 27 | IT-007 | FR-001, FR-012, NFR-002, RULE-002 | AC-001, AC-015, AC-016, AC-019 | completed | `service-status/test/worker.test.js`, manual hosted checks in the runbook (local complete; hosted execution remains pending setup/authorization) |
| 28 | MAN-001 | NFR-003 | AC-017 | cancelled | Manual recorded evidence |

## Implementation Steps

### Step 1: UT-001

- Status: completed.
- Write failing test: Minimal literal schema 1 maintenance, valid token revision, no publication time; shared corpus.
- Target: `app/test/service_status/service_status_document_test.dart`.
- Expected meaningful failure: Missing DTO/decoder or unsupported valid contract.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 2: UT-002

- Status: completed.
- Write failing test: Literal 16,384/16,385-byte bodies, 160/161 title and 2,400/2,401 message scalar boundaries, impossible zoned timestamps, streaming oversize.
- Target: Same Dart suite and `scripts/test_service_status_publish.py` validator group.
- Expected meaningful failure: Unsafe/mismatched acceptance or unbounded decode.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 3: UT-003

- Status: completed.
- Write failing test: Fake clock/repository and delayed dependency init; all modes and lifecycle triggers.
- Target: `app/test/service_status/service_status_controller_test.dart`; startup wiring in `app_test.dart`.
- Expected meaningful failure: No polling/coalescing/deadline or blocked startup.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 4: UT-004

- Status: completed.
- Write failing test: Exact freshness boundaries/renewal, suspension/resume, wall rollback and cancelled late result.
- Target: Same controller suite.
- Expected meaningful failure: Indefinite cover or superseded acceptance.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 5: IT-001

- Status: completed.
- Write failing test: Separate status/AppView servers, captured synthetic auth/device values, redirects; web adapter mocked fetch + hosted checks.
- Target: `app/test/service_status/service_status_http_test.dart`; mobile socket smoke in `critical_journeys_test.dart`.
- Expected meaningful failure: API headers reused, redirects followed or endpoint depends on account.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 6: AT-001

- Status: completed.
- Write failing test: Signed-in/out and failed/pending dependencies/account; splash callback recorder.
- Target: `app/test/service_status/service_status_app_test.dart`.
- Expected meaningful failure: No custom cover before initialization or splash remains.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 7: AT-002

- Status: completed.
- Write failing test: Plain/malicious prose, estimates, retry recorder and internal back event.
- Target: `app/test/service_status/service_status_widgets_test.dart`.
- Expected meaningful failure: Wrong content/actions, hidden screen interaction or missing retry.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 8: AT-005

- Status: completed.
- Write failing test: Status unavailable, API 503, HTML/non-200, announcement/health probe.
- Target: Same app suite and existing error fixtures.
- Expected meaningful failure: Global maintenance inferred or usable UI blocked.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 9: UT-005

- Status: completed.
- Write failing test: Literal revisions A/B, delayed dismissal completion with newer status.
- Target: `app/test/service_status/announcement_dismissal_test.dart`.
- Expected meaningful failure: Same revision resurfaces or new revision suppressed.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 10: IT-002

- Status: completed.
- Write failing test: Mocked adapter for fast tests; real preferences write/reload, fresh instances for device evidence.
- Target: `app/test/service_status/announcement_persistence_test.dart`; `critical_journeys_test.dart`.
- Expected meaningful failure: Dismissal lost, plugin-only cache masks missing storage, or maintenance reused.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 11: AT-003

- Status: completed.
- Write failing test: Current announcement, usable navigation/operations, normal/maintenance replacement.
- Target: `service_status_app_test.dart`.
- Expected meaningful failure: Notice blocks app or simultaneous modes visible.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 12: AT-007

- Status: completed.
- Write failing test: Shared only preference storage across recreated feature/app instances; offline restart.
- Target: `app/test/service_status/service_status_restart_test.dart`.
- Expected meaningful failure: Same-revision flash, missing new fetch or mandatory restored cache.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 13: AT-004

- Status: completed.
- Write failing test: Existing post/project editor fixtures with text/media and delayed operation.
- Target: `app/test/service_status/service_status_editor_test.dart`.
- Expected meaningful failure: Editor unmounted, navigation lost or writes replayed.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 14: AT-006

- Status: completed.
- Write failing test: Inject normal/announcement/expiry and current-account gates.
- Target: `app/test/service_status/service_status_recovery_test.dart`.
- Expected meaningful failure: Relaunch required, wrong-account UI or policy bypass.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 15: REG-001

- Status: completed.
- Write failing test: Genuine 401 control, 503/offline/status outcomes.
- Target: Existing sign-out interceptor/error mapper tests.
- Expected meaningful failure: Session invalidated by status failure.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 16: REG-002

- Status: completed.
- Write failing test: Delayed old-account completion and handoff beneath cover.
- Target: Existing account-switch/OAuth handoff/restricted routing/gate suites.
- Expected meaningful failure: Current-account fencing or eligibility bypassed.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 17: REG-003

- Status: completed.
- Write failing test: Original operation counts/outcomes with maintenance transition.
- Target: Existing composer/video/draft suites and mobile critical journey.
- Expected meaningful failure: Existing completion cleanup changes or extra submission.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 18: UT-006

- Status: completed.
- Write failing test: Explicit offset instants/DST zone test context, absent/past estimate.
- Target: `app/test/service_status/service_status_estimate_test.dart`.
- Expected meaningful failure: Incorrect local display or automatic mode transition.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 19: UT-007

- Status: completed.
- Write failing test: Markup, URL prose and unsupported control fields.
- Target: Document/widgets suites.
- Expected meaningful failure: Code/link/control behavior enabled.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 20: AT-009

- Status: completed.
- Write failing test: Existing English locale/device fallback, light/dark, supported small/web sizes and maximum text scaling.
- Target: `app/test/service_status/service_status_accessibility_test.dart`.
- Expected meaningful failure: Overflow, inaccessible retry/dismissal, exposed covered semantics.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 21: IT-006

- Status: completed.
- Write failing test: Serialized transport, debug ProviderLogger, original cause/stack and prose/secret canaries.
- Target: `app/test/observability/service_status_diagnostics_test.dart`; Python/Node output checks.
- Expected meaningful failure: Leak, missing positive evidence, duplicate issues or reporter affects behavior.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 22: AT-008

- Status: completed.
- Write failing test: Injectable R2/public verifier/confirmation; Render unavailable; no Git/site release.
- Target: `scripts/test_service_status_publish.py` command workflow.
- Expected meaningful failure: Independent validate/revise/clear workflow unavailable.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 23: IT-003

- Status: completed.
- Write failing test: Invalid/declined/EOF inputs, target discovery, fixed temporary payload; reused draft revision.
- Target: Same Python suite.
- Expected meaningful failure: Mutation before confirmation; unchanged revision used for new announcement; credentials exposed.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 24: IT-004

- Status: completed.
- Write failing test: Exact/stale/unreachable/wrong-type public responses and concurrent newer revision.
- Target: Same Python suite.
- Expected meaningful failure: False verified result, unlimited wait, write replay or rollback.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 25: IT-005

- Status: completed.
- Write failing test: Separate fake stores/resources and two website code versions.
- Target: Python command/website suites and Node Worker tests.
- Expected meaningful failure: Status overwritten by unrelated release or Worker deploy mutates object.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 26: REG-004

- Status: completed.
- Write failing test: Status resource separation configuration; existing website allowlist/promotion fixtures.
- Target: Existing `test_web_deploy.py`, `test_cloudflare_pages_build.py`.
- Expected meaningful failure: Status bundled or website release behavior broken.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 27: IT-007

- Status: completed.
- Write failing test: Local fake R2 for handler assertions, then separately authorized preview R2/Worker and real browser/native reads.
- Target: `service-status/test/worker.test.js`, `hosted-status.spec.cjs`, `scripts/service-status-check`.
- Expected meaningful failure: Wrong CORS/cache/type or stale next refresh; mock pass cannot satisfy hosted evidence.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

### Step 28: MAN-001

- Status: cancelled.
- Write failing test: Supported Android/iOS/web screen readers, long text/large scaling, underlying editor.
- Target: Manual recorded evidence.
- Expected meaningful failure: Reading/focus/accessibility fails despite widget semantics.
- Red/initial outcome: see the execution evidence for this test ID below.
- Implementation and green command: see the execution evidence for this test ID below.
- Notes: preserve requirement and criterion links in the test-order table.

## Approval Boundary

The user-selected skill states: "Stop for explicit approval before touching high-risk areas: auth, permissions, billing, payments, migrations, destructive actions, privacy, security, or compliance."

The coding plan includes security/privacy boundaries: strict remote-data validation (UT-001/UT-002), a dedicated anonymous native/web transport and redirect policy (IT-001), credential-safe operator tooling (IT-003), privacy-bounded diagnostics (IT-006), and retained presentation around existing account/auth gates (REG-001–REG-003). The user explicitly approved this concrete local scope on 2026-10-09.

Requested approval covers implementing and testing the approved coding plan locally, including those boundaries. Existing auth/permission policy remains the protected behavior in regression tests. It does not include Cloudflare/Render resource changes, DNS, deployment, live publication, production data access, commits or pushes.

## Evidence And Remaining Dependencies

- Repository was clean at startup (`git status --short` returned no changes).
- Flutter executable is available at `/Users/douglastodd/fvm/default/bin/flutter`.
- First planned command: `just app-test test/service_status/service_status_document_test.dart`, after writing UT-001.
- Local implementation and test execution are recorded below; no live status or infrastructure mutation was performed.
- CPQ-001/GAP-002: hosted resource/access/cache evidence remains pending, requiring separate authorization.
- CPQ-004: real iOS simulator preferences write/reload and fresh-instance evidence passed (IT-002); this is not an OS process-kill claim.
- CPQ-006/GAP-004: real screen-reader checks remain pending; widget semantics cannot satisfy MAN-001.
- GAP-005: retain existing external OAuth/media harness limitations.

## Completion Checklist

- [x] All Must requirements covered by passing tests or documented gaps.
- [x] All selected local automated Must tests passing.
- [ ] Hosted IT-007 read/cache/next-publication evidence (separate resources/authorization required).
- [ ] MAN-001 real screen-reader/platform-back evidence.
- [x] Relevant regression tests passing.
- [x] No unlinked behavior implemented.
- [x] Operational docs updated.
- [x] Final verification and diff review completed.
- [x] Implementation plan updated and read back.
- [x] Implementation review explicitly deferred to the next user-selected stage.

Local implementation is complete. Hosted and real screen-reader release gates remain open. Implementation review is the next optional workflow stage; no commit, push or deployment has been performed.

### Execution evidence: UT-001

Red: focused Flutter test failed with UnimplementedError after establishing the missing decoder interface. Green: just app-test --no-pub test/service_status/service_status_document_test.dart, 2 tests passed. Initial test-fixture/import setup mistakes were corrected before claiming green. Decoder supports all modes and zoned estimates; strict validation follows UT-002.

### Execution evidence: UT-002

Starting bounded validation and shared Dart/Python contract corpus. Streaming transport boundary will be executed under IT-001 after the bounded reader is introduced here.

### Execution evidence: UT-002

Red: unsupported schema and oversized scalar/body fixtures were incorrectly accepted; unbounded streaming reader returned 16385 bytes. Green: 6 Dart contract/body tests and 2 Python corpus/boundary tests passed. Literal shared corpus pins types, modes, revisions, zoned timestamps and surrogate rejection. Flutter required SDK-cache access outside the sandbox; approved local test execution succeeded. No live calls.

### Execution evidence: UT-003

Starting the single controller polling/coalescing/deadline suite. App startup wiring is verified when the lifecycle host is integrated under AT-001, keeping source changes vertical.

### Execution evidence: UT-003

Red: controller/repository interfaces were missing. Green: 4 focused controlled-time tests passed for all-mode 60-second polling, shared refresh future, whole three-second timeout/cancellation, foreground resume, background suspension and disposal. Status startup is non-awaiting. Root wiring evidence remains cross-linked to AT-001. Narrow plan adjustment: explicit Riverpod providers match existing hand-written Notifier tests and avoid generated indirection for this small boundary; no generator required.

### Execution evidence: UT-004

Starting exact freshness, renewal, background age and late-result checks with an injectable monotonic/wall clock.

### Execution evidence: UT-004

Red: maintenance stayed active at five minutes; rollback followed by forward correction revived trust. Green: 8 controller tests passed, including expiry, invalid non-renewal, unchanged renewal, immediate normal/announcement clearing, timeout epoch protection, wall-only suspension age and irreversible invalidation until a successful fetch. No real-time sleeps.

### Execution evidence: IT-001

Starting actual outbound anonymous native HTTP and redirect tests; then conditional browser Fetch boundary. Mobile/hosted evidence remains separate.

### Execution evidence: IT-001

Red: anonymous transport/browser adapter and endpoint validator were absent. Green: 3 actual loopback native tests pass (no protected headers/query/body, redirect destination untouched, invalid content rejected, hung body cancellation) and 1 Chrome Fetch adapter unit test passes (credentials omit, redirect error, no-store, no-referrer). Conditional browser-only test uses an explicit .browser.dart filename so native discovery does not compile JS-only imports. Hosted CORS/cache and mobile socket corroboration are still pending separate evidence.

### Execution evidence: AT-001

Starting root loading/error maintenance and splash removal before dependency/account readiness, using independent status repository overrides.

### Execution evidence: AT-001

Red: App did not start independent status retrieval. Green: 4 signed-in/out pending/failed initialization cases pass and all 11 existing startup tests pass. Fresh maintenance removes splash once without marking account initialization complete; normal reveals retained loading/error state. Existing app tests now override the added network dependency with normal status; production polling is unchanged.

### Execution evidence: AT-002

Starting branded localized maintenance, literal prose/estimate/retry, covered interaction and root back tests.

### Execution evidence: AT-002

Red: localized retry/estimate were absent; root back adapter was absent. Green: 2 presentation tests pass for plain markup/URL prose, no dismissal/link, blocked underlying tap, status-only retry, informational estimate, retained pushed editor on back and normal back behavior after clear. GoRouter does not reflect imperative pushes in the URL by default, so the test uses visible retained route and public canPop behavior rather than changing router settings. Predictive-back/device evidence remains pending.

### Execution evidence: AT-005

Adding unknown status/API failure fallback cases. Existing error taxonomy is retained; tests may already pass because no inference path was introduced.

### Execution evidence: AT-005

Initial fallback cases exposed fixture timing: existing Riverpod retries ApiServerError before terminal error presentation. Disable provider retry in this terminal-state fixture; preserve production retry policy. Actual green verification is recorded below. Normal/announcement replacements are covered by UT-004 and AT-003.

### Execution evidence: UT-005

Starting revision-scoped dismissal, readiness and completion races through the controller boundary.

### Execution evidence: AT-005

Actual green verification after correcting terminal-state fixtures: 11 combined startup/error taxonomy tests passed. Only test ProviderScope disables automatic retry; production retry is unchanged.

### Execution evidence: UT-005

Red: dismissal storage/controller interfaces were absent; later expiry notification reset dismissal readiness. Green: revision A stays dismissed, B/C resurfacing and delayed B write cannot dismiss C, maintenance cannot be dismissed, expiry preserves independent dismissal state. The focused controller dismissal test passes; real preferences adapter follows IT-002.

### Execution evidence: IT-002

Starting preference adapter and real-storage reload/fresh-controller integration. Available iPhone simulator can run the mobile harness; no physical device is targeted.

### Execution evidence: IT-002

Red: preferences implementation was absent. Green: 2 fast preference/dismissal tests passed, with awaited setString, reload and fresh store. Mobile corroboration is running separately on the available iPhone simulator; Xcode dependency preparation is still in progress, so actual persistence evidence is not yet claimed. A dedicated loopback status origin was added to the existing device harness; no production status request is made.

### Execution evidence: AT-003

Starting one non-blocking announcement and replacement presentation. Device evidence may finish independently; it is not a substitute for this widget loop.

### Execution evidence: AT-003

Red: accepted announcements had no UI. Green: 3 widget tests pass, including one notice, usable underlying controls, dismissal, new revision resurfacing and immediate normal/maintenance replacement. The banner occupies a stable bounded scrolling slot; retained content keeps the same tree position.

### Execution evidence: AT-007

Adding fresh-feature restart tests sharing only dismissal storage, with a new fetch and no maintenance restoration during an offline restart.

### Execution evidence: IT-002

Device corroboration now passed: just app-test-integration 022E9DC5-01AE-4399-879D-DDCB0CF9A864 --no-pub --plain-name=status. iPhone 18 Pro iOS 27 simulator: awaited real preferences write/reload, disposed feature state, fresh controller reads A dismissed, B resurfaces, maintenance blocks, actual anonymous HTTP headers verified. Xcode build and 1 device test passed. This is fresh-instance/storage evidence, not an OS process-kill claim.

### Execution evidence: AT-007

Restart regression passes with fresh containers/repositories sharing only dismissal storage. Every instance starts a new fetch; offline restart has no restored maintenance; A stays dismissed, B appears, maintenance unaffected. Fixture initially lacked CraftSky theme extensions required by ChunkyButton; corrected fixture, no product workaround.

### Execution evidence: AT-004

Starting real post/project composer text/media element retention under the status host. Existing submission coordinator is exercised with delayed outcomes and original operation count; detailed API/video/draft regression suites follow REG-003.

### Execution evidence: AT-004

New retention tests passed on first run: actual PostComposerSheet and ProjectComposerSheet keep TextField element/controller and uploaded image state through maintenance and clearing. Existing ComposerSubmissionCoordinator completes one delayed operation/cleanup under the cover, with no replay. No product refactor needed. Detailed actual publication/failure/cancellation cases remain REG-003.

### Execution evidence: AT-006

Starting normal/announcement/expiry transitions around the real ActiveAccountInitializationGate; account fencing/policy/OAuth suites follow REG-002.

### Execution evidence: AT-006

3 recovery cases passed: normal, announcement and injected freshness expiry remove the cover without relaunch; real ActiveAccountInitializationGate keeps current Bob content unmounted until its exact lease is initialized. Registry/session unchanged, no recovery claim, no status-driven operation replay. Fixture import ambiguity corrected before execution; no production gate changes.

### Execution evidence: REG-001

Adding status failures beside true-401 control and existing 503/offline error taxonomy.

### Execution evidence: REG-001

8 interceptor/error taxonomy tests pass, including new status maintenance/offline/invalid/503 results with zero session invalidation and a genuine 401 control that invalidates the exact captured lease. Existing source behavior is unchanged.

### Execution evidence: REG-002

Adding delayed old-account initialization under the cover, then running existing account-switch, restricted-route, OAuth handoff and account-gate suites.

### Execution evidence: REG-002

13 combined recovery, gate, account activation, OAuth route and restricted-route tests pass. New real-gate case switches Alice to Bob under maintenance, rejects late Alice initialization, clears into loading, and reveals content only after exact Bob initialization. Auth/eligibility source remains unchanged; external disposable-PDS OAuth is still the documented harness limitation.

### Execution evidence: REG-003

Extending the real asynchronous draft save fixture under a cover; running existing composer/project/video cancellation and ownership suites. Mobile actual post submission corroboration will use the existing loopback backend.

### Execution evidence: REG-003

32 focused post/project/video/draft tests pass. Existing actual draft-save test now enters maintenance while save is pending; completion and original success feedback proceed under cover with exactly one save, then normal restores host. Existing failure/cancellation/retry/account-switch outcomes are unchanged. iPhone simulator corroboration of one accepted actual HTTP post under maintenance is running separately; no device pass claimed yet.

### Execution evidence: UT-006

Starting explicit-offset/local-time and DST formatting assertions; old estimates remain informational while newly fetched maintenance is fresh.

### Execution evidence: UT-006

Focused estimate tests pass: explicit offsets and London daylight saving transition; past estimate never clears maintenance. Intl CLDR narrow nonbreaking spaces retained.

### Execution evidence: UT-007

Verify literal markup and ignored remote control fields through decoding and presentation.

### Execution evidence: UT-007

Nine decoder/widget tests pass; malicious prose remains literal and unsupported schedules, API destinations and feature controls are ignored. Existing renderer has no link action.

### Execution evidence: AT-009

Exercise retained editor focus plus long text at 320/1280 widths, both themes and maximum supported 1.5 text scale.

### Execution evidence: AT-009

Eleven accessibility/editor tests pass. Red revealed focus restoration and announcement semantics suppressed by Navigator BlockSemantics. Host retains focus and orders banner after Navigator in semantics while positioning it above. Both themes/320 and1280 widths/1.5 scale long text reachable. Real readers remain MAN-001.

### Execution evidence: REG-003

32 local regression tests and real iPhone18Pro iOS27 composer HTTP journey pass. Device fixture must dismiss keyboard before submitting; accepted write completes once under maintenance and appears after clear.

### Execution evidence: IT-006

Verify expected polling has no issue spam, one unexpected consumed failure preserves original cause/stack, serialized sinks redact canaries, reporter failures do not affect UI.

### Execution evidence: IT-006

Nine diagnostics/controller tests pass. SDK serialized issue observed before close; one unexpected failure retains identical original cause/stack; six expected failures produce no issues. Provider debug/state outputs exclude prose/revision canaries. Guarded reporter failure leaves maintenance/retry state intact. Python/Worker output checks follow publishing loops.

### Execution evidence: AT-008

Implement validate/publish/clear via injectable storage/verifier/confirmation with fresh revisions and independent fixed targets.

### Execution evidence: AT-008

Three Python tests pass: validate and independently publish reused draft twice/clear with fresh unique revisions, confirmed exact public content, no input rewrite or Git/website/Render command. Live adapters implemented next.

### Execution evidence: IT-003

Test no mutation for invalid/declined/EOF inputs; exact account/bucket discovery and frozen temp payload with sanitized subprocess output.

### Execution evidence: IT-003

Seven Python tests pass: invalid/declined/EOF input never mutates, explicit environment required, exact authorized account/bucket discovered, immutable 0400 temp payload removed after one upload, credentials never echoed on setup/upload failure. Pinned Wrangler help/source confirms flags and JSON discovery shape.

### Execution evidence: IT-004

Require anonymous fixed-origin public verification with headers/body/revision equality and finite total per-attempt deadlines; uncertain upload remains unverified with no write replay.

### Execution evidence: IT-004

Ten Python tests pass: exact recognized-field/revision comparison, required public JSON/no-store/CORS/nosniff headers, no redirects/cookies, bounded 16385-byte read, three hard three-second child-process deadlines including DNS/TLS/slow body, two one-second waits, no rollback/write retry on uncertainty.

### Execution evidence: IT-005

Prove dedicated production/preview resource configuration is independent of website assets and message updates invoke only fixed R2 object commands.

### Execution evidence: IT-005

Eleven Python tests pass: dedicated fixed production/preview Worker names, domains, private R2 bindings, no assets/shared landing bindings. Publisher adapter exclusively writes selected bucket/app.json. No live resources provisioned.

### Execution evidence: REG-004

Protect unchanged website build/deploy behavior and show status drafts are never bundled in either website version.

### Execution evidence: REG-004

Nine existing website deploy tests and five website build tests pass. Added build/rollback regression proves neither app.json nor service-status drafts can enter website artifacts; no website runtime/allowlist changes.

### Execution evidence: IT-007

Implement fixed-path anonymous Worker, local fake-R2 HTTP/cache/CORS checks and opt-in hosted native/browser checks. Hosted provisioning and publication remain separate unauthorized operations.

### Execution evidence: IT-007

Five local Worker tests pass; Python/Worker recipe and actionlint pass; pinned Wrangler preview dry-run builds with only separate preview R2 binding; one hosted browser smoke is discoverable. No hosted resources provisioned or live publication executed. Hosted CORS/cache/next-publication evidence remains a release gap; mock passes do not satisfy it.

### Execution evidence: MAN-001

Not executed: real VoiceOver/TalkBack/web screen-reader and Android predictive/browser back interaction require available supported devices/readers and separate manual evidence. Widget semantics/layout/focus are passing AT-009 evidence, not a screen-reader claim. Release gate remains open.


### Final verification progress

- Full Flutter unit/widget suite: **2,823 passed, 38 existing skips**, `just app-test --no-pub --reporter expanded`. Full-suite snapshot preceded the additional one-owner provider/cancellation tests; the final focused suite reruns those changed boundaries.
- Chrome Fetch adapter: **1 passed** after formatting/interop cleanup.
- Python operator/contract: **12 passed**; Worker: **5 passed**. Existing website deploy/build regressions: **9 + 5 passed**.
- Worker preview dry-run builds with only its separate preview R2 binding. Hosted Playwright check lists **1 test** but is not executed against Cloudflare.
- `actionlint .github/workflows/backend-ci.yml` and `git diff --check` pass. Final analysis and all four simulator critical journeys passed (see final outcome).
- Additional IT-006 red/green: a provider construction failure was captured twice (Riverpod owner plus controller). Separate dependency resolution now leaves provider failures with their observer; **4 diagnostics/dismissal tests passed**. Throwing cancellation/reporter cannot strand timeout, disposal or retries. Argparse invalid-argument output originally echoed a synthetic credential canary; static bounded parser errors fix that, covered in the 12 Python tests.


### Final outcome (2026-10-09)

- `just app-test --no-pub --reporter expanded`: **2,823 passed, 38 existing skips**. Final changed-boundary rerun `just app-test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart`: **51 passed**.
- Full loopback critical journeys on iPhone 18 Pro / iOS 27 simulator: **4 passed** (real preferences reload/fresh controller, account switch, accepted HTTP composer write under maintenance, notification navigation). No physical device, production backend or PDS used.
- Final Chrome adapter suite: **2 passed**. Additional IT-006 red showed a body-stream network rejection was unclassified; its promise boundary now preserves the original cause/stack in a bounded expected-network wrapper.
- Additional UT-004 red showed a forward wall jump followed by correction could revive an already expired grant. Expiry is now irreversible until a successful validated fetch; all **51** final focused tests pass. The expiry timer also explicitly revokes its grant.
- Python **12** and Node Worker **5** pass; Python rejects nonstandard NaN literals rather than accepting a wire JSON document Dart rejects. No new runtime dependency or AppView API/lexicon/backend change.
- `just web-check`: website build/tooling tests **9 + 5**, and consent browser regressions **10 passed**. Landing runtime/public allowlist remain unchanged.
- `flutter analyze --no-pub`, `actionlint`, `git diff --check`: passing after final browser comment cleanup.
- Worker preview `wrangler deploy --dry-run`: successful; fixed preview R2 binding only. Hosted smoke is discoverable (**1 test**), not executed against hosted resources.
- Runbook: `docs/operations/service-status.md`; host/module overview: `service-status/README.md`. Local recipes and path-aware PR checks validate but never publish.
- Remaining release gaps: separately authorized Cloudflare account/bucket/Worker/DNS setup and live preview publication/cache/browser verification; real VoiceOver/TalkBack/web reader and Android predictive/browser back checks; existing external OAuth/media harness limits. These are not claimed as complete by local fake-R2 or widget passes.
- Source diff reviewed against requirement/test mappings in `04-coding-plan.md`. No AppView, lexicon, Render, website runtime, auth/permission policy, production credentials/data, commits or pushes changed.


## Review simplification progress (2026-10-09)

User approved the concrete simplification proposed in review; no remote publication or deployment is authorized. Existing green behavior tests form the refactor baseline; no artificial red for unchanged behavior.

| Work | Requirements / tests | Status |
|---|---|---|
| Generated serialization plus semantic validation | FR-002/FR-003, UT-001/UT-002/UT-007 | completed |
| Single retrieval boundary and smaller native GET | FR-001/FR-004, RULE-002/RULE-003, IT-001 | completed |
| Direct publisher/verification and consolidated tests | FR-012/NFR-002, AT-008/IT-003/IT-004 | completed |
| Manual hosted check instructions replace browser harness | FR-001/NFR-002, IT-007 | completed |
| Focused regressions, analysis and final notes | all affected requirements | completed |

Generated serialization: red showed missing toMap/toJson; generator plus strict semantic validation passes all 7 model tests. Existing contract corpus caught mapper string coercion, fixed by keeping raw string-type validation. Retrieval refactor now uses Dio's existing response-stream cancellation rather than a second subscription/completer implementation.

### Simplification outcome

- Generated status DTO/enum serialization uses existing dart_mappable. Semantic validation remains explicit; generated stringify/equality/copy methods are disabled. A round-trip test proves supported fields serialize and unknown controls do not.
- Removed the forwarding repository/transport layer and consolidated conditional exports and failure helpers into one repository boundary. Native GET relies on Dio's built-in token/response-stream cancellation instead of a custom subscription, body completer and cancellation exception. Web Fetch keeps anonymous/redirect/cache options with one network-failure boundary and simpler cleanup.
- Consolidated the streaming-body helper test into the real anonymous HTTP boundary test (oversized response alongside invalid content type and redirects). Kept controller, retained UI/editor, session, dismissal and diagnostic outcomes unchanged.
- Publisher uses the configured target directly, without per-message account/bucket discovery or temporary Wrangler configurations. One object upload is followed by one direct public GET and complete recognized-document comparison; removed digest/child-process verification and retry machinery. Validation, fresh revisions, explicit confirmation, sanitized failures and truthful unverified outcomes remain covered.
- Removed scripts/service-status-check and the service-status Playwright configuration/page/spec. Hosted warmed-client CORS/cache/site-independence checks remain explicit manual release checks, not claims established by local tests. Initial account/bucket checks remain in the setup runbook.
- Replaced the 326-line superseded coding plan with the current 82-line design and original stable test order. Updated acceptance-test automation classification and operator runbook.
- Verification: 74 Flutter status/affected-app tests passed; 2 Chrome Fetch tests passed; 11 Python contract/operator tests and 5 Worker tests passed; flutter analyze reports no issues; git diff --check passed. Logs: /tmp/craftsky-status-regressions-final.log, /tmp/craftsky-status-web-simplified.log, /tmp/craftsky-status-operator-final.log, /tmp/craftsky-status-analysis-final.log.
- The filtered generator initially removed unrelated generated files. Full generation restored them; unrelated mapper-format drift was restored to the original tracked bytes. Only the new status mapper is included; no dependency or unrelated generated-output change remains.
- Prior full-suite/simulator results above are historical implementation evidence. They were not rerun for this refactor; affected behavior was verified with the focused regressions above. No hosted verification, deployment, publication, commit or push performed.

### Local development tooling (user-requested)

- Added `scripts/service-status-dev` and just commands: service-status-dev-setup, service-status-dev, service-status-dev-publish [file], service-status-dev-clear, app-run-status [Flutter arguments].
- Setup installs existing pinned web tools if absent, preserves an existing editable draft and loads simulated R2. Serving preserves current object state; publish/clear use only explicit local object operations and fresh revisions. State/draft are ignored under service-status/.wrangler/dev. Port defaults to 8789 with CRAFTSKY_STATUS_DEV_PORT override.
- App launch uses the existing app-run recipe with a per-run status define, plus status-port ADB reverse mappings for connected Android devices. No app env files, production objects or Cloudflare resources are altered.
- Verified with actual Wrangler 4.148.0 local R2 and Worker on port 48789: initial maintenance, two announcement updates with distinct fresh revisions, unchanged source draft, invalid JSON rejected without replacing the live local object, clear to normal, and JSON/no-store/CORS responses. Temporary smoke: /tmp/craftsky-status-dev-smoke.py. App argv/port override checked without launching Flutter; just parsing and quoted draft-file dry runs passed.
- Existing 11 Python and 5 Worker tests pass; git diff --check passes. Local smoke server stopped cleanly. Actual interactive Flutter launch and hosted Cloudflare checks were not performed during this tooling change.

### Announcement modal revision (user-requested)

- User requested a CraftSky-themed modal and confirmed an existing dialog should be reused. Replaced AnnouncementBanner with a thin AnnouncementModal content wrapper around the existing CraftskyDialog and ChunkyButton; shared dialog/theme implementation is unchanged.
- Render the standard dim barrier over the retained app rather than resizing the navigator/feed. Focus and semantics remain inside the visible modal. Button, outside tap, Escape and root back persist the dismissed revision; back does not pop the underlying route. Normal/maintenance replacement removes the modal through existing status state. Failed persistence keeps the modal available for retry.
- Updated FR-006/AC-004 and AT-003 presentation wording to reflect the explicit user change from a non-blocking banner to a dismissible modal. Polling, validation, dismissal persistence and maintenance behavior remain unchanged.
- Focused status/diagnostic suite: 51 passed, including existing-dialog reuse, unchanged underlying control position, blocked background interaction only while visible, all dismissal paths, retained back-stack, status replacement, long scaled text in light/dark narrow/wide layouts, hidden background semantics and editor preservation. Log: /tmp/craftsky-announcement-tests.log.
- flutter analyze reports no issues; git diff --check passes. Real simulator visual/reader checks were not performed in this change; no remote publication, deployment or commit performed.

### Review: notifier access inside build

The host already watches serviceStatusControllerProvider state, so its derived getter reads execute on each state-triggered rebuild; they were not one-time reads. Consolidated notifier access into one explicit watch for controller identity and retained the state watch for status updates. Event callbacks continue to read the notifier when invoked. These values cannot be cached in initState because polling, expiry and dismissal change them. Controller/host behavior suite: 11 passed (/tmp/craftsky-status-build-watch-tests.log).
