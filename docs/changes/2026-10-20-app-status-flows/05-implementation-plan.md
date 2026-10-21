# TDD Implementation Plan: App Status Flows

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Review: `03-document-review.md` (Approved with notes)
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Implement only linked requirements; one test ID at a time, red before implementation.
- Run the smallest relevant test, refactor only while green, record actual evidence.
- Preserve auth/account/policy gates and anonymous transport boundaries.
- No commits, hosted publication, deployment or infrastructure changes authorized.

## Test Order
The coding-plan groups are expanded into individual test-ID loops where new behavior is needed. Existing protective regressions need no artificial red.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-007 | FR-010, FR-012, RULE-002 | AC-014, AC-016 | Schema 2 rejected |
| 2 | UT-009 | FR-014 | AC-023 | No schema gate |
| 3 | UT-001 | RULE-002 | AC-003, AC-010 | No strict version value |
| 4 | UT-002, AT-002 | BR-002, FR-002, FR-007 | AC-003, AC-004, AC-010 | No eligibility |
| 5 | UT-003 | BR-001, RULE-001 | AC-001, AC-002 | Exclusive modes |
| 6 | UT-004/005/008, AT-001/010/011 | FR-001, FR-006, RULE-003 | AC-005, AC-009, AC-018, AC-021 | No update trust |
| 7 | UT-006, AT-006/007, IT-002 | FR-008, FR-009 | AC-012, AC-013 | Root revision identity |
| 8 | AT-003, REG-001/002/003 | FR-003, NFR-001 | AC-004, AC-006 | No update cover |
| 9 | AT-004/005, IT-004 | FR-004, FR-005 | AC-007, AC-008 | No actions |
| 10 | AT-008, IT-002/004 | FR-008, FR-010 | AC-014, AC-022 | No CTA |
| 11 | AT-009, IT-006 | FR-011, NFR-002/003 | AC-015, AC-019/020 | No public image flow |
| 12 | IT-003, REG-004 | FR-013/014 | AC-017, AC-023 | Mode-based tooling |
| 13 | IT-001/002/006, REG-001–004 | FR-001/006/008, NFR-001/003 | AC-005/009/012/020 | Existing regressions |
| 14 | IT-005 | FR-005, NFR-001 | AC-008 | No compiled reload harness |

## Implementation Steps
Executed loops and verification evidence are recorded below. Existing protective regressions were migrated without manufacturing artificial red failures.

## Completion Checklist
- [x] All Must requirements covered by tests or explicit release gaps
- [x] All selected automated Must tests passing; final artifact rerun also passed
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Workflow document review approved; separate implementation review awaits user choice

## Release Evidence Boundaries
MAN-001 official stores, MAN-002 hosted cache recovery and MAN-003 physical back/readers require separate platform/release evidence. Mock storage is not durable device proof. No live changes are part of this stage.

### UT-007 / FR-010, FR-012 / AC-014, AC-016
- Red: Python `unittest ... -k schema_two` and Flutter document test `--name schema.two.accepts` both rejected the literal all-controls fixture with `Status schema`.
- Green: both focused commands passed after independent mapped sections and atomic field validation.
- Generated mappers with `dart run build_runner build`; restored unrelated generated churn from the initially clean checkout. No unrelated refactor.
- Initial Flutter invocation needed SDK-cache write access; corrected a just recipe argument quoting issue before obtaining the meaningful red.

### UT-009 / FR-014 / AC-023
- Red: isolated pinned jsonschema environment loaded correctly; test failed because the formal schema did not exist.
- Green: `/private/tmp/craftsky-status-venv/bin/python -m unittest discover -s scripts -p test_service_status_publish.py -k offline_formal` passed.
- Added Draft 2020-12 schema, local-reference-only validation with non-retrieving registry, safe fail-closed dependency/schema errors. Final clear payload and generated root revision are validated before any store writes.
- Explicit dependency setup is isolated in `/private/tmp`; ordinary system Python lacks jsonschema. No validation command installs dependencies.

### UT-001 / RULE-002 / AC-003, AC-010, AC-016
- Red: Dart test could not resolve the new version value; Python test lacked `compare_versions`.
- Green: focused Dart version test and Python `-k precedence` passed.
- Decimal-string core/prerelease comparison, strict grammar, metadata-free precedence; range validation rejects reversed boundaries. No integer conversion of unbounded components.

### UT-002 / FR-002, FR-007, RULE-003 / AC-003, AC-010, AC-018
- Red: focused eligibility test lacked platform/runtime selectors.
- Added independent package-info FutureProvider, supported-platform provider and pure version eligibility predicates. Build metadata and inclusive boundaries covered; null version fails open. Green: both focused eligibility/version tests passed.

### UT-003 / BR-001, RULE-001 / AC-001, AC-002
- Red: missing shared presentation selector. Green: `status_flows_controller_test.dart` priority scenario passed after adding maintenance → update → eligible message → none selection.
- Runtime changes notify the existing Notifier without rebuilding its request/timer lifetime.

### UT-008 / RULE-001, RULE-003, FR-006 / AC-018, AC-021
- Red: after failure and maintenance expiry, expected message but got stale update gate.
- Green: controller suite passed after finish-path trust invalidation, valid-snapshot restoration and unchanged epoch/freshness rules. Timeout/late success and valid recovery covered.

### UT-006 / FR-008, FR-009 / AC-012, AC-013
- Red: replacement B inherited A's pending dismissal flag.
- Green: controller suite passed after per-revision pending state and serialized persistence. A→B→reused A sequence explicitly permits resurfacing under the approved bounded durable marker.
- Schema-2 persistence uses its new preferences key; legacy key remains untouched. Existing maintenance/message regression fixtures now build independent schema-2 controls with explicit test runtime overrides.

### AT-003 / FR-003, NFR-001 / AC-004, AC-006
- Red: ready-host router test found no update cover. Green after full-page cover/shared blocking selector; route changes, Escape, back and retained editor text covered.
- Additional meaningful red: actual pending/error App hosts returned false to system back under the update cover. Added a route-local common PopScope guard in initialization hosts; all 11 app-host scenarios pass for both blockers and signed-in/out variants.
- Existing editor/recovery/widget regressions passed apart from an initial new-screen semantics assertion, corrected with explicit route child semantics before the focused green run.

### AT-004 / FR-004 / AC-007
- Red: missing external action adapter/Update CTA. Green: iOS and Android official-listing, pending duplicate-tap, false-result feedback/retry and retained-gate tests passed.
- Corrected a duplicate platform override in test setup. No claim of physical store handoff.

### AT-005 / FR-005 / AC-008
- Red: missing reload adapter. Green: explicit user-tap test passed with exactly one injected reload and no refresh-triggered reload.
- Production conditional web adapter calls `window.location.reload()` without changing URL; compiled external browser proof was completed below.

### AT-008 / FR-008, FR-010 / AC-014, AC-022
- Red: missing activation method, then missing CTA host boundary. Green: controller race/session-fallback and external/named CTA widget tests passed.
- Destination invocation starts before persistence; failures retain feedback, success suppresses captured revision before serialized persistence. Failed saves retain only session suppression; old completions cannot overwrite replacement action state or lower blockers.

### IT-004 / FR-010 / AC-014
- Red: no named-destination mapper. Green: all six names map to canonical typed routes and retain actual signed-out GoRouter redirects. Ready App alone supplies routing; loading/error hosts report unavailable actions through ordinary CTA failure feedback.

### AT-009, IT-006 / FR-011, NFR-002/003 / AC-015, AC-019/020
- Red: missing optional image adapter/widget and native transport. Green: pending image leaves text/actions accessible and cancels on cover; real native loopback verifies no credential/cookie/referrer/account/device headers, no redirect following, non-image rejection and >5 MiB streaming rejection.
- Native adapter has a ten-second deadline. Decorative decode requests fit within 1120×1120 and a bounded display area; failures remain optional.

### Contract corpus expansion / UT-007, UT-009
- Shared literal schema-2 corpus covers all-control/clear/minimal snapshots, all six destinations, metadata-equal ranges, invalid siblings/nulls/types, strict versions, malformed/credential-bearing URLs, calendars and scalar/byte boundaries.
- Additional red: Dart normalized away empty URL userinfo and accepted it. Fixed raw-authority rejection; Dart/Python corpus green.
- Additional red: a broken reference in an unused schema branch, then an empty schema file, allowed a clear write. Fixed full local-pointer resolution and expected schema identity before any payload validation/upload. Both zero-write tests green.
- Explicit URI/date-time checker dependencies are pinned and required; no optional checker fallback. Missing/broken validator tests pass without dependency prose in output. Formal structural rejection and semantic reversed-range rejection are independently asserted.

### Operator integration / IT-003, REG-004
- Red: local schema-2 publish uploaded but failed when its summary read the obsolete `mode` field. Green after independent-control summaries and schema-2 full clear.
- CLI/local wrappers and deterministic recipe use one configured offline-validator interpreter; setup is explicit. Python implementation moved into `scripts/service_status_dev.py` behind the retained CLI name.
- 17 Python tests and five Node tests pass, including all-controls → omit-maintenance while preserving minima/message identity → full clear, unchanged input draft, cancellation/EOF, final-payload rejection, safe unverified outcomes and read-only Worker behavior.
- Actual Wrangler 4.148.0 local Worker/R2 smoke passed with disposable `/private/tmp` state: all controls, maintenance removal, unchanged message/minima, full clear, no-store/CORS/nosniff headers. Every object command used `--local`; no remote upload or resource creation. Local Worker stopped afterward.

### Media/accessibility/diagnostic completion / AT-009, IT-006
- 24 layout/focus/media scenarios passed: both blocker types and message, 320/1280 widths, themes, enlarged text, absent/pending/valid/broken/extreme-ratio images, reachable actions and retained focus.
- Real native image transport proves anonymous requests, no cookie reuse, redirect/media/5 MiB rejection and an actual overall ten-second streaming deadline.
- Chrome RequestInit/status-body tests and a real separate-origin browser image CORS/byte/header probe passed (four browser transport tests total).
- Local and serialized Sentry sink tests preserve original causes/stacks and approved operation/classification while excluding message prose, revision, URL/token/cookie and dependency canaries. Throwing reporters do not change business outcomes.
- Additional meaningful red: broken image bytes produced an unexpected issue despite rendering a fallback. Codec-boundary content Exceptions now retain the original cause/stack as expected diagnostics; internal Errors still follow unexpected reporting. Broken-media and serialized sink tests pass.

### Runtime, routing and action completion / UT-008, AT-003/004/005/008, IT-004
- Delayed runtime-version resolution does not rebuild/cancel a status request. Synchronous start and repository-construction failures release a previously accepted update policy; focused controller suite passes.
- All six typed destinations tested through actual signed-out redirects plus actual parser/account-gate states: uninitialized, onboarding, policy-restricted and initialized. No protected pages are directly constructed by status actions.
- Actual post/project editor fixtures retain element/text/media state and one submitted operation under either blocker. Ready/loading/error hosts protect back and splash behavior for signed-in/out states.
- Update actions cover false, thrown, success and duplicate taps. Additional red: a second web tap requested another reload before navigation completed. The web action now retains its pending guard until document unload; focused tests pass.
- Additional red: store launch failure after valid policy removal read a disposed widget ref. Capture the controller/diagnostic owner before await; late completion is safely consumed without touching removed widget state. Focused update suite passes.

### Real storage / IT-002, AT-006/007
- iPhone 18 Pro simulator, iOS 27: real preferences write/reload, fresh adapter/controller, unchanged-message suppression across root revision changes, replacement message eligibility, maintenance and anonymous native transport journey passed.
- Initial simulator attempt could not start because the generated Flutter plugin Swift package was missing. `flutter pub get --offline` regenerated checkout-local tooling from locked cached packages; the retry passed. No native desktop target or Xcode dependency/project edit was introduced.
- Real browser preferences survive an actual document reload in the external runner: dismissed A stays hidden; B becomes eligible. This complements host mock tests rather than claiming mocks prove persistence.

### External compiled reload / IT-005, AT-005
- Debug artifacts A `1.8.0+1` and B `1.9.0+2` use production version/controller/host/actions and an independent loopback status origin. Compiled markers distinguish the actual JavaScript build from fetched version metadata.
- Harness setup fixes: use normal GoRouter path routing rather than a home-only Navigator; distinguish existing startup fragment normalization and same-document history updates from real document reloads. The adapter's first committed reload preserves the URL including fragment; no product routing policy was changed.
- All four external cases passed: A→B, still-old A gates again, unavailable status fails open, and actual browser dismissal persists across reload. Final exact-source artifact rebuild/recheck passed, as recorded in the verification table below.
- Reused the existing main checkout's exact pinned Playwright 1.63.0 and Wrangler 4.148.0 installations; no npm/browser package installation or upgrade. No hosted cache behavior claimed.

## Final Verification (Initial Implementation)

| Check | Result |
|---|---|
| `CRAFTSKY_STATUS_PYTHON=/private/tmp/craftsky-status-venv/bin/python just service-status-test` | Passed: 17 Python + 5 Node tests |
| `just app-test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart test/router/router_redirect_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart` | Passed: 126 tests |
| `just app-analyze` | Passed: no issues |
| Chrome status/image RequestInit + actual image CORS/bytes probe | Passed: 4 tests |
| `just app-test-integration IPHONE_SIMULATOR --no-pub --name status.dismissal` | Passed: real iOS simulator storage/controller journey |
| Local Wrangler Worker/R2 transitions and headers | Passed; disposable local state only, server stopped |
| Final A/B rebuild + dedicated Playwright reload/storage runner | Passed: both final-source builds and all 4 browser cases |
| `git diff --check` | Passed |

## Requirement Traceability

| Requirements | Executed verification |
|---|---|
| BR-001, RULE-001 | Atomic independent snapshot and maintenance→update→message priority/clear controller/host tests |
| BR-002, FR-002, RULE-002 | Strict SemVer metadata/prerelease/large decimal tests, platform minima, inclusive ranges and targeted host tests |
| FR-001, FR-006, RULE-003 | Existing timing/coalescing/epoch/expiry/restart regressions plus failed refresh, late runtime, start/construction failure and later policy recovery |
| FR-003, NFR-001 | Ready/loading/error covers, shared back guards, splash, typed route parsing and actual retained editor/media/submitted-write fixtures |
| FR-004, FR-005 | Official listing fakes, false/thrown/late launcher outcomes, pending guards, real compiled browser reload; MAN-001/002 remain release checks |
| FR-007, FR-008, FR-009 | Parser/eligibility, all dismissal paths, captured-revision serialization, CTA persistence fallback/races, actual simulator/browser preferences and bounded revision reuse |
| FR-010 | Strict HTTPS/closed destination contract, CTA UI outcomes and real router/account/policy gate matrix |
| FR-011, NFR-002 | Native/browser bounded media adapters, content fallback, cancellation, responsive/theme/scaled-text/focus/semantics matrix; MAN-003 remains a release check |
| FR-012, FR-014 | Shared corpus, formal schema/meta-schema, explicit formats/offline local references, semantics and fail-closed exact final-payload validation |
| FR-013 | Hosted fake-store and actual local Worker/R2 independent-edit/clear/identity/confirmation/verification workflows; updated examples/runbook/setup |
| NFR-003 | Anonymous native and actual browser transports, serialized cause/stack/positive-field and protected-canary tests, expected media classification and throwing diagnostics |

## Remaining Release Checks
- MAN-001 not run: physical iOS/Android listing availability and store handoff. Fake launcher tests and simulator storage do not establish this.
- MAN-002 not run: actual authorized-preview Flutter hosting/CDN/bootstrap/service-worker cache recovery. Public website deploy scripts do not identify that hosting path. The existing path strategy normalizes fragments at startup; the adapter itself reloads the current full URL.
- MAN-003 not run: physical VoiceOver/TalkBack/web-reader and predictive/platform back checks. Local semantics/focus/layout and simulator checks remain distinct evidence.
- Physical-device process-kill/restart and Android storage were not exercised; real iOS simulator reload/fresh-controller and real browser document-reload persistence passed.
- Implementation review is the next optional stage, not performed automatically. No commits, push, deployment, hosted publication or production mutation.


## Stage Completion (Initial Implementation)

Implementation complete on 2026-10-20. All selected automated loops and relevant regression gates pass; manual/release gaps above remain explicit. The final diff contains only status feature code, mapped/localized output, linked fixtures/tests, validator/operator tooling, the dedicated browser harness and operational/execution documentation. No app dependency/version, Xcode project, API, lexicon, backend, migration or production infrastructure change remains. Implementation review awaits the user's stage-exit choice. No stage commit was authorized or created.

## IR-001 Correction Pass

User selected the required correction stage after `06-implementation-review.md` reported Changes required. The review artifact remains the original review record; re-review is a separate stage.

| Order | Test ID / Requirements | Correction scope | Status |
|---|---|---|---|
| 1 | UT-006 / FR-008 / AC-012, AC-022 | Delayed A activation after B has been successfully dismissed; order durable updates by user intent | Passed focused red/green |
| 2 | AT-008, IT-002 / FR-008 / AC-012, AC-022 | Newer CTA dismissal, delayed storage, fresh controller/adapter, failed-write recovery and blocker safety | Passed |
| 3 | REG-002/003/004 | Relevant full Flutter regressions and static analysis, inspect final diff | Passed |

No adapter/storage-contract, route, privacy, diagnostic, hosted or dependency change is planned. Keep one durable marker and existing session-only suppression on unsuccessful persistence.

### Correction loop 1: UT-006 / FR-008 / AC-012, AC-022
- Red: `flutter test --no-pub test/service_status/status_flows_controller_test.dart --name late.CTA.success.preserves` failed because B reappeared after A completed. The initial just invocation split a spaced test filter into file arguments; a corrected direct invocation established the clean meaningful red.
- Green: the same focused command passed after assigning action-start order, checking against the latest successfully saved order inside the write queue, and applying the durable marker/state together after successful storage. Older skipped CTA persistence retains captured session suppression. No preference schema/adapter change.
- Supplemental same-behavior verification now covers pending storage and fresh preferences/controller reads; no unrelated refactor.

### Correction loop 1 supplemental coverage: AT-008, IT-002 / FR-008
- 20 focused tests passed across controller, CTA, dismissal and persistence suites. Eleven new regression cases cover the original race, eight explicit/CTA × immediate/pending storage × maintenance/update combinations, and two newer activation/storage failures.
- Fresh preferences adapters/controllers still read B; no history ledger or extra preference keys were added. Captured A stays hidden only in the existing session when its write is superseded. A is eligible after a fresh controller if B is the durable marker, retaining the approved bounded-history contract. Platform storage is mocked for these fresh-adapter tests; earlier actual iOS/browser evidence is not newly claimed for these cases.
- Failed newer activation does not dismiss B; failed newer storage retains B's session suppression without preventing A's successful storage. Both covers remain active when old activation/persistence completes.
- Refactor while green: allocate explicit dismissal order before its state notification, matching CTA order capture. Centralized successful durable-state application prevents completion callbacks from independently replacing the marker.

### Correction final verification

| Check | Result |
|---|---|
| Focused controller, dismissal, persistence and CTA suites | Passed: 20 tests |
| `flutter test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart test/router/router_redirect_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart` from `app/` | Passed: 137 tests; `/private/tmp/craftsky-status-ir001-tests-direct.txt` |
| `just app-analyze` | Passed: no issues; `/private/tmp/craftsky-status-ir001-analyze.txt` |
| `git diff --check` | Passed |

- The broader `just app-test` invocation stalled at its Rive library download before running tests. Stopped only that invocation's setup process and ran the equivalent Flutter test command directly, using the libraries that had already passed the focused suite. This was a tooling/network delay, not a hidden test failure.
- The first analysis run flagged an overlong new test comment; corrected it and reran successfully.
- Correction scope: only `service_status_controller.dart`, `status_flows_controller_test.dart`, `announcement_persistence_test.dart` and this execution document changed during this pass. Successful storage now updates its marker and state together, guarded by action-start order inside the serialized queue. No route, launcher, storage adapter/key, wire contract, media, diagnostic, dependency or generated-output changes.
- Python/Worker, browser build/reload and simulator integration checks were not repeated for this controller-only ordering correction; their earlier evidence remains recorded above. Fresh-adapter/controller ordering cases use mocked platform storage and do not replace physical durability evidence. MAN-001/002/003 and physical process-restart/Android-storage gaps remain unchanged.

### Correction stage completion

IR-001 addressed through UT-006 / AT-008 / IT-002 for FR-008, AC-012/022. The meaningful failing regression is now green, eleven ordering/failure/restart cases were added, and all 137 relevant Flutter tests plus analysis pass. Implementation correction is complete; `06-implementation-review.md` remains the original Changes required review until a separately selected re-review. No commit, push, publication or deployment.

## Trusted-host Image Simplification — D-014 (2026-10-21)

User explicitly requested `Image.network` for a trusted image host. This supersedes the custom image transport decisions in the original implementation/review; anonymous status JSON networking is unchanged. Updated requirements, acceptance test notes, coding plan and operator guidance accordingly.

- Replaced byte-provider rendering with plain `Image.network`, keeping the 200-pixel-high `BoxFit.contain` display, decorative semantics, safe one-time diagnostic reporting and empty error fallback. No app authorization/account/device headers are supplied. Flutter owns networking/cache behavior.
- Removed five custom provider/native/web/body/stub files, the two dedicated image transport test files, the image-only RequestInit case and its unused browser-server route. Removed custom credentials/referrer/redirect/size/deadline/cancellation guarantees for images. HTTPS URL validation remains. No host allowlist or new service was added; operators select a trusted public web-compatible host.
- Test-first evidence: the focused standard-network-provider test failed against the old byte loader (no Image existed while bytes were pending), then passed after the replacement. The retained image suite now exercises actual loopback NetworkImage loading, decoded normal/extreme aspect ratios, pending/broken/absent states, reachable controls and cover removal. Standard networking is tested without adding production injection APIs.
- Test fixture setup corrections: use a real test HTTP client, reset the painting debug hook and close clients before widget-test invariants, and pump asynchronous IO/decode to completion. Dropped decode resizing so very thin images preserve their native aspect ratio. These setup failures were resolved before the final green gate.
- Verification: 12 focused image tests passed; the complete relevant Flutter regression command passed **136 tests** (`/private/tmp/craftsky-status-network-image-tests.txt`); remaining status Chrome suite passed **2 tests** (`/private/tmp/craftsky-status-network-image-browser.txt`); analysis passed with no issues (`/private/tmp/craftsky-status-network-image-analyze.txt`); `git diff --check` and browser fixture server syntax passed. Three test-only analysis findings (unused import, braces, const) were corrected before the clean run.
- Full regression command: `flutter test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart test/router/router_redirect_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart` from `app/`. Browser command: `flutter test --no-pub --platform chrome test/service_status/service_status_repository_web.browser.dart`. Used direct Flutter commands to avoid repeating the earlier redundant Rive download.
- No dependencies, API/lexicon, credentials, deployment or publication changed. Earlier custom-image transport evidence is historical and no longer describes the selected image implementation. Physical store/reader/hosted-cache checks remain open as before; the actual chosen image host was not supplied or tested live.

D-014 implementation complete. The prior implementation approval predates this user amendment; this pass records implementation/test evidence, not a new independent review. No commit or deployment.

## SemVer Package Simplification — D-015 (2026-10-21)

User approved replacing custom Dart version parsing/comparison with the existing `pub_semver` dependency and a thin strict-syntax/build-omission wrapper, dropping arbitrarily large numeric components.

- `StatusAppVersion` now stores a `Version` and delegates comparison to `Version.compareTo`. A strict regex validates the complete input before stripping `+build`; Pub-specific build ordering and permissive leading-zero parsing therefore do not change status semantics. No version-constraint API is used.
- Core and numeric prerelease identifiers are capped at 15 digits (`999999999999999` maximum), below the web exact-integer boundary. This is enforced in the Dart wrapper, Python validator and checked-in JSON Schema. Build metadata remains ignored and has no numeric digit cap; nonnumeric prerelease identifiers retain SemVer lexical precedence. The Python range comparator is retained for operator validation.
- Red: native eligibility test accepted a 16-digit core component that the new contract rejects; shared Python corpus similarly accepted oversized core/numeric prerelease values. Green: both focused suites passed after the package wrapper and aligned numeric grammar.
- Added five shared fixtures for accepted/rejected numeric bounds and ignored oversized build metadata. Native/Chrome eligibility checks also compare adjacent 15-digit core and prerelease values, preserve build equality and reject leading zeroes, invalid grammar and oversized core/prerelease identifiers.
- Fixture-edit recovery: the initial JSON rewrite encountered an intentional unpaired-surrogate fixture and truncated the file before encoding. Restored the full corpus from the current workspace checkpoint, preserved its original literal bytes, appended the five new cases and asserted all 63 original cases were unchanged as JSON values. This setup error was corrected before the meaningful shared-corpus red/green run.
- Updated D-015 requirements/acceptance/coding-plan notes and operator limits. Earlier arbitrary-decimal comparison evidence is historical and superseded for this selected implementation. `pub_semver` was already a direct dependency; pubspec/lockfile were unchanged, with no install or upgrade.
- Verification: six focused native eligibility/document tests passed; all **17 Python + 5 Node** status tests passed; the full relevant Flutter regression command passed **136 tests** (`/private/tmp/craftsky-status-semver-tests.txt`); Chrome eligibility passed **2 tests** (`/private/tmp/craftsky-status-semver-browser.txt`). Static analysis initially flagged an overlong comment; shortened it and reran. Final analysis output is `/private/tmp/craftsky-status-semver-analyze.txt`.
- No application flow, status transport, action, persistence, deployment or publication change. The numeric contract limit is the deliberate D-015 change; unknown/invalid runtime versions retain existing fail-open behavior. Physical/manual release gaps remain as previously recorded.

D-015 implementation complete. Final static analysis reported no issues; `git diff --check` passed. No commit or deployment.
