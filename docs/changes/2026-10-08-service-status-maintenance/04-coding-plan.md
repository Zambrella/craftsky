# Coding Plan: Service Status and Maintenance Messaging

## Inputs and approval

Requirements: `01-requirements.md`; acceptance tests: `02-acceptance-tests.md`; document review: `03-document-review.md` (Approved with notes). Execution evidence is in `05-implementation-plan.md`.

On 2026-10-09 the user approved simplifying the implemented status retrieval and publishing tooling. This revision replaces the original layered transport/repository design, repeated account discovery, subprocess-based verifier and hosted browser harness. Product requirements remain unchanged. Local refactoring is authorized; provisioning, DNS, publication and deployment remain separate actions.

## App implementation

- `ServiceStatusDocument` and its mode enum use the existing `dart_mappable` generator. Generate only decoding/encoding, keeping a safe custom `toString`. Validate schema version, strict wire types, revision, required notice text, size and supplied estimate before accepting a document. Generated mapping alone does not enforce these rules (FR-002/FR-003, RULE-003).
- A single `ServiceStatusRepository` boundary exposes a cancellable request. The native implementation uses an independent Dio GET with no API interceptors, credentials or account dependency. Dio handles response cancellation; a shared small body reader enforces the 16 KiB limit. There is no forwarding repository/transport layer (FR-001, RULE-002/RULE-003).
- The browser implementation uses Fetch with credentials omitted, redirects rejected, no-store and no referrer. It retains the body limit and AbortController because the browser adapter cannot rely on native redirect behavior. Native and web implementations are conditionally exported; no new dependency is required.
- The controller retains launch/resume/manual refresh, coalescing, 60-second foreground polling and the three-second overall deadline. Only successful validated reads renew the five-minute maintenance grant. Late results and expired grants cannot reactivate maintenance (FR-004/FR-008).
- Announcement dismissal persists only the revision. The existing maintenance cover preserves the navigator, editors, focus, sessions and current account/policy gates; accepted writes complete normally without replay (FR-005–FR-009, RULE-001/RULE-004).
- Use the existing diagnostic entry points with bounded classifications and original available causes/stacks. Remote prose, documents and protected values must not enter diagnostics. Expected polling failures do not create repeated issues (NFR-004).

## Hosting and publication

A small read-only Cloudflare Worker serves the private R2 `app.json` object at `https://status.craftsky.social/app.json`. Preview has its own Worker, bucket and domain. Read the binding directly on every request, with no Cache API. Serve JSON, no-store, anonymous CORS and bounded errors. Keep mutable messages outside Worker and landing-site deployments (BR-002, FR-001, NFR-002).

The operator uses:

```sh
scripts/service-status validate /path/to/incident.json
scripts/service-status publish /path/to/incident.json --environment preview
scripts/service-status clear --environment preview
```

`publish`/`clear` select the configured account and fixed environment target, validate the document, assign a new UUID revision, show the public target/content and require confirmation. Then run one pinned Wrangler object upload and make one direct anonymous public GET. Compare the complete recognized document, including revision, and check JSON/no-store/CORS headers. Only verified success exits zero. Invalid/cancelled requests do not upload; failure after an upload attempt reports unverified state without rollback or write retry (FR-006/FR-012).

Account membership and bucket verification happen during initial authorized setup, rather than every message update. Keep credentials in the operator environment or Wrangler login. Use subprocess argument arrays, a temporary payload and suppressed raw dependency output. The direct verifier uses a three-second socket timeout rather than a child process enforcing an overall deadline; the app retains its overall three-second deadline.

## Verification

Keep tests at observable boundaries: document contract and serialization, anonymous HTTP/redirect/body handling, controller polling/expiry/dismissal, retained UI/account behavior and diagnostics, publisher validation/confirmation/upload/verification, and Worker routing/cache/CORS. Consolidate helper-specific checks into boundary tests. Reuse the literal contract corpus across Dart and Python.

Commands:

```sh
just service-status-test
just app-test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart
just app-test --no-pub --platform chrome test/service_status/service_status_repository_web.browser.dart
just app-analyze
```

Real hosted CORS/cache checks are manual release checks in `docs/operations/service-status.md`. After an authorized preview publication, use the same warmed native/browser clients to observe each next published revision during a qualifying refresh. Verify a disposable landing-site deploy/rollback leaves the object unchanged. Local fake-R2 tests cannot establish hosted delivery. Real reader/platform-back checks and existing external OAuth/media harness limits remain release gaps. Do not provision or publish as a side effect of tests.

## Test order and traceability

The original IDs and order remain the execution reference. Refactors start from existing green tests; generated serialization receives a new focused round-trip assertion. Historical red/green evidence remains in the implementation journal.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `app/test/service_status/service_status_document_test.dart` | Minimal literal schema 1 maintenance, valid token revision, no publication time; shared corpus | Missing DTO/decoder or unsupported valid contract |
| 2 | UT-002 | Same Dart suite and `scripts/test_service_status_publish.py` validator group | Literal 16,384/16,385-byte bodies, 160/161 title and 2,400/2,401 message scalar boundaries, impossible zoned timestamps, streaming oversize | Unsafe/mismatched acceptance or unbounded decode |
| 3 | UT-003 | `app/test/service_status/service_status_controller_test.dart`; startup wiring in `app_test.dart` | Fake clock/repository and delayed dependency init; all modes and lifecycle triggers | No polling/coalescing/deadline or blocked startup |
| 4 | UT-004 | Same controller suite | Exact freshness boundaries/renewal, suspension/resume, wall rollback and cancelled late result | Indefinite cover or superseded acceptance |
| 5 | IT-001 | `app/test/service_status/service_status_http_test.dart`; mobile socket smoke in `critical_journeys_test.dart` | Separate status/AppView servers, captured synthetic auth/device values, redirects; web adapter mocked fetch + hosted checks | API headers reused, redirects followed or endpoint depends on account |
| 6 | AT-001 | `app/test/service_status/service_status_app_test.dart` | Signed-in/out and failed/pending dependencies/account; splash callback recorder | No custom cover before initialization or splash remains |
| 7 | AT-002 | `app/test/service_status/service_status_widgets_test.dart` | Plain/malicious prose, estimates, retry recorder and internal back event | Wrong content/actions, hidden screen interaction or missing retry |
| 8 | AT-005 | Same app suite and existing error fixtures | Status unavailable, API 503, HTML/non-200, announcement/health probe | Global maintenance inferred or usable UI blocked |
| 9 | UT-005 | `app/test/service_status/announcement_dismissal_test.dart` | Literal revisions A/B, delayed dismissal completion with newer status | Same revision resurfaces or new revision suppressed |
| 10 | IT-002 | `app/test/service_status/announcement_persistence_test.dart`; `critical_journeys_test.dart` | Mocked adapter for fast tests; real preferences write/reload, fresh instances for device evidence | Dismissal lost, plugin-only cache masks missing storage, or maintenance reused |
| 11 | AT-003 | `service_status_app_test.dart` | Current announcement modal, dismissal restores retained navigation/operations, normal/maintenance replacement | Dismissal fails to restore app or simultaneous modes visible |
| 12 | AT-007 | `app/test/service_status/service_status_restart_test.dart` | Shared only preference storage across recreated feature/app instances; offline restart | Same-revision flash, missing new fetch or mandatory restored cache |
| 13 | AT-004 | `app/test/service_status/service_status_editor_test.dart` | Existing post/project editor fixtures with text/media and delayed operation | Editor unmounted, navigation lost or writes replayed |
| 14 | AT-006 | `app/test/service_status/service_status_recovery_test.dart` | Inject normal/announcement/expiry and current-account gates | Relaunch required, wrong-account UI or policy bypass |
| 15 | REG-001 | Existing sign-out interceptor/error mapper tests | Genuine 401 control, 503/offline/status outcomes | Session invalidated by status failure |
| 16 | REG-002 | Existing account-switch/OAuth handoff/restricted routing/gate suites | Delayed old-account completion and handoff beneath cover | Current-account fencing or eligibility bypassed |
| 17 | REG-003 | Existing composer/video/draft suites and mobile critical journey | Original operation counts/outcomes with maintenance transition | Existing completion cleanup changes or extra submission |
| 18 | UT-006 | `app/test/service_status/service_status_estimate_test.dart` | Explicit offset instants/DST zone test context, absent/past estimate | Incorrect local display or automatic mode transition |
| 19 | UT-007 | Document/widgets suites | Markup, URL prose and unsupported control fields | Code/link/control behavior enabled |
| 20 | AT-009 | `app/test/service_status/service_status_accessibility_test.dart` | Existing English locale/device fallback, light/dark, supported small/web sizes and maximum text scaling | Overflow, inaccessible retry/dismissal, exposed covered semantics |
| 21 | IT-006 | `app/test/observability/service_status_diagnostics_test.dart`; Python/Node output checks | Serialized transport, debug ProviderLogger, original cause/stack and prose/secret canaries | Leak, missing positive evidence, duplicate issues or reporter affects behavior |
| 22 | AT-008 | `scripts/test_service_status_publish.py` command workflow | Injectable R2/public verifier/confirmation; Render unavailable; no Git/site release | Independent validate/revise/clear workflow unavailable |
| 23 | IT-003 | Same Python suite | Invalid/declined/EOF inputs, configured target, temporary payload; reused draft revision | Mutation before confirmation; unchanged revision used for new announcement; credentials exposed |
| 24 | IT-004 | Same Python suite | Exact/stale/unreachable/wrong-type public responses and concurrent newer revision | False verified result, unlimited wait, write replay or rollback |
| 25 | IT-005 | Python command/website suites and Node Worker tests | Separate fake stores/resources and two website code versions | Status overwritten by unrelated release or Worker deploy mutates object |
| 26 | REG-004 | Existing `test_web_deploy.py`, `test_cloudflare_pages_build.py` | Status resource separation configuration; existing website allowlist/promotion fixtures | Status bundled or website release behavior broken |
| 27 | IT-007 | `service-status/test/worker.test.js`, manual checks in the operator runbook | Local fake R2 for handler assertions, then separately authorized preview R2/Worker and real browser/native reads | Wrong CORS/cache/type or stale next refresh; mock pass cannot satisfy hosted evidence |
| 28 | MAN-001 | Manual recorded evidence | Supported Android/iOS/web screen readers, long text/large scaling, underlying editor | Reading/focus/accessibility fails despite widget semantics |

## Requested local development tooling

User requested reusable scripts and just commands after the simplification. `scripts/service-status-dev` reuses document validation and pinned Wrangler, using only simulated R2 with explicit `--local` and the preview bucket. Setup seeds an editable ignored draft; serve preserves local state; publish assigns fresh revisions; clear generates normal. `app-run-status` passes the loopback status URL for a debug run and adds Android status-port reverse mappings through the existing app launch flow. Configuration files remain untouched. See the operator runbook for commands and the configurable local port.

## Announcement presentation revision

User requested the existing CraftskyDialog presentation instead of the banner. Render it above the retained app with the standard dim barrier and ChunkyButton. Persist explicit button, barrier, Escape and system-back dismissal through the existing revision store. Trap focus and hide underlying semantics while visible, then restore retained focus. A normal/maintenance replacement removes the modal without touching routes or drafts.
